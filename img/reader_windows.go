package img

import (
	"log"
	"unsafe"

	"github.com/aarsakian/MFTExtractor/utils"
	"golang.org/x/sys/windows"
)

type DISK_GEOMETRY struct {
	Cylinders         int64
	MediaType         int32
	TracksPerCylinder int32
	SectorsPerTrack   int32
	BytesPerSector    int32
}

type WindowsReader struct {
	a_file string
	fd     windows.Handle
}

func (winreader *WindowsReader) CreateHandler() {
	file_ptr, _ := windows.UTF16PtrFromString(winreader.a_file)
	var templateHandle windows.Handle
	fd, err := windows.CreateFile(file_ptr, windows.FILE_READ_DATA,
		windows.FILE_SHARE_READ, nil,
		windows.OPEN_EXISTING, 0, templateHandle)
	if err != nil {
		log.Fatalln(err)
	}
	winreader.fd = fd
}

func (winreader WindowsReader) CloseHandler() {
	windows.Close(winreader.fd)
}

func (winreader WindowsReader) GetDiskSize() int64 {
	const IOCTL_DISK_GET_DRIVE_GEOMETRY = 0x70000
	const nByte_DISK_GEOMETRY = 24
	disk_geometry := DISK_GEOMETRY{}

	var junk *uint32
	var inBuffer *byte
	err := windows.DeviceIoControl(winreader.fd, IOCTL_DISK_GET_DRIVE_GEOMETRY,
		inBuffer, 0, (*byte)(unsafe.Pointer(&disk_geometry)), nByte_DISK_GEOMETRY, junk, nil)
	if err != nil {
		log.Fatalln(err)
	}

	return disk_geometry.Cylinders * int64(disk_geometry.TracksPerCylinder) *
		int64(disk_geometry.SectorsPerTrack) * int64(disk_geometry.BytesPerSector)
}

func (winreader WindowsReader) ReadFile(buf_pointer int64, length int) []byte {
	buffer := make([]byte, length)

	largeInteger := utils.NewLargeInteger(buf_pointer)
	var bytesRead uint32

	newLowOffset, err := windows.SetFilePointer(winreader.fd, largeInteger.LowPart,
		&largeInteger.HighPart, windows.FILE_BEGIN)
	largeInteger.LowPart = int32(newLowOffset)
	if err != nil {
		log.Fatalln(err)
	}

	err = windows.ReadFile(winreader.fd, buffer, &bytesRead, nil)
	if err != nil {
		log.Fatalln("error reading win32 api file", err)
	}
	return buffer
}
