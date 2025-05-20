package tree

import (
	"fmt"
	"strings"

	"github.com/aarsakian/FileSystemForensics/FS/NTFS/MFT"
	MFTAttributes "github.com/aarsakian/FileSystemForensics/FS/NTFS/MFT/attributes"
	"github.com/aarsakian/FileSystemForensics/logger"
)

/*Thus, a B-tree node is equivalent to a disk block, and a “pointer” value stored
in the tree is actually the number of the block containing the child node (usually
interpreted as an offset from the beginning of the corresponding disk file)*/

type Node struct {
	record        *MFT.Record
	parent        *Node
	children      []*Node
	MinChildEntry uint32
	MaxChildEntry uint32
}

type Tree struct {
	root *Node
}

func (t *Tree) Build(records MFT.Records) {
	msg := fmt.Sprintf("Building tree from %d MFT records ", len(records))
	fmt.Printf(msg + "\n")
	logger.MFTExtractorlogger.Info(msg)

	for idx := range records {
		if records[idx].Entry < 5 { //$MFT entry number 5
			continue
		}

		t.AddRecord(&records[idx])
	}

}

func (t *Tree) AddRecord(record *MFT.Record) {

	if t.root == nil {
		logger.MFTExtractorlogger.Info("initialized root")

		t.root = &Node{record, nil, nil, 0, 0}
	} else {

		t.root.insert(record)

	}

}

func (node *Node) insert(record *MFT.Record) {
	if !node.record.IsFolder() {
		return
	}

	attr := record.FindAttribute("FileName")
	if attr != nil {
		logger.MFTExtractorlogger.Info(fmt.Sprintf("checking %s %d min %d max %d",
			node.record.GetFname(), node.record.Entry, node.MinChildEntry, node.MaxChildEntry))

		fnattr := attr.(*MFTAttributes.FNAttribute)
		if uint64(node.record.Entry) == fnattr.ParRef && node.record.Seq-fnattr.ParSeq < 2 { //record is children
			childNode := Node{record, node, nil, record.Entry, record.Entry}
			node.updateEntryRange(record.Entry)
			node.children = append(node.children, &childNode)

			childNode.parent = node

			logger.MFTExtractorlogger.Info(fmt.Sprintf("added %s Id %d  to %s Id %d", childNode.record.GetFname(),
				childNode.record.Entry, childNode.parent.record.GetFname(), childNode.parent.record.Entry))

		} else {

			for idx := range node.children { //test its children
				/*	logger.MFTExtractorlogger.Info(fmt.Sprintf("children %s %d min %d max %d", node.children[idx].record.GetFname(), node.children[idx].record.Entry,
					node.children[idx].MinChildEntry, node.children[idx].MaxChildEntry))*/
				if !node.children[idx].contains(uint32(fnattr.ParRef)) {

					continue
				}
				node.children[idx].insert(record)

			}
		}
	}

}

func (node Node) contains(entry uint32) bool {

	if node.MinChildEntry <= entry && entry <= node.MaxChildEntry {
		return true

	}

	return false
}

func (node *Node) updateEntryRange(entry uint32) {
	for node != nil {
		//logger.MFTExtractorlogger.Info(fmt.Sprintf("updating %d for %d", node.record.Entry, entry))
		if node.MinChildEntry > entry {
			node.MinChildEntry = entry

		}
		if node.MaxChildEntry < entry {
			node.MaxChildEntry = entry

		}
		node = node.parent
	}

}

func (t Tree) Show() {

	t.root.descend()

}

func (node Node) descend() {
	if node.children == nil {
		return

	}
	node.showChildrenInfo()
	for _, node := range node.children {

		node.descend()

	}
}

func (node Node) showChildrenInfo() {
	msgB := strings.Builder{}
	msgB.Grow(len(node.children) + 1) // for root

	msg := fmt.Sprintf(" %s  %d |_> ", node.record.GetFname(), node.record.Entry)

	msgB.WriteString(msg)

	fmt.Print("\n" + msg)

	for _, childnode := range node.children {
		msg := fmt.Sprintf(" %s %d", childnode.record.GetFname(), childnode.record.Entry)

		fmt.Print(msg)
		msgB.WriteString(msg)

	}

	logger.MFTExtractorlogger.Info(msgB.String())

}
