package disk

import (
	"fmt"

	"github.com/aarsakian/MFTExtractor/FS"
	"github.com/aarsakian/MFTExtractor/FS/NTFS/MFT"
	gptLib "github.com/aarsakian/MFTExtractor/Partition/GPT"
	mbrLib "github.com/aarsakian/MFTExtractor/Partition/MBR"
	"github.com/aarsakian/MFTExtractor/img"
)

type Disk struct {
	MBR         *mbrLib.MBR
	GPT         *gptLib.GPT
	Handler     img.DiskReader
	Partitions  []Partition
	FileSystems []FS.FileSystem
}

func (disk Disk) hasProtectiveMBR() bool {
	return disk.MBR.IsProtective()
}

type Partition interface {
	GetOffset() uint64
	LocateFileSystem(img.DiskReader) FS.FileSystem
}

func (disk *Disk) populateMBR() {
	var mbr mbrLib.MBR
	physicalOffset := int64(0)
	length := int(512) // MBR always at first sector

	data := disk.Handler.ReadFile(physicalOffset, length) // read 1st sector

	mbr.Parse(data)

	disk.MBR = &mbr

}

func (disk *Disk) populateGPT() {

	physicalOffset := int64(512) // gpt always starts at 512

	data := disk.Handler.ReadFile(physicalOffset, 512)

	var gpt gptLib.GPT
	gpt.ParseHeader(data)
	length := gpt.GetPartitionArraySize()

	data = disk.Handler.ReadFile(int64(gpt.Header.PartitionsStartLBA*512), int(length))

	gpt.ParsePartitions(data)

	disk.GPT = &gpt
}

func (disk *Disk) DiscoverPartitions() {

	disk.populateMBR()
	if disk.hasProtectiveMBR() {
		disk.populateGPT()
		for _, partition := range disk.GPT.Partitions {
			disk.Partitions = append(disk.Partitions, partition)
		}

	} else {
		for _, partition := range disk.MBR.Partitions {
			disk.Partitions = append(disk.Partitions, partition)
		}
	}

}

func (disk *Disk) ProcessPartitions(partitionNum int, MFTSelectedEntry int, fromMFTEntry int, toMFTEntry int) {

	for idx, partition := range disk.Partitions {
		if partitionNum != -1 && idx != partitionNum {
			continue
		}

		filesystem := partition.LocateFileSystem(disk.Handler)
		partitionOffsetB := int64(partition.GetOffset() * filesystem.GetBytesPerSector())
		filesystem.Process(disk.Handler, partitionOffsetB, MFTSelectedEntry, fromMFTEntry, toMFTEntry)

		disk.FileSystems = append(disk.FileSystems, filesystem)
	}

}

func (disk Disk) GetFileSystemMetadata() []MFT.Record {
	var records []MFT.Record
	for _, filesystem := range disk.FileSystems {
		records = append(records, filesystem.GetMetadata()...)
	}
	return records
}

func (disk Disk) GetFileContents() map[string][]byte {

	for idx, filesystem := range disk.FileSystems {
		partitionOffsetB := int64(disk.Partitions[idx].GetOffset() * filesystem.GetBytesPerSector())
		return filesystem.GetFileContents(disk.Handler, partitionOffsetB)
	}
	return map[string][]byte{}

}

func (disk Disk) GetSelectedPartition(partitionNum int) Partition {

	if disk.hasProtectiveMBR() {
		return disk.GPT.GetPartition(partitionNum)
	} else {
		return disk.MBR.GetPartition(partitionNum)
	}

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
