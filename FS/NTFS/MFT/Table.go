package MFT

import (
	"errors"
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
	msg := fmt.Sprintf("Processing %d $MFT entries", len(mfttable.Records))
	fmt.Printf(msg + "\n")
	logger.MFTExtractorlogger.Info(msg)

	var record Record
	for i := 0; i < len(data); i += RecordSize {
		if utils.Hexify(data[i:i+4]) == "00000000" { //zero area skip
			continue
		}

		record.Process(data[i : i+RecordSize])

		mfttable.Records[i/RecordSize] = record

		logger.MFTExtractorlogger.Info(fmt.Sprintf("Processed record %d at pos %d", record.Entry, i/RecordSize))
	}

}

func (mfttable *MFTTable) ProcessNonResidentRecords(hD img.DiskReader, partitionOffsetB int64, clusterSizeB int) {
	fmt.Printf("Processing NoN resident attributes of %d records.\n", len(mfttable.Records))
	for idx := range mfttable.Records {
		mfttable.Records[idx].ProcessNoNResidentAttributes(hD, partitionOffsetB, clusterSizeB)
		logger.MFTExtractorlogger.Info(fmt.Sprintf("Processed non resident attribute record %d at pos %d", mfttable.Records[idx].Entry, idx))
	}
}

func (mfttable *MFTTable) CreateLinkedRecords() {
	for idx := range mfttable.Records {
		previdx := idx
		for _, linkedRecordInfo := range mfttable.Records[idx].LinkedRecordsInfo {
			entryId := linkedRecordInfo.Entry
			if int(entryId) > len(mfttable.Records) {
				logger.MFTExtractorlogger.Warning(fmt.Sprintf("Record %d has linked to non existing record %d", mfttable.Records[previdx].Entry, entryId))
				continue
			}
			mfttable.Records[previdx].LinkedRecord = &mfttable.Records[entryId]
			previdx = int(entryId)

		}
	}
}

func (mfttable *MFTTable) FindParentRecords() {

	for idx := range mfttable.Records {
		attr := mfttable.Records[idx].FindAttribute("FileName")
		if attr == nil {
			//logger.MFTExtractorlogger.Warning(fmt.Sprintf("No FileName attribute found at record %d ", mfttable.Records[idx].Entry))
			continue

		}
		fnattr := attr.(*MFTAttributes.FNAttribute)
		parentRecord, err := mfttable.GetParentRecord(fnattr.ParRef)
		if err == nil {
			mfttable.Records[idx].Parent = parentRecord
		} else {
			logger.MFTExtractorlogger.Warning(fmt.Sprintf("No Parent %d Record found for record %d", fnattr.ParRef, mfttable.Records[idx].Entry))
		}

	}
}

func (mfttable MFTTable) GetParentRecord(referencedEntry uint64) (*Record, error) {
	if int(referencedEntry) < len(mfttable.Records) && mfttable.Records[referencedEntry].Entry == uint32(referencedEntry) {
		return &mfttable.Records[referencedEntry], nil
	} else { //brute force seach
		for idx := range mfttable.Records {
			if mfttable.Records[idx].Entry == uint32(referencedEntry) {
				return &mfttable.Records[idx], nil
			}
		}
	}
	return nil, errors.New("no parent record found")
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

		//issue with realsize in 8.3 fnattr
		parentEntry, err := mfttable.GetParentRecord(entry.ParRef)
		if err != nil {
			msg := fmt.Sprintf("Record %d has attribute %s which references non existent $MFT record entry %d.",
				recordId, attrType, entry.Fnattr.ParRef)
			logger.MFTExtractorlogger.Warning(msg)
			continue
		}

		if entry.Fnattr.RealFsize > entry.Fnattr.AllocFsize {
			parentEntry.I30Size = entry.Fnattr.AllocFsize
		} else {
			parentEntry.I30Size = entry.Fnattr.RealFsize
		}

	}

}
