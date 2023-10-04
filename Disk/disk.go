package disk

import (
	"bytes"
	"fmt"
	"sync"

	"github.com/aarsakian/MFTExtractor/FS"
	"github.com/aarsakian/MFTExtractor/FS/NTFS/MFT"
	gptLib "github.com/aarsakian/MFTExtractor/Partition/GPT"
	mbrLib "github.com/aarsakian/MFTExtractor/Partition/MBR"
	"github.com/aarsakian/MFTExtractor/img"
	"github.com/aarsakian/MFTExtractor/utils"
)

type Disk struct {
	MBR        *mbrLib.MBR
	GPT        *gptLib.GPT
	Handler    img.DiskReader
	Partitions []Partition
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
			continue //fs not found
		}
		partitionOffsetB := int64(disk.Partitions[idx].GetOffset() * fs.GetBytesPerSector())
		fs.Process(disk.Handler, partitionOffsetB, MFTSelectedEntries, fromMFTEntry, toMFTEntry)

	}

}

func (disk Disk) GetFileSystemMetadata(partitionNum int) []MFT.Record {
	var records []MFT.Record
	for idx, partition := range disk.Partitions {
		if idx != partitionNum {
			continue
		}
		fs := partition.GetFileSystem()
		records = append(records, fs.GetMetadata()...)
	}
	return records
}

func (disk Disk) Worker(wg *sync.WaitGroup, records chan MFT.Record, results chan<- utils.AskedFile, partitionNum int) {
	partition := disk.Partitions[partitionNum]
	partitionOffsetB := int64(partition.GetOffset())
	fs := partition.GetFileSystem()
	sectorsPerCluster := int(fs.GetSectorsPerCluster())
	bytesPerSector := int(fs.GetBytesPerSector())
	defer wg.Done()

	for record := range records {
		lsize := record.GetLogicalFileSize()

		var dataRuns bytes.Buffer
		dataRuns.Grow(int(lsize))
		if record.LinkedRecord == nil {
			record.LocateData(disk.Handler, partitionOffsetB, sectorsPerCluster, bytesPerSector, &dataRuns)
		} else { // attribute runlist
			record := record.LinkedRecord
			for record != nil {
				record.LocateData(disk.Handler, partitionOffsetB, sectorsPerCluster, bytesPerSector, &dataRuns)
				record = record.LinkedRecord
			}
		}

		results <- utils.AskedFile{Fname: record.GetFname(), Content: dataRuns.Bytes()}
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
