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
	Hash     string
}

func (exp Exporter) ExportData(wg *sync.WaitGroup, results chan utils.AskedFile) {
	defer wg.Done()

	for result := range results {

		exp.CreateFile(result.Fname, result.Content)
	}

}

func (exp Exporter) HashFile(wg *sync.WaitGroup, results chan utils.AskedFile) {
	defer wg.Done()

	for result := range results {
		if exp.Hash == "MD5" {
			fmt.Printf("File %s has %s %s \n", result.Fname, exp.Hash, utils.GetMD5(result.Content))
		} else if exp.Hash == "SHA1" {
			fmt.Printf("File %s has %s %s \n", result.Fname, exp.Hash, utils.GetSHA1(result.Content))
		}

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
