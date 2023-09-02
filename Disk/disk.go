package disk

import (
	"fmt"

	ewfLib "github.com/aarsakian/EWF_Reader/ewf"
	"github.com/aarsakian/MFTExtractor/FS"
	gptLib "github.com/aarsakian/MFTExtractor/Partition/GPT"
	mbrLib "github.com/aarsakian/MFTExtractor/Partition/MBR"
	"github.com/aarsakian/MFTExtractor/img"
)

type Disk struct {
	PhysicalDriveNum int
	Image            *ewfLib.EWF_Image
	MBR              *mbrLib.MBR
	GPT              *gptLib.GPT
}

func (disk Disk) GetPhysicalPath() string {
	return fmt.Sprintf("\\\\.\\PHYSICALDRIVE%d", disk.PhysicalDriveNum)
}

func (disk Disk) GetHandler() img.DiskReader {
	return img.GetHandler(disk.GetPhysicalPath())
}

func (disk Disk) hasProtectiveMBR() bool {
	return disk.MBR.IsProtective()
}

type Partitions []Partition

type Partition interface {
	GetOffset() uint64
	LocateFileSystem([]byte) FS.FileSystem
}

func (disk *Disk) populateMBR() {
	var mbr mbrLib.MBR
	physicalOffset := int64(0)
	length := uint32(512) // MBR always at first sector
	buffer := make([]byte, length)
	if disk.Image != nil {
		buffer = disk.Image.RetrieveData(physicalOffset, int64(length))

	} else {
		hD := img.GetHandler(disk.GetPhysicalPath())
		hD.ReadFile(physicalOffset, buffer) // read 1st sector
		defer hD.CloseHandler()
	}

	mbr.Parse(buffer)

	disk.MBR = &mbr

}

func (disk *Disk) populateGPT() {

	physicalOffset := int64(512) // gpt always starts at 512
	length := uint32(512)
	buffer := make([]byte, length)
	if disk.Image != nil {
		buffer = disk.Image.RetrieveData(physicalOffset, int64(length))
	} else {
		hD := img.GetHandler(disk.GetPhysicalPath())
		hD.ReadFile(physicalOffset, buffer)
		defer hD.CloseHandler()

	}

	var gpt gptLib.GPT
	gpt.ParseHeader(buffer)
	length = gpt.GetPartitionArraySize()
	buffer = make([]byte, length)

	if disk.Image != nil {
		buffer = disk.Image.RetrieveData(physicalOffset, int64(length))
	} else {
		hD := img.GetHandler(disk.GetPhysicalPath())
		hD.ReadFile(int64(gpt.Header.PartitionsStartLBA*512), buffer)
		defer hD.CloseHandler()
	}

	gpt.ParsePartitions(buffer)

	disk.GPT = &gpt
}

func (disk *Disk) Populate() {
	disk.populateMBR()
	if disk.hasProtectiveMBR() {
		disk.populateGPT()
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
