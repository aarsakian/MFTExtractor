package ntfs

import (
	"fmt"

	"github.com/aarsakian/MFTExtractor/img"
	"github.com/aarsakian/MFTExtractor/utils"
)

type NTFS struct {
	JumpInstruction           [3]byte //0-3
	Signature                 string  //4 bytes NTFS 3-7
	NotUsed1                  [4]byte
	BytesPerSector            uint16       // 11-13
	SectorsPerCluster         uint8        //13
	NotUsed2                  [26]byte     //13-39
	TotalSectors              uint64       //39-47
	MFTOffset                 uint64       //48-56
	MFTMirrOffset             uint64       //56-64
	MFTrunlistOffsetsAndSizes *map[int]int //points to $MFT
}

func (ntfs NTFS) GetSectorsPerCluster() uint8 {
	return ntfs.SectorsPerCluster
}

func (ntfs NTFS) GetMFTEntry(hD img.DiskReader, partitionOffset uint32,
	recordOffset int) []byte {

	length := uint32(1024) // len of MFT record

	buffer := make([]byte, length)
	mftOffset := int64(ntfs.MFTOffset * uint64(ntfs.SectorsPerCluster) * 512)
	offsetedRecord := recordOffset
	lengthB := 0
	if offsetedRecord > 0 {
		for offset, len := range *ntfs.MFTrunlistOffsetsAndSizes {
			lengthB += len * int(ntfs.SectorsPerCluster) * 512
			if offsetedRecord >= lengthB { // more than the available clusters in the contiguous area
				mftOffset = int64(offset * int(ntfs.SectorsPerCluster) * 512)
				offsetedRecord -= lengthB
				break
			}

		}

	}

	logicalOffsetB := int64(mftOffset) + int64(offsetedRecord)
	physicalOffset := int64(partitionOffset)*512 + logicalOffsetB
	hD.ReadFile(physicalOffset, buffer)
	return buffer

}

func Parse(drive int, partitionOffset uint32) NTFS {
	physicalOffset := int64(partitionOffset * 512)
	length := uint32(512)

	hD := img.GetHandler(fmt.Sprintf("\\\\.\\PHYSICALDRIVE%d", drive))
	buffer := make([]byte, length)

	hD.ReadFile(physicalOffset, buffer)

	defer hD.CloseHandler()
	var ntfs NTFS
	utils.Unmarshal(buffer, &ntfs)

	return ntfs
}
