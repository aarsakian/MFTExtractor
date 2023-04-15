package ntfs

import (
	"fmt"

	"github.com/aarsakian/MFTExtractor/img"
	"github.com/aarsakian/MFTExtractor/utils"
)

type NTFS struct {
	JumpInstruction   [3]byte //0-3
	Signature         string  //4 bytes NTFS 3-7
	NotUsed1          [4]byte
	BytesPerSector    uint16   // 11-13
	SectorsPerCluster uint8    //13
	NotUsed2          [26]byte //13-39
	TotalSectors      uint64   //39-47
	MFTOffset         uint64   //48-56
	MFTMirrOffset     uint64   //56-64
}

func (ntfs NTFS) GetSectorsPerCluster() uint8 {
	return ntfs.SectorsPerCluster
}

func Parse(drive int, partitionOffset uint32) NTFS {
	offset := int64(partitionOffset * 512)
	length := uint32(512)

	hD := img.GetHandler(fmt.Sprintf("\\\\.\\PHYSICALDRIVE%d", drive))
	buffer := make([]byte, length)

	hD.ReadFile(offset, buffer)

	defer hD.CloseHandler()
	var ntfs NTFS
	utils.Unmarshal(buffer, &ntfs)

	return ntfs
}
