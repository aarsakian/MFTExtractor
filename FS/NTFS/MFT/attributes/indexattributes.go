package attributes

import (
	"fmt"

	"github.com/aarsakian/MFTExtractor/utils"
)

type IndexEntry struct {
	ParRef      uint64
	ParSeq      uint16
	Len         uint16 //8-9
	FilenameLen uint16 //10-11
	Flags       uint32 //12-15
	Fnattr      *FNAttribute
}

type IndexRoot struct {
	Type                 string //0-3 similar to FNA type
	CollationSortingRule string //4-7
	Sizebytes            uint32 //8-11
	Sizeclusters         uint8  //12-12
	Nodeheader           *NodeHeader
	Header               *AttributeHeader
	IndexEntries         []IndexEntry
}

type NodeHeader struct {
	OffsetEntryList          uint32 // 16-20 offset to start of the index entry
	OffsetEndUsedEntryList   uint32 //20-24 where EntryList ends
	OffsetEndEntryListBuffer uint32 //24-28
	Flags                    uint32 //0x01 no children
}

type IndexAllocation struct {
	Signature        string //0-4
	FixupArrayOffset int16  //4-6
	NumEntries       int16  //6-8
	LSN              int64  //8-16
	VCN              int64  //16-24 where the record fits in the tree
	Nodeheader       *NodeHeader
	Header           *AttributeHeader
	IndexEntries     []IndexEntry
}

func (idxEntry IndexEntry) ShowInfo() {
	if idxEntry.Fnattr != nil {
		fmt.Printf("type %s file ref %d idx name %s flags %d \n", idxEntry.Fnattr.GetType(), idxEntry.ParRef,
			idxEntry.Fnattr.Fname, idxEntry.Flags)
	}

}

func (idxRoot *IndexRoot) SetHeader(header *AttributeHeader) {
	idxRoot.Header = header
}

func (idxRoot *IndexRoot) Parse(data []byte) {
	utils.Unmarshal(data[:12], idxRoot)

	var nodeheader *NodeHeader = new(NodeHeader)
	utils.Unmarshal(data[16:32], nodeheader)
	idxRoot.Nodeheader = nodeheader

	idxEntryOffset := 16 + uint16(nodeheader.OffsetEntryList)
	lastIdxEntryOffset := 16 + uint16(nodeheader.OffsetEndEntryListBuffer)

	for idxEntryOffset+16 < lastIdxEntryOffset {
		var idxEntry *IndexEntry = new(IndexEntry)
		utils.Unmarshal(data[idxEntryOffset:idxEntryOffset+16], idxEntry)

		if idxEntry.FilenameLen > 0 {
			var fnattrIDXEntry FNAttribute
			utils.Unmarshal(data[idxEntryOffset+16:idxEntryOffset+16+idxEntry.FilenameLen],
				&fnattrIDXEntry)

			fnattrIDXEntry.Fname =
				utils.DecodeUTF16(data[idxEntryOffset+16+66 : idxEntryOffset+16+
					66+2*uint16(fnattrIDXEntry.Nlen)])
			idxEntry.Fnattr = &fnattrIDXEntry

		}
		idxEntryOffset += idxEntry.Len

		idxRoot.IndexEntries = append(idxRoot.IndexEntries, *idxEntry)
	}
}

func (idxRoot IndexRoot) GetHeader() AttributeHeader {
	return *idxRoot.Header
}

func (idxRoot IndexRoot) IsNoNResident() bool {
	return false // always resident
}

func (idxRoot IndexRoot) FindType() string {
	return idxRoot.Header.GetType()
}

func (idxRoot IndexRoot) ShowInfo() {
	fmt.Printf("type %s nof entries %d\n", idxRoot.FindType(), len(idxRoot.IndexEntries))
	for _, idxEntry := range idxRoot.IndexEntries {
		idxEntry.ShowInfo()
	}
}

func (idxAllocation *IndexAllocation) SetHeader(header *AttributeHeader) {
	idxAllocation.Header = header
}

func (idxAllocation IndexAllocation) GetHeader() AttributeHeader {
	return *idxAllocation.Header
}

func (idxAllocation IndexAllocation) FindType() string {
	return idxAllocation.Header.GetType()
}

func (idxAllocation IndexAllocation) IsNoNResident() bool {
	return idxAllocation.Header.IsNoNResident()
}

func (idxAllocation IndexAllocation) ShowInfo() {
	fmt.Printf("type %s nof entries %d\n", idxAllocation.FindType(), idxAllocation.NumEntries)
	for _, idxEntry := range idxAllocation.IndexEntries {
		idxEntry.ShowInfo()
	}
}

func (idxAllocation *IndexAllocation) Parse(bs []byte) {
	utils.Unmarshal(bs[:24], idxAllocation)

	var nodeheader *NodeHeader = new(NodeHeader)
	utils.Unmarshal(bs[24:24+16], nodeheader)
	idxAllocation.Nodeheader = nodeheader

	idxEntryOffset := nodeheader.OffsetEntryList + 24 // relative to the start of node header
	for idxEntryOffset < nodeheader.OffsetEndUsedEntryList {
		var idxEntry *IndexEntry = new(IndexEntry)
		utils.Unmarshal(bs[idxEntryOffset:idxEntryOffset+16], idxEntry)
		if idxEntry.FilenameLen > 0 {
			var fnattrIDXEntry FNAttribute
			utils.Unmarshal(bs[idxEntryOffset+16:idxEntryOffset+16+uint32(idxEntry.FilenameLen)],
				&fnattrIDXEntry)

			fnattrIDXEntry.Fname =
				utils.DecodeUTF16(bs[idxEntryOffset+16+66 : idxEntryOffset+16+
					66+2*uint32(fnattrIDXEntry.Nlen)])
			idxEntry.Fnattr = &fnattrIDXEntry

		}
		idxEntryOffset += uint32(idxEntry.Len)
		idxAllocation.IndexEntries = append(idxAllocation.IndexEntries, *idxEntry)
	}

}
