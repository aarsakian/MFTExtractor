package gpt

import (
	"fmt"

	"github.com/aarsakian/MFTExtractor/img"
	"github.com/aarsakian/MFTExtractor/utils"
)

type Partitions []Partition

type GPT struct {
	Header     *GPTHeader
	Partitions Partitions
}
type GPTHeader struct {
	StartSignature     [4]byte
	Revision           [4]byte
	HeaderSize         uint32
	HeaderCRC          uint32
	Reserved           [4]byte
	CurrentLBA         uint64 //location of header
	BackupLBA          uint64
	FirstUsableLBA     uint64
	LastUsableLBA      uint64
	DiskGUID           [16]byte
	PartitionsStartLBA uint64 // usually LBA 2
	NofPartitions      uint32
	PartitionSize      uint32
	PartionArrayCRC    uint32
	Reserved2          [418]byte
	EndSignature       [2]byte //510-511
}

type Partition struct {
	PartitionTypeGUID [16]byte
	PartitionGUID     [16]byte
	StartLBA          uint64
	EndLBA            uint64
	Atttributes       [8]byte
	Name              string
}

func Parse(drive int, partitionOffset uint32) GPT {

	var gpt GPT

	physicalOffset := int64(partitionOffset * 512)
	length := uint32(512)

	hD := img.GetHandler(fmt.Sprintf("\\\\.\\PHYSICALDRIVE%d", drive))
	buffer := make([]byte, length)

	hD.ReadFile(physicalOffset, buffer)

	defer hD.CloseHandler()

	var header GPTHeader
	utils.Unmarshal(buffer, &header)
	gpt.Header = &header

	partitionArraySize := header.PartitionSize * header.NofPartitions

	buffer = make([]byte, partitionArraySize)
	hD.ReadFile(int64(header.PartitionsStartLBA*512), buffer)

	gpt.GetPartitions(buffer)
	return gpt

}

func (gpt *GPT) GetPartitions(data []byte) {

	partitions := make([]Partition, gpt.Header.NofPartitions)

	for idx := 0; idx < len(partitions); idx++ {
		var partition Partition
		utils.Unmarshal(data[idx*int(gpt.Header.PartitionSize):(idx+1)*int(gpt.Header.PartitionSize)], &partition)
		partitions[idx] = partition
	}
	gpt.Partitions = partitions
}
