package filestore

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// FileMetadata stores info about an uploaded file
type FileMetadata struct {
	ID          string    `json:"id"`
	Filename    string    `json:"filename"`
	ContentType string    `json:"content_type"`
	Size        int64     `json:"size"`
	CreatedAt   time.Time `json:"created_at"`
}

// FileStore manages file upload/download on disk
type FileStore struct {
	uploadDir string
}

// NewFileStore creates a FileStore and ensures the upload directory exists
func NewFileStore(uploadDir string) (*FileStore, error) {
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create upload dir %s: %w", uploadDir, err)
	}
	return &FileStore{uploadDir: uploadDir}, nil
}

func generateID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// Save writes file bytes to disk and returns metadata
func (fs *FileStore) Save(filename string, data []byte) (*FileMetadata, error) {
	id := generateID()

	// Determine content type from extension
	contentType := "application/octet-stream"
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".pdf":
		contentType = "application/pdf"
	case ".jpg", ".jpeg":
		contentType = "image/jpeg"
	case ".png":
		contentType = "image/png"
	case ".txt":
		contentType = "text/plain"
	case ".json":
		contentType = "application/json"
	}

	// Write file to disk: <uploadDir>/<id>_<filename>
	diskName := fmt.Sprintf("%s_%s", id, filename)
	fullPath := filepath.Join(fs.uploadDir, diskName)

	if err := os.WriteFile(fullPath, data, 0644); err != nil {
		return nil, fmt.Errorf("failed to write file: %w", err)
	}

	meta := &FileMetadata{
		ID:          id,
		Filename:    filename,
		ContentType: contentType,
		Size:        int64(len(data)),
		CreatedAt:   time.Now(),
	}

	log.Printf("[FileStore] Saved file %s (%d bytes) as %s", filename, len(data), id)
	return meta, nil
}

// Get reads a file from disk by ID prefix match
func (fs *FileStore) Get(id string) ([]byte, *FileMetadata, error) {
	entries, err := os.ReadDir(fs.uploadDir)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read upload dir: %w", err)
	}

	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), id+"_") {
			fullPath := filepath.Join(fs.uploadDir, entry.Name())
			data, err := os.ReadFile(fullPath)
			if err != nil {
				return nil, nil, err
			}

			filename := strings.TrimPrefix(entry.Name(), id+"_")
			contentType := "application/octet-stream"
			ext := strings.ToLower(filepath.Ext(filename))
			switch ext {
			case ".pdf":
				contentType = "application/pdf"
			case ".jpg", ".jpeg":
				contentType = "image/jpeg"
			case ".png":
				contentType = "image/png"
			case ".txt":
				contentType = "text/plain"
			}

			return data, &FileMetadata{
				ID:          id,
				Filename:    filename,
				ContentType: contentType,
				Size:        int64(len(data)),
			}, nil
		}
	}

	return nil, nil, fmt.Errorf("file not found: %s", id)
}

// ─── HTTP Handlers ──────────────────────────────────────────────────────────

// HandleUpload handles POST /api/v1/files
// Accepts multipart form with a "file" field, or raw bytes with X-Filename header
func (fs *FileStore) HandleUpload(w http.ResponseWriter, r *http.Request) {
	// CORS
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Filename")
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	var fileData []byte
	var filename string

	// Try multipart first
	if err := r.ParseMultipartForm(50 << 20); err == nil { // 50MB max
		file, header, err := r.FormFile("file")
		if err == nil {
			defer file.Close()
			fileData, _ = io.ReadAll(file)
			filename = header.Filename
		}
	}

	// Fallback: raw body with X-Filename header
	if fileData == nil {
		filename = r.Header.Get("X-Filename")
		if filename == "" {
			filename = "upload.bin"
		}
		fileData, _ = io.ReadAll(r.Body)
	}

	if len(fileData) == 0 {
		http.Error(w, `{"error":"no file data received"}`, http.StatusBadRequest)
		return
	}

	meta, err := fs.Save(filename, fileData)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"file_id":      meta.ID,
		"filename":     meta.Filename,
		"size":         meta.Size,
		"content_type": meta.ContentType,
		"download_url": fmt.Sprintf("/api/v1/files/%s", meta.ID),
	})
}

// HandleDownload handles GET /api/v1/files/{id}
func (fs *FileStore) HandleDownload(w http.ResponseWriter, r *http.Request) {
	// CORS
	w.Header().Set("Access-Control-Allow-Origin", "*")

	fileID := r.PathValue("id")
	if fileID == "" {
		http.Error(w, `{"error":"file ID required"}`, http.StatusBadRequest)
		return
	}

	data, meta, err := fs.Get(fileID)
	if err != nil {
		http.Error(w, `{"error":"file not found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", meta.ContentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, meta.Filename))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", meta.Size))
	w.Write(data)
}
