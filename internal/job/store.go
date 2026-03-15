package job

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

// Status represents the current state of a job.
type Status string

const (
	StatusPending    Status = "pending"
	StatusProcessing Status = "processing"
	StatusDone       Status = "done"
	StatusFailed     Status = "failed"
)

// Job represents a single image conversion task.
type Job struct {
	ID         string
	Status     Status
	Error      string
	CreatedAt  time.Time
	InputPath  string
	OutputPath string
	InputName  string
}

// Store provides thread-safe storage for jobs.
type Store struct {
	m sync.Map
}

// Create initialises a new pending job and stores it.
func (s *Store) Create(inputPath, inputName string) *Job {
	id := generateUUID()
	j := &Job{
		ID:        id,
		Status:    StatusPending,
		CreatedAt: time.Now(),
		InputPath: inputPath,
		InputName: inputName,
	}
	s.m.Store(id, j)
	return j
}

// Get retrieves a job by ID.
func (s *Store) Get(id string) (*Job, bool) {
	v, ok := s.m.Load(id)
	if !ok {
		return nil, false
	}
	return v.(*Job), true
}

// Update stores the job back into the map.
func (s *Store) Update(j *Job) {
	s.m.Store(j.ID, j)
}

// Delete removes a job by ID.
func (s *Store) Delete(id string) {
	s.m.Delete(id)
}

// Range iterates over all jobs. Return false from fn to stop early.
func (s *Store) Range(fn func(id string, j *Job) bool) {
	s.m.Range(func(key, value any) bool {
		return fn(key.(string), value.(*Job))
	})
}

// generateUUID produces a 32-character hex string from 16 random bytes.
func generateUUID() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		panic("crypto/rand failed: " + err.Error())
	}
	return hex.EncodeToString(b)
}
