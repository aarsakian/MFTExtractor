package BTRFS

import (
	"errors"
	"fmt"
	"time"
)

type FilesDirsMap map[uint64]FileDirEntry //inodeid -> FileDirEntry

type FileDirEntry struct {
	Id       int
	SizeB    int
	StBlock  int
	Nlink    int
	Uid      int
	Gid      int
	Index    int
	Name     string
	Type     string
	Flags    string
	ATime    time.Time
	CTime    time.Time
	MTime    time.Time
	OTime    time.Time
	Children []*FileDirEntry
	Parent   *FileDirEntry
	Path     string
	Extents  []Extent
}

type Extent struct {
	Offset int
	LSize  int
	PSize  int
}

func (fileDirEntry FileDirEntry) GetParentId() (int, error) {
	if fileDirEntry.Parent != nil {
		return fileDirEntry.Parent.Id, nil
	} else {
		return -1, errors.New("no parent found")
	}
}

func (fileDirEntry FileDirEntry) GetInfo() string {
	return fmt.Sprintf("Id %d, %s, A %s, C %s, M %s, O %s, %s, Idx %d, lnks %d, exts %s, path %s\n",
		fileDirEntry.Id,
		fileDirEntry.Name, fileDirEntry.ATime, fileDirEntry.CTime,
		fileDirEntry.MTime, fileDirEntry.OTime,
		fileDirEntry.Type, fileDirEntry.Index, fileDirEntry.Nlink,
		fileDirEntry.GetExtentsInfo(), fileDirEntry.Path)
}

func (fileDirEntry FileDirEntry) GetExtentsInfo() string {
	var extentInfo string
	for _, extent := range fileDirEntry.Extents {
		extentInfo += fmt.Sprintf(" off %d Ps %d Ls %d|",
			extent.Offset, extent.LSize, extent.PSize)
	}
	return extentInfo
}

func (fileDirEntry *FileDirEntry) BuildPath() {
	parent := fileDirEntry.Parent
	var paths []string
	for parent != nil {
		paths = append(paths, parent.Name)
		parent = parent.Parent
	}

	for idx := range paths {
		path := paths[len(paths)-idx-1]
		if path == "" {
			continue
		}
		fileDirEntry.Path += "\\" + path
	}

	for idx := range fileDirEntry.Children {
		fileDirEntry.Children[idx].BuildPath()
	}

}
