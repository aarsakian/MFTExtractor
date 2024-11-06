package filtermanager

import (
	"github.com/aarsakian/MFTExtractor/FS/NTFS/MFT"
	"github.com/aarsakian/MFTExtractor/filters"
)

type FilterManager struct {
	filters []filters.Filter
}

func (filterManager *FilterManager) Register(filter filters.Filter) {
	filterManager.filters = append(filterManager.filters, filter)
}

func (filterManager FilterManager) ApplyFilters(records MFT.Records) MFT.Records {
	for _, filter := range filterManager.filters {
		records = filter.Execute(records)
	}
	return records
}
