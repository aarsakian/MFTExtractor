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

type FoldersFilter struct {
	Include bool
}

func (foldersFilter FoldersFilter) Execute(records MFT.Records) MFT.Records {
	if !foldersFilter.Include {
		return records.FilterOutFolders()
	}
	return records
}
