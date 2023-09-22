package attributes

import (
	"fmt"

	"github.com/aarsakian/MFTExtractor/utils"
)

var RecordTypes = map[uint32]string{
	1: "Read Only", 2: "Hidden", 4: "System",
	32: "Archive", 64: "Device", 128: "Normal", 256: "Temporary", 512: "Sparse file",
	1024: "Reparse", 2048: "Compressed", 4096: "Offline",
	8192:  "Content  is not being indexed for faster searches",
	16384: "Encrypted"}

var NameSpaceFlags = map[uint8]string{
	0: "POSIX", 1: "Win32", 2: "Dos", 3: "Win32 & Dos",
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
	atime, ctime, mtime, mfttime := fnattr.GetTimestamps()
	typeAttr := fnattr.FindType()
	fmt.Printf("type %s Par Ref %d name %s atime %s ctime %s mtime %s mfttime %s\n",
		typeAttr, fnattr.ParRef, fnattr.Fname, atime, ctime, mtime, mfttime)
}

func (fnAttr FNAttribute) GetType() string {
	return RecordTypes[fnAttr.Flags]
}

func (fnAttr *FNAttribute) Parse(data []byte) {
	utils.Unmarshal(data[:66], fnAttr)
	fnAttr.Fname = utils.DecodeUTF16(data[66 : 66+2*uint16(fnAttr.Nlen)])
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
