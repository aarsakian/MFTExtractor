package ntfs

import (
	"bytes"
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

func (ntfs NTFS) Process(hD img.DiskReader, partitionOffsetB int64, MFTSelectedEntry int, fromMFTEntry int, toMFTEntry int) {
	length := uint32(1024) // len of MFT record

	physicalOffset := partitionOffsetB + int64(ntfs.MFTOffset)*int64(ntfs.SectorsPerCluster)*int64(ntfs.BytesPerSector)

	bs := make([]byte, length)
	hD.ReadFile(physicalOffset, bs)

	var mfttable *MFT.MFTTable = new(MFT.MFTTable)
	mfttable.ProcessRecords(bs)
	mfttable.DetermineClusterOffsetLength()
	ntfs.MFTTable = mfttable
	// fill buffer before parsing the record

	MFTAreaBuf := ntfs.CollectMFTArea(hD, partitionOffsetB)
	ntfs.ProcessMFT(MFTAreaBuf, MFTSelectedEntry, fromMFTEntry, toMFTEntry)
}

func (ntfs NTFS) CollectMFTArea(hD img.DiskReader, partitionOffsetB int64) []byte {
	var buf bytes.Buffer

	buf.Grow(int(ntfs.MFTTable.Size) * int(ntfs.BytesPerSector) * int(ntfs.SectorsPerCluster)) // allow for MFT size

	for offset, clustr := range *ntfs.MFTTable.RunlistOffsetsAndSizes {
		//inefficient since allocates memory for each round
		tempBuffer := make([]byte, clustr*int(ntfs.SectorsPerCluster)*int(ntfs.BytesPerSector))
		hD.ReadFile(partitionOffsetB+int64(offset)*int64(ntfs.SectorsPerCluster)*int64(ntfs.BytesPerSector), tempBuffer)
		buf.Write(tempBuffer)
	}
	return buf.Bytes()
}

func (ntfs *NTFS) ProcessMFT(data []byte, MFTSelectedEntry int,
	fromMFTEntry int, toMFTEntry int) {
	MFTSizeB := int(ntfs.MFTTable.Size * int(ntfs.BytesPerSector) * int(ntfs.SectorsPerCluster))
	totalRecords := MFTSizeB / MFT.RecordSize
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
	for i := 0; i < MFTSizeB; i += MFT.RecordSize {

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

func (ntfs NTFS) GetMFTEntryOffset(
	recordOffset int) int64 {

	mftOffset := int64(ntfs.MFTOffset * uint64(ntfs.SectorsPerCluster) * uint64(ntfs.BytesPerSector))
	offsetedRecord := recordOffset
	lengthB := 0

	for offset, len := range *ntfs.MFTTable.RunlistOffsetsAndSizes {
		lengthB += len * int(ntfs.SectorsPerCluster) * 512
		if offsetedRecord >= lengthB { // more than the available clusters in the contiguous area
			mftOffset = int64(offset * int(ntfs.SectorsPerCluster) * 512)
			offsetedRecord -= lengthB
			break
		}

	}

	return int64(mftOffset) + int64(offsetedRecord)

}

func (ntfs *NTFS) Parse(buffer []byte) {
	utils.Unmarshal(buffer, ntfs)

}
