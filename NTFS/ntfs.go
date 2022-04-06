package ntfs

type NFTS struct {
	JumpInstruction   [3]byte //0-3
	Signature         string  //4 bytes NTFS 3-7
	NotUsed1          [4]byte
	BytesPerSector    uint32   // 11-13
	SectorsPerCluster uint8    //14
	NotUsed2          [26]byte //14-40
	TotalSectors      uint64   //40-48
	MFTOffset         uint64   //48-56
	MFTMirrOffset     uint64   //56-64
}
