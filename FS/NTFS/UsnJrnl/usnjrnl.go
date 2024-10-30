package UsnJrnl

import (
	"sync"

	"github.com/aarsakian/MFTExtractor/utils"
)

var reasons = map[uint32]string{0x01: "File Overwritten", 0x02: "File or directory added", 0x404: "File or directory truncated"}

var source = map[uint32]string{0x00: "Normal Event", 0x01: "The operation provides information about a change to the file"}

var fileattributes = map[uint32]string{0x01: "Read-only file"}

type Records []Record

type Record struct {
	Size        uint32
	MajorVer    uint16
	MinorVer    uint16
	EntryRef    uint64
	EntrySeq    uint16
	ParRef      uint64
	ParSeq      uint16
	USN         uint64
	EventTime   utils.WindowsTime
	ReasonFlag  uint32
	SourceInfo  uint32
	SecurityId  [4]byte
	FileAttrs   uint32
	FnameLen    uint16 //length of name
	FnameOffset uint16 //format of name 58-60
	Fname       string //special string type without nulls

}

func (records Records) AsyncProcess(wg *sync.WaitGroup, dataClusters <-chan []byte) {
	defer wg.Done()
	offset := 0
	for dataCluster := range dataClusters {
		for offset < len(dataCluster) {
			record := new(Record)
			record.Parse(dataCluster)
			records = append(records, *record)
		}
	}
}

func (record *Record) Parse(data []byte) int {
	utils.Unmarshal(data, record)
	record.Fname = utils.DecodeUTF16(data[60 : 60+2*record.FnameLen])
	return int(60 + 2*record.FnameLen)
}

func (record Record) GetSourceInfo() string {
	return source[record.SourceInfo]
}

func (record Record) GetFileAttributes() string {
	return fileattributes[record.FileAttrs]
}

func (record Record) GetReason() string {
	return reasons[record.ReasonFlag]
}
