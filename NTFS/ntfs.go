package ntfs

import (
	"bytes"
	"fmt"
	"math"

	"github.com/aarsakian/MFTExtractor/MFT"
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
	MFTTable          *MFT.MFTTable
}

func (ntfs NTFS) GetSectorsPerCluster() uint8 {
	return ntfs.SectorsPerCluster
}

func (ntfs NTFS) CollectMFTArea(hD img.DiskReader, partitionOffset uint64) []byte {
	var buf bytes.Buffer

	buf.Grow(int(ntfs.MFTTable.Size)) // allow for MFT size

	partitionOffsetB := int64(partitionOffset) * int64(ntfs.BytesPerSector)

	for offset, clustr := range *ntfs.MFTTable.RunlistOffsetsAndSizes {
		//inefficient since allocates memory for each round
		tempBuffer := make([]byte, clustr*int(ntfs.SectorsPerCluster)*int(ntfs.BytesPerSector))
		hD.ReadFile(partitionOffsetB+int64(offset)*int64(ntfs.SectorsPerCluster)*int64(ntfs.BytesPerSector), tempBuffer)
		buf.Write(tempBuffer)
	}
	return buf.Bytes()
}

func (ntfs *NTFS) ProcessFirstRecord(hD img.DiskReader, partitionOffset uint64) {
	bs := ntfs.GetMFTEntry(hD, partitionOffset, 0)

	var mfttable *MFT.MFTTable = new(MFT.MFTTable)
	ntfs.MFTTable = mfttable
	ntfs.MFTTable.ProcessRecords(bs)
	firstRecord := ntfs.MFTTable.Records[0]
	runlistOffsetsAndSizes := firstRecord.GetRunListSizesAndOffsets()
	mfttable.RunlistOffsetsAndSizes = &runlistOffsetsAndSizes

	mfttable.Size = int(firstRecord.GetTotalRunlistSize() * int(ntfs.BytesPerSector) * int(ntfs.SectorsPerCluster))
}

func (ntfs *NTFS) ProcessMFT(data []byte, MFTSelectedEntry int,
	fromMFTEntry int, toMFTEntry int) {
	totalRecords := int(ntfs.MFTTable.Size) / MFT.RecordSize
	var buf bytes.Buffer
	if fromMFTEntry != -1 {
		totalRecords -= fromMFTEntry
	}
	if fromMFTEntry > totalRecords {
		panic("MFT start entry exceeds $MFT number of records")
	}

	if toMFTEntry != math.MaxUint32 && toMFTEntry > totalRecords {
		panic("MFT end entry exceeds $MFT number of records")
	}
	if toMFTEntry != math.MaxUint32 {
		totalRecords -= toMFTEntry
	}
	if MFTSelectedEntry != -1 {
		totalRecords = 1
	}
	buf.Grow(totalRecords * MFT.RecordSize)
	for i := 0; i < int(ntfs.MFTTable.Size); i += MFT.RecordSize {

		if i/MFT.RecordSize > toMFTEntry {
			break
		}

		if MFTSelectedEntry != -1 && i/MFT.RecordSize != MFTSelectedEntry ||
			fromMFTEntry > i/MFT.RecordSize {
			continue
		}

		buf.Write(data[i : i+MFT.RecordSize])

		if i == MFTSelectedEntry {
			break
		}

	}
	ntfs.MFTTable.ProcessRecords(buf.Bytes())

}

func (ntfs NTFS) FilterRecordsByExtension(extension string) []MFT.Record {

	records := utils.Filter(ntfs.MFTTable.Records, func(record MFT.Record) bool {
		return record.HasFilenameExtension(extension)
	})

	return records
}

func (ntfs NTFS) GetMFTEntry(hD img.DiskReader, partitionOffset uint64,
	recordOffset int) []byte {

	length := uint32(1024) // len of MFT record

	buffer := make([]byte, length)
	mftOffset := int64(ntfs.MFTOffset * uint64(ntfs.SectorsPerCluster) * uint64(ntfs.BytesPerSector))
	offsetedRecord := recordOffset
	lengthB := 0
	if offsetedRecord > 0 {
		for offset, len := range *ntfs.MFTTable.RunlistOffsetsAndSizes {
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
