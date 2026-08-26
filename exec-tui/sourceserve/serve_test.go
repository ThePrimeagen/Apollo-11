package sourceserve

// Tests written first: ./source-serve is a tiny static server for the
// reconstructed Cherry memo. It binds port 42000 and serves the HTML
// from docs/ so the laggy PDF scan can be read in a browser.

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAddr(t *testing.T) {
	t.Run("happy: the server listens on port 42000", func(t *testing.T) {
		if Addr() != ":42000" {
			t.Fatalf("Addr() = %q, want :42000", Addr())
		}
	})
	t.Run("unhappy: some other port is not source-serve", func(t *testing.T) {
		if Addr() == ":8080" || Addr() == ":8000" {
			t.Fatal("source-serve must not default to a generic HTTP port")
		}
	})
}

func TestHandler(t *testing.T) {
	t.Run("happy: GET / serves the memo as HTML", func(t *testing.T) {
		dir := t.TempDir()
		body := "<!DOCTYPE html><title>Exegesis of the 1201 and 1202 Alarms</title><p>Cherry</p>"
		if err := os.WriteFile(filepath.Join(dir, MemoFile), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		srv := httptest.NewServer(Handler(dir))
		t.Cleanup(srv.Close)

		resp, err := http.Get(srv.URL + "/")
		if err != nil {
			t.Fatalf("GET /: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("GET / status %d, want 200", resp.StatusCode)
		}
		if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "text/html") {
			t.Fatalf("Content-Type %q, want text/html", ct)
		}
		got, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(got), "1201 and 1202") {
			t.Fatalf("body is missing the memo title, got %q", got)
		}
	})

	t.Run("unhappy: a missing page is 404", func(t *testing.T) {
		dir := t.TempDir()
		srv := httptest.NewServer(Handler(dir))
		t.Cleanup(srv.Close)

		resp, err := http.Get(srv.URL + "/ghost.html")
		if err != nil {
			t.Fatalf("GET /ghost.html: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("GET /ghost.html status %d, want 404", resp.StatusCode)
		}
	})
}

func TestHandlerStaysInDocs(t *testing.T) {
	t.Run("happy: the memo filename itself is served", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, MemoFile), []byte("<html>ok</html>"), 0o644); err != nil {
			t.Fatal(err)
		}
		srv := httptest.NewServer(Handler(dir))
		t.Cleanup(srv.Close)

		resp, err := http.Get(srv.URL + "/" + MemoFile)
		if err != nil {
			t.Fatalf("GET /%s: %v", MemoFile, err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status %d, want 200", resp.StatusCode)
		}
	})

	t.Run("unhappy: a request cannot escape docs and read a sibling secret", func(t *testing.T) {
		root := t.TempDir()
		docs := filepath.Join(root, "docs")
		if err := os.Mkdir(docs, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, "secret.txt"), []byte("classified"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(docs, MemoFile), []byte("<html>ok</html>"), 0o644); err != nil {
			t.Fatal(err)
		}
		srv := httptest.NewServer(Handler(docs))
		t.Cleanup(srv.Close)

		req, err := http.NewRequest(http.MethodGet, srv.URL, nil)
		if err != nil {
			t.Fatal(err)
		}
		req.URL.Path = "/../secret.txt"
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("escaped GET: %v", err)
		}
		defer resp.Body.Close()
		got, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(got), "classified") {
			t.Fatal("handler leaked a file outside docs")
		}
		if resp.StatusCode == http.StatusOK && strings.Contains(string(got), "classified") {
			t.Fatal("path traversal succeeded")
		}
	})
}
