package MFT

import (
	"bytes"
	"fmt"
	"strings"

	MFTAttributes "github.com/aarsakian/MFTExtractor/FS/NTFS/MFT/attributes"
	"github.com/aarsakian/MFTExtractor/img"

	"github.com/aarsakian/MFTExtractor/utils"
)

var RecordSize = 1024

var IndexEntryFlags = map[string]string{
	"00000001": "Child Node exists",
	"00000002": "Last Entry in list",
}

var SIFlags = map[uint32]string{
	1: "Read Only", 2: "Hidden", 4: "System", 32: "Archive", 64: "Device", 128: "Normal",
	256: "Temporary", 512: "Sparse", 1024: "Reparse Point", 2048: "Compressed",
	4096: "Offline",
	8192: "Not Indexed", 16384: "Encrypted",
}

var MFTflags = map[uint16]string{
	0: "File Unallocted", 1: "File Allocated", 2: "Folder Unalloc", 3: "Folder Allocated",
}

// $MFT table points either to its file path or the buffer containing $MFT
type MFTTable struct {
	Records []Record

	Size int
}

type Records []Record

type Attribute interface {
	FindType() string
	SetHeader(header *MFTAttributes.AttributeHeader)
	GetHeader() MFTAttributes.AttributeHeader
	IsNoNResident() bool
	ShowInfo()
	Parse([]byte)
}

// when attributes span over a record entry
type LinkedRecordInfo struct {
	Entry    uint32
	StartVCN uint64
}

// MFT Record
type Record struct {
	Signature          string //0-3
	UpdateSeqArrOffset uint16 //4-5      offset values are relative to the start of the entry.
	UpdateSeqArrSize   uint16 //6-7
	Lsn                uint64 //8-15       logical File Sequence Number
	Seq                uint16 //16-17   is incremented when the entry is either allocated or unallocated, determined by the OS.
	Linkcount          uint16 //18-19        how many directories have entries for this MFTentry
	AttrOff            uint16 //20-21       //first attr location
	Flags              uint16 //22-23  //tells whether entry is used or not
	Size               uint32 //24-27
	AllocSize          uint32 //28-31
	BaseRef            uint64 //32-39
	NextAttrID         uint16 //40-41 e.g. if it is 6 then there are attributes with 1 to 5
	F1                 uint16 //42-43
	Entry              uint32 //44-48                  ??
	Fncnt              bool
	Attributes         []Attribute
	Bitmap             bool
	LinkedRecordsInfo  []LinkedRecordInfo
	LinkedRecord       *Record // when
	I30Size            uint64
	// fixupArray add the        UpdateSeqArrOffset to find is location

}
type IndexAttributes interface {
	GetIndexEntriesSortedByMFTEntryID() MFTAttributes.IndexEntries
}

func (mfttable *MFTTable) DetermineClusterOffsetLength() {
	firstRecord := mfttable.Records[0]

	mfttable.Size = int(firstRecord.GetTotalRunlistSize("DATA"))

}
func (mfttable *MFTTable) ProcessRecords(data []byte) {

	records := make([]Record, len(data)/RecordSize)

	var record Record
	for i := 0; i < len(data); i += RecordSize {
		//fmt.Println("index ", i)
		if utils.Hexify(data[i:i+4]) == "00000000" { //zero area skip
			continue
		}
		fmt.Printf("Processing $MFT entry %d  out of %d records  \n", record.Entry+1, len(records))
		record.Process(data[i : i+RecordSize])
		records[i/RecordSize] = record
	}
	mfttable.Records = records
}

func (mfttable *MFTTable) ProcessNonResidentRecords(hD img.DiskReader, partitionOffsetB int64, clusterSizeB int) {

	for idx := range mfttable.Records {
		fmt.Printf("Processing NoN resident attributes, record %d of out %d\n", idx+1, len(mfttable.Records))
		mfttable.Records[idx].ProcessNoNResidentAttributes(hD, partitionOffsetB, clusterSizeB)
	}
}

func (mfttable *MFTTable) CreateLinkedRecords() {
	for idx := range mfttable.Records {
		previdx := idx
		for _, linkedRecordInfo := range mfttable.Records[idx].LinkedRecordsInfo {
			entryId := linkedRecordInfo.Entry
			mfttable.Records[previdx].LinkedRecord = &mfttable.Records[entryId]
			previdx = int(entryId)

		}
	}
}

func (mfttable *MFTTable) CalculateFileSizes() {
	for idx := range mfttable.Records {
		//process only I30 records

		if mfttable.Records[idx].HasAttr("Index Root") {
			mfttable.SetI30Size(idx, "Index Root")
		}
		if mfttable.Records[idx].HasAttr("Index Allocation") {
			mfttable.SetI30Size(idx, "Index Allocation")
		}

	}
}

func (mfttable *MFTTable) SetI30Size(recordId int, attrType string) {

	attr := mfttable.Records[recordId].FindAttribute(attrType).(IndexAttributes)

	idxEntries := attr.GetIndexEntriesSortedByMFTEntryID()

	for _, entry := range idxEntries {
		if entry.Fnattr == nil {
			continue
		}
		if entry.ParRef > uint64(len(mfttable.Records)) {
			fmt.Printf("Record %d has FileAttribute in its  %s which references non existent $MFT record entry %d\n",
				recordId, attrType, entry.Fnattr.ParRef)
			continue
		}

		mfttable.Records[entry.ParRef].I30Size = entry.Fnattr.RealFsize

	}

}

func (record *Record) ProcessNoNResidentAttributes(hD img.DiskReader, partitionOffsetB int64, clusterSizeB int) {

	for _, attribute := range record.FindNonResidentAttributes() {
		attrName := attribute.FindType()
		if attrName == "DATA" { //skip Data attributes since point to content only searching for metadata
			continue
		}
		runlist := *attribute.GetHeader().ATRrecordNoNResident.RunList

		length := record.GetTotalRunlistSize(attrName) * clusterSizeB
		if length == 0 { // no runlists found
			fmt.Printf("attribute %s No runlists found \n", attrName)
			continue
		}
		var buf bytes.Buffer
		buf.Grow(length)

		offset := int64(0)

		for (MFTAttributes.RunList{}) != runlist {
			offset += int64(runlist.Offset)

			clusters := int(runlist.Length)

			//inefficient since allocates memory for each round
			if offset*int64(clusterSizeB) >= hD.GetDiskSize()-partitionOffsetB {
				fmt.Printf("attribute %s runlist offset exceeds partition size %d\n", attrName, offset)
				break
			}
			data := hD.ReadFile(partitionOffsetB+offset*int64(clusterSizeB), clusters*clusterSizeB)
			buf.Write(data)

			if runlist.Next == nil {
				break
			}

			runlist = *runlist.Next
		}
		actualLen := int(attribute.GetHeader().ATRrecordNoNResident.ActualLength)
		if actualLen > length {
			fmt.Printf("attribute %s actual length exceeds the runlist length actual %d runlist %d \n", attrName, actualLen, length)
			continue
		}
		attribute.Parse(buf.Bytes()[:actualLen])

		buf.Reset()

	}

}

func (record Record) LocateData(hD img.DiskReader, partitionOffset int64, sectorsPerCluster int, bytesPerSector int, dataRuns *bytes.Buffer) {

	if record.HasResidentDataAttr() {
		dataRuns.Write(record.GetResidentData())
	} else {
		var runlist MFTAttributes.RunList

		runlist = record.GetRunList("DATA")

		offset := partitionOffset // partition in bytes

		diskSize := hD.GetDiskSize()

		for (MFTAttributes.RunList{}) != runlist {
			offset += runlist.Offset * int64(sectorsPerCluster*bytesPerSector)
			if offset > diskSize {
				fmt.Printf("skipped offset %d exceeds disk size! exiting", offset)
				break
			}
			//	fmt.Printf("extracting data from %d len %d \n", offset, runlist.Length)

			data := hD.ReadFile(offset, int(runlist.Length)*sectorsPerCluster*bytesPerSector)

			dataRuns.Write(data)

			if runlist.Next == nil {
				break
			}

			runlist = *runlist.Next
		}

	}
}

func (record Record) FindNonResidentAttributes() []Attribute {
	return utils.Filter(record.Attributes, func(attribute Attribute) bool {
		return attribute.IsNoNResident()
	})
}

func (record Record) FindAttributePtr(attributeName string) Attribute {
	for idx := range record.Attributes {
		if record.Attributes[idx].FindType() == attributeName {

			return record.Attributes[idx]
		}
	}
	return nil
}

func (record Record) FindAttribute(attributeName string) Attribute {
	for _, attribute := range record.Attributes {
		if attribute.FindType() == attributeName {

			return attribute
		}
	}
	return nil
}

func (record Record) HasResidentDataAttr() bool {
	attribute := record.FindAttribute("DATA")
	return attribute != nil && !attribute.IsNoNResident()
}

func (record Record) HasNonResidentAttr() bool {
	for _, attr := range record.Attributes {
		if !attr.IsNoNResident() {
			return true
		}
	}
	return false
}

func (record Record) getType() string {
	return MFTflags[record.Flags]
}

func (record Record) GetRunList(attrType string) MFTAttributes.RunList {

	attr := record.FindAttribute(attrType)
	return *attr.GetHeader().ATRrecordNoNResident.RunList

}

func (record Record) GetRunLists() []MFTAttributes.RunList {
	var runlists []MFTAttributes.RunList
	for _, attribute := range record.Attributes {
		if attribute.IsNoNResident() {

			runlists = append(runlists, *attribute.GetHeader().ATRrecordNoNResident.RunList)
		}
	}
	return runlists
}

func (record Record) ShowVCNs() {
	startVCN, lastVCN := record.getVCNs()
	if startVCN != 0 || lastVCN != 0 {
		fmt.Printf(" startVCN %d endVCN %d", startVCN, lastVCN)
	}

}

func (record Record) ShowIndex() {
	indexAttr := record.FindAttribute("Index Root")
	indexAlloc := record.FindAttribute("Index Allocation")

	if indexAttr != nil {
		idxRoot := indexAttr.(*MFTAttributes.IndexRoot)
		idxRoot.ShowInfo()

	}

	if indexAlloc != nil {
		idx := indexAlloc.(*MFTAttributes.IndexAllocation)
		idx.ShowInfo()

	}

}

func (record Record) getVCNs() (uint64, uint64) {
	for _, attribute := range record.Attributes {
		if attribute.IsNoNResident() {
			return attribute.GetHeader().ATRrecordNoNResident.StartVcn,
				attribute.GetHeader().ATRrecordNoNResident.LastVcn
		}
	}
	return 0, 0

}

func (record Record) ShowAttributes(attrType string) {
	fmt.Printf("%d %d %s \n", record.Entry, record.Seq, record.getType())
	var attributes []Attribute
	if attrType == "any" {
		attributes = record.Attributes
	} else {
		attributes = utils.Filter(record.Attributes, func(attribute Attribute) bool {
			return attribute.FindType() == attrType
		})
	}

	for _, attribute := range attributes {
		attribute.ShowInfo()
	}

}

func (record Record) ShowTimestamps() {
	var attr Attribute
	attr = record.FindAttribute("FileName")
	if attr != nil {
		fnattr := attr.(*MFTAttributes.FNAttribute)
		atime, ctime, mtime, mftime := fnattr.GetTimestamps()
		fmt.Printf("FN a %s c %s m %s mftm %s ", atime, ctime, mtime, mftime)
	}
	attr = record.FindAttribute("Standard Information")
	if attr != nil {
		siattr := attr.(*MFTAttributes.SIAttribute)
		atime, ctime, mtime, mftime := siattr.GetTimestamps()
		fmt.Printf("SI a %s c %s m %s mftm %s ", atime, ctime, mtime, mftime)
	}
}

func (record Record) showInfo() {
	fmt.Printf("record %d type %s\n", record.Entry, record.getType())
}

func (record Record) GetResidentData() []byte {
	return record.FindAttribute("DATA").(*MFTAttributes.DATA).Content

}

func (record Record) GetTotalRunlistSize(attributeType string) int {
	runlist := record.GetRunList(attributeType)
	totalSize := 0

	for (MFTAttributes.RunList{}) != runlist {

		totalSize += int(runlist.Length)

		if runlist.Next == nil {
			break
		}
		runlist = *runlist.Next
	}
	return totalSize
}

func (record Record) ShowRunList() {
	runlists := record.GetRunLists()

	nonResidentAttributes := record.FindNonResidentAttributes()
	for idx := range nonResidentAttributes {

		runlist := runlists[idx]
		for (MFTAttributes.RunList{}) != runlist {

			fmt.Printf(" offs. %d cl len %d cl \n", runlist.Offset, runlist.Length)
			if runlist.Next == nil {
				break
			}
			runlist = *runlist.Next
		}

	}

}

func (record Record) HasFilenameExtension(extension string) bool {
	if record.HasAttr("FileName") {
		fnattr := record.FindAttribute("FileName").(*MFTAttributes.FNAttribute)
		if strings.HasSuffix(fnattr.Fname, extension) {
			return true
		}
	}

	return false
}

func (record Record) HasFilename(filename string) bool {

	return record.GetFname() == filename

}

func (record Record) HasAttr(attrName string) bool {
	return record.FindAttribute(attrName) != nil
}

func (record Record) ShowIsResident() {
	if record.HasAttr("DATA") {
		if record.HasResidentDataAttr() {
			fmt.Printf("Resident")
		} else {
			fmt.Printf("NoN Resident")
		}

	} else {
		fmt.Print("NO DATA attr")
	}
}

func (record Record) ShowFNAModifiedTime() {
	fnattr := record.FindAttribute("FileName").(*MFTAttributes.FNAttribute)
	fmt.Printf("%s ", fnattr.Mtime.ConvertToIsoTime())
}

func (record Record) ShowFNACreationTime() {
	fnattr := record.FindAttribute("FileName").(*MFTAttributes.FNAttribute)
	fmt.Printf("%s ", fnattr.Crtime.ConvertToIsoTime())
}

func (record Record) ShowFNAMFTModifiedTime() {
	fnattr := record.FindAttribute("FileName").(*MFTAttributes.FNAttribute)
	fmt.Printf("%s ", fnattr.MFTmtime.ConvertToIsoTime())
}

func (record Record) ShowFNAMFTAccessTime() {
	fnattr := record.FindAttribute("FileName").(*MFTAttributes.FNAttribute)
	fmt.Printf("%s ", fnattr.Atime.ConvertToIsoTime())
}

func (record *Record) Process(bs []byte) {

	utils.Unmarshal(bs, record)

	if record.Signature == "BAAD" { //skip bad entry
		return
	}

	ReadPtr := record.AttrOff //offset to first attribute
	var linkedRecordsInfo []LinkedRecordInfo
	var attributes []Attribute
	for ReadPtr < 1024 {

		if utils.Hexify(bs[ReadPtr:ReadPtr+4]) == "ffffffff" { //End of attributes
			break
		}

		var attrHeader MFTAttributes.AttributeHeader
		utils.Unmarshal(bs[ReadPtr:ReadPtr+16], &attrHeader)

		if attrHeader.IsLast() { // End of attributes
			break
		}

		if !attrHeader.IsNoNResident() { //Resident Attribute
			var attr Attribute

			var atrRecordResident *MFTAttributes.ATRrecordResident = new(MFTAttributes.ATRrecordResident)

			atrRecordResident.Parse(bs[ReadPtr+16:])
			atrRecordResident.Name = utils.DecodeUTF16(bs[ReadPtr+attrHeader.NameOff : ReadPtr+attrHeader.NameOff+2*uint16(attrHeader.Nlen)])
			attrHeader.ATRrecordResident = atrRecordResident

			if attrHeader.IsFileName() { // File name
				attr = &MFTAttributes.FNAttribute{}
				attr.Parse(bs[ReadPtr+atrRecordResident.OffsetContent:])

			} else if attrHeader.IsReparse() {
				attr = &MFTAttributes.Reparse{}
				attr.Parse(bs[ReadPtr+atrRecordResident.OffsetContent:])

			} else if attrHeader.IsData() {
				attr = &MFTAttributes.DATA{}
				attr.Parse(bs[ReadPtr+
					atrRecordResident.OffsetContent : ReadPtr +
					+uint16(attrHeader.AttrLen)])

			} else if attrHeader.IsObject() {
				attr = &MFTAttributes.ObjectID{}
				attr.Parse(bs[ReadPtr+atrRecordResident.OffsetContent : ReadPtr+
					atrRecordResident.OffsetContent+64])

			} else if attrHeader.IsAttrList() { //Attribute List

				attr = &MFTAttributes.AttributeListEntries{}

				attr.Parse(bs[ReadPtr+atrRecordResident.OffsetContent : ReadPtr+
					atrRecordResident.OffsetContent+uint16(attrHeader.AttrLen)])
				attrListEntries := attr.(*MFTAttributes.AttributeListEntries) //dereference

				for _, entry := range attrListEntries.Entries {
					if entry.GetType() != "DATA" {
						continue
					}
					linkedRecordsInfo = append(linkedRecordsInfo,
						LinkedRecordInfo{Entry: uint32(entry.ParRef), StartVCN: entry.StartVcn})
				}

			} else if attrHeader.IsBitmap() { //BITMAP
				record.Bitmap = true
				attr = &MFTAttributes.BitMap{}
				attr.Parse(bs[ReadPtr+atrRecordResident.OffsetContent:])

			} else if attrHeader.IsVolumeName() { //Volume Name
				attr = &MFTAttributes.VolumeName{}
				attr.Parse(bs[ReadPtr+
					atrRecordResident.OffsetContent : uint32(ReadPtr)+
					uint32(atrRecordResident.OffsetContent)+atrRecordResident.ContentSize])

			} else if attrHeader.IsVolumeInfo() { //Volume Info
				attr = &MFTAttributes.VolumeInfo{}
				attr.Parse(bs[ReadPtr+atrRecordResident.OffsetContent : ReadPtr+
					atrRecordResident.OffsetContent+12])

			} else if attrHeader.IsIndexRoot() { //Index Root
				attr = &MFTAttributes.IndexRoot{}
				attr.Parse(bs[ReadPtr+atrRecordResident.OffsetContent:])

			} else if attrHeader.IsStdInfo() { //Standard Information

				attr = &MFTAttributes.SIAttribute{}
				attr.Parse(bs[ReadPtr+atrRecordResident.OffsetContent : ReadPtr+atrRecordResident.OffsetContent+72])

			} else {
				fmt.Printf("uknown attribute %s \n", attrHeader.GetType())
			}
			if attr != nil {
				attr.SetHeader(&attrHeader)
				attributes = append(attributes, attr)

			}

		} else { //NoN Resident Attribute
			var atrNoNRecordResident *MFTAttributes.ATRrecordNoNResident = new(MFTAttributes.ATRrecordNoNResident)
			utils.Unmarshal(bs[ReadPtr+16:ReadPtr+64], atrNoNRecordResident)

			if ReadPtr+attrHeader.AttrLen <= 1024 {
				var runlist *MFTAttributes.RunList = new(MFTAttributes.RunList)
				runlist.Process(bs[ReadPtr+
					atrNoNRecordResident.RunOff : ReadPtr+attrHeader.AttrLen])
				atrNoNRecordResident.RunList = runlist

			}
			attrHeader.ATRrecordNoNResident = atrNoNRecordResident

			if attrHeader.IsData() {
				data := &MFTAttributes.DATA{}
				data.SetHeader(&attrHeader)
				attributes = append(attributes, data)
			} else if attrHeader.IsIndexAllocation() {
				var idxAllocation *MFTAttributes.IndexAllocation = new(MFTAttributes.IndexAllocation)
				idxAllocation.SetHeader(&attrHeader)
				attributes = append(attributes, idxAllocation)
			} else if attrHeader.IsBitmap() { //BITMAP
				record.Bitmap = true

				var bitmap *MFTAttributes.BitMap = new(MFTAttributes.BitMap)
				bitmap.SetHeader(&attrHeader)
				attributes = append(attributes, bitmap)
			} else if attrHeader.IsAttrList() {
				var attrListEntries *MFTAttributes.AttributeListEntries = new(MFTAttributes.AttributeListEntries)
				attrListEntries.SetHeader(&attrHeader)
				attributes = append(attributes, attrListEntries)
			} else if attrHeader.IsReparse() {
				var reparse *MFTAttributes.Reparse = new(MFTAttributes.Reparse)
				reparse.SetHeader(&attrHeader)
				attributes = append(attributes, reparse)
			} else {
				fmt.Printf("unknown non resident attr %s\n", attrHeader.GetType())
			}

		} //ends non Resident
		ReadPtr = ReadPtr + uint16(attrHeader.AttrLen)

	} //ends while
	record.Attributes = attributes
	record.LinkedRecordsInfo = linkedRecordsInfo
}

func (record Record) ShowFileSize() {

	logical := record.GetLogicalFileSize()
	physical := record.GetPhysicalSize()
	fmt.Printf(" logical: %d (KB), physical: %d (KB)",
		logical/1024, physical/1024)

}

func (record Record) GetPhysicalSize() int64 {
	fnattr := record.FindAttribute("FileName").(*MFTAttributes.FNAttribute)
	return int64(fnattr.AllocFsize)
}

func (record Record) GetLogicalFileSize() int64 {
	attr := record.FindAttribute("FileName")
	if attr != nil {
		fnattr := attr.(*MFTAttributes.FNAttribute)
		if int64(fnattr.RealFsize) != 0 {
			return int64(fnattr.RealFsize)
		}

	}
	return int64(record.I30Size)
}

func (record Record) GetFnames() map[string]string {

	fnAttributes := utils.Filter(record.Attributes, func(attribute Attribute) bool {
		return attribute.FindType() == "FileName"
	})
	fnames := make(map[string]string, len(fnAttributes))
	for _, attr := range fnAttributes {
		fnattr := attr.(*MFTAttributes.FNAttribute)
		fnames[fnattr.GetFileNameType()] = fnattr.Fname

	}

	return fnames

}

func (record Record) GetFname() string {
	fnames := record.GetFnames()
	for _, namescheme := range []string{"POSIX", "Win32", "Win32 & Dos", "Dos"} {
		name, ok := fnames[namescheme]
		if ok {
			return name
		}
	}
	return "-"

}

func (record Record) ShowFileName(fileNameSyntax string) {

	fnames := record.GetFnames()
	for ftype, fname := range fnames {
		if ftype == fileNameSyntax {
			fmt.Printf(" %s ", fname)
		} else {
			fmt.Printf(" %s ", fname)
		}
	}
}

func (records Records) FilterByExtension(extension string) []Record {

	return utils.Filter(records, func(record Record) bool {
		return record.HasFilenameExtension(extension)
	})

}

func addLinkedRecords(records []Record) []Record {
	for _, record := range records {
		linkedRecord := record.LinkedRecord
		for linkedRecord != nil {
			records = append(records, *linkedRecord)
			linkedRecord = linkedRecord.LinkedRecord
		}

	}
	return records
}

func (records Records) FilterByName(filename string) []Record {
	filteredRecords := utils.Filter(records, func(record Record) bool {
		return record.HasFilename(filename)
	})
	return addLinkedRecords(filteredRecords)

}
