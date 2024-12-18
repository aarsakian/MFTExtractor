package ntfs

import (
	"bytes"
	"fmt"
	"math"

	"github.com/aarsakian/MFTExtractor/FS/NTFS/MFT"
	MFTAttributes "github.com/aarsakian/MFTExtractor/FS/NTFS/MFT/attributes"
	"github.com/aarsakian/MFTExtractor/img"
	"github.com/aarsakian/MFTExtractor/logger"
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

func (ntfs NTFS) HasValidSignature() bool {
	return ntfs.Signature == "NTFS"
}
func (ntfs NTFS) GetSectorsPerCluster() int {
	return int(ntfs.SectorsPerCluster)
}

func (ntfs NTFS) GetBytesPerSector() uint64 {
	return uint64(ntfs.BytesPerSector)
}

func (ntfs NTFS) GetSignature() string {
	return ntfs.Signature
}

func (ntfs NTFS) GetMetadata() []MFT.Record {
	return ntfs.MFTTable.Records
}

func (ntfs *NTFS) Process(hD img.DiskReader, partitionOffsetB int64, MFTSelectedEntries []int,
	fromMFTEntry int, toMFTEntry int) {

	length := int(1024) // len of MFT record

	physicalOffset := partitionOffsetB + int64(ntfs.MFTOffset)*int64(ntfs.SectorsPerCluster)*int64(ntfs.BytesPerSector)

	msg := "Reading first record entry to determine the size of $MFT Table at offset %d"
	fmt.Printf(msg+"\n", physicalOffset)
	logger.MFTExtractorlogger.Info(fmt.Sprintf(msg, physicalOffset))

	data := hD.ReadFile(physicalOffset, length)

	var mfttable *MFT.MFTTable = new(MFT.MFTTable)
	mfttable.ProcessRecords(data)
	mfttable.DetermineClusterOffsetLength()
	ntfs.MFTTable = mfttable
	// fill buffer before parsing the record

	MFTAreaBuf := ntfs.CollectMFTArea(hD, partitionOffsetB)
	ntfs.ProcessMFT(MFTAreaBuf, MFTSelectedEntries, fromMFTEntry, toMFTEntry)
	ntfs.MFTTable.ProcessNonResidentRecords(hD, partitionOffsetB, int(ntfs.SectorsPerCluster)*int(ntfs.BytesPerSector))
	if len(MFTSelectedEntries) == 0 && fromMFTEntry == -1 && toMFTEntry == math.MaxUint32 { // additional processing only when user has not selected entries
		msg := "Linking $MFT record non resident $MFT entries"
		fmt.Printf(msg + "\n")
		logger.MFTExtractorlogger.Info(msg)
		ntfs.MFTTable.CreateLinkedRecords()

		msg = "Locating parent $MFT records from Filename attributes"
		fmt.Printf(msg + "\n")
		logger.MFTExtractorlogger.Info(msg)
		ntfs.MFTTable.FindParentRecords()

		msg = "Calculating files sizes from $I30"
		fmt.Printf(msg + "\n")
		logger.MFTExtractorlogger.Info(msg)
		ntfs.MFTTable.CalculateFileSizes()

	}

}

func (ntfs NTFS) CollectUnallocated(hD img.DiskReader, partitionOffsetB int64, blocks chan<- []byte) {

	record := ntfs.MFTTable.Records[0]
	bitmap := record.FindAttribute("BitMap").(*MFTAttributes.BitMap)
	unallocatedClusters := bitmap.GetUnallocatedClusters()
	var buf bytes.Buffer

	blockSize := 1 // nof consecutive clusters
	prevClusterOffset := unallocatedClusters[0]

	for idx, unallocatedCluster := range unallocatedClusters {
		if idx == 0 {
			continue
		}
		if unallocatedCluster-prevClusterOffset <= 1 {
			blockSize += 1
		} else {
			buf.Grow(blockSize * int(ntfs.BytesPerSector))
			firstBlockCluster := unallocatedClusters[idx-blockSize]
			offset := partitionOffsetB + int64(firstBlockCluster)*int64(ntfs.SectorsPerCluster*uint8(ntfs.BytesPerSector))
			buf.Write(hD.ReadFile(offset, int(uint16(blockSize)*(uint16(ntfs.SectorsPerCluster)*ntfs.BytesPerSector))))
			blockSize = 1
			blocks <- buf.Bytes()

		}
		prevClusterOffset = unallocatedCluster

	}
	close(blocks)

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

		data := hD.ReadFile(partitionOffsetB+int64(offset)*int64(ntfs.SectorsPerCluster)*int64(ntfs.BytesPerSector), clusters*int(ntfs.BytesPerSector)*int(ntfs.SectorsPerCluster))
		buf.Write(data)

		if runlist.Next == nil {
			break
		}

		runlist = *runlist.Next
	}
	return buf.Bytes()
}

func (ntfs *NTFS) ProcessMFT(data []byte, MFTSelectedEntries []int,
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
	if len(MFTSelectedEntries) > 0 {
		totalRecords = len(MFTSelectedEntries)
	}
	buf.Grow(totalRecords * MFT.RecordSize)

	for i := 0; i < len(data); i += MFT.RecordSize {
		if i/MFT.RecordSize > toMFTEntry {
			break
		}
		for _, MFTSelectedEntry := range MFTSelectedEntries {

			if i/MFT.RecordSize != MFTSelectedEntry {

				continue
			}

			buf.Write(data[i : i+MFT.RecordSize])

		}
		//buffer full break

		if fromMFTEntry > i/MFT.RecordSize {
			continue
		}
		if len(MFTSelectedEntries) == 0 {
			buf.Write(data[i : i+MFT.RecordSize])
		}
		if buf.Len() == len(MFTSelectedEntries)*MFT.RecordSize {
			break
		}

	}
	ntfs.MFTTable.ProcessRecords(buf.Bytes())

}

func (ntfs *NTFS) Parse(buffer []byte) {
	utils.Unmarshal(buffer, ntfs)

}
