package main

import (
	//"C"

	//"database/sql"

	"flag"
	"fmt"
	"log"
	"math"

	disk "github.com/aarsakian/MFTExtractor/Disk"
	"github.com/aarsakian/MFTExtractor/FS"
	"github.com/aarsakian/MFTExtractor/FS/NTFS/MFT"
	"github.com/aarsakian/MFTExtractor/exporter"
	"github.com/aarsakian/MFTExtractor/img"
	"github.com/aarsakian/MFTExtractor/tree"

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
	MFTSelectedEntry := flag.Int("entry", -1, "select a particular MFT entry")
	showFileName := flag.String("showfilename", "", "show the name of the filename attribute of each MFT record choices: Any, Win32, Dos")
	exportFile := flag.String("filename", "", "file to export")
	isResident := flag.Bool("resident", false, "check whether entry is resident")
	fromMFTEntry := flag.Int("fromEntry", -1, "select entry to start parsing")
	toMFTEntry := flag.Int("toEntry", math.MaxUint32, "select entry to end parsing")
	showRunList := flag.Bool("runlist", false, "show runlist of MFT record attributes")
	showFileSize := flag.Bool("filesize", false, "show file size of a record holding a file")
	showVCNs := flag.Bool("vcns", false, "show the vncs of non resident attributes")
	showAttributes := flag.String("attributes", "", "show attributes")
	showTimestamps := flag.Bool("timestamps", false, "show all timestamps")
	showIndex := flag.Bool("index", false, "show index structures")
	physicalDrive := flag.Int("physicaldrive", -1, "select disk drive number for extraction of non resident files")
	partitionNum := flag.Int("partition", -1, "select partition number")
	showFSStructure := flag.Bool("structure", false, "reconstrut entries tree")
	listPartitions := flag.Bool("listpartitions", false, "list partitions")
	fileExtension := flag.String("extension", "", "search MFT records by extension")

	flag.Parse() //ready to parse

	var partitionOffsetB uint64

	var hD img.DiskReader
	var records MFT.Records
	var partition disk.Partition
	var physicalDisk disk.Disk
	var fs FS.FileSystem

	rp := reporter.Reporter{
		ShowFileName:   *showFileName,
		ShowAttributes: *showAttributes,
		ShowTimestamps: *showTimestamps,
		IsResident:     *isResident,
		ShowRunList:    *showRunList,
		ShowFileSize:   *showFileSize,
		ShowVCNs:       *showVCNs,
		ShowIndex:      *showIndex,
	}

	if *evidencefile != "" && *partitionNum != -1 || *physicalDrive != -1 && *partitionNum != -1 {
		physicalDisk = disk.Disk{}

		if *physicalDrive != -1 && *partitionNum != -1 {

			hD = img.GetHandler(fmt.Sprintf("\\\\.\\PHYSICALDRIVE%d", *physicalDrive), "physicalDrive")

		} else {
			hD = img.GetHandler(*evidencefile, "image")

		}

		physicalDisk.DiscoverPartitions(hD)

		partition = physicalDisk.GetSelectedPartition(*partitionNum)

		partitionOffsetB = uint64(partition.GetOffset() * 512)

		data := hD.ReadFile(int64(partitionOffsetB), 512)

		fs = partition.LocateFileSystem(data)

		records = fs.Process(hD, int64(partitionOffsetB), *MFTSelectedEntry, *fromMFTEntry, *toMFTEntry)
		defer hD.CloseHandler()

	}

	if *fileExtension != "" {
		records = records.FilterByExtension(*fileExtension)
	}

	if *inputfile != "Disk MFT" {
		mftTable := MFT.MFTTable{Filepath: *inputfile}
		mftTable.Populate(*MFTSelectedEntry, *fromMFTEntry, *toMFTEntry)
		records = mftTable.Records
	}
	rp.Show(records)

	if *exportFile != "" {
		records = records.FilterByName(*exportFile)
	}

	if location != "" && *physicalDrive != -1 && *partitionNum != -1 || location != "" && *evidencefile != "" && *partitionNum != -1 {
		if records == nil {
			fmt.Printf("no records found for request file %s", *exportFile)
		}
		sectorsPerCluster := fs.GetSectorsPerCluster()
		exp := exporter.Exporter{Disk: *physicalDrive, PartitionOffset: partitionOffsetB,
			SectorsPerCluster: sectorsPerCluster, Location: location}
		exp.ExportData(records, hD)

	}

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

} //ends for
