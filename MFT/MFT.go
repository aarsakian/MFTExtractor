package MFT

import (
	"fmt"
	"os"
	"strings"

	"github.com/aarsakian/MFTExtractor/utils"
)

var IndexEntryFlags = map[string]string{
	"00000001": "Child Node exists",
	"00000002": "Last Entry in list",
}

var AttrTypes = map[string]string{
	"00000010": "Standard Information", "00000020": "Attribute List",
	"00000030": "FileName", "00000040": "Object ID",
	"00000050": "Security Descriptor", "00000060": "Volume Name",
	"00000070": "Volume Information", "00000080": "DATA",
	"00000090": "Index Root", "000000A0": "Index Allocation",
	"000000B0": "BitMap", "000000C0": "Reparse Point",
	"ffffffff": "Last",
}

var SIFlags = map[uint32]string{
	1: "Read Only", 2: "Hidden", 4: "System", 32: "Archive", 64: "Device", 128: "Normal",
	256: "Temporary", 512: "Sparse", 1024: "Reparse Point", 2048: "Compressed", 4096: "Offline",
	8192: "Not Indexed", 16384: "Encrypted",
}

var NameSpaceFlags = map[uint32]string{
	0: "POSIX", 1: "Win32", 2: "DOS", 3: "Win32 & Dos",
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
	Attributes         []Attribute
	Bitmap             bool
	// fixupArray add the        UpdateSeqArrOffset to find is location

}

type Attribute interface {
	findType() string
	setHeader(header *AttributeHeader)
	getHeader() AttributeHeader
}

type AttributeHeader struct {
	Type              string //        0-3                              type of attribute e.g. $DATA
	AttrLen           uint32 //4-8             length of attribute
	NoNResident       string //8
	Nlen              string
	NameOff           uint16 //name offset 10-12          relative to the start of attribute
	Flags             uint16 //12-14           //compressed,
	ID                uint16 //14-16 type of attribute
	ATRrecordResident *ATRrecordResident
	ATRrecordNoNResID *ATRrecordNoNResident
}

type ATRrecordResident struct {
	ContentSize   uint32 //16-20 size of Resident attribute
	OffsetContent uint16 //20-22 offset to content            soff+ssize=len
	IDxflag       uint16 //22-24
}

type DATA struct {
	Content []byte
	Header  *AttributeHeader
}

type ATRrecordNoNResident struct {
	StartVcn   uint64   //16-24
	LastVcn    uint64   //24-32
	RunOff     uint16   //32-24     offset to the start of the attribute
	Compusize  uint16   //34-36
	F1         uint32   //36-40
	Alen       uint64   //40-48
	NonRessize uint64   //48-56
	Initsize   uint64   //56-64
	RunList    []uint64 //holds an array of the clusters

}

type FNAttribute struct {
	ParRef      uint64
	ParSeq      uint16
	Crtime      utils.WindowsTime
	Mtime       utils.WindowsTime //WindowsTime
	MFTmtime    utils.WindowsTime //WindowsTime
	Atime       utils.WindowsTime //WindowsTime
	AllocFsize  uint64
	RealFsize   uint64
	Flags       uint32 //hIDden Read Only? check Reparse
	Reparse     uint32
	Nlen        uint8  //length of name
	Nspace      uint8  //format of name
	Fname       string //special string type without nulls
	HexFlag     bool
	UnicodeHack bool
	EntryID     uint32 //foreing key
	AttrID      uint16 //for DB use
	Header      *AttributeHeader
}

type ObjectID struct { //unique guID
	ObjID     string //object ID
	OrigVolID string //volume ID
	OrigObjID string //original objID
	OrigDomID string // domain ID
	EntryID   uint32 //foreing key
	AttrID    uint16
	Header    *AttributeHeader
}

type VolumeName struct {
	Name   utils.NoNull
	Header *AttributeHeader
}

type IndexEntry struct {
	MFTfileref  uint64 //0-7
	Len         uint16 //8-9
	FilenameLen uint16 //10-11
	Flags       uint32 //12-15
	Fnattr      *FNAttribute
}

type IndexRoot struct {
	Type                 string //0-4 similar to FNA type
	CollationSortingRule string
	Sizebytes            uint32 //8-12
	Sizeclusters         uint8  //12-12
	nodeheader           *NodeHeader
	Header               *AttributeHeader
}

type NodeHeader struct {
	OffsetEntryList          uint32 // 16-20 offset to start of the index entry
	OffsetEndUsedEntryList   uint32 //20-24 where EntryList ends
	OffsetEndEntryListBuffer uint32 //24-28
	Flags                    uint32 //0x01 no children
}

type IndexAllocation struct {
	Signature        string //0-4
	FixupArrayOffset int16  //4-6
	NumEntries       int16  //6-8
	LSN              int64  //8-16
	VCN              int64  //16-24 where the record fits in the tree
	nodeheader       *NodeHeader
}

type AttributeListEntries struct {
	Entries []AttributeList
	Header  *AttributeHeader
}

type AttributeList struct { //more than one MFT entry to store a file/directory its attributes
	Type       string //        typeif 0-4    # 4
	Len        uint16 //4-6
	Namelen    uint8  //7unsigned char           # 1
	Nameoffset uint8  //8-8               # 1
	StartVcn   uint64 //8-16         # 8
	FileRef    uint64 //16-22      # 6
	Seq        uint16 //       22-24    # 2
	ID         uint8  //     24-26   # 4
	name       utils.NoNull
}

type VolumeInfo struct {
	F1     uint64 //unused
	MajVer string // 8-8
	MinVer string // 9-9
	Flags  uint16 //see table 13.22
	F2     uint32
	Header *AttributeHeader
}

type SIAttribute struct {
	Crtime   utils.WindowsTime
	Mtime    utils.WindowsTime
	MFTmtime utils.WindowsTime
	Atime    utils.WindowsTime
	Dos      uint32
	Maxver   uint32
	Ver      uint32
	ClassID  uint32
	OwnID    uint32
	SecID    uint32
	Quota    uint64
	Usn      uint64
	Header   *AttributeHeader
}

func (record MFTrecord) findAttribute(attributeName string) interface{} {
	for _, attribute := range record.Attributes {
		if attribute.findType() == attributeName {
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

func (fnattr *FNAttribute) setHeader(header *AttributeHeader) {
	fnattr.Header = header
}

func (fnattr FNAttribute) getHeader() AttributeHeader {
	return *fnattr.Header
}

func (fnattr FNAttribute) findType() string {
	return fnattr.Header.getType()
}

func (siattr *SIAttribute) setHeader(header *AttributeHeader) {
	siattr.Header = header
}

func (siattr SIAttribute) getHeader() AttributeHeader {
	return *siattr.Header
}

func (siattr SIAttribute) findType() string {
	return siattr.Header.getType()
}

func (data *DATA) setHeader(header *AttributeHeader) {
	data.Header = header
}

func (data DATA) getHeader() AttributeHeader {
	return *data.Header
}

func (data DATA) findType() string {
	return data.Header.getType()
}

func (objectId ObjectID) setHeader(header *AttributeHeader) {
	objectId.Header = header
}

func (objectId ObjectID) getHeader() AttributeHeader {
	return *objectId.Header
}

func (objectId ObjectID) findType() string {
	return objectId.Header.getType()
}

func (volInfo *VolumeInfo) setHeader(header *AttributeHeader) {
	volInfo.Header = header
}

func (volInfo VolumeInfo) getHeader() AttributeHeader {
	return *volInfo.Header
}

func (volInfo VolumeInfo) findType() string {
	return volInfo.Header.getType()
}

func (volName *VolumeName) setHeader(header *AttributeHeader) {
	volName.Header = header
}

func (volName VolumeName) getHeader() AttributeHeader {
	return *volName.Header
}

func (volName VolumeName) findType() string {
	return volName.Header.getType()
}

func (attrListEntries *AttributeListEntries) setHeader(header *AttributeHeader) {
	attrListEntries.Header = header
}

func (attrListEntries AttributeListEntries) getHeader() AttributeHeader {
	return *attrListEntries.Header
}

func (attrListEntries AttributeListEntries) findType() string {
	return attrListEntries.Header.getType()
}

func (idxRoot *IndexRoot) setHeader(header *AttributeHeader) {
	idxRoot.Header = header
}

func (idxRoot IndexRoot) getHeader() AttributeHeader {
	return *idxRoot.Header
}

func (idxRoot IndexRoot) findType() string {
	return idxRoot.Header.getType()
}

func (attrHeader AttributeHeader) getType() string {
	return AttrTypes[attrHeader.Type]
}

func (attrHeader AttributeHeader) isNoNResident() bool {
	return attrHeader.NoNResident == "1"

}

func (attrHeader AttributeHeader) isLast() bool {
	return attrHeader.getType() == "Last"
}

func (attrHeader AttributeHeader) isFileName() bool {
	return attrHeader.getType() == "FileName"
}

func (attrHeader AttributeHeader) isData() bool {
	return attrHeader.getType() == "DATA"
}

func (attrHeader AttributeHeader) isObject() bool {
	return attrHeader.getType() == "Object ID"
}

func (attrHeader AttributeHeader) isAttrList() bool {
	return attrHeader.getType() == "Attribute List"
}

func (attrHeader AttributeHeader) isBitmap() bool {
	return attrHeader.getType() == "Bitmap"
}

func (attrHeader AttributeHeader) isVolumeName() bool {
	return attrHeader.getType() == "Volume Name"
}

func (attrHeader AttributeHeader) isVolumeInfo() bool {
	return attrHeader.getType() == "Volume Info"
}

func (attrHeader AttributeHeader) isIndexRoot() bool {
	return attrHeader.getType() == "Index Root"
}

func (attrHeader AttributeHeader) isStdInfo() bool {
	return attrHeader.getType() == "Standard Information"
}

func (fnAttr FNAttribute) getType() string {
	return NameSpaceFlags[fnAttr.Flags]
}

func (record MFTrecord) hasResidentDataAttr() bool {
	attribute := record.findAttribute("DATA")
	return attribute != nil && !attribute.(*DATA).getHeader().isNoNResident()
}

func (record MFTrecord) getType() string {
	return MFTflags[record.Flags]
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
	fnattr := record.findAttribute("FileName").(*FNAttribute)
	fmt.Printf("%s ", fnattr.Mtime.ConvertToIsoTime())
}

func (record MFTrecord) ShowFNACreationTime() {
	fnattr := record.findAttribute("FileName").(*FNAttribute)
	fmt.Printf("%s ", fnattr.Crtime.ConvertToIsoTime())
}

func (record MFTrecord) ShowFNAMFTModifiedTime() {
	fnattr := record.findAttribute("FileName").(*FNAttribute)
	fmt.Printf("%s ", fnattr.MFTmtime.ConvertToIsoTime())
}

func (record MFTrecord) ShowFNAMFTAccessTime() {
	fnattr := record.findAttribute("FileName").(*FNAttribute)
	fmt.Printf("%s ", fnattr.Atime.ConvertToIsoTime())
}

func (record MFTrecord) CreateFileFromEntry(exportFiles string) {

	if (exportFiles == "Resident" || exportFiles == "All") &&
		record.hasResidentDataAttr() {
		fnattr := record.findAttribute("FileName").(*FNAttribute)
		data := record.findAttribute("DATA").(DATA)
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
	var attributes []Attribute
	for ReadPtr < 1024 {

		if utils.Hexify(bs[ReadPtr:ReadPtr+4]) == "ffffffff" { //End of attributes
			break
		}

		var attrHeader AttributeHeader
		utils.Unmarshal(bs[ReadPtr:ReadPtr+16], &attrHeader)

		fmt.Printf("%s ", attrHeader.getType())

		if attrHeader.isLast() { // End of attributes
			break
		}

		if !attrHeader.isNoNResident() { //Resident Attribute
			var atrRecordResident ATRrecordResident
			utils.Unmarshal(bs[ReadPtr+16:ReadPtr+24], &atrRecordResident)

			if attrHeader.isFileName() { // File name
				var fnattr *FNAttribute = new(FNAttribute)
				utils.Unmarshal(bs[ReadPtr+atrRecordResident.OffsetContent:ReadPtr+
					atrRecordResident.OffsetContent+66], fnattr)

				fnattr.Fname =
					utils.DecodeUTF16(bs[ReadPtr+atrRecordResident.OffsetContent+66 : ReadPtr+
						atrRecordResident.OffsetContent+66+2*uint16(fnattr.Nlen)])
				fnattr.setHeader(&attrHeader)
				attributes = append(attributes, fnattr)

			} else if attrHeader.isData() {
				data := &DATA{bs[ReadPtr+atrRecordResident.OffsetContent : ReadPtr +
					+uint16(attrHeader.AttrLen)], &attrHeader}
				data.setHeader(&attrHeader)
				attributes = append(attributes, data)

			} else if attrHeader.isObject() {
				var objectattr *ObjectID = new(ObjectID)
				utils.Unmarshal(bs[ReadPtr+atrRecordResident.OffsetContent:ReadPtr+
					atrRecordResident.OffsetContent+64], objectattr)
				objectattr.setHeader(&attrHeader)
				attributes = append(attributes, *objectattr)

			} else if attrHeader.isAttrList() { //Attribute List
				//  runlist:=bs[ReadPtr+atrRecordResident.OffsetContent:uint32(ReadPtr)+atrRecordResident.Len]

				attrLen := uint16(0)
				var attrListEntries *AttributeListEntries = new(AttributeListEntries)
				for atrRecordResident.OffsetContent+24+attrLen < uint16(attrHeader.AttrLen) {
					var attrList AttributeList
					utils.Unmarshal(bs[ReadPtr+atrRecordResident.OffsetContent+attrLen:ReadPtr+
						atrRecordResident.OffsetContent+attrLen+24], &attrList)

					attrList.name = utils.NoNull(bs[ReadPtr+atrRecordResident.OffsetContent+attrLen+
						uint16(attrList.Nameoffset) : ReadPtr+atrRecordResident.OffsetContent+
						attrLen+uint16(attrList.Nameoffset)+uint16(attrList.Namelen)])
					//   runlist=bs[ReadPtr+atrRecordResident.OffsetContent+attrList.len:uint32(ReadPtr)+atrRecordResident.Len]
					attrListEntries.Entries = append(attrListEntries.Entries, attrList)
					attrLen += attrList.Len

				}
				attrListEntries.setHeader(&attrHeader)
				attributes = append(attributes, attrListEntries)

			} else if attrHeader.isBitmap() { //BITMAP
				record.Bitmap = true

			} else if attrHeader.isVolumeName() { //Volume Name
				volumeName := &VolumeName{utils.NoNull(bs[ReadPtr+
					atrRecordResident.OffsetContent : uint32(ReadPtr)+
					uint32(atrRecordResident.OffsetContent)+atrRecordResident.ContentSize]), &attrHeader}
				volumeName.setHeader(&attrHeader)
				attributes = append(attributes, volumeName)

			} else if attrHeader.isVolumeInfo() { //Volume Info
				var volumeInfo *VolumeInfo = new(VolumeInfo)
				utils.Unmarshal(bs[ReadPtr+atrRecordResident.OffsetContent:ReadPtr+
					atrRecordResident.OffsetContent+12], volumeInfo)
				volumeInfo.setHeader(&attrHeader)
				attributes = append(attributes, volumeInfo)

			} else if attrHeader.isIndexRoot() { //Index Root
				var idxRoot *IndexRoot
				utils.Unmarshal(bs[ReadPtr+atrRecordResident.OffsetContent:ReadPtr+
					atrRecordResident.OffsetContent+12], idxRoot)

				var nodeheader NodeHeader
				utils.Unmarshal(bs[ReadPtr+atrRecordResident.OffsetContent+
					16:ReadPtr+atrRecordResident.OffsetContent+32], &nodeheader)
				idxRoot.nodeheader = &nodeheader

				var idxEntry IndexEntry
				utils.Unmarshal(bs[ReadPtr+atrRecordResident.OffsetContent+16+
					uint16(nodeheader.OffsetEntryList):ReadPtr+atrRecordResident.OffsetContent+
					16+uint16(nodeheader.OffsetEndUsedEntryList)], &idxEntry)
				//

				if idxEntry.FilenameLen > 0 {
					var fnattrIDXEntry FNAttribute
					utils.Unmarshal(bs[ReadPtr+atrRecordResident.OffsetContent+16+
						uint16(nodeheader.OffsetEntryList)+16:ReadPtr+atrRecordResident.OffsetContent+16+
						uint16(nodeheader.OffsetEntryList)+16+idxEntry.FilenameLen],
						&fnattrIDXEntry)

				}
				idxRoot.setHeader(&attrHeader)
				attributes = append(attributes, idxRoot)
			} else if attrHeader.isStdInfo() { //Standard Information
				startpoint := ReadPtr + atrRecordResident.OffsetContent
				var siattr *SIAttribute = new(SIAttribute)
				utils.Unmarshal(bs[startpoint:startpoint+72], siattr)
				siattr.setHeader(&attrHeader)
				attributes = append(attributes, siattr)

			}

			ReadPtr = ReadPtr + uint16(attrHeader.AttrLen)

		} else { //NoN Resident Attribute
			var atrNoNRecordResident ATRrecordNoNResident
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
	fnattr := record.findAttribute("FileName").(*FNAttribute)
	fmt.Printf("%s ", fnattr.Fname)

}

func (record MFTrecord) GetBasicInfoFromRecord(file1 *os.File) {

	s := fmt.Sprintf("%d;%d;%s", record.Entry, record.Seq, record.getType())
	fnattr := record.findAttribute("FileName").(*FNAttribute)

	s1 := strings.Join([]string{s, fnattr.Atime.ConvertToIsoTime(),
		fnattr.Crtime.ConvertToIsoTime(),
		fnattr.Mtime.ConvertToIsoTime(), fnattr.Fname,
		fmt.Sprintf(";%d;%d;%s\n", fnattr.ParRef, fnattr.ParSeq,
			fnattr.getType())}, ";")

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
