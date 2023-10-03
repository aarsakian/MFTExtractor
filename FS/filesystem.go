package FS

import (
	ntfs "github.com/aarsakian/MFTExtractor/FS/NTFS"
	"github.com/aarsakian/MFTExtractor/FS/NTFS/MFT"
	"github.com/aarsakian/MFTExtractor/img"
)

type FileSystem interface {
	Process(img.DiskReader, int64, []int, int, int)
	GetSectorsPerCluster() int
	GetBytesPerSector() uint64
	GetFileContents(img.DiskReader, int64, chan ntfs.Task)
	GetMetadata() []MFT.Record
}
