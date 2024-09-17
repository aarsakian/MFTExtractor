package tree

import (
	"fmt"
	"strings"

	"github.com/aarsakian/MFTExtractor/FS/NTFS/MFT"
	MFTAttributes "github.com/aarsakian/MFTExtractor/FS/NTFS/MFT/attributes"
	"github.com/aarsakian/MFTExtractor/logger"
)

/*Thus, a B-tree node is equivalent to a disk block, and a “pointer” value stored
in the tree is actually the number of the block containing the child node (usually
interpreted as an offset from the beginning of the corresponding disk file)*/

type Node struct {
	record   *MFT.Record
	parent   *Node
	children []*Node
}

type Tree struct {
	root *Node
}

func (t *Tree) Build(records MFT.Records) {
	fmt.Printf("Building tree from MFT records \n")
	for idx := range records {
		if records[idx].Entry < 5 { //$MFT entry number 5
			continue
		}
		t.AddRecord(&records[idx])
	}

}

func (t *Tree) AddRecord(record *MFT.Record) {

	if t.root == nil {

		t.root = &Node{record, nil, nil}
	} else {
		t.root.insert(record)
	}

}

func (n *Node) insert(record *MFT.Record) {
	if record.FindAttribute("FileName") != nil {
		fnattr := record.FindAttribute("FileName").(*MFTAttributes.FNAttribute)
		if uint64(n.record.Entry) == fnattr.ParRef && n.record.Seq-fnattr.ParSeq < 2 { //record is children
			childNode := Node{record, n, nil}

			n.children = append(n.children, &childNode)

		} else {
			for _, childNode := range n.children { //test its children
				childNode.insert(record)

			}
		}
	}

}

func (t Tree) Show() {

	t.root.Show()

}

func (n Node) Show() {

	if n.children != nil {

		msg := fmt.Sprintf(" %s  |_> ", n.record.GetFname())
		logger.MFTExtractorlogger.Info(msg)
		fmt.Print("\n" + msg)
	}

	msgB := strings.Builder{}
	msgB.Grow(len(n.children))

	for _, childnode := range n.children {
		msg := fmt.Sprintf(" %s", childnode.record.GetFname())

		fmt.Print(msg)
		msgB.WriteString(msg)

	}

	logger.MFTExtractorlogger.Info(msgB.String())

	for _, node := range n.children {

		node.Show()

	}
}
