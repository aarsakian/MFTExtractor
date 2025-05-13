package lvmlib

import "github.com/aarsakian/MFTExtractor/utils"

type LVM2 struct {
	Header *PhysicalVolLabel
}

type PhysicalVolLabel struct {
	PhysicalVolLabelHeader *PhysicalVolLabelHeader
	PhysicalVolHeader      *PhysicalVolHeader
}

// 32bytes
type PhysicalVolLabelHeader struct {
	LVMSignature  string
	SectorNum     uint8
	Chksum        [4]byte //20offst ot end
	HeaderSize    uint32
	IndicatorType string //8bytes
}

// 40+
type PhysicalVolHeader struct {
	UUID                [32]byte
	VolSize             uint64
	DataAreaDescriptors []DataAreaDescriptor
}

type DataAreaDescriptor struct {
	OffsetB int64 //from volume
	LenB    uint64
}

type MetadataAreaHeader struct {
	Chksum                 [4]byte
	Signature              string //LVM2
	Version                uint32
	Offset                 int64
	Size                   int64
	RawLocationDescriptors [4]RawLocationDescriptor
}

type RawLocationDescriptor struct {
	Offset int64
	Len    uint64
	Chksum [4]byte
	Flags  uint64
}

func (lvm2 *LVM2) Parse(data []byte) {
	header := new(PhysicalVolLabel)
	header.Parse(data[512:])
	lvm2.Header = header

}

func (physicalVolHeader *PhysicalVolHeader) Parse(data []byte) {

	offset, _ := utils.Unmarshal(data[32:], physicalVolHeader)

	dataDescriptor := new(DataAreaDescriptor)
	currOffset, _ := utils.Unmarshal(data[32+offset:], dataDescriptor)

	for dataDescriptor.LenB != 0 && dataDescriptor.OffsetB != 0 {
		physicalVolHeader.DataAreaDescriptors = append(physicalVolHeader.DataAreaDescriptors, *dataDescriptor)
		offset, _ := utils.Unmarshal(data[32+currOffset:], dataDescriptor)

		currOffset += offset

	}

}

func (header *PhysicalVolLabel) Parse(data []byte) {
	phyVolLabelHeader := new(PhysicalVolLabelHeader)
	utils.Unmarshal(data[:32], phyVolLabelHeader)
	physicalVolHeader := new(PhysicalVolHeader)
	physicalVolHeader.Parse(data[32:])

	header.PhysicalVolLabelHeader = phyVolLabelHeader
	header.PhysicalVolHeader = physicalVolHeader

}
