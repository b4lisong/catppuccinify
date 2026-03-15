package api

import "net/http"

// RegisterRoutes wires the API endpoints onto the given ServeMux.
func RegisterRoutes(mux *http.ServeMux, h *Handler) {
	mux.HandleFunc("POST /api/convert", h.HandleConvert)
	mux.HandleFunc("GET /api/status/{job_id}", h.HandleStatus)
	mux.HandleFunc("GET /api/download/{job_id}", h.HandleDownload)
}
