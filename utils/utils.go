package utils

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

type NoNull string

func readEndianInt(barray []byte) uint64 {
	var sum uint64
	sum = 0
	for index, val := range barray {
		sum += uint64(val) << uint(index*8)

	}

	return sum
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
