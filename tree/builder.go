package tree

import (
	"fmt"

	"github.com/aarsakian/MFTExtractor/MFT"
	MFTAttributes "github.com/aarsakian/MFTExtractor/MFT/attributes"
)

type Node struct {
	record   MFT.MFTrecord
	parent   *Node
	children []*Node
}

type Tree struct {
	root *Node
}

func (t *Tree) BuildTree(record MFT.MFTrecord) *Tree {

	if t.root == nil {
		t.root = &Node{record, nil, nil}
	} else {
		t.root.insert(record)
	}

	return t
}

func (n *Node) insert(record MFT.MFTrecord) {
	if record.FindAttribute("FileName") != nil {
		fnattr := record.FindAttribute("FileName").(*MFTAttributes.FNAttribute)
		if uint64(n.record.Entry) == fnattr.ParRef && n.record.Seq-fnattr.ParSeq < 2 {
			childNode := Node{record, n, nil}

			childNode.parent = n
			n.children = append(n.children, &childNode)

		} else {
			for _, childNode := range n.children {
				childNode.insert(record)

			}
		}
	}

}

func (t Tree) Show() {
	fmt.Printf("\n root %d ", t.root.record.Entry)

	t.root.Show()

}

func (n Node) Show() {
	fmt.Printf("\n Parent is")
	n.record.ShowFileName("LONG")
	for _, node := range n.children {
		node.record.ShowFileName("LONG")

	}

	for _, node := range n.children {

		node.Show()

	}
}
