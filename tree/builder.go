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
		if uint64(n.record.Entry) ==
			record.FindAttribute("FileName").(*MFTAttributes.FNAttribute).ParRef {
			childNode := Node{record, n, nil}
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

	for _, node := range n.children {
		fmt.Printf("%d has children %d \n", n.record.Entry, node.record.Entry)
		node.Show()

	}
}
