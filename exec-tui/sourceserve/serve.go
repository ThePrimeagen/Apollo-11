package sourceserve

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
)

const (
	Port     = 42000
	MemoFile = "CherryApollo11Exegesis.html"
)

func Addr() string { return fmt.Sprintf(":%d", Port) }

func Handler(docsDir string) http.Handler {
	files := http.FileServer(http.Dir(docsDir))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.ServeFile(w, r, filepath.Join(docsDir, MemoFile))
			return
		}
		files.ServeHTTP(w, r)
	})
}

// MemoPath is docs/CherryApollo11Exegesis.html, found from this
// package's source so tests and the server agree on one home.
func MemoPath() (string, error) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("cannot locate sourceserve source")
	}
	path := filepath.Join(filepath.Dir(file), "..", "docs", MemoFile)
	if _, err := os.Stat(path); err != nil {
		return "", fmt.Errorf("memo %s: %w", path, err)
	}
	return path, nil
}

func DocsDir() (string, error) {
	path, err := MemoPath()
	if err != nil {
		return "", err
	}
	return filepath.Dir(path), nil
}
