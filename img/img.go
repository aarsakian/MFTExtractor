package img

import (
	"log"

	"github.com/aarsakian/MFTExtractor/utils"
	"golang.org/x/sys/windows"
)

func readDisk(a_file string, byteToRead uint32, lowoffset int32) {

	file_ptr := utils.StringToUTF16Ptr(a_file)
	var templateHandle windows.Handle
	fd, err := windows.CreateFile(file_ptr, windows.FILE_READ_DATA,
		windows.FILE_SHARE_READ, nil,
		windows.OPEN_EXISTING, 0, templateHandle)
	if err != nil {
		log.Fatalln(err)
	}
	var buf_pointer []byte
	windows.SetFilePointer(fd, lowoffset, nil, windows.FILE_BEGIN)

	windows.ReadFile(fd, buf_pointer, &byteToRead, nil)

}
