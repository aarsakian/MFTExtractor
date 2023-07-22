package disk

import (
	"fmt"

	gptLib "github.com/aarsakian/MFTExtractor/GPT"
	mbrLib "github.com/aarsakian/MFTExtractor/MBR"
	ntfsLib "github.com/aarsakian/MFTExtractor/NTFS"
	"github.com/aarsakian/MFTExtractor/img"
)

type Disk struct {
	PhysicalDriveNum int
	PartitionNum     int
}

type Partition interface {
	GetOffset() uint64
	LocateFileSystem(int) ntfsLib.NTFS
}

func (disk Disk) GetPhysicalPath() string {
	return fmt.Sprintf("\\\\.\\PHYSICALDRIVE%d", disk.PhysicalDriveNum)
}

func (disk Disk) GetHandler() img.DiskReader {
	return img.GetHandler(disk.GetPhysicalPath())
}

func (disk Disk) GetPartition() Partition {

	mbr := mbrLib.Parse(disk.PhysicalDriveNum)

	if mbr.IsProtective() {

		gpt := gptLib.Parse(disk.PhysicalDriveNum)
		return gpt.GetPartition(disk.PartitionNum)
	} else {
		return mbr.GetPartition(disk.PartitionNum)
	}
}
