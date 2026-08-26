package sourceserve

// The reconstructed memo lives in docs/, next to the server that
// publishes it — not buried in cmd/ or copied into this package.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMemoHome(t *testing.T) {
	t.Run("happy: the shipped HTML lives in docs/ and is the Cherry memo", func(t *testing.T) {
		path, err := MemoPath()
		if err != nil {
			t.Fatalf("the shipped memo must be findable: %v", err)
		}
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("the shipped memo must read: %v", err)
		}
		if !strings.Contains(string(body), "Exegesis of the 1201 and 1202 Alarms") {
			t.Fatal("docs/CherryApollo11Exegesis.html is not the reconstructed memo")
		}
	})
	t.Run("unhappy: the memo is not copied next to the server package", func(t *testing.T) {
		if _, err := os.Stat(filepath.Join("CherryApollo11Exegesis.html")); err == nil {
			t.Fatal("the HTML must not live in sourceserve/ — it belongs in docs/")
		}
	})
}
