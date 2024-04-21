package disk

import (
	"fmt"
	"sync"

	"github.com/aarsakian/MFTExtractor/FS"
	"github.com/aarsakian/MFTExtractor/FS/NTFS/MFT"
	gptLib "github.com/aarsakian/MFTExtractor/Partition/GPT"
	mbrLib "github.com/aarsakian/MFTExtractor/Partition/MBR"
	"github.com/aarsakian/MFTExtractor/img"
	"github.com/aarsakian/MFTExtractor/logger"
	"github.com/aarsakian/MFTExtractor/utils"
)

type Disk struct {
	MBR        *mbrLib.MBR
	GPT        *gptLib.GPT
	Handler    img.DiskReader
	Partitions []Partition
}

func InitiliazeEvidence(evidencefile string) Disk {
	var hD img.DiskReader
	hD = img.GetHandler(evidencefile, "ewf")
	return Disk{Handler: hD}

}

func InitializePhysicalDisk(physicalDrive int) Disk {
	var hD img.DiskReader
	hD = img.GetHandler(fmt.Sprintf("\\\\.\\PHYSICALDRIVE%d", physicalDrive), "physicalDrive")
	return Disk{Handler: hD}
}

func InitalizeVMDKDisk(vmdkfile string) Disk {
	var hD img.DiskReader
	hD = img.GetHandler(vmdkfile, "vmdk")
	return Disk{Handler: hD}
}

func (disk Disk) Close() {
	disk.Handler.CloseHandler()
}

func (disk Disk) hasProtectiveMBR() bool {
	return disk.MBR.IsProtective()
}

type Partition interface {
	GetOffset() uint64
	LocateFileSystem(img.DiskReader)
	GetFileSystem() FS.FileSystem
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
		for idx := range disk.GPT.Partitions {

			disk.Partitions = append(disk.Partitions, &disk.GPT.Partitions[idx])

		}

	} else {
		for idx := range disk.MBR.Partitions {
			disk.Partitions = append(disk.Partitions, &disk.MBR.Partitions[idx])
		}
	}

}

func (disk *Disk) ProcessPartitions(partitionNum int, MFTSelectedEntries []int, fromMFTEntry int, toMFTEntry int) {

	for idx := range disk.Partitions {
		if partitionNum != -1 && idx != partitionNum {
			continue
		}

		disk.Partitions[idx].LocateFileSystem(disk.Handler)
		fs := disk.Partitions[idx].GetFileSystem()
		if fs == nil {
			msg := "No Known File System found at partition %d (Currently supported NTFS)."
			fmt.Printf(msg+"\n", idx)
			logger.MFTExtractorlogger.Error(fmt.Sprintf(msg, idx))
			continue //fs not found
		}
		partitionOffsetB := int64(disk.Partitions[idx].GetOffset() * fs.GetBytesPerSector())
		msg := "Located %s at %d bytes."
		fmt.Printf(msg+"\n", fs.GetSignature(), partitionOffsetB)
		logger.MFTExtractorlogger.Error(fmt.Sprintf(msg, fs.GetSignature(), partitionOffsetB))

		fs.Process(disk.Handler, partitionOffsetB, MFTSelectedEntries, fromMFTEntry, toMFTEntry)

	}

}

func (disk Disk) GetFileSystemMetadata(partitionNum int) map[int]MFT.Records {

	recordsPerPartition := map[int]MFT.Records{}
	for idx, partition := range disk.Partitions {
		if partitionNum != -1 && idx != partitionNum {
			continue
		}
		fs := partition.GetFileSystem()
		if fs == nil {
			continue
		}
		recordsPerPartition[idx] = fs.GetMetadata()

	}
	return recordsPerPartition
}

func (disk Disk) Worker(wg *sync.WaitGroup, records MFT.Records, results chan<- utils.AskedFile, partitionNum int) {
	defer wg.Done()
	partition := disk.Partitions[partitionNum]

	fs := partition.GetFileSystem()
	sectorsPerCluster := int(fs.GetSectorsPerCluster())
	bytesPerSector := int(fs.GetBytesPerSector())
	partitionOffsetB := int64(partition.GetOffset()) * int64(bytesPerSector)

	for _, record := range records {

		if record.IsFolder() {
			msg := fmt.Sprintf("Record %s Id %d is folder! No data to export.", record.GetFname(), record.Entry)
			logger.MFTExtractorlogger.Warning(msg)
			continue
		}
		fmt.Printf("pulling data file %s Id %d\n", record.GetFname(), record.Entry)

		if record.LinkedRecord == nil {
			record.LocateData(disk.Handler, partitionOffsetB, sectorsPerCluster, bytesPerSector, results)
		} else { // attribute runlist
			record := record.LinkedRecord
			for record.LinkedRecord != nil {
				record.LocateData(disk.Handler, partitionOffsetB, sectorsPerCluster, bytesPerSector, results)
				record = record.LinkedRecord
			}
		}
		// use lsize to make sure that we cannot exceed the logical size

	}
	close(results)

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

func (disk Disk) CollectedUnallocated(wg *sync.WaitGroup, blocks chan<- []byte) {
	for _, partition := range disk.Partitions {

		fs := partition.GetFileSystem()
		if fs == nil {
			continue
		}
		bytesPerSector := int(fs.GetBytesPerSector())
		partitionOffsetB := int64(partition.GetOffset()) * int64(bytesPerSector)

		fs.CollectUnallocated(wg, disk.Handler, partitionOffsetB, blocks)
	}
}
