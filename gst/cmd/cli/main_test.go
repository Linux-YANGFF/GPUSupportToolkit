package main_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLI_NonExistentFile(t *testing.T) {
	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "gst-cli")

	buildCmd := exec.Command("go", "build", "-o", binPath, ".")
	buildCmd.Env = append(os.Environ(), "GOFLAGS=")
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build CLI: %v\n%s", err, out)
	}

	tests := []struct {
		name    string
		args    []string
		wantMsg string
	}{
		{
			name:    "nonexistent file",
			args:    []string{"-parse", "/nonexistent/path/deadc0de_file.log"},
			wantMsg: "无法打开文件",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command(binPath, tt.args...)
			cmd.Env = append(os.Environ(), "GOFLAGS=")
			out, err := cmd.CombinedOutput()

			if !strings.Contains(string(out), tt.wantMsg) {
				t.Errorf("expected output to contain %q, got: %s", tt.wantMsg, string(out))
			}

			if exitErr, ok := err.(*exec.ExitError); ok {
				if exitErr.ExitCode() == 0 {
					t.Error("expected non-zero exit code, got 0")
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestCLI_Help(t *testing.T) {
	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "gst-cli")

	buildCmd := exec.Command("go", "build", "-o", binPath, ".")
	buildCmd.Env = append(os.Environ(), "GOFLAGS=")
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build CLI: %v\n%s", err, out)
	}

	cmd := exec.Command(binPath, "-help")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("unexpected error running help: %v", err)
	}

	if !strings.Contains(string(out), "用法:") {
		t.Errorf("help output missing usage: %s", string(out))
	}
}
