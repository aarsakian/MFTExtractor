package img

import (
	"path"
	"path/filepath"
	"strings"

	extent "github.com/aarsakian/VMDK_Reader/extent"
)

type VMDKReader struct {
	PathToEvidenceFiles string
	fd                  extent.Extents
}

func (imgreader *VMDKReader) CreateHandler() {
	extension := path.Ext(imgreader.PathToEvidenceFiles)
	if strings.ToLower(extension) == ".vmdk" {
		imgreader.fd = extent.ProcessExtents(imgreader.PathToEvidenceFiles)

	} else {
		panic("only VMDK Sparse  images are supported")
	}

}

func (imgreader VMDKReader) CloseHandler() {

}

func (imgreader VMDKReader) ReadFile(physicalOffset int64, length int) []byte {
	return imgreader.fd.RetrieveData(filepath.Dir(imgreader.PathToEvidenceFiles), physicalOffset, int64(length))
}

func (imgreader VMDKReader) GetDiskSize() int64 {
	return imgreader.fd.GetHDSize()
}
