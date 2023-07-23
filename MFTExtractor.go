package main

import (
	"C"

	//"database/sql"

	"flag"
	"fmt"
	"log"
	"math"
	"os"

	disk "github.com/aarsakian/MFTExtractor/Disk"
	"github.com/aarsakian/MFTExtractor/MFT"
	ntfsLib "github.com/aarsakian/MFTExtractor/NTFS"
	reporter "github.com/aarsakian/MFTExtractor/Reporter"
	"github.com/aarsakian/MFTExtractor/img"
	"github.com/aarsakian/MFTExtractor/tree"
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
	inputfile := flag.String("MFT", "Disk MFT", "absolute path to the MFT file")
	exportFiles := flag.Bool("export", false, "export  files")
	MFTSelectedEntry := flag.Int("entry", -1, "select a particular MFT entry")
	showFileName := flag.String("fileName", "", "show the name of the filename attribute of each MFT record choices: Any, Win32, Dos")
	isResident := flag.Bool("resident", false, "check whether entry is resident")
	fromMFTEntry := flag.Int("fromEntry", 0, "select entry to start parsing")
	ToMFTEntry := flag.Int("toEntry", math.MaxUint32, "select entry to end parsing")
	showRunList := flag.Bool("runlist", false, "show runlist of MFT record attributes")
	showFileSize := flag.Bool("filesize", false, "show file size of a record holding a file")
	showVCNs := flag.Bool("vcns", false, "show the vncs of non resident attributes")
	showAttributes := flag.String("attributes", "", "show attributes")
	showTimestamps := flag.Bool("timestamps", false, "show all timestamps")
	showIndex := flag.Bool("index", false, "show index structures")
	physicalDrive := flag.Int("physicalDrive", -1, "select disk drive number for extraction of non resident files")
	partitionNum := flag.Int("partitionNumber", -1, "select partition number")
	showFSStructure := flag.Bool("structure", false, "reconstrut entries tree")
	fileExtension := flag.String("extension", "", "search MFT records by extension")

	flag.Parse() //ready to parse

	var file *os.File
	var err error

	var partitionOffset uint64
	var sectorsPerCluster uint8

	var ntfs ntfsLib.NTFS
	var hd img.DiskReader

	var MFTsize int64

	var record MFT.Record

	bs := make([]byte, 1024) //byte array to hold MFT entries

	if *physicalDrive != -1 && *partitionNum != -1 {
		physicalDisk := disk.Disk{PhysicalDriveNum: *physicalDrive, PartitionNum: *partitionNum}
		partition := physicalDisk.GetPartition()
		partitionOffset = partition.GetOffset()
		ntfs = partition.LocateFileSystem(*physicalDrive)

		sectorsPerCluster = ntfs.GetSectorsPerCluster()

		hd = physicalDisk.GetHandler()

		ntfs.ProcessFirstRecord(hd, partitionOffset)
		// fill buffer before parsing the record
		MFTAreaBuf := ntfs.CollectMFTArea(hd, partitionOffset)
		ntfs.ProcessMFT(MFTAreaBuf)
		if *fileExtension != "" {
			ntfs.LocateRecordsByExtension(*fileExtension)
		}

	}

	if *inputfile != "Disk MFT" {
		mftTable := MFT.MFTTable{Filepath: *inputfile}
		mftTable.Populate()

	}

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

	t := tree.Tree{}

	fmt.Printf("Building tree from MFT records \n")
	for _, record := range records {
		if record.Entry < 5 {
			continue
		}
		t.BuildTree(record)
	}
	if *showFSStructure {
		t.Show()
	}

} //ends for
