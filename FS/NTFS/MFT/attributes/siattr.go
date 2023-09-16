package attributes

import (
	"fmt"

	"github.com/aarsakian/MFTExtractor/utils"
)

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

func (siattr SIAttribute) GetTimestamps() (string, string, string, string) {
	atime := siattr.Atime.ConvertToIsoTime()
	ctime := siattr.Crtime.ConvertToIsoTime()
	mtime := siattr.Mtime.ConvertToIsoTime()
	mftime := siattr.MFTmtime.ConvertToIsoTime()
	return atime, ctime, mtime, mftime
}

func (siattr SIAttribute) ShowInfo() {
	atime, ctime, mtime, mfttime := siattr.GetTimestamps()
	typeAttr := siattr.FindType()
	fmt.Printf("type %s usn  %d atime %s ctime %s mtime %s mfttime %s\n",
		typeAttr, siattr.Usn, atime, ctime, mtime, mfttime)
}
