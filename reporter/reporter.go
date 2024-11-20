package reporter

import (
	"fmt"

	"github.com/aarsakian/MFTExtractor/FS/NTFS/MFT"
	UsnJrnl "github.com/aarsakian/MFTExtractor/FS/NTFS/usnjrnl"
)

type Reporter struct {
	ShowFileName   string
	ShowAttributes string
	ShowTimestamps bool
	IsResident     bool
	ShowRunList    bool
	ShowFileSize   bool
	ShowVCNs       bool
	ShowIndex      bool
	ShowParent     bool
	ShowPath       bool
	ShowUSNJRNL    bool
}

func (rp Reporter) Show(records []MFT.Record, usnjrnlRecords UsnJrnl.Records, partitionId int) {
	for _, record := range records {
		askedToShow := false
		if record.Signature == "" { //empty record
			continue
		}
		if rp.ShowFileName != "" {
			record.ShowFileName(rp.ShowFileName)
			askedToShow = true
		}

		if rp.ShowAttributes != "" {
			record.ShowAttributes(rp.ShowAttributes)
			askedToShow = true
		}

		if rp.ShowTimestamps {
			record.ShowTimestamps()
			askedToShow = true
		}

		if rp.IsResident {
			record.ShowIsResident()
			askedToShow = true
		}

		if rp.ShowRunList {
			record.ShowRunList()
			askedToShow = true
		}

		if rp.ShowFileSize {
			record.ShowFileSize()
			askedToShow = true
		}

		if rp.ShowVCNs {
			record.ShowVCNs()
			askedToShow = true
		}

		if rp.ShowIndex {

			record.ShowIndex()
			askedToShow = true
		}

		if rp.ShowParent {
			record.ShowParentRecordInfo()
		}

		if rp.ShowPath {
			record.ShowPath(partitionId)
		}

		if askedToShow {
			fmt.Printf("\n")
		}

	}
	for _, record := range usnjrnlRecords {
		if rp.ShowUSNJRNL {
			record.ShowInfo()
		}
	}

}
