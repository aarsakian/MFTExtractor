package main

import (
	"fmt"

	gptLib "github.com/aarsakian/MFTExtractor/GPT"
	mbrLib "github.com/aarsakian/MFTExtractor/MBR"
	ntfsLib "github.com/aarsakian/MFTExtractor/NTFS"
	"github.com/aarsakian/MFTExtractor/img"
)

type Disk struct {
	physicalDriveNum int
	partitionNum     int
}

type Partition interface {
	GetOffset() uint64
	LocateFileSystem(int) ntfsLib.NTFS
}

func (disk Disk) GetPhysicalPath() string {
	return fmt.Sprintf("\\\\.\\PHYSICALDRIVE%d", disk.physicalDriveNum)
}

func (disk Disk) GetHandler() img.DiskReader {
	return img.GetHandler(disk.GetPhysicalPath())
}

func (disk Disk) GetPartition() Partition {

	mbr := mbrLib.Parse(disk.physicalDriveNum)

	if mbr.IsProtective() {

		gpt := gptLib.Parse(disk.physicalDriveNum)
		return gpt.GetPartition(disk.partitionNum)
	} else {
		return mbr.GetPartition(disk.partitionNum)
	}
}
