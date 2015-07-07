package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hanwen/go-fuse/fuse"
)

var endpoint = "https://api.github.com/users/%s/gists"

func getGists(n string) ([]Gist, error) {
	url := fmt.Sprintf(endpoint, n)
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

func (g *Gist) DirEntries() []fuse.DirEntry {
	d := make([]fuse.DirEntry, 0, 1)
	for n := range g.Files {
		d = append(d, fuse.DirEntry{Name: n, Mode: fuse.S_IFREG | 0440})
	}
	return d
}

type File struct {
	RawUrl string `json:"raw_url"`
	Name   string `json:"filename"`
	Size   int
}
