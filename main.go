//gist-fs exposes gists through a filesystem
package main

import (
	"log"
	"os"

	"github.com/hanwen/go-fuse/fuse/nodefs"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: gistfs mountpoint")
	}

	gf := NewGistFs("")
	server, _, err := nodefs.MountRoot(os.Args[1], gf, nil)
	if err != nil {
		log.Fatal(err)
	}
	server.Serve()
}
