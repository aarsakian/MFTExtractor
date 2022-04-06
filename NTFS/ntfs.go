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

func Parse(driveNumber int) NTFS {
	offset := int64(124769275 * 512)
	length := uint32(512)
	hD := img.GetHandler("\\\\.\\PHYSICALDRIVE" + fmt.Sprintf("%d", driveNumber))

	defer img.CloseHandler(hD)
	data := img.ReadDisk(hD, offset, length)
	var ntfs NTFS
	utils.Unmarshal(data, &ntfs)
	return ntfs
}
