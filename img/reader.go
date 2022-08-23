package img

import (
	"runtime"
)

type DiskReader interface {
	CreateHandler()
	CloseHandler()
	ReadFile(int64, uint32) []byte
	GetDiskSize() int64
}

func GetHandler(pathToDisk string) DiskReader {
	os := runtime.GOOS
	var dr DiskReader
	switch os {
	case "windows":
		//dr = WindowsReader{pathToDisk: pathToDisk}

	case "linux":
		dr = UnixReader{pathToDisk: pathToDisk}

	}
	dr.CreateHandler()

	return dr
}
