package reporter

import (
	"fmt"

	"github.com/aarsakian/MFTExtractor/MFT"
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
}

func (rp Reporter) Show(records []MFT.Record) {
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

		if askedToShow {
			fmt.Printf("\n")
		}

	}

}
