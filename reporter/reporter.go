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
	ShowFull       bool
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

		if rp.ShowFileName != "" || rp.ShowFull {
			record.ShowFileName(rp.ShowFileName)
			askedToShow = true
		}

		if rp.ShowAttributes != "" || rp.ShowFull {
			record.ShowAttributes(rp.ShowAttributes)
			askedToShow = true
		}

		if rp.ShowTimestamps || rp.ShowFull {
			record.ShowTimestamps()
			askedToShow = true
		}

		if rp.IsResident || rp.ShowFull {
			record.ShowIsResident()
			askedToShow = true
		}

		if rp.ShowRunList || rp.ShowFull {
			record.ShowRunList()
			askedToShow = true
		}

		if rp.ShowFileSize || rp.ShowFull {
			record.ShowFileSize()
			askedToShow = true
		}

		if rp.ShowVCNs || rp.ShowFull {
			record.ShowVCNs()
			askedToShow = true
		}

		if rp.ShowIndex || rp.ShowFull {

			record.ShowIndex()
			askedToShow = true
		}

		if rp.ShowParent || rp.ShowFull {
			record.ShowParentRecordInfo()
		}

		if rp.ShowPath || rp.ShowFull {
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
