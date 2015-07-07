//gist-fs exposes gists through a filesystem
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

			a := &fuse.Attr{Mode: fuse.S_IFREG | 0440, Size: uint64(f.Size)}

			if f.isDir {
				a.Mode = fuse.S_IFDIR | 0550
			}

			return a, fuse.OK
		}
	}

	return nil, fuse.ENOENT
}

func (gf *GistFs) OpenDir(n string, c *fuse.Context) ([]fuse.DirEntry, fuse.Status) {

	dirs := make([]fuse.DirEntry, 0)
	if gf.files != nil && n == "" {
		for _, f := range gf.files {
			d := fuse.DirEntry{Name: f.Name, Mode: fuse.S_IFREG | 0440}
			if f.isDir {
				d.Mode = fuse.S_IFDIR | 0550
			}
			dirs = append(dirs, d)
		}
		return dirs, fuse.OK
	}

	gf.files = make(map[string]File)

	u := gf.user
	if n != "" {
		u = n
	}

	gists, err := getGists(u)
	if err != nil {
		log.Println(err)
		return nil, fuse.ENOENT
	}
	for _, g := range gists {
		for _, f := range g.Files {
			gf.files[n+f.Name] = f
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

	fmt.Printf("res.Header = %+v\n", res.Header)

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
	RawUrl   string `json:"raw_url"`
	Name     string `json:"filename"`
	Size     int
	contents []byte
	isDir    bool
}

func (gf *GistFs) Mkdir(n string, mode uint32, c *fuse.Context) fuse.Status {
	gf.files[n] = File{Name: n, isDir: true}
	return fuse.OK
}

func (gf *GistFs) Open(n string, _ uint32, c *fuse.Context) (nodefs.File, fuse.Status) {
	if gf.files == nil {
		log.Println("Nil map. Files have not been queried yet.")
		return nil, fuse.ENOENT
	}

	if f, ok := gf.files[n]; ok {

		if f.contents != nil {
			//Cached copy
			return nodefs.NewDataFile(f.contents), fuse.OK
		}

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

		f.contents = bs

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
