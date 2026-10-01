package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func webDirectory(t *testing.T) string {
	t.Helper()
	directory := t.TempDir()
	files := map[string]string{
		"index.html": "<html>ring</html>",
		"game.wasm":  "\x00asm",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(directory, name), []byte(content), 0o600); err != nil {
			t.Fatalf("writing %s: %v", name, err)
		}
	}
	return directory
}

func get(t *testing.T, server *http.Server, path string) *httptest.ResponseRecorder {
	t.Helper()
	recorder := httptest.NewRecorder()
	server.Handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
	return recorder
}

func TestServerServesIndexAtRoot(t *testing.T) {
	server, err := newServer([]string{webDirectory(t)})
	if err != nil {
		t.Fatalf("newServer: %v", err)
	}

	response := get(t, server, "/")

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if body := response.Body.String(); body != "<html>ring</html>" {
		t.Errorf("body = %q, want the index page", body)
	}
}

func TestServerServesWASMWithWASMContentType(t *testing.T) {
	server, err := newServer([]string{webDirectory(t)})
	if err != nil {
		t.Fatalf("newServer: %v", err)
	}

	response := get(t, server, "/game.wasm")

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if contentType := response.Header().Get("Content-Type"); contentType != "application/wasm" {
		t.Errorf("Content-Type = %q, want application/wasm", contentType)
	}
}

func TestServerReturnsNotFoundForMissingFile(t *testing.T) {
	server, err := newServer([]string{webDirectory(t)})
	if err != nil {
		t.Fatalf("newServer: %v", err)
	}

	response := get(t, server, "/missing.js")

	if response.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", response.Code, http.StatusNotFound)
	}
}

func TestServerListensOnLocalPort8080(t *testing.T) {
	server, err := newServer([]string{webDirectory(t)})
	if err != nil {
		t.Fatalf("newServer: %v", err)
	}

	if server.Addr != "127.0.0.1:8080" {
		t.Errorf("Addr = %q, want 127.0.0.1:8080", server.Addr)
	}
}

func TestNewServerRejectsBadArguments(t *testing.T) {
	directory := webDirectory(t)
	cases := map[string][]string{
		"no arguments":      {},
		"too many":          {directory, directory},
		"missing directory": {filepath.Join(directory, "absent")},
		"file, not a dir":   {filepath.Join(directory, "index.html")},
	}
	for name, arguments := range cases {
		t.Run(name, func(t *testing.T) {
			server, err := newServer(arguments)
			if err == nil {
				t.Fatal("err = nil, want an error")
			}
			if server != nil {
				t.Errorf("server = %v, want nil on error", server)
			}
		})
	}
}
