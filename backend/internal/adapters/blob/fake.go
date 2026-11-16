// Package blobfake is an in-memory blob.Store used in dev/test and the skeleton build. The real
// S3/MinIO adapter lands beside it (ADR-022/025); modules depend only on platform/blob.Store.
package blobfake

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/storetmobiles2-code/rovo/backend/internal/platform/blob"
)

// Store keeps objects in a map keyed by bucket/key. Presign returns a local, non-functional URL
// good enough for wiring and contract tests (the real adapter signs against S3).
type Store struct {
	mu      sync.RWMutex
	objects map[string][]byte
	baseURL string
}

// New builds an empty fake store. baseURL is used for PublicURL/presign stand-ins.
func New(baseURL string) *Store {
	if baseURL == "" {
		baseURL = "http://localhost:9000"
	}
	return &Store{objects: map[string][]byte{}, baseURL: baseURL}
}

func ref(bucket, key string) string { return bucket + "/" + key }

// PresignPut returns a stand-in upload URL.
func (s *Store) PresignPut(_ context.Context, req blob.PutRequest) (blob.PresignedPut, error) {
	return blob.PresignedPut{
		URL:     fmt.Sprintf("%s/%s?upload=1", s.baseURL, ref(req.Bucket, req.Key)),
		Method:  http.MethodPut,
		Headers: http.Header{"Content-Type": []string{req.ContentType}},
		Expires: time.Now().Add(req.Expiry),
	}, nil
}

// PresignGet returns a stand-in download URL.
func (s *Store) PresignGet(_ context.Context, bucket, key string, expiry time.Duration) (string, error) {
	return fmt.Sprintf("%s/%s?expires=%d", s.baseURL, ref(bucket, key), int(expiry.Seconds())), nil
}

// Put stores bytes.
func (s *Store) Put(_ context.Context, bucket, key, _ string, r io.Reader, _ int64) error {
	b, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.objects[ref(bucket, key)] = b
	s.mu.Unlock()
	return nil
}

// Get streams a stored object.
func (s *Store) Get(_ context.Context, bucket, key string) (io.ReadCloser, error) {
	s.mu.RLock()
	b, ok := s.objects[ref(bucket, key)]
	s.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("blobfake: not found: %s", ref(bucket, key))
	}
	return io.NopCloser(bytes.NewReader(b)), nil
}

// Delete removes an object.
func (s *Store) Delete(_ context.Context, bucket, key string) error {
	s.mu.Lock()
	delete(s.objects, ref(bucket, key))
	s.mu.Unlock()
	return nil
}

// PublicURL builds a stand-in CDN URL.
func (s *Store) PublicURL(bucket, key string) string {
	return fmt.Sprintf("%s/%s", s.baseURL, ref(bucket, key))
}

var _ blob.Store = (*Store)(nil)
