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

type WindowsTime struct {
	Stamp uint64
}

func (winTime *WindowsTime) ConvertToIsoTime() string { //receiver winTime struct
	// t:=math.Pow((uint64(winTime.high)*2),32) + uint64(winTime.low)
	x := winTime.Stamp/10000000 - 116444736*1e2
	unixtime := time.Unix(int64(x), 0).UTC()
	return unixtime.Format("02-01-2006 15:04:05")

}

func ReadEndianInt(barray []byte) uint64 {
	var sum uint64
	sum = 0
	for index, val := range barray {
		sum += uint64(val) << uint(index*8)

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
				binary.Read(bytes.NewBuffer(data[idx:idx+6]), binary.LittleEndian, &temp)
				idx += 6
			} else {
				binary.Read(bytes.NewBuffer(data[idx:idx+8]), binary.LittleEndian, &temp)
				idx += 8
			}
			field.SetUint(temp)
		case reflect.Bool:
			field.SetBool(false)
			idx += 1

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

func StringToUTF16Ptr(str string) *uint16 {
	wchars := utf16.Encode([]rune(str + "\x00"))
	return &wchars[0]
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