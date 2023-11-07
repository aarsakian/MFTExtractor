package img

import (
	"path"
	"strings"
)

type VMDKReader struct {
	PathToEvidenceFiles string
	fd                  vmdk.extent.Extents 
}

func (imgreader *VMDKReader) CreateHandler() {
	extension := path.Ext(imgreader.PathToEvidenceFiles)
	if strings.ToLower(extension) == ".vmdk" {
		var extents vmdk.extent.Extents
		extents = vmdk.extent.ProcessExtents(*imagePath)
		imgreader.fd = extents 

	} else {
		panic("only VMDK Sparse  images are supported")
	}

}

func (imgreader VMDKReader) CloseHandler() {

}

func (imgreader VMDKReader) ReadFile(physicalOffset int64, length int) []byte {
	return imgreader.fd.RetrieveData(physicalOffset, int64(length))
}

func (imgreader VMDKReader) GetDiskSize() int64 {
	return imgreader.fd.GetDiskSize()
}
