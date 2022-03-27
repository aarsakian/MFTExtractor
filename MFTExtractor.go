package main

import (
	"C"
	"bytes"

	//"database/sql"
	"encoding/binary"
	"flag"
	"fmt"
	"log"
	"math"
	"os"

	//	"github.com/coopernurse/gorp"
	//	_ "github.com/mattn/go-sqlite3"
	// "gob"//de-serialization
	// "math"
	"github.com/aarsakian/MFTExtractor/MFT"
)

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

func readEndian(barray []byte) (val interface{}) {
	//conversion function
	//fmt.Println("before conversion----------------",barray)
	//fmt.Printf("len%d ",len(barray))

	switch len(barray) {
	case 8:
		var vale uint64
		binary.Read(bytes.NewBuffer(barray), binary.LittleEndian, &vale)
		val = vale
	case 6:

		var vale uint32
		buf := make([]byte, 6)
		binary.Read(bytes.NewBuffer(barray[:4]), binary.LittleEndian, &vale)
		var vale1 uint16
		binary.Read(bytes.NewBuffer(barray[4:]), binary.LittleEndian, &vale1)
		binary.LittleEndian.PutUint32(buf[:4], vale)
		binary.LittleEndian.PutUint16(buf[4:], vale1)
		val, _ = binary.ReadUvarint(bytes.NewBuffer(buf))

	case 4:
		var vale uint32
		//   fmt.Println("barray",barray)
		binary.Read(bytes.NewBuffer(barray), binary.LittleEndian, &vale)
		val = vale
	case 2:

		var vale uint16

		binary.Read(bytes.NewBuffer(barray), binary.LittleEndian, &vale)
		//   fmt.Println("after conversion vale----------------",barray,vale)
		val = vale

	case 1:

		var vale uint8

		binary.Read(bytes.NewBuffer(barray), binary.LittleEndian, &vale)
		//      fmt.Println("after conversion vale----------------",barray,vale)
		val = vale

	default: //best it would be nil
		var vale uint64

		binary.Read(bytes.NewBuffer(barray), binary.LittleEndian, &vale)
		val = vale
	}

	//     b:=[]byte{0x18,0x2d}

	//    fmt.Println("after conversion val",val)
	return val
}

func readEndianFloat(barray []byte) (val uint64) {

	//    fmt.Printf("len%d ",len(barray))

	binary.Read(bytes.NewBuffer(barray), binary.LittleEndian, &val)
	return val
}

func readEndianString(barray []byte) (val []byte) {

	binary.Read(bytes.NewBuffer(barray), binary.LittleEndian, &val)

	return val
}

func main() {
	//dbmap := initDb()
	//defer dbmap.Db.Close()

	//	save2DB := flag.Bool("db", false, "bool if set an sqlite file will be created, each table will corresponed to an MFT attribute")
	inputfile := flag.String("MFT", "MFT file", "absolute path to the MFT file")
	exportFiles := flag.String("Export", "None", "export resident files")
	MFTSelectedEntry := flag.Int("Entry", -1, "select a particular MFT entry")
	showFileName := flag.Bool("FileName", false, "show the name of the filename attribute of each MFT record")
	isResident := flag.Bool("Resident", false, "check whether entry is resident")
	fromMFTEntry := flag.Int("fromEntry", 0, "select entry to start parsing")
	ToMFTEntry := flag.Int("toEntry", math.MaxUint32, "select entry to end parsing")
	showRunList := flag.Bool("runlist", false, "show runlist of MFT record attributes")

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

	for i := 0; i <= int(fsize.Size()); i += 1024 {
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
			record.GetBasicInfoFromRecord(file1)

			if *exportFiles != "None" {
				record.CreateFileFromEntry(*exportFiles)
			}

			if *showFileName {
				record.ShowFileName()
			}

			if *isResident {
				record.ShowIsResident()
			}

			if *showRunList {
				record.ShowRunList()
			}

			if int(record.Entry) == *MFTSelectedEntry {
				break
			}

		}

	}
} //ends for
