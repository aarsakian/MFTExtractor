package ntfs

import (
	"bytes"
	"math"

	"github.com/aarsakian/MFTExtractor/FS/NTFS/MFT"
	MFTAttributes "github.com/aarsakian/MFTExtractor/FS/NTFS/MFT/attributes"
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

func (ntfs NTFS) Process(hD img.DiskReader, partitionOffsetB int64, MFTSelectedEntry int, fromMFTEntry int, toMFTEntry int) []MFT.Record {
	length := int(1024) // len of MFT record

	physicalOffset := partitionOffsetB + int64(ntfs.MFTOffset)*int64(ntfs.SectorsPerCluster)*int64(ntfs.BytesPerSector)

	data := hD.ReadFile(physicalOffset, length)

	var mfttable *MFT.MFTTable = new(MFT.MFTTable)
	mfttable.ProcessRecords(data)
	mfttable.DetermineClusterOffsetLength()
	ntfs.MFTTable = mfttable
	// fill buffer before parsing the record

	MFTAreaBuf := ntfs.CollectMFTArea(hD, partitionOffsetB)
	ntfs.ProcessMFT(MFTAreaBuf, MFTSelectedEntry, fromMFTEntry, toMFTEntry)
	return ntfs.MFTTable.Records
}

func (ntfs NTFS) CollectMFTArea(hD img.DiskReader, partitionOffsetB int64) []byte {
	var buf bytes.Buffer

	length := int(ntfs.MFTTable.Size) * int(ntfs.BytesPerSector) * int(ntfs.SectorsPerCluster) // allow for MFT size
	buf.Grow(length)

	runlist := ntfs.MFTTable.Records[0].GetRunList() // first record $MFT
	offset := 0

	for (MFTAttributes.RunList{}) != runlist {
		offset += int(runlist.Offset)

		clusters := int(runlist.Length)

		//inefficient since allocates memory for each round

		data := hD.ReadFile(partitionOffsetB+int64(offset)*int64(ntfs.SectorsPerCluster)*int64(ntfs.BytesPerSector), clusters*int(ntfs.BytesPerSector)*int(ntfs.SectorsPerCluster))
		buf.Write(data)

		if runlist.Next == nil {
			break
		}

		runlist = *runlist.Next
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

func (ntfs *NTFS) Parse(buffer []byte) {
	utils.Unmarshal(buffer, ntfs)

}
