package attributes

import "github.com/aarsakian/MFTExtractor/utils"

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
