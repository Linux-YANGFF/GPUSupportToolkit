package platform

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func createTempFile(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")
	t.Cleanup(func() { os.Remove(path) })
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	return path
}

func TestNewStreamReader(t *testing.T) {
	tempPath := ""
	if f, err := os.CreateTemp("", "gst-test-*.log"); err == nil {
		f.WriteString("test\nline\n")
		tempPath = f.Name()
		f.Close()
		defer os.Remove(tempPath)
	}

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{"valid file", tempPath, false},
		{"nonexistent", "/nonexistent/path/xyz_deadbeef.log", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sr, err := NewStreamReader(tt.path)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				if sr != nil {
					t.Error("expected nil reader on error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if sr == nil {
				t.Fatal("expected non-nil StreamReader")
			}
			sr.Close()
		})
	}
}

func TestStreamReaderReadLines(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		expectLines int
	}{
		{"three lines", "line1\nline2\nline3\n", 3},
		{"single line", "only one line\n", 1},
		{"empty content", "", 0},
		{"with blank lines", "a\n\nb\n\nc\n", 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := createTempFile(t, tt.content)
			sr, err := NewStreamReader(path)
			if err != nil {
				t.Fatalf("NewStreamReader: %v", err)
			}
			defer sr.Close()

			count := 0
			for range sr.ReadLines() {
				count++
			}

			if count != tt.expectLines {
				t.Errorf("expected %d lines, got %d", tt.expectLines, count)
			}
		})
	}
}

func TestStreamReaderGetFileSize(t *testing.T) {
	content := "hello world\n"
	path := createTempFile(t, content)
	sr, err := NewStreamReader(path)
	if err != nil {
		t.Fatalf("NewStreamReader: %v", err)
	}
	defer sr.Close()

	expected := int64(len(content))
	if got := sr.GetFileSize(); got != expected {
		t.Errorf("GetFileSize = %d, want %d", got, expected)
	}
}

func TestStreamReaderClose(t *testing.T) {
	path := createTempFile(t, "data\n")
	sr, err := NewStreamReader(path)
	if err != nil {
		t.Fatalf("NewStreamReader: %v", err)
	}

	if err := sr.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
}

func TestDetectOS(t *testing.T) {
	result := DetectOS()

	validOS := map[string]bool{
		"ubuntu": true, "kylin": true, "uos": true,
		"debian": true, "rhel": true, "other": true,
	}

	if !validOS[result] {
		t.Errorf("DetectOS returned unexpected value: %q", result)
	}
	t.Logf("DetectOS = %q", result)
}

func TestIsSupportedOS(t *testing.T) {
	osType := DetectOS()
	supported := IsSupportedOS()

	switch osType {
	case "ubuntu", "kylin", "uos", "debian":
		if !supported {
			t.Errorf("IsSupportedOS should be true for %q, got false", osType)
		}
	case "rhel", "other":
		if supported {
			t.Errorf("IsSupportedOS should be false for %q, got true", osType)
		}
	}
}

func TestGetEnvInfo(t *testing.T) {
	info := GetEnvInfo()
	if info == nil {
		t.Fatal("GetEnvInfo returned nil")
	}

	home := os.Getenv("HOME")
	if info.Home != home {
		t.Errorf("Home = %q, want %q", info.Home, home)
	}

	t.Logf("EnvInfo: Display=%q XDGConfig=%q Home=%q",
		info.Display, info.XDGConfig, info.Home)
}

func TestSearchPattern(t *testing.T) {
	content := "glBindBuffer: count=1, time=100\n"
	content += "glDrawElements: count=1, time=200\n"
	content += "glBindBuffer: count=2, time=300\n"
	path := createTempFile(t, content)

	tests := []struct {
		name    string
		pattern string
		want    int
		wantErr bool
	}{
		{"find glBindBuffer", "glBindBuffer", 2, false},
		{"find glDrawElements", "glDrawElements", 1, false},
		{"no match", "nonexistent_func", 0, false},
		{"invalid regex", "[", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			positions, err := SearchPattern(path, tt.pattern)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error for invalid pattern")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(positions) != tt.want {
				t.Errorf("expected %d matches, got %d (positions=%v)",
					tt.want, len(positions), positions)
			}
		})
	}
}

func TestWriteIndexReadIndex(t *testing.T) {
	dir := t.TempDir()
	indexPath := filepath.Join(dir, "test.idx")

	original := &LogIndex{
		Version:    1,
		FilePath:   "/tmp/test.log",
		FileSize:   1024,
		TotalLines: 100,
		FrameStarts: []int64{0, 512, 768},
	}

	if err := WriteIndex(indexPath, original); err != nil {
		t.Fatalf("WriteIndex: %v", err)
	}

	loaded, err := ReadIndex(indexPath)
	if err != nil {
		t.Fatalf("ReadIndex: %v", err)
	}

	if loaded.FilePath != original.FilePath {
		t.Errorf("FilePath = %q, want %q", loaded.FilePath, original.FilePath)
	}
	if loaded.FileSize != original.FileSize {
		t.Errorf("FileSize = %d, want %d", loaded.FileSize, original.FileSize)
	}
	if loaded.TotalLines != original.TotalLines {
		t.Errorf("TotalLines = %d, want %d", loaded.TotalLines, original.TotalLines)
	}
	if len(loaded.FrameStarts) != len(original.FrameStarts) {
		t.Errorf("FrameStarts len = %d, want %d",
			len(loaded.FrameStarts), len(original.FrameStarts))
	}
	fmt.Println("Test completed successfully")
}

func TestReadIndex_Nonexistent(t *testing.T) {
	_, err := ReadIndex("/nonexistent/path/xyz_deadbeef.idx")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestReadIndex_InvalidMagic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.idx")
	if err := os.WriteFile(path, []byte("BADM"), 0644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	_, err := ReadIndex(path)
	if err == nil {
		t.Error("expected error for invalid magic")
	}
}

func TestReadIndex_InvalidVersion(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "badver.idx")
	magicBytes := []byte{0x49, 0x54, 0x53, 0x47}
	versionBytes := []byte{0xFF, 0x00, 0x00, 0x00}
	badData := append(magicBytes, versionBytes...)
	if err := os.WriteFile(path, badData, 0644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	_, err := ReadIndex(path)
	if err == nil {
		t.Error("expected error for unsupported version")
	}
}

func TestWriteIndex_EmptyFrameStarts(t *testing.T) {
	dir := t.TempDir()
	indexPath := filepath.Join(dir, "empty_frames.idx")
	original := &LogIndex{
		FilePath:    "/tmp/test.log",
		FileSize:    512,
		TotalLines:  50,
		FrameStarts: []int64{},
	}
	if err := WriteIndex(indexPath, original); err != nil {
		t.Fatalf("WriteIndex: %v", err)
	}
	loaded, err := ReadIndex(indexPath)
	if err != nil {
		t.Fatalf("ReadIndex: %v", err)
	}
	if len(loaded.FrameStarts) != 0 {
		t.Errorf("expected 0 frame starts, got %d", len(loaded.FrameStarts))
	}
}

func TestStreamReaderClose_NilFile(t *testing.T) {
	sr := &StreamReader{file: nil}
	if err := sr.Close(); err != nil {
		t.Errorf("Close with nil file should not error, got: %v", err)
	}
}

func TestGetOSVersion(t *testing.T) {
	name, version := GetOSVersion()
	if name == "" {
		t.Error("GetOSVersion name should not be empty")
	}
	t.Logf("OS: name=%q version=%q", name, version)
}

func TestIsKylinV10(t *testing.T) {
	result := IsKylinV10()
	t.Logf("IsKylinV10 = %v", result)
}

func TestCheckDesktopEnvironment(t *testing.T) {
	hasDesktop, err := CheckDesktopEnvironment()
	if err != nil {
		t.Logf("CheckDesktopEnvironment returned error (expected in headless env): %v", err)
	}
	if hasDesktop {
		t.Log("Desktop environment detected")
	}
}

func TestSearchPattern_NonexistentFile(t *testing.T) {
	_, err := SearchPattern("/nonexistent/path/xyz_deadbeef.log", "pattern")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestStreamReader_SeekToLine(t *testing.T) {
	content := "line1\nline2\nline3\nline4\nline5\n"
	path := createTempFile(t, content)
	sr, err := NewStreamReader(path)
	if err != nil {
		t.Fatalf("NewStreamReader: %v", err)
	}
	defer sr.Close()

	if err := sr.SeekToLine(3); err != nil {
		t.Fatalf("SeekToLine: %v", err)
	}
}

func TestStreamReader_ReadLinesWithProgress(t *testing.T) {
	content := "a\nb\nc\nd\ne\n"
	path := createTempFile(t, content)
	sr, err := NewStreamReader(path)
	if err != nil {
		t.Fatalf("NewStreamReader: %v", err)
	}
	defer sr.Close()

	var progresses []float64
	ch := sr.ReadLinesWithProgress(func(p float64) {
		progresses = append(progresses, p)
	})

	count := 0
	for range ch {
		count++
	}

	if count != 5 {
		t.Errorf("expected 5 lines, got %d", count)
	}
	if len(progresses) == 0 {
		t.Error("expected progress callbacks")
	}
}

func TestWriteIndex_InvalidPath(t *testing.T) {
	err := WriteIndex("/nonexistent/dir/test.idx", &LogIndex{FilePath: "test.log"})
	if err == nil {
		t.Error("expected error for invalid path")
	}
}

func TestReadIndex_Truncated(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "trunc.idx")
	if err := os.WriteFile(path, []byte{0x49, 0x54, 0x53}, 0644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	_, err := ReadIndex(path)
	if err == nil {
		t.Error("expected error for truncated file")
	}
}
