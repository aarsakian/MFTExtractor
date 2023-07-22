package MBR

import (
	"fmt"

	"github.com/aarsakian/MFTExtractor/img"
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

func (mbr MBR) IsProtective() bool {
	return mbr.Partitions[0].Type == 0xEE // 1st partition flag
}

func (mbr MBR) GetPartitionOffset(partitionNum int) uint32 {
	return mbr.Partitions[partitionNum].StartLBA
}

func GetPartitions(data []byte) Partitions {
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

func Parse(drive int) MBR {
	var mbr MBR

	offset := int64(0)
	length := uint32(512) // MBR always at first sector

	hD := img.GetHandler(fmt.Sprintf("\\\\.\\PHYSICALDRIVE%d", drive))
	buffer := make([]byte, length)

	hD.ReadFile(offset, buffer) // read 1st sector

	defer hD.CloseHandler()

	utils.Unmarshal(buffer, &mbr)
	mbr.Partitions = GetPartitions(buffer[446:510])

	return mbr
}
