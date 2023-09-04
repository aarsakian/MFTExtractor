package img

type DiskReader interface {
	CreateHandler()
	CloseHandler()
	ReadFile(int64, int) []byte
	GetDiskSize() int64
}

func GetHandler(pathToDisk string, mode string) DiskReader {

	var dr DiskReader
	switch mode {
	case "physicalDrive":
		dr = &WindowsReader{a_file: pathToDisk}

	case "linux":
		//	dr = UnixReader{pathToDisk: pathToDisk}
	case "image":
		dr = &ImageReader{PathToEvidenceFiles: pathToDisk}
	}
	dr.CreateHandler()

	return dr
}
