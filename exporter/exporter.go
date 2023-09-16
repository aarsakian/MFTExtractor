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
	SectorsPerCluster int
	Disk              int
	PartitionOffset   uint64
}

func (exp Exporter) ExportData(records []MFT.Record, hD img.DiskReader) {

	var retrievedData []byte
	for _, record := range records {
		fmt.Printf("about to export data to file %s", record.GetFname())
		if !record.HasAttr("DATA") {
			continue
		}
		if record.HasResidentDataAttr() {
			retrievedData = record.GetResidentData()
		} else {
			runlist := record.GetRunList()
			_, lsize := record.GetFileSize()

			var dataRuns bytes.Buffer
			dataRuns.Grow(int(lsize))

			offset := int64(exp.PartitionOffset) // partition in bytes

			diskSize := hD.GetDiskSize()

			for (MFTAttributes.RunList{}) != runlist {
				offset += runlist.Offset * int64(exp.SectorsPerCluster) * 512
				if offset > diskSize {
					fmt.Printf("skipped offset %d exceeds disk size! exiting", offset)
					break
				}
				//	fmt.Printf("extracting data from %d len %d \n", offset, runlist.Length)

				data := hD.ReadFile(offset, int(runlist.Length*uint64(exp.SectorsPerCluster)*512))

				dataRuns.Write(data)

				if runlist.Next == nil {
					break
				}

				runlist = *runlist.Next
			}
			retrievedData = dataRuns.Bytes()
		}

		exp.CreateFile(record.GetFname(), retrievedData)

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
