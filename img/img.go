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

func GetHandler(a_file string) windows.Handle {
	file_ptr, _ := windows.UTF16PtrFromString(a_file)
	var templateHandle windows.Handle
	fd, err := windows.CreateFile(file_ptr, windows.FILE_READ_DATA,
		windows.FILE_SHARE_READ, nil,
		windows.OPEN_EXISTING, 0, templateHandle)
	if err != nil {
		log.Fatalln(err)
	}

	return fd
}

func CloseHandler(fd windows.Handle) {
	windows.Close(fd)
}
func ReadDisk(fd windows.Handle, offset int64, bytesToRead uint32) []byte {

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

func GetDiskSize(hD windows.Handle) int64 {
	const IOCTL_DISK_GET_DRIVE_GEOMETRY = 0x70000
	const nByte_DISK_GEOMETRY = 24
	disk_geometry := DISK_GEOMETRY{}

	var junk *uint32
	var inBuffer *byte
	err := windows.DeviceIoControl(hD, IOCTL_DISK_GET_DRIVE_GEOMETRY,
		inBuffer, 0, (*byte)(unsafe.Pointer(&disk_geometry)), nByte_DISK_GEOMETRY, junk, nil)
	if err != nil {
		log.Fatalln(err)
	}

	return disk_geometry.Cylinders * int64(disk_geometry.TracksPerCylinder) * int64(disk_geometry.SectorsPerTrack) * int64(disk_geometry.BytesPerSector)
}
