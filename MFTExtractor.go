package main

import (
	"C"
	"bytes"
	"errors"
	"reflect"
	"strconv"

	//"database/sql"
	"encoding/binary"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"
	"unicode/utf16"
	"unicode/utf8"
	//	"github.com/coopernurse/gorp"
	//	_ "github.com/mattn/go-sqlite3"
	// "gob"//de-serialization
	// "math"
)

type MFTrecord struct {
	Signature          string //0-3
	UpdateSeqArrOffset uint16 //4-5      offset values are relative to the start of the entry.
	UpdateSeqArrSize   uint16 //6-7
	Lsn                uint64 //8-15       logical File Sequence Number
	Seq                uint16 //16-17   is incremented when the entry is either allocated or unallocated, determined by the OS.
	Linkcount          uint16 //18-19        how many directories have entries for this MFTentry
	AttrOff            uint16 //20-21       //first attr location
	Flags              uint16 //22-23  //tells whether entry is used or not
	Size               uint32 //24-27
	AllocSize          uint32 //28-31
	BaseRef            uint64 //32-39
	NextAttrID         uint16 //40-41 e.g. if it is 6 then there are attributes with 1 to 5
	F1                 uint16 //42-43
	Entry              uint32 //44-48                  ??
	Fncnt              bool
	Data               []byte
	Bitmap             bool
	// fixupArray add the        UpdateSeqArrOffset to find is location

}

type ATRrecordResident struct {
	Type          string //        0-3                              type of attribute e.g. $DATA
	Len           uint32 //4-8             length of attribute
	Res           string //8
	Nlen          string
	NameOff       uint16 //name offset 10-12          relative to the start of attribute
	Flags         uint16 //12-14           //compressed,
	ID            uint16 //14-16 type of attribute
	ContentSize   uint32 //16-20 size of resIDent attribute
	OffsetContent uint16 //20-22 offset to content            soff+ssize=len
	IDxflag       uint16 //22-24
	EntryID       uint32 //foreing key
	AttrID        uint16 //for DB use

}

type DATAResident struct {
	Header  *ATRrecordResident
	Content []byte
}

type ATRrecordNoNResIDent struct {
	Type       string //        0-3                              type of attribute e.g. $DATA
	Len        uint32 //4-8             length of attribute
	Res        string //8 bool in original
	Nlen       string
	NameOff    uint16 //name offset 10-12          relative to the start of attribute
	Flags      uint16 //12-14           //compressed,
	ID         uint16 //14-16
	StartVcn   uint64 //16-24
	LastVcn    uint64 //24-32
	RunOff     uint16 //32-24     offset to the start of the attribute
	Compusize  uint16 //34-36
	F1         uint32 //36-40
	Alen       uint64 //40-48
	NonRessize uint64 //48-56
	Initsize   uint64 //56-64
	EntryID    uint32 //foreing key
	AttrID     uint16 //for DB use

}

type WindowsTime struct {
	Stamp uint64
}

type FNAttribute struct {
	ParRef      uint64
	ParSeq      uint16
	Crtime      WindowsTime
	Mtime       WindowsTime //WindowsTime
	MFTmtime    WindowsTime //WindowsTime
	Atime       WindowsTime //WindowsTime
	AllocFsize  uint64
	RealFsize   uint64
	Flags       uint32 //hIDden Read Only? check Reparse
	Reparse     uint32
	Nlen        uint8  //length of name
	Nspace      uint8  //format of name
	Fname       string //special string type without nulls
	HexFlag     bool
	UnicodeHack bool
	EntryID     uint32 //foreing key
	AttrID      uint16 //for DB use
}

type ObjectID struct { //unique guID
	ObjID     string //object ID
	OrigVolID string //volume ID
	OrigObjID string //original objID
	OrigDomID string // domain ID
	EntryID   uint32 //foreing key
	AttrID    uint16
}

type VolumeName struct {
	Name    NoNull
	EntryID uint32 //foreing key
	AttrID  uint16 //for DB use
}

type IndexEntry struct {
	MFTfileref uint64 //0-7
	Len        uint16 //8-9
	Contentlen uint16 //10-11
	Flags      string //12-15

}

type IndexRoot struct {
	Type string //0-4 similar to FNA type
	// CollationSortingRule string
	Sizebytes    uint32 //8-12
	Sizeclusters uint8  //12-12
	nodeheader   NodeHeader
}

type NodeHeader struct {
	OffsetEntryList          uint32 // 16-20 see 13.14
	OffsetEndUsedEntryList   uint32 //20-24 where EntryList ends
	OffsetEndEntryListBuffer uint32 //24-28
	Flags                    string
}
type IndexAllocation struct {
	Signature        string //0-4
	FixupArrayOffset int16  //4-6
	NumEntries       int16  //6-8
	LSN              int64  //8-16
	VCN              int64  //16-24 where the record fits in the tree
	nodeheader       NodeHeader
}
type AttributeList struct { //more than one MFT entry to store a file/directory its attributes
	Type       string //        typeif 0-4    # 4
	len        uint16 //4-6
	namelen    uint8  //7unsigned char           # 1
	nameoffset uint8  //8-8               # 1
	StartVcn   uint64 //8-16         # 8
	fileRef    uint64 //16-22      # 6
	seq        uint16 //       22-24    # 2
	ID         uint8  //     24-26   # 4
	name       NoNull
}

type VolumeInfo struct {
	F1      uint64 //unused
	MajVer  string
	MinVer  string
	Flags   string //see table 13.22
	F2      uint32
	EntryID uint32 //foreing key
	AttrID  uint16 //for DB use

}

type SIAttribute struct {
	Crtime   WindowsTime
	Mtime    WindowsTime
	MFTmtime WindowsTime
	Atime    WindowsTime
	Dos      uint32
	Maxver   uint32
	Ver      uint32
	ClassID  uint32
	OwnID    uint32
	SecID    uint32
	Quota    uint64
	Usn      uint64
	EntryID  uint32 //foreing key
	AttrID   uint16 //for DB use
}

func Bytereverse(barray []byte) []byte { //work with indexes
	//  fmt.Println("before",barray)
	for i, j := 0, len(barray)-1; i < j; i, j = i+1, j-1 {

		barray[i], barray[j] = barray[j], barray[i]

	}

	//  binary.Read(bytes.NewBuffer(barray)  ,binary.LittleEndian,&val )
	//     fmt.Println("after",barray)
	return barray

}

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

func Hexify(barray []byte) string {

	return hex.EncodeToString(barray)

}

func stringifyGuIDs(barray []byte) string {
	s := []string{Hexify(barray[0:4]), Hexify(barray[4:6]), Hexify(barray[6:8]), Hexify(barray[8:10]), Hexify(barray[10:16])}
	return strings.Join(s, "-")
}

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

func (winTime *WindowsTime) convertToIsoTime() string { //receiver winTime struct
	var localTime string
	if winTime.Stamp != 0 {
		// t:=math.Pow((uint64(winTime.high)*2),32) + uint64(winTime.low)
		x := winTime.Stamp/10000000 - 116444736*1e2
		unixtime := time.Unix(int64(x), 0).UTC()
		localTime = unixtime.Format("02-01-2006 15:04:05")
		// fmt.Println("time",unixtime.Format("02-01-2006 15:04:05"))

	} else {
		localTime = "---"
	}
	return localTime
}

func readEndianFloat(barray []byte) (val uint64) {

	//    fmt.Printf("len%d ",len(barray))

	binary.Read(bytes.NewBuffer(barray), binary.LittleEndian, &val)
	return val
}

func readEndianInt(barray []byte) uint64 {

	//fmt.Println("------",barray,barray[len(barray)-1])
	var sum uint64
	sum = 0
	for index, val := range barray {
		sum += uint64(val) << uint(index*8)

		//fmt.Println(sum)
	}

	return sum
}

type NoNull string

func readEndianString(barray []byte) (val []byte) {

	binary.Read(bytes.NewBuffer(barray), binary.LittleEndian, &val)

	return val
}

func (atrRecordNoNResident ATRrecordNoNResIDent) ProcessRunList(runlist []byte) {
	clusterPtr := uint64(0)

	// fmt.Printf("LEN %d RUNLIST %x\n" ,len(runlist),runlist)
	for clusterPtr < uint64(len(runlist)) { // length of bytes of runlist
		ClusterOffsB, ClusterLenB := determineClusterOffsetLength(runlist[clusterPtr])

		if ClusterLenB != 0 && ClusterOffsB != 0 {
			//   fmt.Println("reading from",uint64(ReadPtr)+uint64(atrRecordResident.RunOff)+uint64(index),"ews ",
			//     uint64(ReadPtr)+uint64(atrRecordResident.RunOff)+uint64(index)+ClusterLen+ClusterOffs,
			//    "atrRecordResident star                     ts at",ReadPtr+atrRecordResident.RunOff,"atrRecordResident LEN",uint16(atrRecordResident.Len),"reading at",uint64(ReadPtr)+uint64(atrRecordResident.RunOff)+uint64(index)+ClusterLen+ClusterOffs)

			ClustersLen := readEndianInt(runlist[clusterPtr+1 : clusterPtr+ClusterLenB+1])

			ClustersOff := readEndianInt(runlist[clusterPtr+1+ClusterLenB : clusterPtr+ClusterLenB+ClusterOffsB+1])
			fmt.Printf("len of %d clusterlen %d and clust %d clustoff %d came from %x \n", ClusterLenB, ClustersLen, ClusterOffsB, ClustersOff, runlist[clusterPtr])
			//readEndianInt(bs[uint64(ReadPtr)+uint64(atrRecordResident.RunOff)+1:uint64(ReadPtr)+uint64(atrRecordResident.RunOff)+ClusterLen+1]))
			/*s := []string{fmt.Sprintf(";%d;%d", ClustersOff, ClustersLen)}
			_, err := file1.WriteString(strings.Join(s, " "))
			if err != nil {
				// handle the error here
				fmt.Printf("err %s\n", err)
				return
			}*/

			//fmt.Println("lenght of runlist",len(runlist),"cluster len" ,ClusterLen+ClusterOffs,"runlist",runlist)

			clusterPtr += ClusterLenB + ClusterOffsB

		} else {
			break
		}
	}
}

func determineClusterOffsetLength(val byte) (uint64, uint64) {

	var err error

	clusterOffs := uint64(0)
	clusterLen := uint64(0)

	val1 := (fmt.Sprintf("%x", val))

	if len(val1) == 2 { //requires non zero hex

		clusterLen, err = strconv.ParseUint(val1[1:2], 8, 8)
		if err != nil {
			fmt.Printf("error finding cluster length %s", err)
		}

		clusterOffs, err = strconv.ParseUint(val1[0:1], 8, 8)
		if err != nil {
			fmt.Printf("error finding cluster offset %s", err)
		}

	}
	//  fmt.Printf("Cluster located at %s and lenght %s\n",ClusterOffs, ClusterLen)
	return clusterOffs, clusterLen

}

func DecodeUTF16(b []byte) string {
	utf := make([]uint16, (len(b)+(2-1))/2) //2 bytes for one char?
	for i := 0; i+(2-1) < len(b); i += 2 {
		utf[i/2] = binary.LittleEndian.Uint16(b[i:])
	}
	if len(b)/2 < len(utf) {
		utf[len(utf)-1] = utf8.RuneError
	}
	return string(utf16.Decode(utf))

}
func (str *NoNull) PrintNulls() string {
	var newstr []string
	for _, v := range *str {
		if v != 0 {

			newstr = append(newstr, string(v))

		}
	}
	return strings.Join(newstr, "")
}

func Unmarshal(data []byte, v interface{}) error {
	idx := 0
	structValPtr := reflect.ValueOf(v)
	structType := reflect.TypeOf(v)
	if structType.Elem().Kind() != reflect.Struct {
		return errors.New("must be a struct")
	}
	for i := 0; i < structValPtr.Elem().NumField(); i++ {
		field := structValPtr.Elem().Field(i) //StructField type
		switch field.Kind() {
		case reflect.String:
			name := structType.Elem().Field(i).Name
			if name == "Signature" {
				field.SetString(string(data[idx : idx+4]))
				idx += 4
			} else if name == "Type" {
				field.SetString(Hexify(Bytereverse(data[idx : idx+4])))
				idx += 4
			} else if name == "Res" || name == "Len" {
				field.SetString(Hexify(Bytereverse(data[idx : idx+2])))
				idx += 2
			} else if name == "ObjID" || name == "OrigVolID" ||
				name == "OrigObjID" || name == "OrigDomID" {
				field.SetString(stringifyGuIDs(data[idx : idx+16]))
				idx += 16
			}
		case reflect.Struct:
			var windowsTime WindowsTime
			Unmarshal(data[idx:idx+8], &windowsTime)
			//	windowsTimePtr := &windowsTime
			field.Set(reflect.ValueOf(windowsTime))
			idx += 8
		case reflect.Uint16:
			var temp uint16
			binary.Read(bytes.NewBuffer(data[idx:idx+2]), binary.LittleEndian, &temp)
			field.SetUint(uint64(temp))
			idx += 2
		case reflect.Uint32:
			var temp uint32
			binary.Read(bytes.NewBuffer(data[idx:idx+4]), binary.LittleEndian, &temp)
			field.SetUint(uint64(temp))
			idx += 4
		case reflect.Uint64:
			var temp uint64
			binary.Read(bytes.NewBuffer(data[idx:idx+8]), binary.LittleEndian, &temp)
			field.SetUint(temp)
			idx += 8
		case reflect.Bool:
			field.SetBool(false)
			idx += 1

		}

	}
	return nil
}

func main() {
	//dbmap := initDb()
	//defer dbmap.Db.Close()

	//	save2DB := flag.Bool("db", false, "bool if set an sqlite file will be created, each table will corresponed to an MFT attribute")
	inputfile := flag.String("MFT", "MFT file", "absolute path to the MFT file")
	flag.Parse() //ready to parse

	//err := dbmap.TruncateTables()
	//checkErr(err, "TruncateTables failed")
	IndexEntryFlags := map[string]string{
		"00000001": "Child Node exists",
		"00000002": "Last Entry in list",
	}

	AttrTypes := map[string]string{
		"00000010": "Standard Information", "00000020": "Attribute List", "00000030": "File Name", "00000040": "Object ID",
		"00000050": "Security Descriptor", "00000060": "Volume Name", "00000070": "Volume Information", "00000080": "Data",
		"00000090": "Index Root", "000000A0": "Index Allocation", "000000B0": "BitMap", "000000C0": "Reparse Point",
	}

	Flags := map[uint32]string{
		1: "Read Only", 2: "HIDden", 4: "System", 32: "Archive", 64: "Device", 128: "Normal",
		256: "Temporary", 512: "Sparse", 1024: "Reparse Point", 2048: "Compressed", 4096: "Offline",
		8192: "Not Indexed", 16384: "Encrypted",
	}

	MFTflags := map[uint16]string{
		0: "File Unallocted", 1: "File Allocated", 2: "Folder Unalloc", 3: "Folder Allocated",
	}
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
	file1, err := os.OpenFile("MFToutput.txt", os.O_RDWR|os.O_CREATE, 0666)

	if err != nil {
		// handle the error here
		fmt.Printf("err %s", err)
		return
	}
	defer file.Close()
	defer file1.Close()

	var k = 0
	_, err1 := file1.WriteString("FILE SIZE------------------------------------------------------------" + fmt.Sprintf("%d", fsize.Size()) + string(10))
	if err1 != nil {
		// handle the error here
		fmt.Printf("err %s\n", err)
		return
	}
	bs := make([]byte, 1024) //new byte array of length MFT entry

	for i := 0; i <= int(fsize.Size()); i += 1024 {
		fmt.Printf("Processing $MFT entry %d \n", i/1024)
		_, err := file.ReadAt(bs, int64(i))
		// fmt.Printf("\n I read %s and out is %d\n",hex.Dump(bs[20:22]), readEndian(bs[20:22]).(uint16))
		if err != nil {
			fmt.Printf("error reading file --->%s", err)
			return
		}

		if string(bs[:4]) == "FILE" {

			var record MFTrecord
			Unmarshal(bs, &record)

			/*		if *save2DB {
					dbmap.Insert(&r)
					checkErr(err, "Insert failed")
				}*/

			_, err1 := file1.WriteString(fmt.Sprintf("\n%d;%d;%s", record.Entry, record.Seq,
				MFTflags[record.UpdateSeqArrSize]))
			if err1 != nil {
				// handle the error here
				// fmt.Printf("err %s\n",err)
				return
			}

			// err = dbmap.Insert(&r)
			// checkErr(err, "Insert failed")

			if record.Signature == "BAAD" { //skip bad entry
				continue
			}

			ReadPtr := record.AttrOff //offset to first attribute
			//  fmt.Println("ResIDent? ",Bytereverse(bs[ReadPtr+8:ReadPtr+9]))
			for ReadPtr < 1024 {

				if hex.EncodeToString(bs[ReadPtr:ReadPtr+4]) == "ffffffff" { //End of attributes
					break
				}

				// fmt.Printf("Type %s ResIDnet  Attr %b Endian \n",hex.EncodeToString(bs[ReadPtr:ReadPtr+4]),readEndianString(bs[ReadPtr:ReadPtr+4]))
				if Hexify(bs[ReadPtr+8:ReadPtr+9]) == "00" { //ResIDent Attribute
					var atrRecordResident ATRrecordResident
					Unmarshal(bs[ReadPtr:ReadPtr+24], &atrRecordResident)

					/*	if *save2DB {
						dbmap.Insert(&atrRecordResident)
						checkErr(err, "Insert failed")
					}*/
					//   fmt.Printf("ResIDent type %s where data length %d and starts at %d ,Attribute length %d  equal>%b \n",atrRecordResident.Type,atrRecordResident.ssize,atrRecordResident.OffsetContent,atrRecordResident.Len,uint32(atrRecordResident.OffsetContent)+atrRecordResident.ssize==atrRecordResident.Len)
					s := strings.Join([]string{";", AttrTypes[atrRecordResident.Type]}, "")
					_, err := file1.WriteString(s)
					if err != nil {
						// handle the error here
						fmt.Printf("err %s\n", err)
						return
					}
					if atrRecordResident.Type == "ffffffff" { // End of attributes
						break

					} else if atrRecordResident.Type == "00000030" { // File name
						var fnattr FNAttribute
						Unmarshal(bs[ReadPtr+atrRecordResident.OffsetContent:ReadPtr+atrRecordResident.OffsetContent+66], &fnattr)

						fnattr.Fname =
							DecodeUTF16(bs[ReadPtr+atrRecordResident.OffsetContent+66 : ReadPtr+atrRecordResident.OffsetContent+66+2*uint16(readEndian(bs[ReadPtr+atrRecordResident.OffsetContent+64:ReadPtr+atrRecordResident.OffsetContent+65]).(uint8))])
						//	false, false, record.Entry, 0}
						//  fmt.Println("\nFNA ",bs[ReadPtr+atrRecordResident.OffsetContent:ReadPtr+atrRecordResident.OffsetContent+65],bs[ReadPtr+atrRecordResident.OffsetContent:ReadPtr+atrRecordResident.OffsetContent+6],readEndian(bs[ReadPtr+atrRecordResident.OffsetContent:ReadPtr+atrRecordResident.OffsetContent+6]).(uint64),
						//	"PAREF",fnattr.ParRef,"SQ",fnattr.fname,"FLAG",fnattr.flags)
						//   fmt.Printf("time Mod %s time Accessed %s time Created %s Filename %s\n ", fnattr.atime.convertToIsoTime(),fnattr.crtime.convertToIsoTime(),fnattr.mtime.convertToIsoTime(),fnattr.fname )
						//    fmt.Println(strings.TrimSpace(string(bs[ReadPtr+atrRecordResident.OffsetContent+66:ReadPtr+atrRecordResident.OffsetContent+66+2*uint16(readEndian(bs[ReadPtr+atrRecordResident.OffsetContent+64:ReadPtr+atrRecordResident.OffsetContent+65]).(uint8))])))
						/*if *save2DB {
							dbmap.Insert(&fnattr)
							checkErr(err, "Insert failed")
						}*/
						s := strings.Join([]string{fmt.Sprintf(";%d", ReadPtr+atrRecordResident.OffsetContent), ";", fnattr.Atime.convertToIsoTime(), ";", fnattr.Crtime.convertToIsoTime(),
							";", fnattr.Mtime.convertToIsoTime(), ";", fnattr.Fname, fmt.Sprintf(";%d;%d;%s", fnattr.ParRef, fnattr.ParSeq, Flags[fnattr.Flags])}, "")

						_, err := file1.WriteString(s) //(string(10)) breaks line
						if err != nil {
							// handle the error here
							fmt.Printf("err %s\n", err)
							return
						}
					} else if atrRecordResident.Type == "00000080" {
						record.Data = bs[ReadPtr+atrRecordResident.OffsetContent : ReadPtr+atrRecordResident.OffsetContent+uint16(atrRecordResident.ContentSize)]

						/*	_, err := file1.WriteString(";" + strconv.FormatBool(record.Data))
							if err != nil {
								// handle the error here
								fmt.Printf("err %s\n", err)
								return
							}*/

					} else if atrRecordResident.Type == "00000040" {
						var objectattr ObjectID
						Unmarshal(bs[ReadPtr+atrRecordResident.OffsetContent:ReadPtr+atrRecordResident.OffsetContent+64], &objectattr)

						// fmt.Println("file unique ID ",objectattr.objID)
						/*	if *save2DB {
							dbmap.Insert(&objectattr)
							checkErr(err, "Insert failed")
						}*/
						s := []string{";", objectattr.ObjID}
						_, err := file1.WriteString(strings.Join(s, " "))
						if err != nil {
							// handle the error here
							fmt.Printf("err %s\n", err)
							return
						}
					} else if atrRecordResident.Type == "00000020" { //Attribute List
						//  runlist:=bs[ReadPtr+atrRecordResident.OffsetContent:uint32(ReadPtr)+atrRecordResident.Len]

						attrLen := uint16(0)
						for atrRecordResident.OffsetContent+24+attrLen < uint16(atrRecordResident.Len) {
							//fmt.Println("TEST",len(runlist),26+attrLen+atrRecordResident.OffsetContent, uint16(atrRecordResident.Len))
							var attrList AttributeList
							Unmarshal(bs[ReadPtr+atrRecordResident.OffsetContent+attrLen:ReadPtr+atrRecordResident.OffsetContent+attrLen+24], &attrList)
							attrList.name = NoNull(bs[ReadPtr+atrRecordResident.OffsetContent+attrLen+uint16(attrList.nameoffset) : ReadPtr+atrRecordResident.OffsetContent+attrLen+uint16(attrList.nameoffset)+uint16(attrList.namelen)])
							//     fmt.Println("START VCN",attrList.StartVcn)

							s := []string{"Type of Attr in Run list", fmt.Sprintf("Attribute starts at %d", ReadPtr),
								AttrTypes[attrList.Type], fmt.Sprintf("length %d ", attrList.len), fmt.Sprintf("start VCN %d ", attrList.StartVcn),
								"MFT Record Number", fmt.Sprintf("%d Name %s", attrList.fileRef, attrList.name),
								"Attribute ID ", fmt.Sprintf("%d ", attrList.ID), string(10)}
							_, err := file1.WriteString(strings.Join(s, " "))
							if err != nil {
								// handle the error here
								fmt.Printf("err %s\n", err)
								return
							}

							//   runlist=bs[ReadPtr+atrRecordResident.OffsetContent+attrList.len:uint32(ReadPtr)+atrRecordResident.Len]
							attrLen += attrList.len

						}
					} else if atrRecordResident.Type == "000000b0" { //BITMAP
						record.Bitmap = true

					} else if atrRecordResident.Type == "00000060" { //Volume Name
						volname := VolumeName{NoNull(bs[ReadPtr+atrRecordResident.OffsetContent : ReadPtr+atrRecordResident.OffsetContent+16]), record.Entry, 0}
						/*		if *save2DB {
								dbmap.Insert(&volname)
								checkErr(err, "Insert failed")
							}*/

						s := []string{";", volname.Name.PrintNulls()}
						_, err := file1.WriteString(strings.Join(s, "s"))
						if err != nil {
							// handle the error here
							fmt.Printf("err %s\n", err)
							return
						}
					} else if atrRecordResident.Type == "00000070" { //Volume Info
						volinfo := VolumeInfo{readEndian(bs[ReadPtr+atrRecordResident.OffsetContent : ReadPtr+atrRecordResident.OffsetContent+8]).(uint64),
							Hexify(Bytereverse(bs[ReadPtr+atrRecordResident.OffsetContent+8 : ReadPtr+atrRecordResident.OffsetContent+9])),
							Hexify(Bytereverse(bs[ReadPtr+atrRecordResident.OffsetContent+9 : ReadPtr+atrRecordResident.OffsetContent+10])),
							Hexify(Bytereverse(bs[ReadPtr+atrRecordResident.OffsetContent+10 : ReadPtr+atrRecordResident.OffsetContent+12])),
							readEndian(bs[ReadPtr+atrRecordResident.OffsetContent+12 : ReadPtr+atrRecordResident.OffsetContent+16]).(uint32),
							record.Entry, 0}
						/*	if *save2DB {
							dbmap.Insert(&volinfo)
							checkErr(err, "Insert failed")
						}*/

						s := []string{"Vol Info flags", volinfo.Flags, string(10)}
						_, err := file1.WriteString(strings.Join(s, " "))
						if err != nil {
							// handle the error here
							fmt.Printf("err %s\n", err)
							return
						}
					} else if atrRecordResident.Type == "00000090" { //Index Root

						nodeheader := NodeHeader{readEndian(bs[ReadPtr+atrRecordResident.OffsetContent+16 : ReadPtr+atrRecordResident.OffsetContent+20]).(uint32), readEndian(bs[ReadPtr+atrRecordResident.OffsetContent+20 : ReadPtr+atrRecordResident.OffsetContent+24]).(uint32),
							readEndian(bs[ReadPtr+atrRecordResident.OffsetContent+24 : ReadPtr+atrRecordResident.OffsetContent+28]).(uint32), Hexify(Bytereverse(bs[ReadPtr+atrRecordResident.OffsetContent+28 : ReadPtr+atrRecordResident.OffsetContent+32]))}
						IDxroot := IndexRoot{string(bs[ReadPtr+atrRecordResident.OffsetContent : ReadPtr+atrRecordResident.OffsetContent+4]), readEndian(bs[ReadPtr+atrRecordResident.OffsetContent+8 : ReadPtr+atrRecordResident.OffsetContent+12]).(uint32),
							readEndian(bs[ReadPtr+atrRecordResident.OffsetContent+12 : ReadPtr+atrRecordResident.OffsetContent+13]).(uint8), nodeheader}

						IDxentry := IndexEntry{readEndian(bs[ReadPtr+atrRecordResident.OffsetContent+32 : ReadPtr+atrRecordResident.OffsetContent+40]).(uint64), readEndian(bs[ReadPtr+atrRecordResident.OffsetContent+40 : ReadPtr+atrRecordResident.OffsetContent+42]).(uint16),
							readEndian(bs[ReadPtr+atrRecordResident.OffsetContent+42 : ReadPtr+atrRecordResident.OffsetContent+44]).(uint16), Hexify(Bytereverse(bs[ReadPtr+atrRecordResident.OffsetContent+44 : ReadPtr+atrRecordResident.OffsetContent+48]))}
						//
						s := []string{IDxroot.Type, ";", fmt.Sprintf(";%d", IDxroot.Sizeclusters), ";", fmt.Sprintf("%d;", 16+IDxroot.nodeheader.OffsetEntryList),
							fmt.Sprintf(";%d", 16+IDxroot.nodeheader.OffsetEndUsedEntryList), fmt.Sprintf("allocated ends at %d", 16+IDxroot.nodeheader.OffsetEndEntryListBuffer),
							fmt.Sprintf("MFT entry%d ", IDxentry.MFTfileref), "FLags", IndexEntryFlags[IDxentry.Flags]}
						//fmt.Sprintf("%x",bs[uint32(ReadPtr)+uint32(atrRecordResident.OffsetContent)+32:uint32(ReadPtr)+uint32(atrRecordResident.OffsetContent)+16+IDxroot.nodeheader.OffsetEndEntryListBuffer]

						_, err := file1.WriteString(strings.Join(s, " "))
						if err != nil {
							// handle the error here
							fmt.Printf("err %s\n", err)
							return
						}
					} else if atrRecordResident.Type == "00000010" { //Standard Information
						startpoint := ReadPtr + atrRecordResident.OffsetContent
						var siattr SIAttribute
						Unmarshal(bs[startpoint:startpoint+72], &siattr)

						/*	if *save2DB {
							dbmap.Insert(&siattr)
							checkErr(err, "Insert failed")
						}*/
						s := []string{fmt.Sprintf(";%d", startpoint), ";", siattr.Crtime.convertToIsoTime(),
							";", siattr.Atime.convertToIsoTime(), ";", siattr.Mtime.convertToIsoTime(), ";", siattr.MFTmtime.convertToIsoTime()}
						_, err := file1.WriteString(strings.Join(s, ""))
						if err != nil {
							// handle the error here
							fmt.Printf("err %s\n", err)
							return
						}

					}

					ReadPtr = ReadPtr + uint16(atrRecordResident.Len)

				} else { //NoN ResIDent Attribute
					var atrNoNRecordResident ATRrecordNoNResIDent
					Unmarshal(bs[ReadPtr:ReadPtr+64], &atrNoNRecordResident)

					//        fmt.Println("NON ResIDent type ",atrRecordResident.Type,atrRecordResident.Len,ReadPtr)
					//  buffer:=[] byte{}// a slice
					/*		if *save2DB {
							dbmap.Insert(&atrRecordResident)
							checkErr(err, "Insert failed")
						}*/

					s := []string{";", AttrTypes[atrNoNRecordResident.Type], fmt.Sprintf(";%d", ReadPtr), ";false", fmt.Sprintf(";%d;%d", atrNoNRecordResident.StartVcn, atrNoNRecordResident.LastVcn)}
					_, err := file1.WriteString(strings.Join(s, ""))
					if err != nil {
						// handle the error here
						fmt.Printf("err %s\n", err)
						return
					}
					if atrNoNRecordResident.Type == "ffffffff" { // End of attributes
						break

					} else if atrNoNRecordResident.Type == "00000080" {

						if uint32(ReadPtr)+atrNoNRecordResident.Len <= 1024 {
							atrNoNRecordResident.ProcessRunList(bs[ReadPtr+atrNoNRecordResident.RunOff : uint32(ReadPtr)+atrNoNRecordResident.Len])

						}
					}

					//s := [] string {fmt.Sprintf("Start VCN %d END VCN %d",atrRecordResident.StartVcn,atrRecordResident.LastVcn ), string(10)}
					// _,err:=file1.WriteString(strings.Join(s," "))
					//  if err != nil {
					// handle the error here
					//   fmt.Printf("err %s\n",err)
					//     return
					//  }

					/*else if  atrRecordResident.Type == "000000a0" {//Index Allcation
						     nodeheader := NodeHeader {readEndian(bs[ReadPtr+atrRecordResident.OffsetContent+16:ReadPtr+atrRecordResident.OffsetContent+20]).(uint32),readEndian(bs[ReadPtr+atrRecordResident.OffsetContent+20:ReadPtr+atrRecordResident.OffsetContent+24]).(uint32),
						    readEndian(bs[ReadPtr+atrRecordResident.OffsetContent+24:ReadPtr+atrRecordResident.OffsetContent+28]).(uint32)}
					        IDxall := IndexAllocation{string(bs[ReadPtr+atrRecordResident.OffsetContent:ReadPtr+atrRecordResident.OffsetContent+4]),readEndian(bs[ReadPtr+atrRecordResident.OffsetContent+4:ReadPtr+atrRecordResident.OffsetContent+6]).(uint16),readEndian(bs[ReadPtr+atrRecordResident.OffsetContent+6:ReadPtr+atrRecordResident.OffsetContent+8]).(uint16),
						    readEndian(bs[ReadPtr+atrRecordResident.OffsetContent+16:ReadPtr+atrRecordResident.OffsetContent+24]).(uint64), nodeheader}

						 s := [] string  {"Index Allocation Type ",IDxall.Type,fmt.Sprintf("VCN %d  ",IDxall.VCN),"Index entry start",fmt.Sprintf("%d",IDxall.nodeheader.OffsetEntryList),
						        fmt.Sprintf(" used portion ends at %d",IDxall.nodeheader.OffsetEndUsedEntryList),fmt.Sprintf("allocated ends at %d",IDxall.nodeheader.OffsetEndEntryListBuffer)  ,string(10)}
						  _,err:=file1.WriteString(strings.Join(s," "))
						  if err != nil {
						    // handle the error here
						    fmt.Printf("err %s\n",err)
						      return
						    }

					   }*/
					if atrNoNRecordResident.Len > 0 {
						ReadPtr = ReadPtr + uint16(atrNoNRecordResident.Len)
					}

				} //ends non resIDent

			} //ends while

		} //ends if
		k++
	}
} //ends for
