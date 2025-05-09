package repository

import (
	"context"
	"net/url"
	"s3proxy/domain/entity"
	"s3proxy/domain/repository"
	"s3proxy/infrastructure/config"
	"sync"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/rs/zerolog/log"
)

// S3RepositoryImpl implements the S3Repository interface
type S3RepositoryImpl struct {
	client   *minio.Client
	bucket   string
	endpoint string
}

// Global instance and initialization tracking
var (
	globalRepository *S3RepositoryImpl
	once             sync.Once
	initErr          error
)

// NewS3Repository creates a new S3Repository instance
func NewS3Repository() (repository.S3Repository, error) {
	// Initialize the global repository if needed
	once.Do(func() {
		client, err := minio.New(config.AppConfig.S3.Endpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(config.AppConfig.S3.Key.Access, config.AppConfig.S3.Key.Secret, ""),
			Secure: true,
		})

		if err != nil {
			log.Error().Caller().Err(err).Msg("Failed to create MinIO client")
			initErr = err
			return
		}

		globalRepository = &S3RepositoryImpl{
			client:   client,
			bucket:   config.AppConfig.S3.Bucket,
			endpoint: config.AppConfig.S3.Endpoint,
		}
	})

	if initErr != nil {
		return nil, initErr
	}

	return globalRepository, nil
}

// ListObjects returns all objects in the bucket
func (r *S3RepositoryImpl) ListObjects(ctx context.Context) (*entity.ObjectCollection, error) {
	collection := entity.NewObjectCollection()

	// Create a done channel to control the listing
	objectCh := r.client.ListObjects(ctx, r.bucket, minio.ListObjectsOptions{Recursive: true})

	for object := range objectCh {
		if object.Err != nil {
			log.Error().Err(object.Err).Msg("Error listing objects")
			continue
		}

		// Convert MinIO object to domain entity
		s3Object := &entity.S3Object{
			Key:          object.Key,
			Size:         object.Size,
			LastModified: object.LastModified,
			ETag:         object.ETag,
			ContentType:  object.ContentType,
		}

		collection.Add(s3Object)
	}

	return collection, nil
}

// FindObject tries to find an object by path
func (r *S3RepositoryImpl) FindObject(ctx context.Context, objectPath string) (string, error) {
	// Try decoding the path if it might be URL-encoded
	decodedPath, decodeErr := url.QueryUnescape(objectPath)
	if decodeErr == nil && decodedPath != objectPath {
		log.Debug().Msgf("Successfully decoded path: %s -> %s", objectPath, decodedPath)
	}

	if objectPath == "" {
		return "", nil
	}

	// Get all objects
	collection, err := r.ListObjects(ctx)
	if err != nil {
		return "", err
	}

	// 1. Try exact match first with original and decoded path
	if obj := collection.FindByPath(objectPath); obj != nil {
		log.Debug().Msgf("Found exact match: %s", obj.Key)
		return obj.Key, nil
	}

	// Try with decoded path if available
	if decodeErr == nil && decodedPath != objectPath {
		if obj := collection.FindByPath(decodedPath); obj != nil {
			log.Debug().Msgf("Found exact match with decoded path: %s", obj.Key)
			return obj.Key, nil
		}
	}

	// 2. Try case-insensitive match
	if obj := collection.FindByCaseInsensitivePath(objectPath); obj != nil {
		log.Info().Msgf("Found case-insensitive match: %s for request: %s", obj.Key, objectPath)
		return obj.Key, nil
	}

	// Try with decoded path if available
	if decodeErr == nil && decodedPath != objectPath {
		if obj := collection.FindByCaseInsensitivePath(decodedPath); obj != nil {
			log.Info().Msgf("Found case-insensitive match with decoded path: %s for request: %s",
				obj.Key, decodedPath)
			return obj.Key, nil
		}
	}

	// 3. Try fuzzy match by components
	// Choose which path to use for component analysis
	analysisPath := objectPath
	if decodeErr == nil && decodedPath != objectPath {
		// If we successfully decoded a different path, use that for component analysis
		analysisPath = decodedPath
	}

	// Try fuzzy matching
	bestMatch, score := collection.FindByFilenameFuzzy(analysisPath)

	// If we found a reasonably good match, use it
	if bestMatch != nil && score > 30 { // Threshold for accepting a match
		log.Info().Msgf("Found fuzzy match: %s (score: %d) for request: %s",
			bestMatch.Key, score, objectPath)
		return bestMatch.Key, nil
	}

	log.Debug().Msgf("Object not found in bucket: %s", objectPath)
	return "", nil
}

// GetPresignedURL generates a pre-signed URL for an object
func (r *S3RepositoryImpl) GetPresignedURL(ctx context.Context, objectKey string, expiryMinutes int) (string, error) {
	reqParams := make(url.Values)
	reqParams.Set("response-content-disposition", "inline")

	// Calculate expiry time
	expires := time.Duration(expiryMinutes) * time.Minute

	// Generate presigned URL
	presignedURL, err := r.client.PresignedGetObject(ctx, r.bucket, objectKey, expires, reqParams)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to generate presigned URL for %s", objectKey)
		return "", err
	}

	return presignedURL.String(), nil
}

// GetBucketName returns the current bucket name
func (r *S3RepositoryImpl) GetBucketName() string {
	return r.bucket
}

// GetEndpoint returns the S3 endpoint
func (r *S3RepositoryImpl) GetEndpoint() string {
	return r.endpoint
}
