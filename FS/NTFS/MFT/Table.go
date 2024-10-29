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
	Records Records
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

		err := record.Process(data[i : i+RecordSize])
		if err != nil {
			logger.MFTExtractorlogger.Error(err)
			continue
		}

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
	//recreate chain  for fragmented $MFT records (attrList present)
	for idx := range mfttable.Records {

		for _, linkedRecordInfo := range mfttable.Records[idx].LinkedRecordsInfo {
			//cannot point to itself
			if mfttable.Records[idx].Entry == linkedRecordInfo.RefEntry {
				continue
			}

			linkedRecord, err := mfttable.GetRecord(linkedRecordInfo.RefEntry)
			linkedRecord.OriginLinkedRecord = &mfttable.Records[idx]

			if err != nil {
				logger.MFTExtractorlogger.Warning(fmt.Sprintf("Record %d has linked to non existing record %d",
					mfttable.Records[idx].Entry, linkedRecordInfo.RefEntry))
				continue
			}
			mfttable.Records[idx].LinkedRecords = append(mfttable.Records[idx].LinkedRecords, linkedRecord)

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
		parentRecord, err := mfttable.GetRecord(uint32(fnattr.ParRef))
		if err == nil {
			mfttable.Records[idx].Parent = parentRecord
		} else {
			logger.MFTExtractorlogger.Warning(fmt.Sprintf("No Parent %d Record found for record %d", fnattr.ParRef, mfttable.Records[idx].Entry))
		}

	}
}

func (mfttable MFTTable) GetRecord(referencedEntry uint32) (*Record, error) {
	if int(referencedEntry) < len(mfttable.Records) &&
		mfttable.Records[referencedEntry].Entry == referencedEntry {
		return &mfttable.Records[referencedEntry], nil
	} else { //brute force seach
		for idx := range mfttable.Records {
			if mfttable.Records[idx].Entry == referencedEntry {
				return &mfttable.Records[idx], nil
			}
		}
	}
	return nil, errors.New("no record found")
}

func (mfttable *MFTTable) CalculateFileSizes() {

	for idx := range mfttable.Records {
		//process only I30 records
		if !mfttable.Records[idx].IsFolder() {
			continue
		}
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

	idxEntries := attr.GetIndexEntriesSortedByMFTEntry()

	for _, idxEntry := range idxEntries {
		if idxEntry.Fnattr == nil {
			continue
		}

		//issue with realsize in 8.3 fnattr
		referencedEntry, err := mfttable.GetRecord(uint32(idxEntry.ParRef))

		if err != nil {
			msg := fmt.Sprintf("Record %d has attribute %s which references non existent $MFT record entry %d.",
				recordId, attrType, idxEntry.Fnattr.ParRef)
			logger.MFTExtractorlogger.Warning(msg)
			continue
		}

		// set file size omit folders
		if referencedEntry.IsFolder() {
			continue
		}

		if idxEntry.Fnattr.RealFsize > idxEntry.Fnattr.AllocFsize {

			referencedEntry.I30Size = idxEntry.Fnattr.AllocFsize
		} else {
			referencedEntry.I30Size = idxEntry.Fnattr.RealFsize
		}

	}

}
