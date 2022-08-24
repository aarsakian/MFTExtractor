package main

import (
	"C"

	//"database/sql"

	"flag"
	"fmt"
	"log"
	"math"
	"os"

	"github.com/aarsakian/MFTExtractor/MFT"
	ntfsLib "github.com/aarsakian/MFTExtractor/NTFS"
)
import "github.com/aarsakian/MFTExtractor/tree"

func checkErr(err error, msg string) {
	if err != nil {
		log.Fatalln(msg, err)
	}
}

/*func initDb() *gorp.DbMap {
	// connect to db using standard Go database/sql API
	// use whatever database/sql driver you wish
	db, err := sql.Open("sqlite3", "./mft.sqlite")
	checkErr(err, "sql.Open failed")

	// construct a gorp DbMap
	dbmap := &gorp.DbMap{Db: db, Dialect: gorp.SqliteDialect{}}

	// add a table, setting the table name to 'posts' and
	// specifying that the ID property is an auto incrementing PK
	dbmap.AddTableWithName(MFTrecord{}, "MFTrecord").SetKeys(false, "Entry")
	dbmap.AddTableWithName(ATRrecordResIDent{}, "ATRrecordResIDent").SetKeys(true, "AttrID")
	dbmap.AddTableWithName(ATRrecordNoNResIDent{}, "ATRrecordNoNResIDent").SetKeys(true, "AttrID")
	dbmap.AddTableWithName(FNAttribute{}, "FNAttribute")
	dbmap.AddTableWithName(SIAttribute{}, "SIAttribute")
	dbmap.AddTableWithName(ObjectID{}, "ObjectID")
	dbmap.AddTableWithName(VolumeInfo{}, "VolumeInfo")
	dbmap.AddTableWithName(VolumeName{}, "VolumeName")
	// create the table. in a production system you'd generally
	// use a migration tool, or create the tables via scripts
	err = dbmap.CreateTablesIfNotExists()
	checkErr(err, "Create tables failed")

	return dbmap
}*/

func main() {
	//dbmap := initDb()
	//defer dbmap.Db.Close()

	//	save2DB := flag.Bool("db", false, "bool if set an sqlite file will be created, each table will corresponed to an MFT attribute")
	inputfile := flag.String("MFT", "MFT file", "absolute path to the MFT file")
	exportFiles := flag.String("export", "None", "export resident files")
	MFTSelectedEntry := flag.Int("entry", -1, "select a particular MFT entry")
	showFileName := flag.Bool("fileName", false, "show the name of the filename attribute of each MFT record")
	isResident := flag.Bool("resident", false, "check whether entry is resident")
	fromMFTEntry := flag.Int("fromEntry", 0, "select entry to start parsing")
	ToMFTEntry := flag.Int("toEntry", math.MaxUint32, "select entry to end parsing")
	showRunList := flag.Bool("runlist", false, "show runlist of MFT record attributes")
	showFileSize := flag.Bool("filesize", false, "show file size of a record holding a file")
	showVCNs := flag.Bool("vcns", false, "show the vncs of non resident attributes")
	showAttributes := flag.Bool("attributes", false, "show attributes")
	showTimestamps := flag.Bool("timestamps", false, "show all timestamps")
	showIndex := flag.Bool("index", false, "show index structures")
	physicalDrive := flag.String("physicalDrive", "", "use physical drive information for extraction of non resident files")

	flag.Parse() //ready to parse

	//err := dbmap.TruncateTables()
	//checkErr(err, "TruncateTables failed")

	//	fmt.Println(*inputfile, os.Args[1])
	file, err := os.Open(*inputfile) //

	if err != nil {
		// handle the error here
		fmt.Printf("err %s for reading the MFT ", err)
		return
	}

	// get the file size
	fsize, err := file.Stat() //file descriptor
	if err != nil {
		return
	}
	// read the file
	file1, err := os.OpenFile("MFToutput.csv", os.O_RDWR|os.O_CREATE, 0666)

	if err != nil {
		// handle the error here
		fmt.Printf("err %s", err)
		return
	}
	defer file.Close()
	defer file1.Close()

	bs := make([]byte, 1024) //byte array to hold MFT entries

	var records []MFT.MFTrecord

	for i := 0; i < int(fsize.Size()); i += 1024 {
		_, err := file.ReadAt(bs, int64(i))
		// fmt.Printf("\n I read %s and out is %d\n",hex.Dump(bs[20:22]), readEndian(bs[20:22]).(uint16))
		if err != nil {
			fmt.Printf("error reading file --->%s", err)
			return
		}

		if i/1024 > *ToMFTEntry {
			break
		}

		if *MFTSelectedEntry != -1 && i/1024 != *MFTSelectedEntry ||
			*fromMFTEntry > i/1024 || i/1024 > *ToMFTEntry {
			continue
		}

		if string(bs[:4]) == "FILE" {
			var record MFT.MFTrecord

			record.Process(bs)

			if *exportFiles != "None" && *physicalDrive != "" {
				ntfs := ntfsLib.Parse(*physicalDrive)
				record.CreateFileFromEntry(ntfs.SectorsPerCluster, *physicalDrive)

			}
			if *showFileName {
				record.ShowFileName("ANY")
			}

			if *showAttributes {
				record.ShowAttributes()
			}

			if *showTimestamps {
				record.ShowTimestamps()
			}

			if *isResident {
				record.ShowIsResident()
			}

			if *showRunList {
				record.ShowRunList()
			}

			if *showFileSize {
				record.ShowFileSize()
			}

			if *showVCNs {
				record.ShowVCNs()
			}

			if *showIndex {
				record.ShowIndex()
			}

			records = append(records, record)

			if int(record.Entry) == *MFTSelectedEntry {
				break
			}

		}

	}
	t := tree.Tree{}

	for _, record := range records {
		if record.Entry < 5 {
			continue
		}
		t.BuildTree(record)
	}
	t.Show()

} //ends for
