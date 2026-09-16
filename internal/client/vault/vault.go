package vault

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

type FileInfo struct {
	Path    string
	Size    int64
	ModTime time.Time
	Hash    string
}

func ScanVault(dir string) ([]FileInfo, error) {
	var files []FileInfo

	stat, err := os.Stat(dir)
	if err != nil {
		return nil, err
	}
	if !stat.IsDir() {
		return nil, os.ErrNotExist
	}

	err = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("vault: walk %s: %w", path, err)
		}

		if !d.IsDir() {
			name, err := filepath.Rel(dir, path)
			if err != nil {
				return fmt.Errorf("vault: relative path for %s: %w", path, err)
			}

			info, err := d.Info()
			if err != nil {
				return fmt.Errorf("vault: stat %s: %w", path, err)
			}

			hash, err := hashFile(path)
			if err != nil {
				return fmt.Errorf("vault: hash %s: %w", path, err)
			}

			files = append(files, FileInfo{
				Path:    name,
				Size:    info.Size(),
				ModTime: info.ModTime(),
				Hash:    hash,
			})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return files, nil
}

func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}
