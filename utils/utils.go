package utils

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode/utf16"
	"unicode/utf8"
)

type NoNull string

type LargeInteger struct {
	QuadPart int64
	HighPart int32
	LowPart  int32
}

func NewLargeInteger(val int64) LargeInteger {

	return LargeInteger{QuadPart: val, HighPart: int32(val >> 32),
		LowPart: int32(val & 0xFFFFFFFF)}

}

type WindowsTime struct {
	Stamp uint64
}

func Filter[T any](s []T, f func(T) bool) []T {
	var r []T
	for _, v := range s {
		if f(v) {
			r = append(r, v)
		}
	}
	return r
}

func (winTime *WindowsTime) ConvertToIsoTime() string { //receiver winTime struct
	// t:=math.Pow((uint64(winTime.high)*2),32) + uint64(winTime.low)
	x := winTime.Stamp/10000000 - 116444736*1e2
	unixtime := time.Unix(int64(x), 0).UTC()
	return unixtime.Format("02-01-2006 15:04:05")

}

func ReadEndianUInt(barray []byte) uint64 {
	var sum uint64
	sum = 0
	for index, val := range barray {
		sum += uint64(val) << uint(index*8)

	}

	return sum
}

func ReadEndianInt(barray []byte) int64 {
	var sum int64
	sum = 0
	for index, val := range barray {
		sum += int64(val) << int(index*8)

	}

	return sum
}

func DetermineClusterOffsetLength(val byte) (uint64, uint64) {

	var err error

	clusterOffs := uint64(0)
	clusterLen := uint64(0)

	val1 := (fmt.Sprintf("%x", val))

	if len(val1) == 2 { //requires non zero hex

		clusterLen, err = strconv.ParseUint(val1[1:2], 8, 8)
		if err != nil {
			//fmt.Printf("error finding cluster length %s", err)
		}

		clusterOffs, err = strconv.ParseUint(val1[0:1], 8, 8)
		if err != nil {
			//fmt.Printf("error finding cluster offset %s", err)
		}

	}
	//  fmt.Printf("Cluster located at %s and lenght %s\n",ClusterOffs, ClusterLen)
	return clusterOffs, clusterLen

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

func Hexify(barray []byte) string {

	return hex.EncodeToString(barray)

}

func stringifyGuIDs(barray []byte) string {
	s := []string{Hexify(barray[0:4]), Hexify(barray[4:6]), Hexify(barray[6:8]), Hexify(barray[8:10]), Hexify(barray[10:16])}
	return strings.Join(s, "-")
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
			if name == "Signature" || name == "CollationSortingRule" {
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
			} else if name == "MajVer" || name == "MinVer" {
				field.SetString(Hexify(Bytereverse(data[idx : idx+1])))
				idx += 1
			}
		case reflect.Struct:
			var windowsTime WindowsTime
			Unmarshal(data[idx:idx+8], &windowsTime)
			field.Set(reflect.ValueOf(windowsTime))
			idx += 8
		case reflect.Uint8:
			var temp uint8
			binary.Read(bytes.NewBuffer(data[idx:idx+1]), binary.LittleEndian, &temp)
			field.SetUint(uint64(temp))
			idx += 1
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
			name := structType.Elem().Field(i).Name
			if name == "ParRef" {
				buf := make([]byte, 8)
				copy(buf, data[idx:idx+6])
				binary.Read(bytes.NewBuffer(buf), binary.LittleEndian, &temp)
				idx += 6
			} else if name == "ChildVCN" {
				len := structValPtr.Elem().FieldByName("Len").Uint()
				binary.Read(bytes.NewBuffer(data[len-8:len]), binary.LittleEndian, &temp)

			} else {
				binary.Read(bytes.NewBuffer(data[idx:idx+8]), binary.LittleEndian, &temp)
				idx += 8
			}
			field.SetUint(temp)
		case reflect.Bool:
			field.SetBool(false)
			idx += 1
		case reflect.Array:
			idx += field.Len()

		}

	}
	return nil
}

func Bytereverse(barray []byte) []byte { //work with indexes
	//  fmt.Println("before",barray)
	for i, j := 0, len(barray)-1; i < j; i, j = i+1, j-1 {

		barray[i], barray[j] = barray[j], barray[i]

	}
	return barray

}

func WriteFile(filename string, content []byte) {
	file, err := os.Create(filename)
	if err != nil {
		// handle the error here
		fmt.Printf("err %s opening the file \n", err)

	}

	bytesWritten, err := file.Write(content)
	if err != nil {
		fmt.Printf("err %s writing the file \n", err)

	}

	fmt.Printf("wrote file %s total %d bytes \n",
		filename, bytesWritten)

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

func WriteToCSV(file *os.File, data string) {
	_, err := file.WriteString(data)
	if err != nil {
		// handle the error here
		fmt.Printf("err %s\n", err)
		return
	}
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

func readEndianFloat(barray []byte) (val uint64) {

	//    fmt.Printf("len%d ",len(barray))

	binary.Read(bytes.NewBuffer(barray), binary.LittleEndian, &val)
	return val
}

func readEndianString(barray []byte) (val []byte) {

	binary.Read(bytes.NewBuffer(barray), binary.LittleEndian, &val)

	return val
}
