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

type MFTrecord struct {
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

func (record MFTrecord) containsAttribute(attributeName string) bool {
	for _, attribute := range record.Attributes {
		if attribute.FindType() == attributeName {
			return true
		}
	}
	return false
}

func (record MFTrecord) FindAttribute(attributeName string) attributes.Attribute {
	for _, attribute := range record.Attributes {
		if attribute.FindType() == attributeName {
			return attribute
		}
	}
	return nil
}

func (record MFTrecord) hasResidentDataAttr() bool {
	attribute := record.FindAttribute("DATA")
	return attribute != nil && !attribute.IsNoNResident()
}

func (record MFTrecord) getType() string {
	return MFTflags[record.Flags]
}

func (record MFTrecord) getRunList() MFTAttributes.RunList {
	for _, attribute := range record.Attributes {
		if attribute.IsNoNResident() &&
			attribute.GetHeader().ATRrecordNoNResident.RunList != nil {
			return *attribute.GetHeader().ATRrecordNoNResident.RunList
		}
	}
	return MFTAttributes.RunList{}
}

func (record MFTrecord) ShowVCNs() {
	startVCN, lastVCN := record.getVCNs()
	if startVCN != 0 || lastVCN != 0 {
		fmt.Printf(" startVCN %d endVCN %d", startVCN, lastVCN)
	}

}

func (record MFTrecord) ShowIndex() {
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

func (record MFTrecord) getVCNs() (uint64, uint64) {
	for _, attribute := range record.Attributes {
		if attribute.IsNoNResident() {
			return attribute.GetHeader().ATRrecordNoNResident.StartVcn,
				attribute.GetHeader().ATRrecordNoNResident.LastVcn
		}
	}
	return 0, 0

}

func (record MFTrecord) ShowAttributes(attrType string) {
	fmt.Printf("\n %d %d %s ", record.Entry, record.Seq, record.getType())
	fnAttributes := utils.Filter(record.Attributes, func(attribute MFTAttributes.Attribute) bool {
		return attribute.FindType() == attrType
	})
	for _, attribute := range fnAttributes {
		attribute.ShowInfo()
	}

}

func (record MFTrecord) ShowTimestamps() {
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

func (record MFTrecord) showInfo() {
	fmt.Printf("record %d type %s\n", record.Entry, record.getType())
}

func (record MFTrecord) getData(sectorsPerCluster uint8, disk string) []byte {

	if record.hasResidentDataAttr() {

		return record.FindAttribute("DATA").(*MFTAttributes.DATA).Content

	} else {
		runlist := record.getRunList()
		var dataRuns [][]byte
		offset := int64(0)
		hD := img.GetHandler(disk)
		diskSize := hD.GetDiskSize()

		for (MFTAttributes.RunList{}) != runlist {
			offset += runlist.Offset * int64(sectorsPerCluster) * 512
			if offset > diskSize {
				fmt.Printf("skipped offset %d exceeds disk size! exiting", offset)
				break
			}
			fmt.Printf("extracting data from %d len %d \n", offset, runlist.Length)
			data := hD.ReadFile(offset,
				uint32(runlist.Length*8*512))
			dataRuns = append(dataRuns, data)

			if runlist.Next == nil {
				break
			}

			runlist = *runlist.Next
		}
		return bytes.Join(dataRuns, []byte(""))

	}

}

func (record MFTrecord) ShowRunList() {
	runlist := record.getRunList()
	totalSize := 0
	for (MFTAttributes.RunList{}) != runlist {
		totalSize += int(runlist.Length)
		fmt.Printf(" offs. %d sector len %d ", runlist.Offset*8, runlist.Length*8)
		if runlist.Next == nil {
			fmt.Printf("total size %d clusters", totalSize)
			break
		}
		runlist = *runlist.Next
	}

}

func (record MFTrecord) hasAttr(attrName string) bool {
	return record.FindAttribute(attrName) != nil
}

func (record MFTrecord) ShowIsResident() {
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

func (record MFTrecord) ShowFNAModifiedTime() {
	fnattr := record.FindAttribute("FileName").(*MFTAttributes.FNAttribute)
	fmt.Printf("%s ", fnattr.Mtime.ConvertToIsoTime())
}

func (record MFTrecord) ShowFNACreationTime() {
	fnattr := record.FindAttribute("FileName").(*MFTAttributes.FNAttribute)
	fmt.Printf("%s ", fnattr.Crtime.ConvertToIsoTime())
}

func (record MFTrecord) ShowFNAMFTModifiedTime() {
	fnattr := record.FindAttribute("FileName").(*MFTAttributes.FNAttribute)
	fmt.Printf("%s ", fnattr.MFTmtime.ConvertToIsoTime())
}

func (record MFTrecord) ShowFNAMFTAccessTime() {
	fnattr := record.FindAttribute("FileName").(*MFTAttributes.FNAttribute)
	fmt.Printf("%s ", fnattr.Atime.ConvertToIsoTime())
}

func (record MFTrecord) CreateFileFromEntry(clusterPerSector uint8, disk string) {
	fnattr := record.FindAttribute("FileName").(*MFTAttributes.FNAttribute)

	data := record.getData(clusterPerSector, disk)
	utils.WriteFile(fnattr.Fname, data)

}

func (record *MFTrecord) Process(bs []byte) {

	utils.Unmarshal(bs, record)

	if record.Signature == "BAAD" { //skip bad entry
		return
	}

	ReadPtr := record.AttrOff //offset to first attribute
	//fmt.Printf("\n Processing $MFT entry %d ", record.Entry)
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

func (record MFTrecord) ShowFileSize() {

	attr := record.FindAttribute("FileName")
	if attr != nil {
		fnattr := attr.(*MFTAttributes.FNAttribute)
		fmt.Printf(" allocated: %d (KB), real: %d (KB)",
			fnattr.AllocFsize/1024, fnattr.RealFsize/1024)
	}

}

func (record MFTrecord) ShowFileName(fileNameSyntax string) {
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

func (record MFTrecord) GetBasicInfoFromRecord(file1 *os.File) {

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
