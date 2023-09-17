package attributes

import (
	"fmt"

	"github.com/aarsakian/MFTExtractor/utils"
)

var AttrTypes = map[string]string{
	"00000010": "Standard Information", "00000020": "Attribute List",
	"00000030": "FileName", "00000040": "Object ID",
	"00000050": "Security Descriptor", "00000060": "Volume Name",
	"00000070": "Volume Information", "00000080": "DATA",
	"00000090": "Index Root", "000000a0": "Index Allocation",
	"000000b0": "BitMap", "000000c0": "Reparse Point",
	"000000e0": "Extended Attribute", "000000f0": "Extended Attribute Information",
	"ffffffff": "Last",
}

type AttributeHeader struct {
	Type                 string //        0-3                              type of attribute e.g. $DATA
	AttrLen              uint32 //4-8             length of attribute
	NoNResident          uint8  //8
	Nlen                 string
	NameOff              uint16 //name offset 10-12          relative to the start of attribute
	Flags                uint16 //12-14           //compressed,
	ID                   uint16 //14-16 type of attribute
	ATRrecordResident    *ATRrecordResident
	ATRrecordNoNResident *ATRrecordNoNResident
}

type ATRrecordResident struct {
	ContentSize   uint32 //16-20 size of Resident attribute
	OffsetContent uint16 //20-22 offset to content            soff+ssize=len
	IDxflag       uint16 //22-24
}

type ATRrecordNoNResident struct {
	StartVcn     uint64   //16-24
	LastVcn      uint64   //24-32
	RunOff       uint16   //32-34     offset to the start of the attribute
	Compusize    uint16   //34-36
	F1           uint32   //36-40
	Length       uint64   //40-48
	ActualLength uint64   //48-56
	InitLength   uint64   //56-64
	RunList      *RunList //holds a linked list of runs

}

type Reparse struct {
	Flags                 uint32
	Size                  uint16
	Unused                [2]byte
	TargetNameOffset      int16
	TargetLen             uint16
	TargetPrintNameOffset int16
	TargetPrintNameLen    uint16
	Header                *AttributeHeader
	Name                  string
	PrintName             string
}

type RunList struct {
	Offset int64
	Length uint64
	Next   *RunList
}

type ObjectID struct { //unique guID
	ObjID     string //object ID
	OrigVolID string //volume ID
	OrigObjID string //original objID
	OrigDomID string // domain ID
	Header    *AttributeHeader
}

type BitMap struct {
	AllocationStatus []byte
	Header           *AttributeHeader
}

type VolumeName struct {
	Name   utils.NoNull
	Header *AttributeHeader
}

type IndexEntry struct {
	ParRef      uint64
	ParSeq      uint16
	Len         uint16 //8-9
	FilenameLen uint16 //10-11
	Flags       uint32 //12-15
	Fnattr      *FNAttribute
}

type IndexRoot struct {
	Type                 string //0-3 similar to FNA type
	CollationSortingRule string //4-7
	Sizebytes            uint32 //8-11
	Sizeclusters         uint8  //12-12
	Nodeheader           *NodeHeader
	Header               *AttributeHeader
	IndexEntries         []IndexEntry
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
	Nodeheader       *NodeHeader
	Header           *AttributeHeader
	IndexEntries     []IndexEntry
}

type VolumeInfo struct {
	F1     uint64 //unused
	MajVer string // 8-8
	MinVer string // 9-9
	Flags  uint16 //see table 13.22
	F2     uint32
	Header *AttributeHeader
}

func (idxEntry IndexEntry) ShowInfo() {
	if idxEntry.Fnattr != nil {
		fmt.Printf("type %s file ref %d idx name %s flags %d \n", idxEntry.Fnattr.GetType(), idxEntry.ParRef,
			idxEntry.Fnattr.Fname, idxEntry.Flags)
	}

}

func (objectId *ObjectID) SetHeader(header *AttributeHeader) {
	objectId.Header = header
}

func (objectId ObjectID) GetHeader() AttributeHeader {
	return *objectId.Header
}

func (objectId ObjectID) FindType() string {
	return objectId.Header.GetType()
}

func (objectId ObjectID) IsNoNResident() bool {
	return objectId.Header.IsNoNResident()
}

func (objectId ObjectID) ShowInfo() {
	fmt.Printf("type %s\n", objectId.FindType())
}

func (idxAllocation *IndexAllocation) SetHeader(header *AttributeHeader) {
	idxAllocation.Header = header
}

func (idxAllocation IndexAllocation) GetHeader() AttributeHeader {
	return *idxAllocation.Header
}

func (idxAllocation IndexAllocation) FindType() string {
	return idxAllocation.Header.GetType()
}

func (idxAllocation IndexAllocation) IsNoNResident() bool {
	return idxAllocation.Header.IsNoNResident()
}

func (idxAllocation IndexAllocation) ShowInfo() {
	fmt.Printf("type %s nof entries %d\n", idxAllocation.FindType(), idxAllocation.NumEntries)
	for _, idxEntry := range idxAllocation.IndexEntries {
		idxEntry.ShowInfo()
	}
}

func (bitmap *BitMap) SetHeader(header *AttributeHeader) {
	bitmap.Header = header
}

func (bitmap BitMap) GetHeader() AttributeHeader {
	return *bitmap.Header
}

func (bitmap BitMap) FindType() string {
	return bitmap.Header.GetType()
}

func (bitmap BitMap) IsNoNResident() bool {
	return bitmap.Header.IsNoNResident()
}

func (bitmap BitMap) ShowInfo() {
	fmt.Printf("type %s \n", bitmap.FindType())
	pos := 1
	for _, byteval := range bitmap.AllocationStatus {
		bitmask := uint8(0x01)
		shifter := 0
		for bitmask < 128 {

			bitmask = 1 << shifter
			fmt.Printf("cluster/entry  %d status %d \t", pos, byteval&bitmask)
			pos++
			shifter++
		}

	}
}

func (idxAllocation *IndexAllocation) Parse(bs []byte) {
	utils.Unmarshal(bs[:24], idxAllocation)

	var nodeheader *NodeHeader = new(NodeHeader)
	utils.Unmarshal(bs[24:24+16], nodeheader)
	idxAllocation.Nodeheader = nodeheader

	idxEntryOffset := nodeheader.OffsetEntryList + 24 // relative to the start of node header
	for idxEntryOffset < nodeheader.OffsetEndUsedEntryList {
		var idxEntry *IndexEntry = new(IndexEntry)
		utils.Unmarshal(bs[idxEntryOffset:idxEntryOffset+16], idxEntry)
		if idxEntry.FilenameLen > 0 {
			var fnattrIDXEntry FNAttribute
			utils.Unmarshal(bs[idxEntryOffset+16:idxEntryOffset+16+uint32(idxEntry.FilenameLen)],
				&fnattrIDXEntry)

			fnattrIDXEntry.Fname =
				utils.DecodeUTF16(bs[idxEntryOffset+16+66 : idxEntryOffset+16+
					66+2*uint32(fnattrIDXEntry.Nlen)])
			idxEntry.Fnattr = &fnattrIDXEntry

		}
		idxEntryOffset += uint32(idxEntry.Len)
		idxAllocation.IndexEntries = append(idxAllocation.IndexEntries, *idxEntry)
	}

}

func (reparse *Reparse) SetHeader(header *AttributeHeader) {
	reparse.Header = header
}

func (reparse Reparse) GetHeader() AttributeHeader {
	return *reparse.Header
}

func (reparse Reparse) IsNoNResident() bool {
	return reparse.Header.IsNoNResident()
}

func (reparse Reparse) FindType() string {
	return reparse.Header.GetType()
}

func (reparse Reparse) ShowInfo() {
	fmt.Printf("Type %s Target Name %s Print Name %s", reparse.FindType(),
		reparse.Name, reparse.PrintName)
}

func (volInfo *VolumeInfo) SetHeader(header *AttributeHeader) {
	volInfo.Header = header
}

func (volInfo VolumeInfo) GetHeader() AttributeHeader {
	return *volInfo.Header
}

func (volInfo VolumeInfo) IsNoNResident() bool {
	return volInfo.Header.IsNoNResident()
}

func (volInfo VolumeInfo) FindType() string {
	return volInfo.Header.GetType()
}

func (volinfo VolumeInfo) ShowInfo() {

}

func (volName *VolumeName) SetHeader(header *AttributeHeader) {
	volName.Header = header
}

func (volName VolumeName) GetHeader() AttributeHeader {
	return *volName.Header
}

func (volName VolumeName) FindType() string {
	return volName.Header.GetType()
}

func (volName VolumeName) IsNoNResident() bool {
	return volName.Header.IsNoNResident()
}

func (volName VolumeName) ShowInfo() {

}

func (idxRoot *IndexRoot) SetHeader(header *AttributeHeader) {
	idxRoot.Header = header
}

func (idxRoot IndexRoot) GetHeader() AttributeHeader {
	return *idxRoot.Header
}

func (idxRoot IndexRoot) IsNoNResident() bool {
	return false // always resident
}

func (idxRoot IndexRoot) FindType() string {
	return idxRoot.Header.GetType()
}

func (idxRoot IndexRoot) ShowInfo() {
	fmt.Printf("type %s nof entries %d\n", idxRoot.FindType(), len(idxRoot.IndexEntries))
	for _, idxEntry := range idxRoot.IndexEntries {
		idxEntry.ShowInfo()
	}
}

func (attrHeader AttributeHeader) GetType() string {
	return AttrTypes[attrHeader.Type]
}

func (attrHeader AttributeHeader) IsLast() bool {
	return attrHeader.GetType() == "Last"
}

func (attrHeader AttributeHeader) IsFileName() bool {
	return attrHeader.GetType() == "FileName"
}

func (attrHeader AttributeHeader) IsData() bool {
	return attrHeader.GetType() == "DATA"
}

func (attrHeader AttributeHeader) IsReparse() bool {
	return attrHeader.GetType() == "Reparse Point"
}

func (attrHeader AttributeHeader) IsObject() bool {
	return attrHeader.GetType() == "Object ID"
}

func (attrHeader AttributeHeader) IsAttrList() bool {
	return attrHeader.GetType() == "Attribute List"
}

func (attrHeader AttributeHeader) IsBitmap() bool {
	return attrHeader.GetType() == "BitMap"
}

func (attrHeader AttributeHeader) IsVolumeName() bool {
	return attrHeader.GetType() == "Volume Name"
}

func (attrHeader AttributeHeader) IsIndexAllocation() bool {
	return attrHeader.GetType() == "Index Allocation"
}

func (attrHeader AttributeHeader) IsVolumeInfo() bool {
	return attrHeader.GetType() == "Volume Info"
}

func (attrHeader AttributeHeader) IsIndexRoot() bool {
	return attrHeader.GetType() == "Index Root"
}

func (attrHeader AttributeHeader) IsStdInfo() bool {
	return attrHeader.GetType() == "Standard Information"
}

func (attrHeader AttributeHeader) IsNoNResident() bool {
	return attrHeader.NoNResident == 1
}

func (prevRunlist *RunList) Process(runlists []byte) {
	clusterPtr := uint64(0)

	for clusterPtr < uint64(len(runlists)) { // length of bytes of runlist
		ClusterOffsB, ClusterLenB := utils.DetermineClusterOffsetLength(runlists[clusterPtr])

		if ClusterLenB != 0 && ClusterOffsB != 0 {
			clustersLen := utils.ReadEndianUInt(runlists[clusterPtr+1 : clusterPtr+
				ClusterLenB+1])

			clustersOff := utils.ReadEndianInt(runlists[clusterPtr+1+
				ClusterLenB : clusterPtr+ClusterLenB+ClusterOffsB+1])

			runlist := RunList{Offset: clustersOff, Length: clustersLen}

			if clusterPtr == 0 {
				*prevRunlist = runlist
			} else {
				prevRunlist.Next = &runlist
				prevRunlist = &runlist
			}

			//		prevRunlist = runlist
			clusterPtr += ClusterLenB + ClusterOffsB + 1

		} else {
			break
		}
	}

}
