package disk

import (
	"fmt"

	ewfLib "github.com/aarsakian/EWF_Reader/ewf"
	gptLib "github.com/aarsakian/MFTExtractor/GPT"
	mbrLib "github.com/aarsakian/MFTExtractor/MBR"
	ntfsLib "github.com/aarsakian/MFTExtractor/NTFS"
	"github.com/aarsakian/MFTExtractor/img"
)

type Disk struct {
	PhysicalDriveNum int
	PartitionNum     int
	Image            *ewfLib.EWF_Image
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
	var mbr mbrLib.MBR
	physicalOffset := int64(0)
	length := uint32(512) // MBR always at first sector
	buffer := make([]byte, length)
	if disk.Image != nil {
		buffer = disk.Image.RetrieveData(physicalOffset, int64(length))

	} else {
		hD := img.GetHandler(fmt.Sprintf("\\\\.\\PHYSICALDRIVE%d", disk.PhysicalDriveNum))
		hD.ReadFile(physicalOffset, buffer) // read 1st sector
		defer hD.CloseHandler()
	}

	mbr.Parse(buffer)

	if mbr.IsProtective() {

		physicalOffset = int64(512) // gpt always starts at 512
		length = uint32(512)
		buffer = make([]byte, length)
		if disk.Image != nil {
			buffer = disk.Image.RetrieveData(physicalOffset, int64(length))
		} else {
			hD := img.GetHandler(fmt.Sprintf("\\\\.\\PHYSICALDRIVE%d", disk.PhysicalDriveNum))
			hD.ReadFile(physicalOffset, buffer)
			defer hD.CloseHandler()

		}

		var gpt gptLib.GPT
		gpt.ParseHeader(buffer)

		buffer = make([]byte, gpt.GetPartitionArraySize())

		if disk.Image != nil {
			buffer = disk.Image.RetrieveData(physicalOffset, int64(length))
		} else {
			hD := img.GetHandler(fmt.Sprintf("\\\\.\\PHYSICALDRIVE%d", disk.PhysicalDriveNum))
			hD.ReadFile(int64(gpt.Header.PartitionsStartLBA*512), buffer)
			defer hD.CloseHandler()
		}

		gpt.ParsePartitions(buffer)

		return gpt.GetPartition(disk.PartitionNum)
	} else {
		return mbr.GetPartition(disk.PartitionNum)
	}
}
