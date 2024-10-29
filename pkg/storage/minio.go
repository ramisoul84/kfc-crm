package storage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"mime"
	"net/http"
	"sync"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/ramisoul84/kfc-crm/internal/config"
)

var (
	ErrEmptyFile        = errors.New("storage: empty file")
	ErrTooLarge         = errors.New("storage: file exceeds size limit")
	ErrUnsupportedImage = errors.New("storage: unsupported image type")
)

// allowedImageTypes maps a content type to its file extension. The value is
// used only to pick a stable extension; the URL is what callers persist.
var allowedImageTypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

// PhotoStorage stores photo blobs in MinIO and reports back the public URL
// (for clients) and the object key (for deletion).
type PhotoStorage interface {
	// Upload stores an image for the given owner and returns its public URL
	// and MinIO object key. The content type is sniffed from the bytes; the
	// provided mimeType is used only as a fallback (e.g. webp).
	Upload(ctx context.Context, ownerID string, data []byte, mimeType string) (url, key string, err error)
	Delete(ctx context.Context, key string) error
}

type minioStorage struct {
	client    *minio.Client
	bucket    string
	publicURL string
	maxBytes  int64

	ensureOnce sync.Once
	ensureErr  error
}

// NewMinIOStorage builds the client. It performs no network I/O, so a MinIO
// outage cannot block startup; the bucket is initialised on first upload.
func NewMinIOStorage(cfg config.MinIOConfig) (PhotoStorage, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("minio: create client: %w", err)
	}

	return &minioStorage{
		client:    client,
		bucket:    cfg.Bucket,
		publicURL: cfg.PublicURL,
		maxBytes:  cfg.MaxUploadBytes,
	}, nil
}

func (s *minioStorage) Upload(ctx context.Context, ownerID string, data []byte, mimeType string) (string, string, error) {
	if err := s.ensure(ctx); err != nil {
		return "", "", err
	}

	if len(data) == 0 {
		return "", "", ErrEmptyFile
	}
	if s.maxBytes > 0 && int64(len(data)) > s.maxBytes {
		return "", "", ErrTooLarge
	}

	ext := extensionFor(data, mimeType)
	if ext == "" {
		return "", "", ErrUnsupportedImage
	}

	// Key layout: photos/{ownerID}/{uuid}.ext. Owner (item) partitioning keeps
	// per-item listing cheap and object keys reliably unique.
	key := fmt.Sprintf("photos/%s/%s%s", ownerID, uuid.NewString(), ext)

	_, err := s.client.PutObject(
		ctx,
		s.bucket,
		key,
		bytes.NewReader(data),
		int64(len(data)),
		minio.PutObjectOptions{ContentType: mimeTypeFor(data, mimeType)},
	)
	if err != nil {
		return "", "", fmt.Errorf("minio: upload: %w", err)
	}

	return s.publicURL + "/" + key, key, nil
}

func (s *minioStorage) Delete(ctx context.Context, key string) error {
	if key == "" {
		return nil
	}
	if err := s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("minio: delete %q: %w", key, err)
	}
	return nil
}

// ensure initialises the bucket exactly once (create if missing + public-read
// policy so Nginx can serve objects without MinIO credentials).
func (s *minioStorage) ensure(ctx context.Context) error {
	s.ensureOnce.Do(func() {
		exists, err := s.client.BucketExists(ctx, s.bucket)
		if err != nil {
			s.ensureErr = fmt.Errorf("minio: check bucket %q: %w", s.bucket, err)
			return
		}
		if !exists {
			if err := s.client.MakeBucket(ctx, s.bucket, minio.MakeBucketOptions{}); err != nil {
				s.ensureErr = fmt.Errorf("minio: create bucket %q: %w", s.bucket, err)
				return
			}
		}

		policy := fmt.Sprintf(`{
			"Version": "2012-10-17",
			"Statement": [{
				"Effect":    "Allow",
				"Principal": {"AWS": ["*"]},
				"Action":    ["s3:GetObject"],
				"Resource":  ["arn:aws:s3:::%s/*"]
			}]
		}`, s.bucket)
		if err := s.client.SetBucketPolicy(ctx, s.bucket, policy); err != nil {
			s.ensureErr = fmt.Errorf("minio: set bucket policy: %w", err)
		}
	})
	return s.ensureErr
}

// extensionFor resolves an extension for the image. It trusts the sniffed
// magic bytes first; the caller-supplied mimeType (from the upload form) acts
// as a fallback for types the sniffer cannot detect (e.g. webp).
func extensionFor(data []byte, mimeType string) string {
	if ext, ok := allowedImageTypes[http.DetectContentType(data)]; ok {
		return ext
	}
	if mediaType, _, err := mime.ParseMediaType(mimeType); err == nil {
		return allowedImageTypes[mediaType]
	}
	return ""
}

// mimeTypeFor returns the content type to persist on the object.
func mimeTypeFor(data []byte, provided string) string {
	if mt := http.DetectContentType(data); mt != "application/octet-stream" {
		return mt
	}
	if mediaType, _, err := mime.ParseMediaType(provided); err == nil {
		return mediaType
	}
	return provided
}
