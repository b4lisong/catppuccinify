package job

import (
	"context"
	"log"
	"os"
	"time"
)

// StartCleanup launches a background goroutine that periodically removes
// jobs older than maxAge along with their associated files.
func StartCleanup(ctx context.Context, store *Store, maxAge time.Duration, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				now := time.Now()
				store.Range(func(id string, j *Job) bool {
					if now.Sub(j.CreatedAt) <= maxAge {
						return true
					}
					if j.InputPath != "" {
						if err := os.Remove(j.InputPath); err != nil && !os.IsNotExist(err) {
							log.Printf("cleanup: failed to remove input %s: %v", j.InputPath, err)
						}
					}
					if j.OutputPath != "" {
						if err := os.Remove(j.OutputPath); err != nil && !os.IsNotExist(err) {
							log.Printf("cleanup: failed to remove output %s: %v", j.OutputPath, err)
						}
					}
					store.Delete(id)
					return true
				})
			}
		}
	}()
}
