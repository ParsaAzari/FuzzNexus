package scan

import (
	"log/slog"
	"strings"
	"sync"
	"testing"

	"go.uber.org/goleak"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

func TestMaskSensitiveData(t *testing.T) {
	input := "Request failed with Authorization: Bearer secret-token-12345 and Cookie: session_id=abcdef123456\nSet-Cookie: auth=xyz789;"
	masked := MaskSensitiveData(input)

	if strings.Contains(masked, "secret-token-12345") {
		t.Errorf("Expected token to be masked, got: %s", masked)
	}
	if strings.Contains(masked, "session_id=abcdef123456") {
		t.Errorf("Expected cookie to be masked, got: %s", masked)
	}
	if strings.Contains(masked, "auth=xyz789") {
		t.Errorf("Expected set-cookie to be masked, got: %s", masked)
	}
	if !strings.Contains(masked, "Authorization: ***") {
		t.Errorf("Expected 'Authorization: ***', got: %s", masked)
	}
}

func TestSafeGoRecoversPanic(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)

	var recoveredErr error
	logger := slog.Default()

	SafeGo(logger, "test.panic_task", func(err error) {
		recoveredErr = err
		wg.Done()
	}, func() {
		panic("deliberate test panic")
	})

	wg.Wait()

	if recoveredErr == nil {
		t.Fatal("Expected panic callback to be called, got nil")
	}
	if !strings.Contains(recoveredErr.Error(), "deliberate test panic") {
		t.Errorf("Expected error to contain panic message, got: %v", recoveredErr)
	}
}
