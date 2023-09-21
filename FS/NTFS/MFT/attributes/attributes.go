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
	"00000100": "Logged Utility Stream",
	"ffffffff": "Last",
}

type AttributeHeader struct {
	Type                 string //        0-3                              type of attribute e.g. $DATA
	AttrLen              uint16 //4-8             length of attribute??? practice shown 4-6
	Uknown               [2]byte
	NoNResident          uint8  //8
	Nlen                 uint8  //9
	NameOff              uint16 //name offset 10-12          relative to the start of attribute
	Flags                uint16 //12-14           //compressed,
	ID                   uint16 //14-16 type of attribute
	ATRrecordResident    *ATRrecordResident
	ATRrecordNoNResident *ATRrecordNoNResident
}

type ATRrecordResident struct {
	ContentSize   uint32 //16-20 size of Resident attribute
	OffsetContent uint16 //20-22 offset to content            soff+ssize=len
	IdxFlags      uint16
	Name          string
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

type VolumeInfo struct {
	F1     uint64 //unused
	MajVer string // 8-8
	MinVer string // 9-9
	Flags  uint16 //see table 13.22
	F2     uint32
	Header *AttributeHeader
}

func (atrRecordResident *ATRrecordResident) Parse(data []byte) {
	utils.Unmarshal(data[:8], atrRecordResident)
}

func (objectId *ObjectID) SetHeader(header *AttributeHeader) {
	objectId.Header = header
}

func (objectId ObjectID) GetHeader() AttributeHeader {
	return *objectId.Header
}

func (objectId *ObjectID) Parse(data []byte) {
	utils.Unmarshal(data, objectId)
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

func (bitmap *BitMap) SetHeader(header *AttributeHeader) {
	bitmap.Header = header
}

func (bitmap BitMap) GetHeader() AttributeHeader {
	return *bitmap.Header
}

func (bitmap *BitMap) Parse(data []byte) {
	bitmap.AllocationStatus = data
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

func (reparse *Reparse) SetHeader(header *AttributeHeader) {
	reparse.Header = header
}

func (reparse Reparse) GetHeader() AttributeHeader {
	return *reparse.Header
}

func (reparse *Reparse) Parse(data []byte) {
	utils.Unmarshal(data[:16], reparse)

	reparse.Name = utils.DecodeUTF16(data[16+
		uint16(reparse.TargetNameOffset) : 16+uint16(reparse.TargetNameOffset)+reparse.TargetLen])
	reparse.PrintName = utils.DecodeUTF16(data[16+uint16(reparse.TargetPrintNameOffset) : 16+
		uint16(reparse.TargetPrintNameLen)])
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

func (volInfo *VolumeInfo) Parse(data []byte) {
	utils.Unmarshal(data, volInfo)
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

func (volName *VolumeName) Parse(data []byte) {
	volName.Name = utils.NoNull(data)

}

func (volName VolumeName) FindType() string {
	return volName.Header.GetType()
}

func (volName VolumeName) IsNoNResident() bool {
	return volName.Header.IsNoNResident()
}

func (volName VolumeName) ShowInfo() {

}

func (attrHeader AttributeHeader) GetType() string {
	attrType, ok := AttrTypes[attrHeader.Type]
	if ok {
		return attrType
	} else {
		return fmt.Sprintf("%x \n", attrHeader.Type)
	}
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
	return attrHeader.GetType() == "Volume Information"
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
