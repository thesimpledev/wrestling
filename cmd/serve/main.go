// Command serve serves a directory of static files over HTTP on
// 127.0.0.1:8080, for trying the web build locally.
package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

const listenAddress = "127.0.0.1:8080"

func main() {
	server, err := newServer(os.Args[1:])
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("serving on http://%s", server.Addr)
	log.Fatal(server.ListenAndServe())
}

// newServer builds the file server from the command line arguments, which
// must be exactly one existing directory.
func newServer(arguments []string) (*http.Server, error) {
	if len(arguments) != 1 {
		return nil, errors.New("usage: serve <directory>")
	}

	root, err := os.OpenRoot(arguments[0])
	if err != nil {
		return nil, fmt.Errorf("opening directory: %w", err)
	}

	return &http.Server{
		Addr:              listenAddress,
		Handler:           http.FileServerFS(root.FS()),
		ReadHeaderTimeout: 10 * time.Second,
	}, nil
}
