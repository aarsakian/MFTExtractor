package main

import (
	//"C"

	//"database/sql"

	"flag"
	"fmt"
	"log"
	"math"
	"path"
	"strings"

	ewfLib "github.com/aarsakian/EWF_Reader/ewf"

	disk "github.com/aarsakian/MFTExtractor/Disk"
	ntfsLib "github.com/aarsakian/MFTExtractor/FS/NTFS"
	"github.com/aarsakian/MFTExtractor/MFT"
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
	inputfile := flag.String("MFT", "Disk MFT", "absolute path to the MFT file")
	evidencefile := flag.String("evidence", "", "path to image file")
	exportLocation := flag.String("export", "", "the path to export  files")
	MFTSelectedEntry := flag.Int("entry", -1, "select a particular MFT entry")
	showFileName := flag.String("fileName", "", "show the name of the filename attribute of each MFT record choices: Any, Win32, Dos")
	isResident := flag.Bool("resident", false, "check whether entry is resident")
	fromMFTEntry := flag.Int("fromEntry", -1, "select entry to start parsing")
	toMFTEntry := flag.Int("toEntry", math.MaxUint32, "select entry to end parsing")
	showRunList := flag.Bool("runlist", false, "show runlist of MFT record attributes")
	showFileSize := flag.Bool("filesize", false, "show file size of a record holding a file")
	showVCNs := flag.Bool("vcns", false, "show the vncs of non resident attributes")
	showAttributes := flag.String("attributes", "", "show attributes")
	showTimestamps := flag.Bool("timestamps", false, "show all timestamps")
	showIndex := flag.Bool("index", false, "show index structures")
	physicalDrive := flag.Int("physicalDrive", -1, "select disk drive number for extraction of non resident files")
	partitionNum := flag.Int("partitionNumber", -1, "select partition number")
	showFSStructure := flag.Bool("structure", false, "reconstrut entries tree")
	listPartitions := flag.Bool("listpartitions", false, "list partitions")
	//fileExtension := flag.String("extension", "", "search MFT records by extension")

	flag.Parse() //ready to parse

	var partitionOffset uint64

	var ntfs ntfsLib.NTFS
	var hD img.DiskReader
	var records []MFT.Record

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

	if *physicalDrive != -1 && *partitionNum != -1 {
		physicalDisk := disk.Disk{PhysicalDriveNum: *physicalDrive}
		physicalDisk.Populate()
		if *listPartitions {
			physicalDisk.ListPartitions()
		}
		partition := physicalDisk.GetSelectedPartition(*partitionNum)

		partitionOffsetB := int64(partition.GetOffset() * 512)
		length := uint32(512)
		buffer := make([]byte, length)

		hD = img.GetHandler(fmt.Sprintf("\\\\.\\PHYSICALDRIVE%d", *physicalDrive))
		hD.ReadFile(partitionOffsetB, buffer)
		defer hD.CloseHandler()

		fs := partition.LocateFileSystem(buffer)

		fs.Process(hD, partitionOffsetB, *MFTSelectedEntry, *fromMFTEntry, *toMFTEntry)

		/*if *fileExtension != "" {
			records = ntfs.FilterRecordsByExtension(*fileExtension)
		} else {
			records = ntfs.MFTTable.Records
		}*/

	}

	if *evidencefile != "" {
		extension := path.Ext(*evidencefile)
		if strings.ToLower(extension) == ".e01" {
			var ewf_image ewfLib.EWF_Image
			filenames := utils.FindEvidenceFiles(*evidencefile)

			ewf_image.ParseEvidence(filenames)

			physicalDisk := disk.Disk{Image: &ewf_image}
			physicalDisk.Populate()
			partition := physicalDisk.GetSelectedPartition(*partitionNum)
			partitionOffset = partition.GetOffset()

			length := uint32(512)
			buffer := make([]byte, length)

			partition.LocateFileSystem(buffer)

		}
	}

	if *inputfile != "Disk MFT" {
		mftTable := MFT.MFTTable{Filepath: *inputfile}
		mftTable.Populate(*MFTSelectedEntry, *fromMFTEntry, *toMFTEntry)
		records = mftTable.Records
	}
	rp.Show(records)

	if *exportLocation != "" && *physicalDrive != -1 && *partitionNum != -1 {
		sectorsPerCluster := ntfs.GetSectorsPerCluster()
		exp := exporter.Exporter{Disk: *physicalDrive, PartitionOffset: partitionOffset,
			SectorsPerCluster: sectorsPerCluster, Location: *exportLocation}
		exp.ExportData(records)

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
