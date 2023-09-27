package FS

import (
	"github.com/aarsakian/MFTExtractor/FS/NTFS/MFT"
	"github.com/aarsakian/MFTExtractor/img"
)

type FileSystem interface {
	Process(img.DiskReader, int64, int, int, int)
	GetSectorsPerCluster() int
	GetBytesPerSector() uint64
	GetFileContents(img.DiskReader, int64) map[string][]byte
	GetMetadata() []MFT.Record
}
