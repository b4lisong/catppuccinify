package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/balisong/catppuccinify/internal/job"
)

// Handler holds dependencies for the HTTP handlers.
type Handler struct {
	Store       *job.Store
	TempDir     string
	ProcessFunc func(j *job.Job)
}

// jsonError writes a JSON error response.
func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// HandleConvert accepts an image upload and queues a conversion job.
func (h *Handler) HandleConvert(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 10<<20)

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		jsonError(w, "File exceeds maximum size of 10 MB", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("image")
	if err != nil {
		jsonError(w, "No image file provided", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Read first 512 bytes to detect content type.
	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		jsonError(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	contentType := http.DetectContentType(buf[:n])
	if contentType != "image/png" && contentType != "image/jpeg" && contentType != "image/webp" {
		jsonError(w, "Unsupported format. Please upload PNG, JPEG, or WebP", http.StatusBadRequest)
		return
	}

	// Seek back to beginning after sniffing.
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		jsonError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Save to temp dir with job-based naming.
	ext := filepath.Ext(header.Filename)
	if ext == "" {
		ext = ".bin"
	}

	flavor := r.FormValue("flavor")
	j := h.Store.Create("", header.Filename, flavor)

	inputPath := filepath.Join(h.TempDir, j.ID+"_original"+ext)
	tmpFile, err := os.Create(inputPath)
	if err != nil {
		h.Store.Delete(j.ID)
		jsonError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if _, err := io.Copy(tmpFile, file); err != nil {
		tmpFile.Close()
		os.Remove(inputPath)
		h.Store.Delete(j.ID)
		jsonError(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	tmpFile.Close()

	j.InputPath = inputPath
	h.Store.Update(j)

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
		jsonError(w, "Job not found or expired", http.StatusNotFound)
		return
	}

	j, ok := h.Store.Get(jobID)
	if !ok {
		jsonError(w, "Job not found or expired", http.StatusNotFound)
		return
	}

	resp := map[string]string{
		"job_id":   j.ID,
		"status":   string(j.Status),
		"progress": strconv.Itoa(j.Progress),
	}
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
		jsonError(w, "Job not found or expired", http.StatusNotFound)
		return
	}

	j, ok := h.Store.Get(jobID)
	if !ok {
		jsonError(w, "Job not found or expired", http.StatusNotFound)
		return
	}

	if j.Status != job.StatusDone {
		jsonError(w, "Image is still processing", http.StatusConflict)
		return
	}

	w.Header().Set("Content-Disposition", `attachment; filename="catppuccinified.png"`)
	http.ServeFile(w, r, j.OutputPath)
}
