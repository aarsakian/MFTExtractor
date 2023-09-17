package MFT

import (
	"bytes"
	"fmt"
	"math"
	"os"
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
	Records  []Record
	Filepath string
	Size     int
}

type Records []Record

type Attribute interface {
	FindType() string
	SetHeader(header *MFTAttributes.AttributeHeader)
	GetHeader() MFTAttributes.AttributeHeader
	IsNoNResident() bool
	ShowInfo()
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
	// fixupArray add the        UpdateSeqArrOffset to find is location

}

func (mfttable *MFTTable) Populate(MFTSelectedEntry int, fromMFTEntry int, ToMFTEntry int) {
	file, err := os.Open(mfttable.Filepath)
	if err != nil {
		// handle the error here
		fmt.Printf("err %s for reading the MFT ", err)
		return
	}
	defer file.Close()

	fsize, err := file.Stat() //file descriptor
	if err != nil {
		fmt.Printf("error getting the file size\n")
		return
	}
	mfttable.Size = int(fsize.Size())

	var buf bytes.Buffer
	// collect data of $MFT

	bs := make([]byte, RecordSize)
	// find buffer size
	totalRecords := int(mfttable.Size) / RecordSize
	if fromMFTEntry != -1 {
		totalRecords -= fromMFTEntry
	}
	if ToMFTEntry != math.MaxUint32 {
		totalRecords -= ToMFTEntry
	}
	if MFTSelectedEntry != -1 {
		totalRecords = 1
	}
	buf.Grow(totalRecords * RecordSize)
	for i := 0; i < int(mfttable.Size); i += 1024 {

		if i/1024 > ToMFTEntry {
			break
		}

		if MFTSelectedEntry != -1 && i/1024 != MFTSelectedEntry ||
			fromMFTEntry > i/1024 {
			continue
		}

		_, err = file.ReadAt(bs, int64(i))

		if err != nil {
			fmt.Printf("error reading file --->%s", err)
			return
		}
		buf.Write(bs)

		if i == MFTSelectedEntry {
			break
		}

	}
	mfttable.ProcessRecords(buf.Bytes())

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
		fmt.Printf("Processing $MFT entry %d  out of %d records \n", record.Entry+1, len(records))
		record.Process(data[i : i+RecordSize])
		records[i/RecordSize] = record
	}
	mfttable.Records = records
}

func (mfttable *MFTTable) ProcessNonResidentRecords(hD img.DiskReader, partitionOffsetB int64, clusterSizeB int) {

	for idx := range mfttable.Records {
		mfttable.Records[idx].ProcessNoNResidentAttributes(hD, partitionOffsetB, clusterSizeB)
	}
}

func (record *Record) ProcessNoNResidentAttributes(hD img.DiskReader, partitionOffsetB int64, clusterSizeB int) {
	for _, attribute := range record.FindNonResidentAttributes() {
		attrName := attribute.FindType()
		runlist := record.GetRunList(attrName)
		length := record.GetTotalRunlistSize(attrName) * clusterSizeB

		var buf bytes.Buffer
		buf.Grow(length)

		offset := int64(0)

		for (MFTAttributes.RunList{}) != runlist {
			offset += int64(runlist.Offset)

			clusters := int(runlist.Length)

			//inefficient since allocates memory for each round
			data := hD.ReadFile(partitionOffsetB+offset*int64(clusterSizeB), clusters*clusterSizeB)
			buf.Write(data)

			if runlist.Next == nil {
				break
			}

			runlist = *runlist.Next
		}
		if attrName == "Index Allocation" {
			idxAllocation := attribute.(*MFTAttributes.IndexAllocation)
			idxAllocation.Parse(buf.Bytes())
		} else if attrName == "BitMap" {
			bitmap := attribute.(*MFTAttributes.BitMap)
			bitmap.AllocationStatus = buf.Bytes()
		}

	}

}

func (record Record) FindNonResidentAttributes() []Attribute {
	return utils.Filter(record.Attributes, func(attribute Attribute) bool {
		return attribute.IsNoNResident()
	})
}

func (record Record) containsAttribute(attributeName string) bool {
	for _, attribute := range record.Attributes {
		if attribute.FindType() == attributeName {
			return true
		}
	}
	return false
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
	attributes := utils.Filter(record.Attributes, func(attribute Attribute) bool {
		return attribute.FindType() == attrType && attribute.IsNoNResident()
	})

	if len(attributes) == 1 && attributes[0].GetHeader().ATRrecordNoNResident.RunList != nil {
		return *attributes[0].GetHeader().ATRrecordNoNResident.RunList
	}

	return MFTAttributes.RunList{}
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

		for _, idxEntry := range idxRoot.IndexEntries {
			if idxEntry.Fnattr == nil {
				continue
			}
			idxEntry.ShowInfo()
		}

	}

	if indexAlloc != nil {
		idx := indexAlloc.(*MFTAttributes.IndexAllocation)
		if idx.IndexEntries[0].Fnattr != nil {
			fmt.Printf("idx alloc %s ",
				idx.IndexEntries[0].Fnattr.Fname)
		}

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
	//fmt.Printf("\n %d %d %s ", record.Entry, record.Seq, record.getType())
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
	runlist := record.GetRunList("DATA")

	for (MFTAttributes.RunList{}) != runlist {

		fmt.Printf(" offs. %d cl len %d cl \n", runlist.Offset, runlist.Length)
		if runlist.Next == nil {
			break
		}
		runlist = *runlist.Next
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
			var atrRecordResident *MFTAttributes.ATRrecordResident = new(MFTAttributes.ATRrecordResident)
			utils.Unmarshal(bs[ReadPtr+16:ReadPtr+24], atrRecordResident)
			attrHeader.ATRrecordResident = atrRecordResident

			if attrHeader.IsFileName() { // File name
				var fnattr *MFTAttributes.FNAttribute = new(MFTAttributes.FNAttribute)
				utils.Unmarshal(bs[ReadPtr+atrRecordResident.OffsetContent:ReadPtr+
					atrRecordResident.OffsetContent+66], fnattr)

				fnattr.Fname =
					utils.DecodeUTF16(bs[ReadPtr+atrRecordResident.OffsetContent+66 : ReadPtr+
						atrRecordResident.OffsetContent+66+2*uint16(fnattr.Nlen)])
				fnattr.SetHeader(&attrHeader)
				attributes = append(attributes, fnattr)

			} else if attrHeader.IsReparse() {
				var reparse *MFTAttributes.Reparse = new(MFTAttributes.Reparse)
				utils.Unmarshal(bs[ReadPtr+atrRecordResident.OffsetContent:ReadPtr+
					atrRecordResident.OffsetContent+16], reparse)
				reparse.Name = utils.DecodeUTF16(bs[ReadPtr+atrRecordResident.OffsetContent+16+
					uint16(reparse.TargetNameOffset) : ReadPtr+atrRecordResident.OffsetContent+
					16+uint16(reparse.TargetNameOffset)+reparse.TargetLen])
				reparse.PrintName = utils.DecodeUTF16(bs[ReadPtr+atrRecordResident.OffsetContent+16+
					uint16(reparse.TargetPrintNameOffset) : ReadPtr+atrRecordResident.OffsetContent+16+
					uint16(reparse.TargetPrintNameLen)])
			} else if attrHeader.IsData() {
				data := &MFTAttributes.DATA{Content: bs[ReadPtr+
					atrRecordResident.OffsetContent : ReadPtr +
					+uint16(attrHeader.AttrLen)], Header: &attrHeader}
				attributes = append(attributes, data)

			} else if attrHeader.IsObject() {
				var objectattr *MFTAttributes.ObjectID = new(MFTAttributes.ObjectID)
				utils.Unmarshal(bs[ReadPtr+atrRecordResident.OffsetContent:ReadPtr+
					atrRecordResident.OffsetContent+64], objectattr)
				objectattr.SetHeader(&attrHeader)
				attributes = append(attributes, objectattr)

			} else if attrHeader.IsAttrList() { //Attribute List

				attrLen := uint16(0)
				var attrListEntries *MFTAttributes.AttributeListEntries = new(MFTAttributes.AttributeListEntries)

				for atrRecordResident.OffsetContent+24+attrLen < uint16(attrHeader.AttrLen) {
					var attrList MFTAttributes.AttributeList
					utils.Unmarshal(bs[ReadPtr+atrRecordResident.OffsetContent+attrLen:ReadPtr+
						atrRecordResident.OffsetContent+attrLen+24], &attrList)
					attrList.Name = utils.NoNull(bs[ReadPtr+atrRecordResident.OffsetContent+attrLen+
						uint16(attrList.Nameoffset) : ReadPtr+atrRecordResident.OffsetContent+
						attrLen+uint16(attrList.Nameoffset)+2*uint16(attrList.Namelen)])
					//   runlist=bs[ReadPtr+atrRecordResident.OffsetContent+attrList.len:uint32(ReadPtr)+atrRecordResident.Len]
					attrListEntries.Entries = append(attrListEntries.Entries, attrList)
					attrLen += attrList.Len

				}
				attrListEntries.SetHeader(&attrHeader)
				attributes = append(attributes, attrListEntries)

			} else if attrHeader.IsBitmap() { //BITMAP
				record.Bitmap = true
				var bitmap *MFTAttributes.BitMap = new(MFTAttributes.BitMap)
				bitmap.AllocationStatus = bs[ReadPtr+
					atrRecordResident.OffsetContent : uint32(ReadPtr)+
					uint32(atrRecordResident.OffsetContent)+atrRecordResident.ContentSize]
				bitmap.SetHeader(&attrHeader)
				attributes = append(attributes, bitmap)

			} else if attrHeader.IsVolumeName() { //Volume Name
				volumeName := &MFTAttributes.VolumeName{Name: utils.NoNull(bs[ReadPtr+
					atrRecordResident.OffsetContent : uint32(ReadPtr)+
					uint32(atrRecordResident.OffsetContent)+atrRecordResident.ContentSize]),
					Header: &attrHeader}
				volumeName.SetHeader(&attrHeader)
				attributes = append(attributes, volumeName)

			} else if attrHeader.IsVolumeInfo() { //Volume Info
				var volumeInfo *MFTAttributes.VolumeInfo = new(MFTAttributes.VolumeInfo)
				utils.Unmarshal(bs[ReadPtr+atrRecordResident.OffsetContent:ReadPtr+
					atrRecordResident.OffsetContent+12], volumeInfo)
				volumeInfo.SetHeader(&attrHeader)
				attributes = append(attributes, volumeInfo)

			} else if attrHeader.IsIndexRoot() { //Index Root
				var idxRoot *MFTAttributes.IndexRoot = new(MFTAttributes.IndexRoot)
				utils.Unmarshal(bs[ReadPtr+atrRecordResident.OffsetContent:ReadPtr+
					atrRecordResident.OffsetContent+12], idxRoot)

				var nodeheader *MFTAttributes.NodeHeader = new(MFTAttributes.NodeHeader)
				utils.Unmarshal(bs[ReadPtr+atrRecordResident.OffsetContent+
					16:ReadPtr+atrRecordResident.OffsetContent+32], nodeheader)
				idxRoot.Nodeheader = nodeheader

				idxEntryOffset := ReadPtr + atrRecordResident.OffsetContent + 16 + uint16(nodeheader.OffsetEntryList)
				lastIdxEntryOffset := ReadPtr + atrRecordResident.OffsetContent + 16 + uint16(nodeheader.OffsetEndEntryListBuffer)

				for idxEntryOffset+16 < lastIdxEntryOffset {
					var idxEntry *MFTAttributes.IndexEntry = new(MFTAttributes.IndexEntry)
					utils.Unmarshal(bs[idxEntryOffset:idxEntryOffset+16], idxEntry)

					if idxEntry.FilenameLen > 0 {
						var fnattrIDXEntry MFTAttributes.FNAttribute
						utils.Unmarshal(bs[idxEntryOffset+16:idxEntryOffset+16+idxEntry.FilenameLen],
							&fnattrIDXEntry)

						fnattrIDXEntry.Fname =
							utils.DecodeUTF16(bs[idxEntryOffset+16+66 : idxEntryOffset+16+
								66+2*uint16(fnattrIDXEntry.Nlen)])
						idxEntry.Fnattr = &fnattrIDXEntry

					}
					idxEntryOffset += idxEntry.Len

					idxRoot.IndexEntries = append(idxRoot.IndexEntries, *idxEntry)
				}

				idxRoot.SetHeader(&attrHeader)
				attributes = append(attributes, idxRoot)
			} else if attrHeader.IsStdInfo() { //Standard Information
				startpoint := ReadPtr + atrRecordResident.OffsetContent
				var siattr *MFTAttributes.SIAttribute = new(MFTAttributes.SIAttribute)
				utils.Unmarshal(bs[startpoint:startpoint+72], siattr)
				siattr.SetHeader(&attrHeader)
				attributes = append(attributes, siattr)

			}

		} else { //NoN Resident Attribute
			var atrNoNRecordResident *MFTAttributes.ATRrecordNoNResident = new(MFTAttributes.ATRrecordNoNResident)
			utils.Unmarshal(bs[ReadPtr+16:ReadPtr+64], atrNoNRecordResident)

			if uint32(ReadPtr)+attrHeader.AttrLen <= 1024 {
				var runlist *MFTAttributes.RunList = new(MFTAttributes.RunList)
				runlist.Process(bs[ReadPtr+
					atrNoNRecordResident.RunOff : uint32(ReadPtr)+attrHeader.AttrLen])
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
}

func (record Record) ShowFileSize() {

	allocated, real := record.GetFileSize()
	fmt.Printf(" logical: %d (KB), physical: %d (KB)",
		allocated/1024, real/1024)

}

func (record Record) GetFileSize() (logical int64, physical int64) {
	attr := record.FindAttribute("FileName")
	if attr != nil {
		fnattr := attr.(*MFTAttributes.FNAttribute)
		return int64(fnattr.AllocFsize), int64(fnattr.RealFsize)
	}
	return 0, 0
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

func (records Records) FilterByName(filename string) []Record {
	return utils.Filter(records, func(record Record) bool {
		return record.HasFilename(filename)
	})

}
