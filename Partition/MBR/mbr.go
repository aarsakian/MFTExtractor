package MBR

import (
	ntfsLib "github.com/aarsakian/MFTExtractor/NTFS"
	"github.com/aarsakian/MFTExtractor/utils"
)

type Partitions []Partition

type MBR struct {
	BootCode   [446]byte //0-445
	Partitions Partitions
	Signature  [2]byte //510-511
}

type Partition struct {
	Flag     uint8
	StartCHS [3]byte
	Type     uint8
	EndCHS   [3]byte
	StartLBA uint32
	Size     uint32 //sectors

}

func (partition Partition) GetOffset() uint64 {
	return uint64(partition.StartLBA)
}

func (partition Partition) LocateFileSystem(physicalDriveNum int) ntfsLib.NTFS {
	return ntfsLib.Parse(physicalDriveNum, uint64(partition.StartLBA))
}

func (mbr MBR) IsProtective() bool {
	return mbr.Partitions[0].Type == 0xEE // 1st partition flag
}

func (mbr MBR) GetPartition(partitionNum int) Partition {
	return mbr.Partitions[partitionNum]
}

func LocatePartitions(data []byte) Partitions {
	pos := 0
	var partitions Partitions
	for pos < len(data) {
		var partition *Partition = new(Partition) //explicit is better
		utils.Unmarshal(data[pos:pos+16], partition)
		partitions = append(partitions, *partition)
		pos += 16
	}
	return partitions
}

func (mbr *MBR) Parse(buffer []byte) {

	utils.Unmarshal(buffer, &mbr)
	mbr.Partitions = LocatePartitions(buffer[446:510])

}
