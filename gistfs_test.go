package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/hanwen/go-fuse/fuse/nodefs"
)

const ret = `
[{
	"url": "foo",
	"id": "123abc",
	"files": {
		"file1": {
			"raw_url": "%s/raw/123abc",
			"size": 123
		}
	}
}]`

func TestGistFs(t *testing.T) {
	root := NewGistFs("")
	mount, _ := ioutil.TempDir("", "")
	defer os.RemoveAll(mount)

	state, _, err := nodefs.MountRoot(mount, root, nil)
	defer state.Unmount()
	if err != nil {
		t.Errorf("Error mounting root: %v\n", err)
	}

	go state.Serve()

	dirs, err := ioutil.ReadDir(mount)
	if err != nil {
		t.Fatalf("Could not read mounted directory")
	}

	if len(dirs) != 0 {
		t.Errorf("Wrong number of directories: %d", len(dirs))
	}

	err = os.Mkdir(mount+"/andrewstuart", 0770)
	if err != nil {
		t.Fatalf("Mkdir could not create a new directory: %v", err)
	}

	dirs, err = ioutil.ReadDir(mount)
	if err != nil {
		t.Fatalf("Error reading directory after mkdir: %v", err)
	}

	if len(dirs) != 1 {
		t.Errorf("Wrong number of directories: %d, should be 1", len(dirs))
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, ret, r.Host)
	}))
	defer ts.Close()

	endpoint = ts.URL + "/%s"

	dirs, err = ioutil.ReadDir(mount + "/andrewstuart")
	if err != nil {
		t.Fatalf("Error reading gists: %v", err)
	}

	if len(dirs) != 1 {
		t.Errorf("Wrong number of listings returned. %d, should be 1", len(dirs))
	}
}
