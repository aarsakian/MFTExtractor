package attributes

import (
	"fmt"

	"github.com/aarsakian/MFTExtractor/utils"
)

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
