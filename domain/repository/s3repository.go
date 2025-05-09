package repository

import (
	"context"
	"s3proxy/domain/entity"
)

// S3Repository defines the interface for S3 storage operations
type S3Repository interface {
	// ListObjects returns all objects in the bucket
	ListObjects(ctx context.Context) (*entity.ObjectCollection, error)
	
	// FindObject tries to find an object by path, using different matching strategies
	// Returns the matching object key or empty string if not found
	FindObject(ctx context.Context, objectPath string) (string, error)
	
	// GetPresignedURL generates a pre-signed URL for an object
	GetPresignedURL(ctx context.Context, objectKey string, expiryMinutes int) (string, error)
	
	// GetBucketName returns the current bucket name
	GetBucketName() string
	
	// GetEndpoint returns the S3 endpoint
	GetEndpoint() string
}