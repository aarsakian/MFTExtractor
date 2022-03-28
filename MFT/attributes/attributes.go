package attributes

import "github.com/aarsakian/MFTExtractor/utils"

var NameSpaceFlags = map[uint32]string{
	0: "POSIX", 1: "Win32", 2: "DOS", 3: "Win32 & Dos",
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

type Attribute interface {
	FindType() string
	SetHeader(header *AttributeHeader)
	GetHeader() AttributeHeader
	IsNoNResident() bool
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
	Header      *AttributeHeader
}

type ObjectID struct { //unique guID
	ObjID     string //object ID
	OrigVolID string //volume ID
	OrigObjID string //original objID
	OrigDomID string // domain ID
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
	Type                 string //0-3 similar to FNA type
	CollationSortingRule string //4-7
	Sizebytes            uint32 //8-11
	Sizeclusters         uint8  //12-12
	Nodeheader           *NodeHeader
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
	Nodeheader       *NodeHeader
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
	Name       utils.NoNull
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

func (fnattr *FNAttribute) SetHeader(header *AttributeHeader) {
	fnattr.Header = header
}

func (fnattr FNAttribute) GetHeader() AttributeHeader {
	return *fnattr.Header
}

func (fnattr FNAttribute) FindType() string {
	return fnattr.Header.GetType()
}

func (fnAttr FNAttribute) GetType() string {
	return NameSpaceFlags[fnAttr.Flags]
}

func (fnAttr FNAttribute) IsNoNResident() bool {
	return fnAttr.Header.IsNoNResident()
}

func (siattr *SIAttribute) SetHeader(header *AttributeHeader) {
	siattr.Header = header
}

func (siattr SIAttribute) GetHeader() AttributeHeader {
	return *siattr.Header
}

func (siattr SIAttribute) FindType() string {
	return siattr.Header.GetType()
}

func (siattr SIAttribute) IsNoNResident() bool {
	return siattr.Header.IsNoNResident() // always resident
}

func (data *DATA) SetHeader(header *AttributeHeader) {
	data.Header = header
}

func (data DATA) GetHeader() AttributeHeader {
	return *data.Header
}

func (data DATA) FindType() string {
	return data.Header.GetType()
}
func (data DATA) IsNoNResident() bool {
	return data.Header.IsNoNResident()
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

func (attrListEntries *AttributeListEntries) SetHeader(header *AttributeHeader) {
	attrListEntries.Header = header
}

func (attrListEntries AttributeListEntries) GetHeader() AttributeHeader {
	return *attrListEntries.Header
}

func (attrListEntries AttributeListEntries) FindType() string {
	return attrListEntries.Header.GetType()
}

func (attrListEntries AttributeListEntries) IsNoNResident() bool {
	return attrListEntries.Header.IsNoNResident()
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

func (attrHeader AttributeHeader) IsObject() bool {
	return attrHeader.GetType() == "Object ID"
}

func (attrHeader AttributeHeader) IsAttrList() bool {
	return attrHeader.GetType() == "Attribute List"
}

func (attrHeader AttributeHeader) IsBitmap() bool {
	return attrHeader.GetType() == "Bitmap"
}

func (attrHeader AttributeHeader) IsVolumeName() bool {
	return attrHeader.GetType() == "Volume Name"
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
	return attrHeader.NoNResident == "1"
}
