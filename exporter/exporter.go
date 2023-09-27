package exporter

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/aarsakian/MFTExtractor/utils"
)

type Exporter struct {
	Location string
}

func (exp Exporter) ExportData(filesData map[string][]byte) {
	for fname, itsData := range filesData {
		exp.CreateFile(fname, itsData)
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
