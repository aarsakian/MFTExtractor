package reporter

import "github.com/aarsakian/MFTExtractor/MFT"

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

func (rp Reporter) Show(record MFT.Record) {
	if rp.ShowFileName != "" {
		record.ShowFileName(rp.ShowFileName)
	}

	if rp.ShowAttributes != "" {
		record.ShowAttributes(rp.ShowAttributes)
	}

	if rp.ShowTimestamps {
		record.ShowTimestamps()
	}

	if rp.IsResident {
		record.ShowIsResident()
	}

	if rp.ShowRunList {
		record.ShowRunList()
	}

	if rp.ShowFileSize {
		record.ShowFileSize()
	}

	if rp.ShowVCNs {
		record.ShowVCNs()
	}

	if rp.ShowIndex {
		record.ShowIndex()
	}
}
