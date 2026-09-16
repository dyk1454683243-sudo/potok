package protocol

import "time"

type Manifest struct {
	Files []ManifestFile `json:"files"`
}

type ManifestFile struct {
	Path    string    `json:"path"`
	BlobID  string    `json:"blob_id"`
	Size    int64     `json:"size"`
	ModTime time.Time `json:"mod_time"`
}
