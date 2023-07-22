package ntfs

import (
	"bytes"
	"fmt"

	"github.com/aarsakian/MFTExtractor/MFT"
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
	MFTRecords                []MFT.Record
	MFTSize                   int
}

func (ntfs NTFS) GetSectorsPerCluster() uint8 {
	return ntfs.SectorsPerCluster
}

func (ntfs NTFS) CollectMFTArea(hD img.DiskReader, partitionOffset uint64) []byte {
	var buf bytes.Buffer

	buf.Grow(ntfs.MFTSize) // allow for MFT size

	partitionOffsetB := int64(partitionOffset) * int64(ntfs.BytesPerSector)

	for offset, clustr := range *ntfs.MFTrunlistOffsetsAndSizes {
		//inefficient since allocates memory for each round
		tempBuffer := make([]byte, clustr*int(ntfs.SectorsPerCluster)*int(ntfs.BytesPerSector))
		hD.ReadFile(partitionOffsetB+int64(offset)*int64(ntfs.SectorsPerCluster)*int64(ntfs.BytesPerSector), tempBuffer)
		buf.Write(tempBuffer)
	}
	return buf.Bytes()
}

func (ntfs *NTFS) ProcessFirstRecord(hD img.DiskReader, partitionOffset uint64) {
	bs := ntfs.GetMFTEntry(hD, partitionOffset, 0)
	ntfs.ProcessRecords(bs)
	firstRecord := ntfs.MFTRecords[0]
	runlistOffsetsAndSizes := firstRecord.GetRunListSizesAndOffsets()
	ntfs.MFTrunlistOffsetsAndSizes = &runlistOffsetsAndSizes

	ntfs.MFTSize = int(firstRecord.GetTotalRunlistSize() * int(ntfs.BytesPerSector) * int(ntfs.SectorsPerCluster))
}

func (ntfs *NTFS) ProcessRecords(data []byte) {
	var record MFT.Record
	for i := 0; i < len(data); i += MFT.RecordSize {

		record.Process(data[i : i+MFT.RecordSize])
		ntfs.MFTRecords = append(ntfs.MFTRecords, record) //copies values
	}
}

func (ntfs NTFS) GetMFTEntry(hD img.DiskReader, partitionOffset uint64,
	recordOffset int) []byte {

	length := uint32(1024) // len of MFT record

	buffer := make([]byte, length)
	mftOffset := int64(ntfs.MFTOffset * uint64(ntfs.SectorsPerCluster) * uint64(ntfs.BytesPerSector))
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

func Parse(drive int, partitionOffset uint64) NTFS {
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
