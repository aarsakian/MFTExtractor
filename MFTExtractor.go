package main

import (
	//"C"

	//"database/sql"

	"flag"
	"log"
	"math"
	"strings"
	"time"

	ntfslib "github.com/aarsakian/MFTExtractor/FS/NTFS"
	"github.com/aarsakian/MFTExtractor/FS/NTFS/MFT"
	UsnJrnl "github.com/aarsakian/MFTExtractor/FS/NTFS/usnjrnl"
	"github.com/aarsakian/MFTExtractor/disk"
	"github.com/aarsakian/MFTExtractor/exporter"
	"github.com/aarsakian/MFTExtractor/filtermanager"
	"github.com/aarsakian/MFTExtractor/filters"
	MFTExtractorLogger "github.com/aarsakian/MFTExtractor/logger"
	"github.com/aarsakian/MFTExtractor/tree"
	"github.com/aarsakian/MFTExtractor/utils"
	VMDKLogger "github.com/aarsakian/VMDK_Reader/logger"

	"github.com/aarsakian/MFTExtractor/reporter"
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
	inputfile := flag.String("MFT", "", "absolute path to the MFT file")
	evidencefile := flag.String("evidence", "", "path to image file (EWF formats are supported)")
	vmdkfile := flag.String("vmdk", "", "path to vmdk file (Sparse formats are supported)")

	flag.StringVar(&location, "location", "", "the path to export  files")
	MFTSelectedEntries := flag.String("entries", "", "select particular MFT entries, use comma as a seperator.")
	showFileName := flag.String("showfilename", "", "show the name of the filename attribute of each MFT record choices: Any, Win32, Dos")
	exportFiles := flag.String("filenames", "", "files to export use comma for each file")
	exportFilesPath := flag.String("path", "", "base path of files to exported must be absolute e.g. C:\\MYFILES\\ABC translates to MYFILES\\ABC")
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

	showFStree := flag.Bool("tree", false, "reconstrut entries tree")
	showParent := flag.Bool("parent", false, "show information about parent record")
	showUsnjrnl := flag.Bool("showusn", false, "show information about usnjrnl records")

	listPartitions := flag.Bool("listpartitions", false, "list partitions")
	fileExtensions := flag.String("extensions", "", "search MFT records by extensions use , for each extension")
	collectUnallocated := flag.Bool("unallocated", false, "collect unallocated area of a file system")
	hashFiles := flag.String("hash", "", "select hash md5 or sha1 for exported files.")
	logactive := flag.Bool("log", false, "enable logging")
	showPath := flag.Bool("showpath", false, "show the full path of the selected files.")
	strategy := flag.String("strategy", "overwrite", "what strategy will use for files sharing the same file name supported is use Id default is ovewrite.")
	usnjrnl := flag.Bool("usnjrnl", false, "show information about changes to files and folders.")

	flag.Parse() //ready to parse

	var records MFT.Records
	var usnjrnlRecords UsnJrnl.Records

	entries := utils.GetEntriesInt(*MFTSelectedEntries)
	fileNamesToExport := utils.GetEntries(*exportFiles)

	if *usnjrnl {
		fileNamesToExport = append(fileNamesToExport, "$UsnJrnl")
	}

	t := tree.Tree{}

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
		ShowPath:       *showPath,
		ShowUSNJRNL:    *showUsnjrnl,
	}

	if *logactive {
		now := time.Now()
		logfilename := "logs" + now.Format("2006-01-02T15_04_05") + ".txt"
		MFTExtractorLogger.InitializeLogger(*logactive, logfilename)
		VMDKLogger.InitializeLogger(*logactive, logfilename)

	}

	exp := exporter.Exporter{Location: location, Hash: *hashFiles, Strategy: *strategy}

	flm := filtermanager.FilterManager{}

	if len(fileNamesToExport) != 0 {
		flm.Register(filters.FoldersFilter{Include: false})
		flm.Register(filters.NameFilter{Filenames: fileNamesToExport})
	}

	if *fileExtensions != "" {
		flm.Register(filters.ExtensionsFilter{Extensions: strings.Split(*fileExtensions, ",")})
	}

	if *exportFilesPath != "" {
		flm.Register(filters.PathFilter{NamePath: *exportFilesPath})
	}

	if *evidencefile != "" || *physicalDrive != -1 || *vmdkfile != "" {
		physicalDisk := new(disk.Disk)
		physicalDisk.Initialize(*evidencefile, *physicalDrive, *vmdkfile)

		recordsPerPartition := physicalDisk.Process(*partitionNum, entries, *fromMFTEntry, *toMFTEntry)
		defer physicalDisk.Close()
		if *listPartitions {
			physicalDisk.ListPartitions()
		}

		if *collectUnallocated {
			exp.ExportUnallocated(*physicalDisk)
		}

		for partitionId, records := range recordsPerPartition {

			records = flm.ApplyFilters(records)

			if location != "" {
				exp.ExportRecords(records, *physicalDisk, partitionId)
				if *hashFiles != "" {
					exp.HashFiles(records)
				}
			}

			if *usnjrnl {
				usnjrnlRecords = UsnJrnl.Process(records, *physicalDisk, partitionId)
			}

			rp.Show(records, usnjrnlRecords, partitionId)

			if *showFStree {
				t.Build(records)
				t.Show()

			}
		}

	} else if *inputfile != "Disk MFT" {

		data, fsize, err := utils.ReadFile(*inputfile)
		if err != nil {
			return
		}
		var ntfs ntfslib.NTFS

		ntfs.MFTTable = &MFT.MFTTable{Size: fsize}
		ntfs.ProcessMFT(data, entries, *fromMFTEntry, *toMFTEntry)

		records = flm.ApplyFilters(ntfs.MFTTable.Records)

		if len(records) == 0 {
			return
		}

		rp.Show(records, usnjrnlRecords, 0)

	}

	if *showFStree {
		t.Build(records)
		t.Show()
	}

} //ends for
