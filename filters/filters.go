package filters

import "github.com/aarsakian/MFTExtractor/FS/NTFS/MFT"

type Filter interface {
	Execute(records MFT.Records) MFT.Records
}

type NameFilter struct {
	Filenames []string
}

func (nameFilter NameFilter) Execute(records MFT.Records) MFT.Records {
	return records.FilterByNames(nameFilter.Filenames)
}

type PathFilter struct {
	NamePath string
}

func (pathFilter PathFilter) Execute(records MFT.Records) MFT.Records {
	return records.FilterByPath(pathFilter.NamePath)
}

type ExtensionsFilter struct {
	Extensions []string
}

func (extensionsFilter ExtensionsFilter) Execute(records MFT.Records) MFT.Records {
	return records.FilterByExtensions(extensionsFilter.Extensions)
}

type OrphansFilter struct {
	Include bool
}

func (orphansFilter OrphansFilter) Execute(records MFT.Records) MFT.Records {
	if orphansFilter.Include {
		return records.FilterOrphans()
	}
	return records
}

type DeletedFilter struct {
	Include bool
}

func (deletedFilter DeletedFilter) Execute(records MFT.Records) MFT.Records {
	if deletedFilter.Include {
		return records.FilterDeleted(deletedFilter.Include)
	}
	return records
}

type FoldersFilter struct {
	Include bool
}

func (foldersFilter FoldersFilter) Execute(records MFT.Records) MFT.Records {
	if !foldersFilter.Include {
		return records.FilterOutFolders()
	}
	return records
}

type PrefixesSuffixesFilter struct {
	Prefixes []string
	Suffixes []string
}

func (prefSufFilter PrefixesSuffixesFilter) Execute(records MFT.Records) MFT.Records {
	for idx, prefix := range prefSufFilter.Prefixes {
		records = records.FilterByPrefixSuffix(prefix, prefSufFilter.Suffixes[idx])
	}

	return records

}
