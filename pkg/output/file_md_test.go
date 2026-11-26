package output

import (
	"os"
	"strings"
	"testing"

	"github.com/parsaazari/fuzznexus/pkg/fuzznexus"
)

func TestWriteMarkdownDoesNotReuseFNHASH(t *testing.T) {
	filename := t.TempDir() + "/results.md"
	config := &fuzznexus.Config{}
	results := []fuzznexus.Result{
		{Input: map[string][]byte{"FUZZ": []byte("first"), "FNHASH": []byte("first-hash")}, Url: "https://example.test/first"},
		{Input: map[string][]byte{"FUZZ": []byte("second")}, Url: "https://example.test/second"},
	}

	if err := writeMarkdown(filename, config, results); err != nil {
		t.Fatalf("writeMarkdown() error = %v", err)
	}
	data, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	text := string(data)
	if !strings.Contains(text, "first-hash") {
		t.Fatalf("markdown output does not contain the first result hash: %s", text)
	}
	if strings.Count(text, "first-hash") != 1 {
		t.Fatalf("markdown output reused the first result hash: %s", text)
	}
}
