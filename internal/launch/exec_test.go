package launch

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestPositionalArgIsNotExecuted(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "PWNED")
	payload := "'; touch " + marker + "; echo '"

	// Same argv shape Build produces, but with a printf command we can capture.
	argv := []string{"-c", `printf 'ARG=%s' "$1"`, "carryon", payload}
	out, err := exec.Command("/bin/sh", argv...).Output()
	if err != nil {
		t.Fatal(err)
	}

	if got := string(out); !strings.HasPrefix(got, "ARG=") || !strings.Contains(got, payload) {
		t.Errorf("payload not passed literally: %q", got)
	}
	if _, err := os.Stat(marker); err == nil {
		t.Fatal("INJECTION: marker file was created — payload executed")
	}
}
