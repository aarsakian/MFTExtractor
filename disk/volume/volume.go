package volume

import "github.com/aarsakian/MFTExtractor/img"

type Volume interface {
	Process(img.DiskReader, int64, []int, int, int)
	GetSectorsPerCluster() int
	GetBytesPerSector() uint64
	GetInfo() string
	//GetFSMetadata() []MFT.Record
	CollectUnallocated(img.DiskReader, int64, chan<- []byte)
	GetSignature() string
}
