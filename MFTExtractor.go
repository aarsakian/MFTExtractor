package main

import (
	//"C"

	//"database/sql"

	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"sync"

	disk "github.com/aarsakian/MFTExtractor/Disk"
	ntfslib "github.com/aarsakian/MFTExtractor/FS/NTFS"
	"github.com/aarsakian/MFTExtractor/FS/NTFS/MFT"
	"github.com/aarsakian/MFTExtractor/exporter"
	"github.com/aarsakian/MFTExtractor/img"
	"github.com/aarsakian/MFTExtractor/tree"
	"github.com/aarsakian/MFTExtractor/utils"

	reporter "github.com/aarsakian/MFTExtractor/Reporter"
)

func checkErr(err error, msg string) {
	if err != nil {
		log.Fatalln(msg, err)
	}
}

func main() {
	//dbmap := initDb()
	//defer dbmap.Db.Close()

	//	save2DB := flag.Bool("db", false, "bool if set an sqlite file will be created, each table will corresponed to an MFT attribute")
	var location string
	inputfile := flag.String("MFT", "Disk MFT", "absolute path to the MFT file")
	evidencefile := flag.String("evidence", "", "path to image file")
	flag.StringVar(&location, "location", "", "the path to export  files")
	MFTSelectedEntries := flag.String("entries", "", "select particular MFT entries, use comma as a seperator.")
	showFileName := flag.String("showfilename", "", "show the name of the filename attribute of each MFT record choices: Any, Win32, Dos")
	exportFiles := flag.String("filenames", "", "files to export use comma for each file")
	isResident := flag.Bool("resident", false, "check whether entry is resident")
	fromMFTEntry := flag.Int("fromEntry", -1, "select entry to start parsing")
	toMFTEntry := flag.Int("toEntry", math.MaxUint32, "select entry to end parsing")
	showRunList := flag.Bool("runlist", false, "show runlist of MFT record attributes")
	showFileSize := flag.Bool("filesize", false, "show file size of a record holding a file")
	showVCNs := flag.Bool("vcns", false, "show the vncs of non resident attributes")
	showAttributes := flag.String("attributes", "", "show attributes (write any for all attributes)")
	showTimestamps := flag.Bool("timestamps", false, "show all timestamps")
	showIndex := flag.Bool("index", false, "show index structures")
	physicalDrive := flag.Int("physicaldrive", -1, "select disk drive number for extraction of non resident files")
	partitionNum := flag.Int("partition", -1, "select partition number")
	showFSStructure := flag.Bool("structure", false, "reconstrut entries tree")
	showParent := flag.Bool("parent", false, "show information about parent record")
	listPartitions := flag.Bool("listpartitions", false, "list partitions")
	fileExtension := flag.String("extension", "", "search MFT records by extension")
	collectUnallocated := flag.Bool("unallocated", false, "collect unallocated area of a file system")

	flag.Parse() //ready to parse

	var hD img.DiskReader
	var records MFT.Records
	var recordsPerPartition map[int]MFT.Records
	var physicalDisk disk.Disk

	entries := utils.GetEntriesInt(*MFTSelectedEntries)
	fileNamesToExport := utils.GetEntries(*exportFiles)

	rp := reporter.Reporter{
		ShowFileName:   *showFileName,
		ShowAttributes: *showAttributes,
		ShowTimestamps: *showTimestamps,
		IsResident:     *isResident,
		ShowRunList:    *showRunList,
		ShowFileSize:   *showFileSize,
		ShowVCNs:       *showVCNs,
		ShowIndex:      *showIndex,
		ShowParent:     *showParent,
	}

	if *evidencefile != "" || *physicalDrive != -1 {

		if *physicalDrive != -1 {

			hD = img.GetHandler(fmt.Sprintf("\\\\.\\PHYSICALDRIVE%d", *physicalDrive), "physicalDrive")

		} else {
			hD = img.GetHandler(*evidencefile, "image")

		}
		physicalDisk = disk.Disk{Handler: hD}
		physicalDisk.DiscoverPartitions()

		if *listPartitions {
			physicalDisk.ListPartitions()
		}
		physicalDisk.ProcessPartitions(*partitionNum, entries, *fromMFTEntry, *toMFTEntry)
		recordsPerPartition = physicalDisk.GetFileSystemMetadata(*partitionNum)

		defer hD.CloseHandler()

		if *collectUnallocated {
			physicalDisk.CollectedUnallocated()
		}
		for partitionId, records := range recordsPerPartition {
			if *exportFiles != "" {
				records = records.FilterByNames(fileNamesToExport)
			}

			if *fileExtension != "" {
				records = records.FilterByExtension(*fileExtension)
			}

			if location != "" && len(records) != 0 {

				results := make(chan utils.AskedFile, len(records))
				wg := new(sync.WaitGroup)
				wg.Add(2)

				exp := exporter.Exporter{Location: location}

				go physicalDisk.Worker(wg, records, results, partitionId) //producer
				go exp.ExportData(wg, results)                            //consumer

				wg.Wait()

			}
			rp.Show(records)
		}

	}

	if *inputfile != "Disk MFT" {

		file, err := os.Open(*inputfile)
		if err != nil {
			// handle the error here
			fmt.Printf("err %s for reading the MFT ", err)
			return
		}
		defer file.Close()

		finfo, err := file.Stat() //file descriptor
		if err != nil {
			fmt.Printf("error getting the file size\n")
			return
		}
		data := make([]byte, finfo.Size())

		file.Read(data)
		if err != nil {
			fmt.Printf("error reading $MFT file.\n")
			return
		}
		var ntfs ntfslib.NTFS

		ntfs.MFTTable = &MFT.MFTTable{Size: int(finfo.Size())}
		ntfs.ProcessMFT(data, entries, *fromMFTEntry, *toMFTEntry)

		records = ntfs.MFTTable.Records

		if *exportFiles != "" {
			records = records.FilterByNames(fileNamesToExport)
		}

		if *fileExtension != "" {
			records = records.FilterByExtension(*fileExtension)
		}
		rp.Show(records)
		t := tree.Tree{}

		fmt.Printf("Building tree from MFT records \n")

		if *showFSStructure {
			for idx := range records {
				if idx < 5 {
					continue
				}
				t.BuildTree(&records[idx])
			}
			t.Show()

		}
	}

} //ends for
