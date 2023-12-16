package exporter

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/aarsakian/MFTExtractor/FS/NTFS/MFT"
	"github.com/aarsakian/MFTExtractor/utils"
)

type Exporter struct {
	Location string
	Hash     string
}

func (exp Exporter) ExportData(wg *sync.WaitGroup, results <-chan utils.AskedFile) {
	defer wg.Done()

	for result := range results {

		exp.CreateFile(result.Fname, result.Content)

	}

}

func (exp Exporter) SetFilesToLogicalSize(records []MFT.Record) {
	for _, record := range records {
		fname := record.GetFname()
		e := os.Truncate(filepath.Join(exp.Location, fname), record.GetLogicalFileSize())
		if e != nil {
			fmt.Printf("ERROR %s", e)
		}

	}
}

func (exp Exporter) HashFiles(records []MFT.Record) {

	if exp.Hash != "MD5" && exp.Hash != "SHA1" {
		fmt.Printf("Only Supported Hashes are MD5 or SHA1 and not %s!\n", exp.Hash)
		return
	}
	fmt.Printf("Hashing Stage\n")
	for _, record := range records {
		fname := record.GetFname()

		data, e := os.ReadFile(filepath.Join(exp.Location, fname))
		if e != nil {
			fmt.Printf("ERROR %s", e)
		}
		if exp.Hash == "MD5" {
			fmt.Printf("File %s has %s %s \n", fname, exp.Hash, utils.GetMD5(data))
		} else if exp.Hash == "SHA1" {
			fmt.Printf("File %s has %s %s \n", fname, exp.Hash, utils.GetSHA1(data))
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
