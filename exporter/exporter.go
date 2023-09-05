package exporter

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/aarsakian/MFTExtractor/FS/NTFS/MFT"
	MFTAttributes "github.com/aarsakian/MFTExtractor/FS/NTFS/MFT/attributes"
	"github.com/aarsakian/MFTExtractor/img"
	"github.com/aarsakian/MFTExtractor/utils"
)

type Exporter struct {
	Location          string
	SectorsPerCluster uint8
	Disk              int
	PartitionOffset   uint64
}

func (exp Exporter) ExportData(records []MFT.Record, hD img.DiskReader) {
	var data []byte
	for _, record := range records {
		if !record.HasAttr("DATA") {
			continue
		}
		if record.HasResidentDataAttr() {
			data = record.GetResidentData()
		} else {
			runlist := record.GetRunList()
			lsize, _ := record.GetFileSize()

			var dataRuns bytes.Buffer
			dataRuns.Grow(int(lsize))

			offset := int64(exp.PartitionOffset) * 512 // partition in bytes

			diskSize := hD.GetDiskSize()

			for (MFTAttributes.RunList{}) != runlist {
				offset += runlist.Offset * int64(exp.SectorsPerCluster) * 512
				if offset > diskSize {
					fmt.Printf("skipped offset %d exceeds disk size! exiting", offset)
					break
				}
				//	fmt.Printf("extracting data from %d len %d \n", offset, runlist.Length)

				data := hD.ReadFile(offset, int(runlist.Length*8*512))

				dataRuns.Write(data)

				if runlist.Next == nil {
					break
				}

				runlist = *runlist.Next
			}
			data = dataRuns.Bytes()
		}
		exp.CreateFile(record.GetFname(), data)

	}

}

func (exp Exporter) CreateFile(fname string, data []byte) {
	fullpath := filepath.Join(exp.Location, fname)

	err := os.MkdirAll(exp.Location, 0750)
	if err != nil && !os.IsExist(err) {
		fmt.Println(err)
	}
	utils.WriteFile(fullpath, data)

}
