package FS

import (
	"sync"

	"github.com/aarsakian/MFTExtractor/FS/NTFS/MFT"
	"github.com/aarsakian/MFTExtractor/img"
)

type FileSystem interface {
	Process(img.DiskReader, int64, []int, int, int)
	GetSectorsPerCluster() int
	GetBytesPerSector() uint64
	GetMetadata() []MFT.Record
	CollectUnallocated(*sync.WaitGroup, img.DiskReader, int64, chan<- []byte)
	GetSignature() string
}
