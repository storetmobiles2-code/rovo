// Package blob is the S3-compatible object-storage port (DB-D13, ADR-022/025). Objects are
// referenced by (bucket, key); URLs are built at request time. The real S3 adapter and a fake
// live under internal/adapters/blob; modules depend only on this interface.
package blob

import (
	"context"
	"io"
	"net/http"
	"time"
)

// PutRequest is a presigned-upload request.
type PutRequest struct {
	Bucket      string
	Key         string
	ContentType string
	MaxBytes    int64
	Expiry      time.Duration
}

// PresignedPut is what a client needs to upload directly to object storage.
type PresignedPut struct {
	URL     string
	Method  string
	Headers http.Header
	Expires time.Time
}

// Store is the object-storage interface.
type Store interface {
	// PresignPut returns a short-lived upload URL (customer/partner uploads, 11 §2.3).
	PresignPut(ctx context.Context, req PutRequest) (PresignedPut, error)
	// PresignGet returns a short-lived download URL (≤ 60 s for KYC, 12 §6.2).
	PresignGet(ctx context.Context, bucket, key string, expiry time.Duration) (string, error)
	// Put uploads bytes (server-side re-encode of images, invoice/report writes).
	Put(ctx context.Context, bucket, key, contentType string, r io.Reader, size int64) error
	// Get streams an object (audited KYC view, private file serving).
	Get(ctx context.Context, bucket, key string) (io.ReadCloser, error)
	// Delete removes an object (retention, erasure).
	Delete(ctx context.Context, bucket, key string) error
	// PublicURL builds a CDN URL for a public object, or "" if the bucket is private.
	PublicURL(bucket, key string) string
}
