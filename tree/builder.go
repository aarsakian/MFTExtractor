package tree

import (
	"fmt"

	"github.com/aarsakian/MFTExtractor/FS/NTFS/MFT"
	MFTAttributes "github.com/aarsakian/MFTExtractor/FS/NTFS/MFT/attributes"
)

/*Thus, a B-tree node is equivalent to a disk block, and a “pointer” value stored
in the tree is actually the number of the block containing the child node (usually
interpreted as an offset from the beginning of the corresponding disk file)*/

type Node struct {
	record   MFT.Record
	parent   *Node
	children []*Node
}

type Tree struct {
	root *Node
}

func (t *Tree) BuildTree(record MFT.Record) *Tree {

	if t.root == nil {
		t.root = &Node{record, nil, nil}
	} else {
		t.root.insert(record)
	}

	return t
}

func (n *Node) insert(record MFT.Record) {
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
	n.record.ShowFileName("Win32")
	for _, node := range n.children {
		node.record.ShowFileName("Win32")

	}

	for _, node := range n.children {

		node.Show()

	}
}
