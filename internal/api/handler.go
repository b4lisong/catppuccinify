package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/balisong/catppuccinify/internal/job"
)

// Handler holds dependencies for the HTTP handlers.
type Handler struct {
	Store       *job.Store
	TempDir     string
	ProcessFunc func(j *job.Job)
}

// HandleConvert accepts an image upload and queues a conversion job.
func (h *Handler) HandleConvert(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 10<<20)

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "file too large or bad form data", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "missing file field", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Read first 512 bytes to detect content type.
	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		http.Error(w, "failed to read file", http.StatusInternalServerError)
		return
	}
	contentType := http.DetectContentType(buf[:n])
	if !strings.HasPrefix(contentType, "image/") {
		http.Error(w, "file is not an image", http.StatusBadRequest)
		return
	}

	// Seek back to beginning after sniffing.
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		http.Error(w, "failed to read file", http.StatusInternalServerError)
		return
	}

	// Save to temp dir with a unique name.
	ext := filepath.Ext(header.Filename)
	tmpFile, err := os.CreateTemp(h.TempDir, "upload-*"+ext)
	if err != nil {
		http.Error(w, "failed to save file", http.StatusInternalServerError)
		return
	}
	defer tmpFile.Close()

	if _, err := io.Copy(tmpFile, file); err != nil {
		http.Error(w, "failed to save file", http.StatusInternalServerError)
		return
	}

	j := h.Store.Create(tmpFile.Name(), header.Filename)

	go func() {
		j.Status = job.StatusProcessing
		h.Store.Update(j)

		func() {
			defer func() {
				if rec := recover(); rec != nil {
					j.Status = job.StatusFailed
					j.Error = fmt.Sprintf("%v", rec)
					h.Store.Update(j)
				}
			}()
			h.ProcessFunc(j)
		}()

		if j.Status == job.StatusProcessing {
			j.Status = job.StatusDone
			h.Store.Update(j)
		}
	}()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"job_id": j.ID})
}

// HandleStatus returns the current status of a job.
func (h *Handler) HandleStatus(w http.ResponseWriter, r *http.Request) {
	jobID := r.PathValue("job_id")
	if jobID == "" {
		http.Error(w, "missing job_id", http.StatusBadRequest)
		return
	}

	j, ok := h.Store.Get(jobID)
	if !ok {
		http.Error(w, "job not found", http.StatusNotFound)
		return
	}

	resp := map[string]string{"status": string(j.Status)}
	if j.Status == job.StatusFailed {
		resp["error"] = j.Error
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// HandleDownload serves the converted output file for a completed job.
func (h *Handler) HandleDownload(w http.ResponseWriter, r *http.Request) {
	jobID := r.PathValue("job_id")
	if jobID == "" {
		http.Error(w, "missing job_id", http.StatusBadRequest)
		return
	}

	j, ok := h.Store.Get(jobID)
	if !ok {
		http.Error(w, "job not found", http.StatusNotFound)
		return
	}

	if j.Status != job.StatusDone {
		http.Error(w, "job not done", http.StatusBadRequest)
		return
	}

	// Replace original extension with .png for the download name.
	name := j.InputName
	ext := filepath.Ext(name)
	if ext != "" {
		name = strings.TrimSuffix(name, ext) + ".png"
	} else {
		name = name + ".png"
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, name))
	http.ServeFile(w, r, j.OutputPath)
}
