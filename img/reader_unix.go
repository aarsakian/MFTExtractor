package img

import (
	"fmt"
	"log"

	"golang.org/x/sys/unix"
)

type UnixReader struct {
	pathToDisk string
	fd         int
}

func (unixreader UnixReader) CreateHandler() {
	fd, err := unix.Open(unixreader.pathToDisk, unix.O_RDONLY, 0)
	if err != nil {
		log.Fatal(err)
	}
	unixreader.fd = fd
}

func (unixreader UnixReader) ReadFile(buf_pointer int64, bytesToRead uint32) []byte {
	buffer := make([]byte, bytesToRead)
	unix.Seek(unixreader.fd, buf_pointer, unix.SEEK_SET)
	_, err := unix.Read(unixreader.fd, buffer)
	if err != nil {
		log.Fatal("error reading", err)
	}

	fmt.Printf("offset %d \n", buf_pointer)
	return buffer

}

func (unixreader UnixReader) CloseHandler() {
	unix.Close(unixreader.fd)
}

func (unixreader UnixReader) GetDiskSize() int64 {
	return 0
}
