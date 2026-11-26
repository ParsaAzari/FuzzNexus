package output

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/parsaazari/fuzznexus/pkg/fuzznexus"
)

func TestAuditLogger(t *testing.T) {
	file, _ := os.CreateTemp("", "prefix")
	filename := file.Name()
	file.Close()
	audit, err := NewAuditLogger(filename)

	if err != nil {
		t.Errorf("Error creating audit logger: %s", err)
	}

	if audit == nil {
		t.Errorf("Audit was nil, expected non-nil")
	}

	err = os.Chmod(filename, 000)
	if err != nil {
		t.Errorf("Error executing Chmod: %s", err)
	}

	reopened, err := NewAuditLogger(filename)

	if err == nil {
		t.Errorf("Error was nil, expected non-nil")
	}

	if reopened != nil {
		t.Errorf("Audit was non-nil, expected nil")
	}

	err = os.Chmod(filename, 6550)
	if err != nil {
		t.Errorf("Error executing Chmod: %s", err)
	}

	// Close before removing rather than in a deferred call: Windows refuses to
	// delete a file that is still open.
	if audit != nil {
		audit.Close()
	}

	err = os.Remove(filename)
	if err != nil {
		t.Errorf("Error executing Remove: %s", err)
	}

}

func TestAuditLogWrite(t *testing.T) {
	headers := make(map[string]string)
	headers["foo"] = "bar"
	headers["baz"] = "wibble"
	headers["Content-Type"] = "application/json"

	data := "{\"quote\":\"I'll still be here tomorrow to high five you yesterday, my friend. Peace.\"}"

	req := fuzznexus.Request{Method: "POST", Url: "http://example.com/aaaa", Headers: headers, Data: []byte(data)}
	config := fuzznexus.Config{Method: "POST", Url: "http://example.com/aaaa", Headers: headers, Data: data}

	file, err := os.CreateTemp("", "prefix")
	if err != nil {
		t.Fatal(err)
	}
	filename := file.Name()
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	audit, err := NewAuditLogger(filename)

	if err != nil {
		t.Fatalf("Error creating audit logger: %s", err)
	}

	if audit == nil {
		t.Fatal("Audit was nil, expected non-nil")
	}

	// test attempting to log a bung object and erroring
	type A struct{ A *A } // cyclic struct to break JSON Marshalling
	a := A{}
	a.A = &a

	err = audit.Write(a)
	if err == nil {
		t.Errorf("Error was nil, expected non-nil")
	}

	// test writing the config and request objects
	err = audit.Write(config)
	if err != nil {
		t.Errorf("Error was not nil, expected nil")
	}

	err = audit.Write(req)
	if err != nil {
		t.Errorf("Error was not nil, expected nil")
	}

	audit.Close()

	// re-read and check the audit log
	f, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal("Error reading audit log:", err)
	}

	type auditRecord struct {
		Type string          `json:"Type"`
		Data json.RawMessage `json:"Data"`
	}
	lines := strings.Split(strings.TrimSpace(string(f)), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected two audit records, got %d", len(lines))
	}
	records := make([]auditRecord, len(lines))
	for i, line := range lines {
		if err := json.Unmarshal([]byte(line), &records[i]); err != nil {
			t.Fatalf("invalid audit record %d: %v", i, err)
		}
	}
	if records[0].Type != "fuzznexus.Config" || records[1].Type != "fuzznexus.Request" {
		t.Fatalf("unexpected audit record types: %q, %q", records[0].Type, records[1].Type)
	}

	var gotConfig map[string]interface{}
	if err := json.Unmarshal(records[0].Data, &gotConfig); err != nil {
		t.Fatalf("invalid config record: %v", err)
	}
	if gotConfig["method"] != config.Method || gotConfig["url"] != config.Url || gotConfig["postdata"] != config.Data {
		t.Fatalf("config record does not contain the written config: %#v", gotConfig)
	}

	var gotRequest map[string]interface{}
	if err := json.Unmarshal(records[1].Data, &gotRequest); err != nil {
		t.Fatalf("invalid request record: %v", err)
	}
	if gotRequest["Method"] != req.Method || gotRequest["Url"] != req.Url || gotRequest["Data"] != base64.StdEncoding.EncodeToString(req.Data) {
		t.Fatalf("request record does not contain the written request: %#v", gotRequest)
	}

	// writing to a closed audit log should fail
	err = audit.Write(config)
	if err == nil {
		t.Errorf("Error was nil, expected non-nil")
	}
}
