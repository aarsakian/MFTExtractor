package exporter

import (
	"bytes"
	"fmt"

	"github.com/aarsakian/MFTExtractor/MFT"
	MFTAttributes "github.com/aarsakian/MFTExtractor/MFT/attributes"
	"github.com/aarsakian/MFTExtractor/img"
	"github.com/aarsakian/MFTExtractor/utils"
)

type Exporter struct {
	location          string
	SectorsPerCluster uint8
	Disk              int
	PartitionOffset   uint64
}

func (exp Exporter) ExportData(records []MFT.Record) {
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
			hD := img.GetHandler(fmt.Sprintf("\\\\.\\PHYSICALDRIVE%d", exp.Disk))
			diskSize := hD.GetDiskSize()

			for (MFTAttributes.RunList{}) != runlist {
				offset += runlist.Offset * int64(exp.SectorsPerCluster) * 512
				if offset > diskSize {
					fmt.Printf("skipped offset %d exceeds disk size! exiting", offset)
					break
				}
				//	fmt.Printf("extracting data from %d len %d \n", offset, runlist.Length)
				buffer := make([]byte, uint32(runlist.Length*8*512))
				hD.ReadFile(offset, buffer)

				dataRuns.Write(buffer)

				if runlist.Next == nil {
					break
				}

				runlist = *runlist.Next
			}
			data = dataRuns.Bytes()
		}
		exp.CreateFile(record.GetFnames()["win32"], data)

	}

}

func (exp Exporter) CreateFile(fname string, data []byte) {

	utils.WriteFile(fname, data)

}
