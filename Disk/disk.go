package disk

import (
	"fmt"

	"github.com/aarsakian/MFTExtractor/FS"
	gptLib "github.com/aarsakian/MFTExtractor/Partition/GPT"
	mbrLib "github.com/aarsakian/MFTExtractor/Partition/MBR"
	"github.com/aarsakian/MFTExtractor/img"
)

type Disk struct {
	MBR *mbrLib.MBR
	GPT *gptLib.GPT
}

func (disk Disk) hasProtectiveMBR() bool {
	return disk.MBR.IsProtective()
}

type Partitions []Partition

type Partition interface {
	GetOffset() uint64
	LocateFileSystem(img.DiskReader) FS.FileSystem
}

func (disk *Disk) populateMBR(hD img.DiskReader) {
	var mbr mbrLib.MBR
	physicalOffset := int64(0)
	length := int(512) // MBR always at first sector

	data := hD.ReadFile(physicalOffset, length) // read 1st sector

	mbr.Parse(data)

	disk.MBR = &mbr

}

func (disk *Disk) populateGPT(hD img.DiskReader) {

	physicalOffset := int64(512) // gpt always starts at 512

	data := hD.ReadFile(physicalOffset, 512)

	var gpt gptLib.GPT
	gpt.ParseHeader(data)
	length := gpt.GetPartitionArraySize()

	data = hD.ReadFile(int64(gpt.Header.PartitionsStartLBA*512), int(length))

	gpt.ParsePartitions(data)

	disk.GPT = &gpt
}

func (disk *Disk) DiscoverPartitions(hD img.DiskReader) {

	disk.populateMBR(hD)
	if disk.hasProtectiveMBR() {
		disk.populateGPT(hD)
	}

}

func (disk Disk) GetSelectedPartition(partitionNum int) Partition {
	var partition Partition
	if disk.hasProtectiveMBR() {
		partition = disk.GPT.GetPartition(partitionNum)
	} else {
		partition = disk.MBR.GetPartition(partitionNum)
	}
	return partition

}

func (disk Disk) ListPartitions() {

	if disk.hasProtectiveMBR() {

		partitions := disk.GPT.Partitions
		for _, partition := range partitions {
			fmt.Printf("%s\n", partition.GetPartitionType())
		}
	} else {
		partitions := disk.MBR.Partitions
		for _, partition := range partitions {
			fmt.Printf("%s\n", partition.GetPartitionType())
		}

	}

}
