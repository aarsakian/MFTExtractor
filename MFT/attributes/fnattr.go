package attributes

import "github.com/aarsakian/MFTExtractor/utils"

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

func (fnattr *FNAttribute) SetHeader(header *AttributeHeader) {
	fnattr.Header = header
}

func (fnattr FNAttribute) GetHeader() AttributeHeader {
	return *fnattr.Header
}

func (fnattr FNAttribute) FindType() string {
	return fnattr.Header.GetType()
}

func (fnattr FNAttribute) ShowInfo() {

}

func (fnAttr FNAttribute) GetType() string {
	return RecordTypes[fnAttr.Flags]
}

func (fnAttr FNAttribute) GetFileNameType() string {
	return NameSpaceFlags[fnAttr.Nspace]
}

func (fnAttr FNAttribute) GetTimestamps() (string, string, string, string) {
	atime := fnAttr.Atime.ConvertToIsoTime()
	ctime := fnAttr.Crtime.ConvertToIsoTime()
	mtime := fnAttr.Mtime.ConvertToIsoTime()
	mftime := fnAttr.MFTmtime.ConvertToIsoTime()
	return atime, ctime, mtime, mftime
}

func (fnAttr FNAttribute) IsNoNResident() bool {
	return fnAttr.Header.IsNoNResident()
}
