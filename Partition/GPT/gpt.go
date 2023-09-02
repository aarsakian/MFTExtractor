package gpt

import (
	"fmt"

	"github.com/aarsakian/MFTExtractor/FS"
	ntfsLib "github.com/aarsakian/MFTExtractor/FS/NTFS"
	"github.com/aarsakian/MFTExtractor/utils"
)

var PartitionTypeGuids = map[string]string{
	"Basic Data": "",
}

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

func (partition Partition) GetPartitionType() string {
	return fmt.Sprintf("%x-%x-%x-%x-%x", utils.Bytereverse(partition.PartitionTypeGUID[0:4]),
		utils.Bytereverse(partition.PartitionTypeGUID[4:6]), utils.Bytereverse(partition.PartitionTypeGUID[6:8]),
		partition.PartitionTypeGUID[8:10], partition.PartitionTypeGUID[10:])
}

func (partition Partition) IdentifyType() string {
	return PartitionTypeGuids[partition.GetPartitionType()]
}

func (gpt *GPT) ParseHeader(buffer []byte) {

	var header GPTHeader
	utils.Unmarshal(buffer, &header)
	gpt.Header = &header

}

func (gpt GPT) GetPartitionArraySize() uint32 {
	return gpt.Header.PartitionSize * gpt.Header.NofPartitions
}

func (gpt *GPT) ParsePartitions(data []byte) {

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

func (partition Partition) LocateFileSystem(buffer []byte) FS.FileSystem {
	var ntfs ntfsLib.NTFS
	ntfs.Parse(buffer)
	return ntfs
}
