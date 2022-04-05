package img

import (
	"log"

	"github.com/aarsakian/MFTExtractor/utils"
	"golang.org/x/sys/windows"
)

func ReadDisk(a_file string, offset int64, bytesToRead uint32) []byte {

	file_ptr, _ := windows.UTF16PtrFromString(a_file)
	var templateHandle windows.Handle
	fd, err := windows.CreateFile(file_ptr, windows.FILE_READ_DATA,
		windows.FILE_SHARE_READ, nil,
		windows.OPEN_EXISTING, 0, templateHandle)
	if err != nil {
		log.Fatalln(err)
	}

	defer windows.Close(fd)
	buf_pointer := make([]byte, bytesToRead)
	largeInteger := utils.NewLargeInteger(offset)
	var bytesRead uint32

	newLowOffset, err := windows.SetFilePointer(fd, largeInteger.LowPart, &largeInteger.HighPart, windows.FILE_BEGIN)
	largeInteger.LowPart = int32(newLowOffset)
	if err != nil {
		log.Fatalln(err)
	}

	err = windows.ReadFile(fd, buf_pointer, &bytesRead, nil)
	if err != nil {
		log.Fatalln(err)
	}
	return buf_pointer

}
