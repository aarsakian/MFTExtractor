package disk

import (
	"fmt"
	"sync"

	"github.com/aarsakian/MFTExtractor/FS"
	"github.com/aarsakian/MFTExtractor/FS/NTFS/MFT"
	gptLib "github.com/aarsakian/MFTExtractor/disk/partition/GPT"
	mbrLib "github.com/aarsakian/MFTExtractor/disk/partition/MBR"
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

func (disk *Disk) Initialize(evidencefile string, physicaldrive int, vmdkfile string) {
	var hD img.DiskReader
	if evidencefile != "" {

		hD = img.GetHandler(evidencefile, "ewf")

	} else if physicaldrive != -1 {

		hD = img.GetHandler(fmt.Sprintf("\\\\.\\PHYSICALDRIVE%d", physicaldrive), "physicalDrive")

	} else {

		hD = img.GetHandler(vmdkfile, "vmdk")

	}
	disk.Handler = hD
}

func (disk *Disk) Process(partitionNum int, MFTentries []int, fromMFTEntry int, toMFTEntry int) map[int]MFT.Records {

	disk.DiscoverPartitions()

	filesystemsOffsetMap := disk.ProcessPartitions(partitionNum)

	for fileSystemOffset, fs := range filesystemsOffsetMap {
		partitionOffsetB := int64(fileSystemOffset * fs.GetBytesPerSector())

		fs.Process(disk.Handler, partitionOffsetB, MFTentries, fromMFTEntry, toMFTEntry)
	}

	return disk.GetFileSystemMetadata(partitionNum)
}

func (disk Disk) Close() {
	disk.Handler.CloseHandler()
}

func (disk Disk) hasProtectiveMBR() bool {
	return disk.MBR.IsProtective()
}

type FileSystemOffsetMap map[uint64]FS.FileSystem

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
	offset, err := mbr.GetExtendedPartitionOffset()
	if err == nil {
		data := disk.Handler.ReadFile(physicalOffset+int64(offset)*512, length)
		mbr.DiscoverExtendedPartitions(data, offset)

	}
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
		for idx := range disk.MBR.ExtendedPartitions {
			disk.Partitions = append(disk.Partitions, &disk.MBR.ExtendedPartitions[idx])
		}
	}

}

func (disk *Disk) ProcessPartitions(partitionNum int) FileSystemOffsetMap {
	filesystems := make(FileSystemOffsetMap)

	for idx := range disk.Partitions {
		if partitionNum != -1 && idx+1 != partitionNum {
			continue
		}

		disk.Partitions[idx].LocateFileSystem(disk.Handler)
		parttionOffset := disk.Partitions[idx].GetOffset()
		fs := disk.Partitions[idx].GetFileSystem()
		if fs == nil {
			msg := "No Known File System found at partition %d (Currently supported NTFS)."
			logger.MFTExtractorlogger.Error(fmt.Sprintf(msg, idx))
			continue //fs not found
		}
		msg := "Partition %d  %s at %d sector"
		fmt.Printf(msg+"\n", idx+1, fs.GetSignature(), parttionOffset)
		logger.MFTExtractorlogger.Info(fmt.Sprintf(msg, idx+1, fs.GetSignature(), parttionOffset))

		filesystems[parttionOffset] = fs
	}

	return filesystems
}

func (disk Disk) GetFileSystemMetadata(partitionNum int) map[int]MFT.Records {

	recordsPerPartition := map[int]MFT.Records{}
	for idx, partition := range disk.Partitions {
		if partitionNum != -1 && idx+1 != partitionNum {
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

func (disk Disk) AsyncWorker(wg *sync.WaitGroup, record MFT.Record, dataClusters chan<- []byte, partitionNum int) {
	defer wg.Done()
	partition := disk.Partitions[partitionNum]

	fs := partition.GetFileSystem()
	sectorsPerCluster := int(fs.GetSectorsPerCluster())
	bytesPerSector := int(fs.GetBytesPerSector())
	partitionOffsetB := int64(partition.GetOffset()) * int64(bytesPerSector)

	if record.IsFolder() {
		msg := fmt.Sprintf("Record %s Id %d is folder! No data to export.", record.GetFname(), record.Entry)
		logger.MFTExtractorlogger.Warning(msg)
		close(dataClusters)
		return
	}
	fmt.Printf("pulling data file %s Id %d\n", record.GetFname(), record.Entry)

	if len(record.LinkedRecords) == 0 {
		record.LocateDataAsync(disk.Handler, partitionOffsetB, sectorsPerCluster, bytesPerSector, dataClusters)
	} else { // attribute runlist

		for _, linkedRecord := range record.LinkedRecords {
			linkedRecord.LocateDataAsync(disk.Handler, partitionOffsetB, sectorsPerCluster, bytesPerSector, dataClusters)

		}
	}
	// use lsize to make sure that we cannot exceed the logical size

	close(dataClusters)

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

		if len(record.LinkedRecords) == 0 {
			record.LocateData(disk.Handler, partitionOffsetB, sectorsPerCluster, bytesPerSector, results)
		} else { // attribute runlist

			for _, linkedRecord := range record.LinkedRecords {
				linkedRecord.LocateData(disk.Handler, partitionOffsetB, sectorsPerCluster, bytesPerSector, results)

			}
		}
		// use lsize to make sure that we cannot exceed the logical size

	}
	close(results)

}

func (disk Disk) ListPartitions() {

	if disk.hasProtectiveMBR() {
		fmt.Printf("GPT:\n")

		for idx, partition := range disk.GPT.Partitions {
			fmt.Printf("%d %s\n", idx, partition.GetPartitionType())
		}
	} else {
		fmt.Printf("MBR:\n")

		for idx, partition := range disk.MBR.Partitions {
			fmt.Printf("%d  %s at %d\n", idx, partition.GetPartitionType(), partition.GetOffset())
		}
		for idx, extPartition := range disk.MBR.ExtendedPartitions {
			fmt.Printf("extended %d  %s at %d\n", idx, extPartition.Partition.GetPartitionType(), extPartition.Partition.GetOffset())
		}

	}

}

func (disk Disk) CollectedUnallocated(blocks chan<- []byte) {
	for _, partition := range disk.Partitions {

		fs := partition.GetFileSystem()
		if fs == nil {
			continue
		}
		bytesPerSector := int(fs.GetBytesPerSector())
		partitionOffsetB := int64(partition.GetOffset()) * int64(bytesPerSector)

		fs.CollectUnallocated(disk.Handler, partitionOffsetB, blocks)
	}
}
