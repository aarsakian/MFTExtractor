package main

import (
	//"C"

	//"database/sql"

	"flag"
	"log"
	"math"
	"strings"
	"time"

	EWFLogger "github.com/aarsakian/EWF_Reader/logger"

	"github.com/aarsakian/MFTExtractor/FS/NTFS/MFT"
	UsnJrnl "github.com/aarsakian/MFTExtractor/FS/NTFS/usnjrnl"
	"github.com/aarsakian/MFTExtractor/disk"
	"github.com/aarsakian/MFTExtractor/disk/volume"
	"github.com/aarsakian/MFTExtractor/exporter"
	"github.com/aarsakian/MFTExtractor/filtermanager"
	"github.com/aarsakian/MFTExtractor/filters"
	FSLogger "github.com/aarsakian/MFTExtractor/logger"
	"github.com/aarsakian/MFTExtractor/reporter"
	"github.com/aarsakian/MFTExtractor/tree"
	"github.com/aarsakian/MFTExtractor/utils"
	VMDKLogger "github.com/aarsakian/VMDK_Reader/logger"
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

	buildtree := flag.Bool("tree", false, "reconstrut entries tree")

	showtree := flag.Bool("showtree", false, "show tree")
	showParent := flag.Bool("parent", false, "show information about parent record")
	showUsnjrnl := flag.Bool("showusn", false, "show information about usnjrnl records")
	showFull := flag.Bool("showfull", false, "show full information about record")

	orphans := flag.Bool("orphans", false, "show information only for orphan records")
	deleted := flag.Bool("deleted", false, "show deleted records")

	listPartitions := flag.Bool("listpartitions", false, "list partitions")
	fileExtensions := flag.String("extensions", "", "search MFT records by extensions use , for each extension")
	collectUnallocated := flag.Bool("unallocated", false, "collect unallocated area of a file system")
	hashFiles := flag.String("hash", "", "select hash md5 or sha1 for exported files.")
	volinfo := flag.Bool("volinfo", false, "show volume information")
	logactive := flag.Bool("log", false, "enable logging")
	showPath := flag.Bool("showpath", false, "show the full path of the selected files.")
	strategy := flag.String("strategy", "overwrite", "what strategy will use for files sharing the same file name supported is use Id default is ovewrite.")
	usnjrnl := flag.Bool("usnjrnl", false, "show information about changes to files and folders.")

	flag.Parse() //ready to parse

	var records MFT.Records
	var usnjrnlRecords UsnJrnl.Records
	var fileNamesToExport []string

	entries := utils.GetEntriesInt(*MFTSelectedEntries)

	if *usnjrnl {
		fileNamesToExport = append(fileNamesToExport, "$UsnJrnl")
	}

	recordsTree := tree.Tree{}

	rp := reporter.Reporter{
		ShowFileName:   *showFileName,
		ShowAttributes: *showAttributes,
		ShowTimestamps: *showTimestamps,
		IsResident:     *isResident,
		ShowFull:       *showFull,
		ShowRunList:    *showRunList,
		ShowFileSize:   *showFileSize,
		ShowVCNs:       *showVCNs,
		ShowIndex:      *showIndex,
		ShowParent:     *showParent,
		ShowPath:       *showPath,
		ShowUSNJRNL:    *showUsnjrnl,
		ShowTree:       *showtree,
	}

	if *logactive {
		now := time.Now()
		logfilename := "logs" + now.Format("2006-01-02T15_04_05") + ".txt"
		FSLogger.InitializeLogger(*logactive, logfilename)
		VMDKLogger.InitializeLogger(*logactive, logfilename)
		EWFLogger.InitializeLogger(*logactive, logfilename)

	}

	exp := exporter.Exporter{Location: location, Hash: *hashFiles, Strategy: *strategy}

	flm := filtermanager.FilterManager{}

	if *exportFiles != "" {
		fileNamesToExport = append(fileNamesToExport, utils.GetEntries(*exportFiles)...)
		flm.Register(filters.FoldersFilter{Include: false})
		flm.Register(filters.NameFilter{Filenames: fileNamesToExport})
	}

	if *fileExtensions != "" {
		flm.Register(filters.ExtensionsFilter{Extensions: strings.Split(*fileExtensions, ",")})
	}

	if *exportFilesPath != "" {
		flm.Register(filters.PathFilter{NamePath: *exportFilesPath})
	}

	if *orphans {
		flm.Register(filters.OrphansFilter{Include: *orphans})
	}

	if *deleted {
		flm.Register(filters.DeletedFilter{Include: *deleted})
	}

	if *evidencefile != "" || *physicalDrive != -1 || *vmdkfile != "" {
		physicalDisk := new(disk.Disk)
		physicalDisk.Initialize(*evidencefile, *physicalDrive, *vmdkfile)

		recordsPerPartition := physicalDisk.Process(*partitionNum, entries, *fromMFTEntry, *toMFTEntry)
		defer physicalDisk.Close()
		if *listPartitions {
			physicalDisk.ListPartitions()
		}

		if *volinfo {
			physicalDisk.ShowVolumeInfo()
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

			if *buildtree {
				recordsTree.Build(records)

			}

			rp.Show(records, usnjrnlRecords, partitionId, recordsTree)

		}

	} else if *inputfile != "Disk MFT" {

		data, fsize, err := utils.ReadFile(*inputfile)
		if err != nil {
			return
		}
		var ntfs volume.NTFS

		ntfs.MFT = &MFT.MFTTable{Size: fsize}
		ntfs.ProcessMFT(data, entries, *fromMFTEntry, *toMFTEntry)

		records = flm.ApplyFilters(ntfs.MFT.Records)

		if *buildtree {
			recordsTree.Build(records)

		}

		rp.Show(records, usnjrnlRecords, 0, recordsTree)

	}

} //ends for
