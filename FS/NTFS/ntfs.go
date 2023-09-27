package ntfs

import (
	"bytes"
	"fmt"
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

func (ntfs NTFS) GetSectorsPerCluster() int {
	return int(ntfs.SectorsPerCluster)
}

func (ntfs NTFS) GetBytesPerSector() uint64 {
	return uint64(ntfs.BytesPerSector)
}

func (ntfs NTFS) GetFileContents(hD img.DiskReader, partitionOffset int64) map[string][]byte {
	retrievedDataRecords := make(map[string][]byte)
	for _, record := range ntfs.MFTTable.Records {

		if !record.HasAttr("DATA") {
			continue
		}
		fname := record.GetFname()
		if record.HasResidentDataAttr() {
			retrievedDataRecords[fname] = record.GetResidentData()
		} else {
			runlist := record.GetRunList("DATA")
			_, lsize := record.GetFileSize()

			var dataRuns bytes.Buffer
			dataRuns.Grow(int(lsize))

			offset := partitionOffset // partition in bytes

			diskSize := hD.GetDiskSize()

			for (MFTAttributes.RunList{}) != runlist {
				offset += runlist.Offset * int64(ntfs.SectorsPerCluster) * 512
				if offset > diskSize {
					fmt.Printf("skipped offset %d exceeds disk size! exiting", offset)
					break
				}
				//	fmt.Printf("extracting data from %d len %d \n", offset, runlist.Length)

				data := hD.ReadFile(offset, int(runlist.Length*uint64(ntfs.SectorsPerCluster)*512))

				dataRuns.Write(data)

				if runlist.Next == nil {
					break
				}

				runlist = *runlist.Next
			}
			retrievedDataRecords[fname] = dataRuns.Bytes()
		}

	}
	return retrievedDataRecords

}

func (ntfs NTFS) GetMetadata() []MFT.Record {
	return ntfs.MFTTable.Records
}

func (ntfs *NTFS) Process(hD img.DiskReader, partitionOffsetB int64, MFTSelectedEntry int, fromMFTEntry int, toMFTEntry int) {
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
	ntfs.MFTTable.ProcessNonResidentRecords(hD, partitionOffsetB, int(ntfs.SectorsPerCluster)*int(ntfs.BytesPerSector))

}

func (ntfs NTFS) CollectMFTArea(hD img.DiskReader, partitionOffsetB int64) []byte {
	var buf bytes.Buffer

	length := int(ntfs.MFTTable.Size) * int(ntfs.BytesPerSector) * int(ntfs.SectorsPerCluster) // allow for MFT size
	buf.Grow(length)

	runlist := ntfs.MFTTable.Records[0].GetRunList("DATA") // first record $MFT
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

	totalRecords := len(data) / MFT.RecordSize
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
	for i := 0; i < len(data); i += MFT.RecordSize {

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
