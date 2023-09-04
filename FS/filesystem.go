package FS

import (
	"github.com/aarsakian/MFTExtractor/MFT"
	"github.com/aarsakian/MFTExtractor/img"
)

type FileSystem interface {
	Process(img.DiskReader, int64, int, int, int) []MFT.Record
}
