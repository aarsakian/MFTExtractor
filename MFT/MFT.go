package MFT

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/aarsakian/MFTExtractor/MFT/attributes"
	MFTAttributes "github.com/aarsakian/MFTExtractor/MFT/attributes"
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
	Attributes         []MFTAttributes.Attribute
	Bitmap             bool
	// fixupArray add the        UpdateSeqArrOffset to find is location

}

func (record Record) containsAttribute(attributeName string) bool {
	for _, attribute := range record.Attributes {
		if attribute.FindType() == attributeName {
			return true
		}
	}
	return false
}

func (record Record) FindAttribute(attributeName string) attributes.Attribute {
	for _, attribute := range record.Attributes {
		if attribute.FindType() == attributeName {

			return attribute
		}
	}
	return nil
}

func (record Record) hasResidentDataAttr() bool {
	attribute := record.FindAttribute("DATA")
	return attribute != nil && !attribute.IsNoNResident()
}

func (record Record) getType() string {
	return MFTflags[record.Flags]
}

func (record Record) getRunList() MFTAttributes.RunList {
	for _, attribute := range record.Attributes {
		if attribute.IsNoNResident() &&
			attribute.GetHeader().ATRrecordNoNResident.RunList != nil {
			return *attribute.GetHeader().ATRrecordNoNResident.RunList
		}
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
	fnAttributes := utils.Filter(record.Attributes, func(attribute MFTAttributes.Attribute) bool {
		return attribute.FindType() == attrType
	})
	for _, attribute := range fnAttributes {
		attribute.ShowInfo()
	}

}

func (record Record) ShowTimestamps() {
	var attr attributes.Attribute
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

func (record Record) getData(sectorsPerCluster uint8, disk int, partitionOffset uint64) []byte {

	if record.hasResidentDataAttr() {

		return record.FindAttribute("DATA").(*MFTAttributes.DATA).Content

	} else {
		runlist := record.getRunList()
		lsize, _ := record.GetFileSize()

		var dataRuns bytes.Buffer
		dataRuns.Grow(int(lsize))

		offset := int64(partitionOffset) * 512 // partition in bytes
		hD := img.GetHandler(fmt.Sprintf("\\\\.\\PHYSICALDRIVE%d", disk))
		diskSize := hD.GetDiskSize()

		for (MFTAttributes.RunList{}) != runlist {
			offset += runlist.Offset * int64(sectorsPerCluster) * 512
			if offset > diskSize {
				fmt.Printf("skipped offset %d exceeds disk size! exiting", offset)
				break
			}
			//	fmt.Printf("extracting data from %d len %d \n", offset, runlist.Length)
			buffer := make([]byte, uint32(runlist.Length*8*512))
			hD.ReadFile(offset, buffer)

			dataRuns.Write(buffer)

			if runlist.Next == nil {
				break
			}

			runlist = *runlist.Next
		}
		return dataRuns.Bytes()

	}

}

func (record Record) GetRunListSizesAndOffsets() map[int]int {
	runlist := record.getRunList()

	offsetLenMap := map[int]int{}
	for (MFTAttributes.RunList{}) != runlist {
		offsetLenMap[int(runlist.Offset)] = int(runlist.Length)

		if runlist.Next == nil {
			break
		}
		runlist = *runlist.Next
	}
	return offsetLenMap
}

func (record Record) GetTotalRunlistSize() int {
	offsetLenMap := record.GetRunListSizesAndOffsets()
	totalSize := 0
	for _, length := range offsetLenMap {
		totalSize += int(length)
	}
	return totalSize

}

func (record Record) ShowRunList() {
	offsetLenMap := record.GetRunListSizesAndOffsets()
	totalSize := 0
	for offset, length := range offsetLenMap {
		totalSize += int(length)
		fmt.Printf(" offs. %d cl len %d cl \n", offset*8, length*8)
	}

}

func (record Record) HasFilenameExtension(extension string) bool {
	if record.hasAttr("FileName") {
		fnattr := record.FindAttribute("FileName").(*MFTAttributes.FNAttribute)
		if strings.HasSuffix(fnattr.Fname, extension) {
			return true
		}
	}

	return false
}

func (record Record) hasAttr(attrName string) bool {
	return record.FindAttribute(attrName) != nil
}

func (record Record) ShowIsResident() {
	if record.hasAttr("DATA") {
		if record.hasResidentDataAttr() {
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

func (record Record) CreateFileFromEntry(clusterPerSector uint8, disk int, partitionOffset uint64) {
	fnattr := record.FindAttribute("FileName").(*MFTAttributes.FNAttribute)

	data := record.getData(clusterPerSector, disk, partitionOffset)
	utils.WriteFile(fnattr.Fname, data)

}

func (record *Record) Process(bs []byte) {

	utils.Unmarshal(bs, record)

	if record.Signature == "BAAD" { //skip bad entry
		return
	}

	ReadPtr := record.AttrOff //offset to first attribute
	fmt.Printf("Processing $MFT entry %d \n", record.Entry)
	var attributes []MFTAttributes.Attribute
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

				for idxEntryOffset+16 < uint16(nodeheader.OffsetEndEntryListBuffer) {
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
					idxEntryOffset = idxEntryOffset + idxEntry.Len

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
				utils.Unmarshal(bs[ReadPtr+64:ReadPtr+64+24], idxAllocation)

				var nodeheader *MFTAttributes.NodeHeader = new(MFTAttributes.NodeHeader)
				utils.Unmarshal(bs[ReadPtr+64+24:ReadPtr+64+24+16], nodeheader)

				var idxEntry *MFTAttributes.IndexEntry = new(MFTAttributes.IndexEntry)
				nodeheaderEndOffs := ReadPtr + 64 + 24 + 16
				utils.Unmarshal(bs[nodeheaderEndOffs+
					uint16(nodeheader.OffsetEntryList):nodeheaderEndOffs+
					uint16(nodeheader.OffsetEntryList)+15], idxEntry)

				idxAllocation.Nodeheader = nodeheader
				idxAllocation.SetHeader(&attrHeader)
				idxAllocation.IndexEntries = append(idxAllocation.IndexEntries, *idxEntry)
				attributes = append(attributes, idxAllocation)
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

func (record Record) ShowFileName(fileNameSyntax string) {
	fnAttributes := utils.Filter(record.Attributes, func(attribute MFTAttributes.Attribute) bool {
		return attribute.FindType() == "FileName"
	})
	if len(fnAttributes) != 0 {
		for _, attr := range fnAttributes {

			fnattr := attr.(*MFTAttributes.FNAttribute)
			if fileNameSyntax == "Win32" && fnattr.GetFileNameType() == "Win32 & Dos" {
				fmt.Printf(" %s ", fnattr.Fname)
			} else if fileNameSyntax == fnattr.GetFileNameType() { //Dos
				fmt.Printf(" %s ", fnattr.Fname)
			} else if fileNameSyntax == "ANY" {
				fmt.Printf(" %s ", fnattr.Fname)
			}
		}

	}

}

func (record Record) GetBasicInfoFromRecord(file1 *os.File) {

	s := fmt.Sprintf("%d;%d;%s", record.Entry, record.Seq, record.getType())
	attr := record.FindAttribute("FileName")
	if attr != nil {
		fnattr := attr.(*MFTAttributes.FNAttribute)

		s1 := strings.Join([]string{s, fnattr.Atime.ConvertToIsoTime(),
			fnattr.Crtime.ConvertToIsoTime(),
			fnattr.Mtime.ConvertToIsoTime(), fnattr.Fname,
			fmt.Sprintf(";%d;%d;%s\n", fnattr.ParRef, fnattr.ParSeq,
				fnattr.GetType())}, ";")

		utils.WriteToCSV(file1, s1)
	}

	/*

						// fmt.Println("file unique ID ",objectattr.objID)
							if *save2DB {
								dbmap.Insert(&objectattr)
								checkErr(err, "Insert failed")
							}
							s := []string{";", objectattr.ObjID}
							_, err := file1.WriteString(strings.Join(s, " "))
							if err != nil {
								// handle the error here
								fmt.Printf("err %s\n", err)
								return
							}


			s := []string{"Type of Attr in Run list", fmt.Sprintf("Attribute starts at %d", ReadPtr),
				AttrTypes[attrList.Type], fmt.Sprintf("length %d ", attrList.Len), fmt.Sprintf("start VCN %d ", attrList.StartVcn),
				"MFT Record Number", fmt.Sprintf("%d Name %s", attrList.FileRef, attrList.name),
				"Attribute ID ", fmt.Sprintf("%d ", attrList.ID), string(10)}
			_, err := file1.WriteString(strings.Join(s, " "))
			if err != nil {
				// handle the error here
				fmt.Printf("err %s\n", err)
				return
			}

			s := []string{";", volname.Name.PrintNulls()}
			_, err := file1.WriteString(strings.Join(s, "s"))
			if err != nil {
				// handle the error here
				fmt.Printf("err %s\n", err)
				return
			}

			s := []string{"Vol Info flags", volinfo.Flags, string(10)}
			_, err := file1.WriteString(strings.Join(s, " "))
			if err != nil {
				// handle the error here
				fmt.Printf("err %s\n", err)
				return
			}

			s := []string{idxRoot.Type, ";", fmt.Sprintf(";%d", idxRoot.Sizeclusters), ";", fmt.Sprintf("%d;", 16+idxRoot.nodeheader.OffsetEntryList),
			fmt.Sprintf(";%d", 16+idxRoot.nodeheader.OffsetEndUsedEntryList), fmt.Sprintf("allocated ends at %d", 16+idxRoot.nodeheader.OffsetEndEntryListBuffer),
			fmt.Sprintf("MFT entry%d ", idxEntry.MFTfileref), "FLags", IndexEntryFlags[idxEntry.Flags]}
		//fmt.Sprintf("%x",bs[uint32(ReadPtr)+uint32(atrRecordResident.OffsetContent)+32:uint32(ReadPtr)+uint32(atrRecordResident.OffsetContent)+16+IDxroot.nodeheader.OffsetEndEntryListBuffer]
		s1 := []string{"Filename idx Entry", fnattrIDXEntry.Fname}
		file1.WriteString(strings.Join(s1, " "))
		}



	*/
}
