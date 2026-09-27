package tarball

import (
	"archive/tar"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

func createTarGz(t *testing.T, entries map[string]string) string {
	t.Helper()

	tarGzPath := filepath.Join(t.TempDir(), "test.tar.gz")
	file, err := os.Create(tarGzPath)
	if err != nil {
		t.Fatalf("create tar.gz: %v", err)
	}
	defer func() { _ = file.Close() }()

	gzipWriter := gzip.NewWriter(file)
	tarWriter := tar.NewWriter(gzipWriter)

	for name, content := range entries {
		if err := tarWriter.WriteHeader(&tar.Header{Name: name, Typeflag: tar.TypeReg, Size: int64(len(content))}); err != nil {
			t.Fatalf("write header: %v", err)
		}
		if _, err := tarWriter.Write([]byte(content)); err != nil {
			t.Fatalf("write content: %v", err)
		}
	}

	if err := tarWriter.Close(); err != nil {
		t.Fatalf("close tar writer: %v", err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatalf("close gzip writer: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("close file: %v", err)
	}

	return tarGzPath
}

func TestExtractTarGzStripsTopLevelDir(t *testing.T) {
	destPath := t.TempDir()
	tarGzPath := createTarGz(t, map[string]string{
		"repo-sha123/note.txt": "hello",
	})

	if err := ExtractTarGz(tarGzPath, destPath); err != nil {
		t.Fatalf("extract: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(destPath, "note.txt"))
	if err != nil {
		t.Fatalf("read extracted file: %v", err)
	}
	if string(content) != "hello" {
		t.Fatalf("content = %q, want %q", content, "hello")
	}
	if _, err := os.Stat(tarGzPath); !os.IsNotExist(err) {
		t.Fatal("tar.gz should be removed after extraction")
	}
}

func TestExtractTarGzRejectsPathTraversal(t *testing.T) {
	destPath := t.TempDir()
	tarGzPath := createTarGz(t, map[string]string{
		"repo-sha123/../../../escaped.txt": "evil",
	})

	if err := ExtractTarGz(tarGzPath, destPath); err == nil {
		t.Fatal("expected error for path traversal entry")
	}

	if _, err := os.Stat(filepath.Join(destPath, "..", "escaped.txt")); !os.IsNotExist(err) {
		t.Fatal("file escaped destination directory")
	}
}

func TestExtractTarGzSkipsRootEntries(t *testing.T) {
	destPath := t.TempDir()
	tarGzPath := createTarGz(t, map[string]string{
		"repo-sha123/nested/deep.txt": "deep",
	})

	if err := ExtractTarGz(tarGzPath, destPath); err != nil {
		t.Fatalf("extract: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(destPath, "nested", "deep.txt"))
	if err != nil {
		t.Fatalf("read nested file: %v", err)
	}
	if string(content) != "deep" {
		t.Fatalf("content = %q", content)
	}
}
