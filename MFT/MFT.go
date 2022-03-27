package MFT

import (
	"fmt"
	"os"
	"strings"

	"github.com/aarsakian/MFTExtractor/MFT/attributes"
	MFTAttributes "github.com/aarsakian/MFTExtractor/MFT/attributes"
	"github.com/aarsakian/MFTExtractor/utils"
)

var IndexEntryFlags = map[string]string{
	"00000001": "Child Node exists",
	"00000002": "Last Entry in list",
}

var SIFlags = map[uint32]string{
	1: "Read Only", 2: "Hidden", 4: "System", 32: "Archive", 64: "Device", 128: "Normal",
	256: "Temporary", 512: "Sparse", 1024: "Reparse Point", 2048: "Compressed", 4096: "Offline",
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

func (record MFTrecord) findAttribute(attributeName string) attributes.Attribute {
	for _, attribute := range record.Attributes {
		if attribute.FindType() == attributeName {
			return attribute
		}
	}
	return nil
}

func ProcessRunList(runlist []byte) []uint64 {
	clusterPtr := uint64(0)
	var clusters []uint64

	// fmt.Printf("LEN %d RUNLIST %x\n" ,len(runlist),runlist)
	for clusterPtr < uint64(len(runlist)) { // length of bytes of runlist
		ClusterOffsB, ClusterLenB := utils.DetermineClusterOffsetLength(runlist[clusterPtr])

		if ClusterLenB != 0 && ClusterOffsB != 0 {
			clustersLen := utils.ReadEndianInt(runlist[clusterPtr+1 : clusterPtr+ClusterLenB+1])

			clustersOff := utils.ReadEndianInt(runlist[clusterPtr+1+ClusterLenB : clusterPtr+ClusterLenB+ClusterOffsB+1])
			fmt.Printf("len of %d clusterlen %d and clust %d clustoff %d came from %x \n",
				ClusterLenB, clustersLen, ClusterOffsB, clustersOff, runlist[clusterPtr])
			for nextCluster := uint64(1); nextCluster <= clustersLen; nextCluster++ {

				clusters = append(clusters, clustersOff)
				clustersOff++
			}
			clusterPtr += ClusterLenB + ClusterOffsB

		} else {
			break
		}
	}
	return clusters
}

func (record MFTrecord) hasResidentDataAttr() bool {
	attribute := record.findAttribute("DATA")
	return attribute != nil && !attribute.IsNoNResident()
}

func (record MFTrecord) getType() string {
	return MFTflags[record.Flags]
}

func (record MFTrecord) getRunList() []uint64 {
	for _, attribute := range record.Attributes {
		if attribute.IsNoNResident() {
			return attribute.GetHeader().ATRrecordNoNResID.RunList
		}
	}
	return nil
}

func (record MFTrecord) ShowRunList() {
	runlists := record.getRunList()
	runs := ""
	for cluster := range runlists {
		runs += fmt.Sprintf("%d ", cluster)
	}
}

func (record MFTrecord) hasDataAttr() bool {
	return record.findAttribute("DATA") != nil
}

func (record MFTrecord) ShowIsResident() {
	if record.hasDataAttr() {
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
	fnattr := record.findAttribute("FileName").(*MFTAttributes.FNAttribute)
	fmt.Printf("%s ", fnattr.Mtime.ConvertToIsoTime())
}

func (record MFTrecord) ShowFNACreationTime() {
	fnattr := record.findAttribute("FileName").(*MFTAttributes.FNAttribute)
	fmt.Printf("%s ", fnattr.Crtime.ConvertToIsoTime())
}

func (record MFTrecord) ShowFNAMFTModifiedTime() {
	fnattr := record.findAttribute("FileName").(*MFTAttributes.FNAttribute)
	fmt.Printf("%s ", fnattr.MFTmtime.ConvertToIsoTime())
}

func (record MFTrecord) ShowFNAMFTAccessTime() {
	fnattr := record.findAttribute("FileName").(*MFTAttributes.FNAttribute)
	fmt.Printf("%s ", fnattr.Atime.ConvertToIsoTime())
}

func (record MFTrecord) CreateFileFromEntry(exportFiles string) {

	if (exportFiles == "Resident" || exportFiles == "All") &&
		record.hasResidentDataAttr() {
		fnattr := record.findAttribute("FileName").(*MFTAttributes.FNAttribute)
		data := record.findAttribute("DATA").(*MFTAttributes.DATA)
		utils.WriteFile(fnattr.Fname, data.Content)

	} else if (exportFiles == "NoNResident" || exportFiles == "All") &&
		!record.hasResidentDataAttr() {

	}

}

func (record *MFTrecord) Process(bs []byte) {

	utils.Unmarshal(bs, record)

	if record.Signature == "BAAD" { //skip bad entry
		return
	}

	ReadPtr := record.AttrOff //offset to first attribute
	fmt.Printf("\n Processing $MFT entry %d %s ", record.Entry, record.getType())
	var attributes []MFTAttributes.Attribute
	for ReadPtr < 1024 {

		if utils.Hexify(bs[ReadPtr:ReadPtr+4]) == "ffffffff" { //End of attributes
			break
		}

		var attrHeader MFTAttributes.AttributeHeader
		utils.Unmarshal(bs[ReadPtr:ReadPtr+16], &attrHeader)

		fmt.Printf("%s ", attrHeader.GetType())

		if attrHeader.IsLast() { // End of attributes
			break
		}

		if !attrHeader.IsNoNResident() { //Resident Attribute
			var atrRecordResident MFTAttributes.ATRrecordResident
			utils.Unmarshal(bs[ReadPtr+16:ReadPtr+24], &atrRecordResident)

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
						attrLen+uint16(attrList.Nameoffset)+uint16(attrList.Namelen)])
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
				var idxRoot *MFTAttributes.IndexRoot
				utils.Unmarshal(bs[ReadPtr+atrRecordResident.OffsetContent:ReadPtr+
					atrRecordResident.OffsetContent+12], idxRoot)

				var nodeheader MFTAttributes.NodeHeader
				utils.Unmarshal(bs[ReadPtr+atrRecordResident.OffsetContent+
					16:ReadPtr+atrRecordResident.OffsetContent+32], &nodeheader)
				idxRoot.Nodeheader = &nodeheader

				var idxEntry MFTAttributes.IndexEntry
				utils.Unmarshal(bs[ReadPtr+atrRecordResident.OffsetContent+16+
					uint16(nodeheader.OffsetEntryList):ReadPtr+atrRecordResident.OffsetContent+
					16+uint16(nodeheader.OffsetEndUsedEntryList)], &idxEntry)
				//

				if idxEntry.FilenameLen > 0 {
					var fnattrIDXEntry MFTAttributes.FNAttribute
					utils.Unmarshal(bs[ReadPtr+atrRecordResident.OffsetContent+16+
						uint16(nodeheader.OffsetEntryList)+16:ReadPtr+atrRecordResident.OffsetContent+16+
						uint16(nodeheader.OffsetEntryList)+16+idxEntry.FilenameLen],
						&fnattrIDXEntry)

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

			ReadPtr = ReadPtr + uint16(attrHeader.AttrLen)

		} else { //NoN Resident Attribute
			var atrNoNRecordResident MFTAttributes.ATRrecordNoNResident
			utils.Unmarshal(bs[ReadPtr+16:ReadPtr+64], &atrNoNRecordResident)

			if uint32(ReadPtr)+attrHeader.AttrLen <= 1024 {
				atrNoNRecordResident.RunList = ProcessRunList(bs[ReadPtr+
					atrNoNRecordResident.RunOff : uint32(ReadPtr)+attrHeader.AttrLen])

			}

			/*else if  atrRecordResident.Type == "000000a0" {//Index Allcation
					 nodeheader := NodeHeader {readEndian(bs[ReadPtr+atrRecordResident.OffsetContent+16:ReadPtr+atrRecordResident.OffsetContent+20]).(uint32),readEndian(bs[ReadPtr+atrRecordResident.OffsetContent+20:ReadPtr+atrRecordResident.OffsetContent+24]).(uint32),
					readEndian(bs[ReadPtr+atrRecordResident.OffsetContent+24:ReadPtr+atrRecordResident.OffsetContent+28]).(uint32)}
					IDxall := IndexAllocation{string(bs[ReadPtr+atrRecordResident.OffsetContent:ReadPtr+atrRecordResident.OffsetContent+4]),readEndian(bs[ReadPtr+atrRecordResident.OffsetContent+4:ReadPtr+atrRecordResident.OffsetContent+6]).(uint16),readEndian(bs[ReadPtr+atrRecordResident.OffsetContent+6:ReadPtr+atrRecordResident.OffsetContent+8]).(uint16),
					readEndian(bs[ReadPtr+atrRecordResident.OffsetContent+16:ReadPtr+atrRecordResident.OffsetContent+24]).(uint64), nodeheader}

				 s := [] string  {"Index Allocation Type ",IDxall.Type,fmt.Sprintf("VCN %d  ",IDxall.VCN),"Index entry start",fmt.Sprintf("%d",IDxall.nodeheader.OffsetEntryList),
						fmt.Sprintf(" used portion ends at %d",IDxall.nodeheader.OffsetEndUsedEntryList),fmt.Sprintf("allocated ends at %d",IDxall.nodeheader.OffsetEndEntryListBuffer)  ,string(10)}
				  _,err:=file1.WriteString(strings.Join(s," "))
				  if err != nil {
					// handle the error here
					fmt.Printf("err %s\n",err)
					  return
					}

			   }*/

			ReadPtr = ReadPtr + uint16(attrHeader.AttrLen)

		} //ends non Resident

	} //ends while
	record.Attributes = attributes
}

func (record MFTrecord) ShowFileName() {
	fnattr := record.findAttribute("FileName").(*MFTAttributes.FNAttribute)
	fmt.Printf("%s ", fnattr.Fname)

}

func (record MFTrecord) GetBasicInfoFromRecord(file1 *os.File) {

	s := fmt.Sprintf("%d;%d;%s", record.Entry, record.Seq, record.getType())
	fnattr := record.findAttribute("FileName").(*MFTAttributes.FNAttribute)

	s1 := strings.Join([]string{s, fnattr.Atime.ConvertToIsoTime(),
		fnattr.Crtime.ConvertToIsoTime(),
		fnattr.Mtime.ConvertToIsoTime(), fnattr.Fname,
		fmt.Sprintf(";%d;%d;%s\n", fnattr.ParRef, fnattr.ParSeq,
			fnattr.GetType())}, ";")

	utils.WriteToCSV(file1, s1)

	//	false, false, record.Entry, 0}
	//  fmt.Println("\nFNA ",bs[ReadPtr+atrRecordResident.OffsetContent:ReadPtr+atrRecordResident.OffsetContent+65],bs[ReadPtr+atrRecordResident.OffsetContent:ReadPtr+atrRecordResident.OffsetContent+6],readEndian(bs[ReadPtr+atrRecordResident.OffsetContent:ReadPtr+atrRecordResident.OffsetContent+6]).(uint64),
	//	"PAREF",fnattr.ParRef,"SQ",fnattr.fname,"FLAG",fnattr.flags)
	//   fmt.Printf("time Mod %s time Accessed %s time Created %s Filename %s\n ", fnattr.atime.convertToIsoTime(),fnattr.crtime.convertToIsoTime(),fnattr.mtime.convertToIsoTime(),fnattr.fname )
	//    fmt.Println(strings.TrimSpace(string(bs[ReadPtr+atrRecordResident.OffsetContent+66:ReadPtr+atrRecordResident.OffsetContent+66+2*uint16(readEndian(bs[ReadPtr+atrRecordResident.OffsetContent+64:ReadPtr+atrRecordResident.OffsetContent+65]).(uint8))])))
	/*if *save2DB {
						dbmap.Insert(&fnattr)
						checkErr(err, "Insert failed")
					}



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

	_, err := file1.WriteString(strings.Join(s, " "))
	if err != nil {
	// handle the error here
	fmt.Printf("err %s\n", err)
	return
	s1 := []string{"Filename idx Entry", fnattrIDXEntry.Fname}
	file1.WriteString(strings.Join(s1, " "))
	}

	_, err := file1.WriteString(strings.Join(s, " "))
	if err != nil {
	// handle the error here
	fmt.Printf("err %s\n", err)
	return

	s := []string{fmt.Sprintf(";%d", startpoint), ";", siattr.Crtime.convertToIsoTime(),
	";", siattr.Atime.convertToIsoTime(), ";", siattr.Mtime.convertToIsoTime(), ";",
	siattr.MFTmtime.convertToIsoTime()}
	writeToCSV(file1, strings.Join(s, ""))

	s := []string{";", AttrTypes[atrNoNRecordResident.Type], fmt.Sprintf(";%d", ReadPtr), ";false", fmt.Sprintf(";%d;%d", atrNoNRecordResident.StartVcn, atrNoNRecordResident.LastVcn)}
	_, err := file1.WriteString(strings.Join(s, ""))
	if err != nil {
		// handle the error here
		fmt.Printf("err %s\n", err)
		return
	}

				//s := [] string {fmt.Sprintf("Start VCN %d END VCN %d",atrRecordResident.StartVcn,atrRecordResident.LastVcn ), string(10)}
				// _,err:=file1.WriteString(strings.Join(s," "))
				//  if err != nil {
				// handle the error here
				//   fmt.Printf("err %s\n",err)
				//     return
				//  }

	*/
}
