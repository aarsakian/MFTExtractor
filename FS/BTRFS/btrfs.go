package BTRFS

import (
	"fmt"

	"github.com/aarsakian/FileSystemForensics/FS/BTRFS/leafnode"
	"github.com/aarsakian/FileSystemForensics/img"
	"github.com/aarsakian/FileSystemForensics/logger"
	"github.com/aarsakian/FileSystemForensics/utils"
)

const SYSTEMCHUNKARRSIZE = 2048
const SUPERBLOCKSIZE = 4096

const OFFSET_TO_SUPERBLOCK = 0x10000 //64KB fixed position

type BTRFS struct {
	Superblock *Superblock
	Trees      []Tree
}

type Tree struct {
	ParentId      uint64
	Name          string
	LogicalOffset uint64
	Nodes         GenericNodesPtr //low level slice of nodes
	Uuid          string
	FilesDirsMap  FilesDirsMap //map for files and folders per inodeid
}

type Superblock struct { //4096B
	Chksum                      [32]byte
	UUID                        [16]byte
	PhysicalAddress             uint64
	Flags                       uint64
	Signature                   string //8bytes 68 offset
	Generation                  uint64
	LogicalAddressRootTree      uint64
	LogicalAddressRootChunkTree uint64
	LogicalAddressLogRootTree   uint64
	LogRootTransid              uint64
	VolumeSizeB                 uint64
	VolumeUsedSizeB             uint64
	ObjectID                    uint64
	NofVolumeDevices            uint64
	SectorSize                  uint32
	NodeSize                    uint32
	LeafSize                    uint32
	StripeSize                  uint32
	SystemChunkArrSize          uint32
	ChunkRootGeneration         uint64
	CompatFlags                 uint64
	CompatROFlags               uint64
	InCompatFlags               uint64
	CsumType                    uint16
	RootLevel                   uint8
	ChunkRootLevel              uint8
	LogRootLevel                uint8
	DevItem                     *leafnode.DevItem
	Label                       [256]byte //299
	Reserved                    [256]byte
	SystemChunkArr              [SYSTEMCHUNKARRSIZE]byte //811

}

func (btrfs *BTRFS) Process(hD img.DiskReader, partitionOffsetB int64, selectedEntries []int,
	fromEntry int, toEntry int) {

	length := int(1024) // len of MFT record

	msg := "Reading superblock at offset %d"
	fmt.Printf(msg+"\n", partitionOffsetB)
	logger.MFTExtractorlogger.Info(fmt.Sprintf(msg, partitionOffsetB))

	data := hD.ReadFile(partitionOffsetB, length)
	btrfs.Parse(data)

}

func (btrfs *BTRFS) Parse(data []byte) {
	logger.MFTExtractorlogger.Info("Parsing superblock")
	superblock := new(Superblock)
	utils.Unmarshal(data, superblock)
	devItem := new(leafnode.DevItem)
	devItem.Parse(data)
	superblock.DevItem = devItem
	btrfs.Superblock = superblock

}

func (btrfs BTRFS) VerifySuperBlock(data []byte) bool {
	return utils.ToUint32(btrfs.Superblock.Chksum[:]) == utils.CalcCRC32(data[32:])

}

func (btrfs BTRFS) ParseSystemChunks() GenericNodesPtr {
	// (KEY, CHUNK_ITEM) pairs for all SYSTEM chunks
	logger.MFTExtractorlogger.Info("Parsing bootstrap data for system chunks.")

	curOffset := 0
	var genericNodes GenericNodesPtr
	data := btrfs.Superblock.SystemChunkArr[:btrfs.Superblock.SystemChunkArrSize]
	for curOffset < len(data) {
		key := new(leafnode.Key)
		curOffset += key.Parse(data[curOffset:])

		chunkItem := new(leafnode.ChunkItem)
		curOffset += chunkItem.Parse(data[curOffset:])

		leafNode := leafnode.LeafNode{}
		leafNode.Items = append(leafNode.Items, leafnode.Item{Key: key})
		leafNode.DataItems = append(leafNode.DataItems, chunkItem)
		node := GenericNode{LeafNode: &leafNode}

		genericNodes = append(genericNodes, &node)

	}
	return genericNodes
}

func (btrfs BTRFS) GetSectorsPerCluster() int {
	return int(btrfs.Superblock.NodeSize)
}

func (btrfs BTRFS) GetBytesPerSector() uint64 {
	return uint64(btrfs.Superblock.SectorSize)
}

func (btrfs BTRFS) GetMetadata() []FileDirEntry {
	//return btrfs.tree.fileDirEntries
	return nil
}

func (btrfs BTRFS) CollectUnallocated(img.DiskReader, int64, chan<- []byte) {

}

func (btrfs BTRFS) GetSignature() string {
	return btrfs.Superblock.Signature
}
