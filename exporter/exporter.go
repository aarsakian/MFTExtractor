package exporter

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/aarsakian/MFTExtractor/utils"
)

type Exporter struct {
	Location string
}

func (exp Exporter) ExportData(wg *sync.WaitGroup, results chan []byte) {
	defer wg.Done()
	for data := range results {
		exp.CreateFile("TESTER", data)
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
