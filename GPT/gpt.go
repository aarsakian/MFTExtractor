package gpt

import (
	"fmt"

	ntfsLib "github.com/aarsakian/MFTExtractor/NTFS"
	"github.com/aarsakian/MFTExtractor/img"
	"github.com/aarsakian/MFTExtractor/utils"
)

type Partitions []Partition

type GPT struct {
	Header     *GPTHeader
	Partitions Partitions
}
type GPTHeader struct {
	StartSignature     [8]byte
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

func Parse(drive int) GPT {

	var gpt GPT

	physicalOffset := int64(512) // gpt always starts at 512
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

	gpt.LocatePartitions(buffer)
	return gpt

}

func (gpt *GPT) LocatePartitions(data []byte) {

	partitions := make([]Partition, gpt.Header.NofPartitions)

	for idx := 0; idx < len(partitions); idx++ {
		var partition Partition
		utils.Unmarshal(data[idx*int(gpt.Header.PartitionSize):(idx+1)*int(gpt.Header.PartitionSize)], &partition)
		partitions[idx] = partition
	}
	gpt.Partitions = partitions
}

func (gpt GPT) GetPartition(partitionNum int) Partition {
	return gpt.Partitions[partitionNum]
}

func (partition Partition) GetOffset() uint64 {
	return partition.StartLBA
}

func (partition Partition) LocateFileSystem(physicalDriveNum int) ntfsLib.NTFS {
	return ntfsLib.Parse(physicalDriveNum, uint64(partition.StartLBA))
}
