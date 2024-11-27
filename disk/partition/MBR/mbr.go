package MBR

import (
	"errors"

	FS "github.com/aarsakian/MFTExtractor/FS"
	ntfsLib "github.com/aarsakian/MFTExtractor/FS/NTFS"
	"github.com/aarsakian/MFTExtractor/img"
	"github.com/aarsakian/MFTExtractor/utils"
)

var PartitionTypes = map[uint8]string{0x07: "HPFS/NTFS/exFAT",
	0x0c: "W95 FAT32 (LBA)",
	0x0f: "Extended",
	0x27: "Hidden NTFS Win"}

type MBR struct {
	BootCode           [446]byte //0-445
	Partitions         []Partition
	ExtendedPartitions []ExtendedPartition
	Signature          [2]byte //510-511
}

type ExtendedPartition struct {
	Partition   *Partition
	TableOffset int
}
type Partition struct {
	Flag     uint8
	StartCHS [3]byte
	Type     uint8
	EndCHS   [3]byte
	StartLBA uint32
	Size     uint32 //sectors
	FS       FS.FileSystem
}

func (partition Partition) GetOffset() uint64 {
	return uint64(partition.StartLBA)
}

func (partition Partition) GetPartitionType() string {
	return PartitionTypes[partition.Type]
}

func (partition *Partition) LocateFileSystem(hD img.DiskReader) {
	partitionOffetB := uint64(partition.GetOffset() * 512)
	data := hD.ReadFile(int64(partitionOffetB), 512)
	if partition.Type == 0x07 || partition.Type == 0x17 {
		var ntfs *ntfsLib.NTFS = new(ntfsLib.NTFS)
		ntfs.Parse(data)
		if ntfs.HasValidSignature() {
			partition.FS = ntfs
		} else {
			partition.FS = nil
		}
	}

}

func (extPartition ExtendedPartition) GetOffset() uint64 {
	return uint64(extPartition.Partition.StartLBA) + uint64(extPartition.TableOffset)
}

func (extPartition *ExtendedPartition) LocateFileSystem(hD img.DiskReader) {
	extPartition.Partition.LocateFileSystem(hD)
}

func (mbr MBR) IsProtective() bool {
	return mbr.Partitions[0].Type == 0xEE // 1st partition flag
}

func (mbr MBR) GetPartition(partitionNum int) Partition {
	return mbr.Partitions[partitionNum]
}

func LocatePartitions(data []byte) []Partition {
	pos := 0
	var partitions []Partition
	for pos < len(data) {
		var partition *Partition = new(Partition) //explicit is better
		utils.Unmarshal(data[pos:pos+16], partition)
		partitions = append(partitions, *partition)
		pos += 16
	}

	return partitions
}

func (mbr *MBR) DiscoverExtendedPartitions(buffer []byte, offset int) {
	var extPartitions []ExtendedPartition
	partitions := LocatePartitions(buffer[446:510])
	for idx := range partitions {
		extPartitions = append(extPartitions, ExtendedPartition{Partition: &partitions[idx], TableOffset: offset})
	}
	mbr.ExtendedPartitions = extPartitions
}

func (mbr *MBR) Parse(buffer []byte) {

	utils.Unmarshal(buffer, mbr)
	mbr.Partitions = LocatePartitions(buffer[446:510])

}

func (mbr MBR) GetExtendedPartitionOffset() (int, error) {
	for _, partition := range mbr.Partitions {
		if partition.Type == 0x0f {
			return int(partition.GetOffset()), nil
		}
	}
	return -1, errors.New("extended partition not found")
}

func (mbr *MBR) UpdateExtendedPartitionsOffsets(extendedTableSectorOffset uint32) {
	for idx := range mbr.Partitions {
		if mbr.Partitions[idx].Type != 0x0f {
			continue
		}
		mbr.Partitions[idx].StartLBA += extendedTableSectorOffset
	}
}

func (partiton Partition) GetFileSystem() FS.FileSystem {
	return partiton.FS
}

func (extPartition ExtendedPartition) GetFileSystem() FS.FileSystem {
	return extPartition.Partition.FS
}
