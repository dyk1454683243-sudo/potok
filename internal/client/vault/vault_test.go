package vault

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, dir, rel string, content []byte) {
	t.Helper()
	path := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("WriteFile(%s): %v", path, err)
	}
}

func hashOf(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

func findByPath(t *testing.T, files []FileInfo, path string) FileInfo {
	t.Helper()
	for _, f := range files {
		if f.Path == path {
			return f
		}
	}
	t.Fatalf("no FileInfo with Path %q in %+v", path, files)
	return FileInfo{}
}

func TestScanVault(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "note.md", []byte("hello"))
	writeFile(t, root, "daily/2026-09-16.md", []byte("today's entry"))
	writeFile(t, root, "attachments/photo.png", []byte("not actually a png"))

	files, err := ScanVault(root)
	if err != nil {
		t.Fatalf("ScanVault() error: %v", err)
	}
	if len(files) != 3 {
		t.Fatalf("got %d files, want 3: %+v", len(files), files)
	}

	note := findByPath(t, files, "note.md")
	if note.Size != int64(len("hello")) {
		t.Errorf("note.md Size = %d, want %d", note.Size, len("hello"))
	}
	if note.ModTime.IsZero() {
		t.Error("note.md ModTime is zero")
	}
	if want := hashOf([]byte("hello")); note.Hash != want {
		t.Errorf("note.md Hash = %q, want %q", note.Hash, want)
	}

	// Nested path is preserved, not collapsed to the basename.
	daily := findByPath(t, files, filepath.Join("daily", "2026-09-16.md"))
	if want := hashOf([]byte("today's entry")); daily.Hash != want {
		t.Errorf("daily entry Hash = %q, want %q", daily.Hash, want)
	}

	// Non-markdown files are included too - the scan isn't note-only.
	findByPath(t, files, filepath.Join("attachments", "photo.png"))
}

func TestScanVaultSameBasenameDifferentDirs(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "folder1/note.md", []byte("first"))
	writeFile(t, root, "folder2/note.md", []byte("second"))

	files, err := ScanVault(root)
	if err != nil {
		t.Fatalf("ScanVault() error: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("got %d files, want 2: %+v", len(files), files)
	}

	first := findByPath(t, files, filepath.Join("folder1", "note.md"))
	second := findByPath(t, files, filepath.Join("folder2", "note.md"))

	if first.Hash == second.Hash {
		t.Error("distinct files under the same basename got the same Hash - content wasn't actually distinguished")
	}
	if first.Path == second.Path {
		t.Error("distinct files under the same basename collapsed to the same Path")
	}
}

func TestScanVaultEmptyDir(t *testing.T) {
	root := t.TempDir()

	files, err := ScanVault(root)
	if err != nil {
		t.Fatalf("ScanVault() error: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("got %d files, want 0: %+v", len(files), files)
	}
}

func TestScanVaultNonexistentPath(t *testing.T) {
	_, err := ScanVault(filepath.Join(t.TempDir(), "does-not-exist"))
	if err == nil {
		t.Fatal("expected an error for a nonexistent path, got nil")
	}
}

func TestScanVaultPathIsAFile(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "not-a-directory")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := ScanVault(file)
	if err == nil {
		t.Fatal("expected an error when the vault path is a file, got nil")
	}
}

func TestScanVaultUnreadableSubdirSurfacesError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("running as root - permission bits don't block access")
	}

	root := t.TempDir()
	writeFile(t, root, "visible.md", []byte("x"))

	blocked := filepath.Join(root, "blocked")
	if err := os.Mkdir(blocked, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, root, "blocked/secret.md", []byte("x"))
	if err := os.Chmod(blocked, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(blocked, 0o755) })

	_, err := ScanVault(root)
	if err == nil {
		t.Fatal("expected ScanVault to surface an error for the unreadable subdirectory, got nil")
	}
}

func TestScanVaultDirectoriesAreNotEntries(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "notes/inbox.md", []byte("x"))

	files, err := ScanVault(root)
	if err != nil {
		t.Fatalf("ScanVault() error: %v", err)
	}

	for _, f := range files {
		if f.Path == "notes" {
			t.Errorf("directory %q was included as a file entry", f.Path)
		}
	}
}
