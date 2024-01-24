package MFT

import (
	"fmt"

	MFTAttributes "github.com/aarsakian/MFTExtractor/FS/NTFS/MFT/attributes"
	"github.com/aarsakian/MFTExtractor/img"
	"github.com/aarsakian/MFTExtractor/logger"
	"github.com/aarsakian/MFTExtractor/utils"
)

// $MFT table points either to its file path or the buffer containing $MFT
type MFTTable struct {
	Records []Record
	Size    int
}

func (mfttable *MFTTable) ProcessRecords(data []byte) {

	mfttable.Records = make([]Record, len(data)/RecordSize)
	fmt.Printf("Processing %d $MFT entries.\n", len(mfttable.Records))

	var record Record
	for i := 0; i < len(data); i += RecordSize {
		if utils.Hexify(data[i:i+4]) == "00000000" { //zero area skip
			continue
		}

		record.Process(data[i : i+RecordSize])

		mfttable.Records[i/RecordSize] = record
	}

}

func (mfttable *MFTTable) ProcessNonResidentRecords(hD img.DiskReader, partitionOffsetB int64, clusterSizeB int) {
	fmt.Printf("Processing NoN resident attributes of %d records.\n", len(mfttable.Records))
	for idx := range mfttable.Records {
		mfttable.Records[idx].ProcessNoNResidentAttributes(hD, partitionOffsetB, clusterSizeB)
	}
}

func (mfttable *MFTTable) CreateLinkedRecords() {
	for idx := range mfttable.Records {
		previdx := idx
		for _, linkedRecordInfo := range mfttable.Records[idx].LinkedRecordsInfo {
			entryId := linkedRecordInfo.Entry
			mfttable.Records[previdx].LinkedRecord = &mfttable.Records[entryId]
			previdx = int(entryId)

		}
	}
}

func (mfttable *MFTTable) FindParentRecords() {
	for idx := range mfttable.Records {
		attr := mfttable.Records[idx].FindAttribute("FileName")
		if attr == nil {
			//	fmt.Printf("No FileName attribute found at record %d\n", idx)
			continue

		}
		fnattr := attr.(*MFTAttributes.FNAttribute)
		mfttable.Records[idx].Parent = &mfttable.Records[fnattr.ParRef]
	}
}

func (mfttable *MFTTable) CalculateFileSizes() {
	for idx := range mfttable.Records {
		//process only I30 records

		if mfttable.Records[idx].HasAttr("Index Root") {
			mfttable.SetI30Size(idx, "Index Root")
		}
		if mfttable.Records[idx].HasAttr("Index Allocation") {
			mfttable.SetI30Size(idx, "Index Allocation")
		}

	}
}

func (mfttable *MFTTable) SetI30Size(recordId int, attrType string) {

	attr := mfttable.Records[recordId].FindAttribute(attrType).(IndexAttributes)

	idxEntries := attr.GetIndexEntriesSortedByMFTEntryID()

	for _, entry := range idxEntries {
		if entry.Fnattr == nil {
			continue
		}
		if entry.ParRef > uint64(len(mfttable.Records)) {
			msg := fmt.Sprintf("Record %d has FileAttribute in its  %s which references non existent $MFT record entry %d.",
				recordId, attrType, entry.Fnattr.ParRef)
			logger.MFTExtractorlogger.Warning(msg)
			continue
		}

		mfttable.Records[entry.ParRef].I30Size = entry.Fnattr.RealFsize

	}

}
