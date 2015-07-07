package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/hanwen/go-fuse/fuse"
	"github.com/hanwen/go-fuse/fuse/nodefs"
	"github.com/hanwen/go-fuse/fuse/pathfs"
)

type GistFs struct {
	pathfs.FileSystem
	user  string
	files map[string]File
}

func (gf *GistFs) GetAttr(n string, c *fuse.Context) (*fuse.Attr, fuse.Status) {
	if n == "" {
		return &fuse.Attr{Mode: fuse.S_IFDIR | 0770}, fuse.OK
	}

	if gf.files != nil {
		if f, ok := gf.files[n]; ok {
			return &fuse.Attr{Mode: fuse.S_IFREG | 0440, Size: uint64(f.Size)}, fuse.OK
		}
	}

	return &fuse.Attr{Mode: fuse.S_IFREG | 0770}, fuse.OK
}

func (gf *GistFs) OpenDir(n string, c *fuse.Context) ([]fuse.DirEntry, fuse.Status) {

	dirs := make([]fuse.DirEntry, 0)
	if gf.files != nil {
		for _, f := range gf.files {
			dirs = append(dirs, fuse.DirEntry{Name: f.Name, Mode: fuse.S_IFREG | 0440})
		}
		return dirs, fuse.OK
	}

	gf.files = make(map[string]File)

	gists, err := getGists(gf.user)
	if err != nil {
		log.Println(err)
		return nil, fuse.ENOENT
	}
	for _, g := range gists {
		for _, f := range g.Files {
			gf.files[f.Name] = f
			dirs = append(dirs, fuse.DirEntry{Name: f.Name, Mode: fuse.S_IFREG | 0440})
		}
	}
	return dirs, fuse.OK
}

func getGists(n string) ([]Gist, error) {
	url := fmt.Sprintf("https://api.github.com/users/%s/gists", n)
	res, err := http.Get(url)
	defer res.Body.Close()
	if err != nil {
		return nil, err
	}

	dec := json.NewDecoder(res.Body)
	gists := make([]Gist, 0, 10)
	err = dec.Decode(&gists)
	if err != nil {
		return nil, err
	}

	return gists, nil
}

type Gist struct {
	Url   string
	Id    string
	Files map[string]File
}

type File struct {
	RawUrl string `json:"raw_url"`
	Name   string `json:"filename"`
	Size   int
}

func (gf *GistFs) Open(n string, _ uint32, c *fuse.Context) (nodefs.File, fuse.Status) {
	if gf.files == nil {
		log.Println("Nil map. Files have not been queried yet.")
		return nil, fuse.ENOENT
	}

	if f, ok := gf.files[n]; ok {
		res, err := http.Get(f.RawUrl)
		defer res.Body.Close()
		if err != nil {
			log.Println(err)
			return nil, fuse.ENOENT
		}

		bs, err := ioutil.ReadAll(res.Body)
		if err != nil {
			log.Println(err)
			return nil, fuse.ENOENT
		}

		return nodefs.NewDataFile(bs), fuse.OK
	}
	return nil, fuse.ENOENT
}

func main() {
	if len(os.Args) < 3 {
		log.Fatal("More Args Pls. path username")
	}

	gf := &GistFs{FileSystem: pathfs.NewDefaultFileSystem(), user: os.Args[2]}

	nfs := pathfs.NewPathNodeFs(gf, nil)
	server, _, err := nodefs.MountRoot(os.Args[1], nfs.Root(), nil)
	if err != nil {
		log.Fatal(err)
	}
	server.Serve()
}
