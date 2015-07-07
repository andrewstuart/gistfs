package main

import (
	"github.com/hanwen/go-fuse/fuse"
	"github.com/hanwen/go-fuse/fuse/nodefs"
)

type GistNode struct {
	nodefs.Node
	name string
}

func NewGistFs(n string) *GistNode {
	gn := &GistNode{Node: nodefs.NewDefaultNode(), name: n}
	return gn
}

func (gn *GistNode) OpenDir(c *fuse.Context) ([]fuse.DirEntry, fuse.Status) {
	ents := make([]fuse.DirEntry, 0)

	if gn.name != "" && len(gn.Inode().Children()) == 0 {
		gists, err := getGists(gn.name)
		if err != nil {
			return nil, fuse.ENOENT
		}
		for _, g := range gists {
			for n := range g.Files {
				newNode := NewGistFs(n)
				in := gn.Inode().NewChild(n, false, newNode)
				gn.Inode().AddChild(n, in)
			}
		}
	}

	for k, c := range gn.Inode().Children() {
		e := fuse.DirEntry{Name: k, Mode: fuse.S_IFREG | 0660}
		if c.IsDir() {
			e.Mode = fuse.S_IFDIR | 0770
		}
		ents = append(ents, e)
	}

	return ents, fuse.OK
}

func (gn *GistNode) Mkdir(n string, mode uint32, c *fuse.Context) (*nodefs.Inode, fuse.Status) {
	mode = mode | fuse.S_IFDIR

	gf := NewGistFs(n)

	in := gn.Inode().NewChild(n, true, gf)
	gn.Inode().AddChild(n, in)
	return in, fuse.OK
}

func (gn *GistNode) GetAttr(a *fuse.Attr, f nodefs.File, c *fuse.Context) fuse.Status {
	switch {
	case gn.Inode().IsDir():
		a.Mode = fuse.S_IFDIR | 0770
	default:
		a.Mode = fuse.S_IFREG | 0660
	}
	return fuse.OK
}
