//go:build integration

package main_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

var binaryPath string

func TestMain(m *testing.M) {
	tmp, err := os.MkdirTemp("", "fmc-*")
	if err != nil {
		panic("failed to create temp dir: " + err.Error())
	}
	defer os.RemoveAll(tmp)

	bin := filepath.Join(tmp, "fmc")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}

	cmd := exec.Command("go", "build", "-o", bin, ".")
	if out, err := cmd.CombinedOutput(); err != nil {
		panic("build failed: " + string(out))
	}

	binaryPath = bin
	os.Exit(m.Run())
}

func runFmc(t *testing.T, args ...string) string {
	t.Helper()
	cmd := exec.Command(binaryPath, args...)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("fmc exited with error: %v\noutput: %s", err, out)
	}
	return string(out)
}

func assertFileStatus(t *testing.T, output, filename, expectedStatus string) {
	t.Helper()
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, filename) {
			if !strings.Contains(line, expectedStatus) {
				t.Errorf("file %q: expected status %q in output line:\n  %s", filename, expectedStatus, line)
			}
			return
		}
	}
	t.Errorf("file %q: no output line found\nfull output:\n%s", filename, output)
}

func TestPlacementAudit(t *testing.T) {
	output := runFmc(t, "--placementAudit", "--dir", "example-files")

	cases := []struct {
		file   string
		status string
	}{
		{"gang-of-four.md", string("ok")},
		{"quasi-design-patterns.md", string("ok")},
		{"zabout.md", "missing"},
		{"whitespace-before-fm.md", "misplaced_whitespace_only"},
		{"content-before-fm.md", "manual_review"},
	}

	for _, tt := range cases {
		t.Run(tt.file, func(t *testing.T) {
			assertFileStatus(t, output, tt.file, tt.status)
		})
	}
}
