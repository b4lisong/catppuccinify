package main

import (
	"context"
	"image"
	"image/png"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	_ "image/jpeg"

	_ "golang.org/x/image/webp"

	"github.com/balisong/catppuccinify/internal/api"
	"github.com/balisong/catppuccinify/internal/converter"
	"github.com/balisong/catppuccinify/internal/job"
)

func main() {
	tempDir, err := os.MkdirTemp("", "catppuccinify-*")
	if err != nil {
		log.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	store := &job.Store{}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	job.StartCleanup(ctx, store, 10*time.Minute, 60*time.Second)

	h := &api.Handler{
		Store:   store,
		TempDir: tempDir,
		ProcessFunc: func(j *job.Job) {
			f, err := os.Open(j.InputPath)
			if err != nil {
				j.Status = job.StatusFailed
				j.Error = "Conversion failed. Please try again"
				return
			}
			defer f.Close()

			img, _, err := image.Decode(f)
			if err != nil {
				j.Status = job.StatusFailed
				j.Error = "Could not read image. File may be corrupted"
				return
			}

			result := converter.Convert(img)

			outPath := filepath.Join(tempDir, j.ID+"_output.png")
			out, err := os.Create(outPath)
			if err != nil {
				j.Status = job.StatusFailed
				j.Error = "Conversion failed. Please try again"
				return
			}
			defer out.Close()

			if err := png.Encode(out, result); err != nil {
				j.Status = job.StatusFailed
				j.Error = "Conversion failed. Please try again"
				return
			}

			j.OutputPath = outPath
		},
	}

	mux := http.NewServeMux()
	api.RegisterRoutes(mux, h)
	mux.Handle("GET /", http.FileServer(http.Dir("static")))

	srv := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	go func() {
		<-ctx.Done()
		log.Println("shutting down server...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Printf("server shutdown error: %v", err)
		}
	}()

	log.Println("listening on :8080")
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}
