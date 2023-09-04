package img

import (
	ewfLib "github.com/aarsakian/EWF_Reader/ewf"

	"github.com/aarsakian/MFTExtractor/utils"
)

type ImageReader struct {
	PathToEvidenceFiles string
	fd                  ewfLib.EWF_Image
}

func (imgreader *ImageReader) CreateHandler() {
	var ewf_image ewfLib.EWF_Image
	filenames := utils.FindEvidenceFiles(imgreader.PathToEvidenceFiles)

	ewf_image.ParseEvidence(filenames)

	imgreader.fd = ewf_image
}

func (imgreader ImageReader) CloseHandler() {

}

func (imgreader ImageReader) ReadFile(physicalOffset int64, length int) []byte {
	return imgreader.fd.RetrieveData(physicalOffset, int64(length))
}

func (imgreader ImageReader) GetDiskSize() int64 {
	return 0
}
