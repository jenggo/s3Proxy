package pkg

import (
	"context"
	"net/url"
	"s3proxy/types"
	"strings"
	"sync"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// S3 is the client for S3 operations, exported for global use
type S3 struct {
	client *minio.Client
}

// Global initialization tracking
var (
	Client  *S3
	once    sync.Once
	initErr error
	logger  zerolog.Logger
)

// InitMinio initializes the global MinIO client
func InitMinio() error {
	// Initialize the package logger
	logger = log.With().Caller().Logger()

	once.Do(func() {
		client, err := minio.New(types.Config.S3.Endpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(types.Config.S3.Key.Access, types.Config.S3.Key.Secret, ""),
			Secure: true,
		})

		if err != nil {
			logger.Error().Err(err).Send()
			initErr = err
			return
		}

		Client = &S3{client: client}
	})

	return initErr
}

// NewMinio creates a new MinIO client (kept for backward compatibility)
func NewMinio() (*S3, error) {
	// If global client exists, return it
	if Client != nil {
		return Client, nil
	}

	// Otherwise create a new instance (fallback)
	client, err := minio.New(types.Config.S3.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(types.Config.S3.Key.Access, types.Config.S3.Key.Secret, ""),
		Secure: true,
	})

	if err != nil {
		logger.Error().Err(err).Send()
		return nil, err
	}

	s3Client := &S3{client: client}

	// Set global client if not already set
	if Client == nil {
		Client = s3Client
	}

	return s3Client, nil
}

func (s3 *S3) List(ctx context.Context, baseURL string) (list []types.List) {
	objectList := s3.client.ListObjects(ctx, types.Config.S3.Bucket, minio.ListObjectsOptions{Recursive: true})

	for object := range objectList {
		if object.Err != nil {
			continue
		}

		// Skip if it's a folder (S3 represents folders as objects ending with '/')
		// Some S3 implementations also create 0-byte objects to represent folders
		if strings.HasSuffix(object.Key, "/") {
			logger.Debug().Msgf("Skipping folder: %s", object.Key)
			continue
		}

		// Skip if it's a zero-size object that looks like a folder marker
		// (doesn't have a file extension but has path separators)
		if object.Size == 0 {
			// Extract the filename portion (after the last slash)
			filename := object.Key
			lastSlashIndex := strings.LastIndex(object.Key, "/")
			if lastSlashIndex >= 0 {
				filename = object.Key[lastSlashIndex+1:]
			}

			// If the filename has no extension and is empty or doesn't contain a dot,
			// it's likely a folder marker
			if filename == "" || !strings.Contains(filename, ".") {
				logger.Debug().Msgf("Skipping likely folder marker: %s (size: %d)", object.Key, object.Size)
				continue
			}
		}

		// Security check: Reject files with suspicious patterns
		if strings.Contains(object.Key, "..") {
			logger.Warn().Msgf("Skipping file with suspicious path: %s", object.Key)
			continue
		}

		f := types.List{
			Name: object.Key,
			// Use the original object key in the URL for maximum compatibility
			Url: baseURL + "/" + url.QueryEscape(object.Key),
		}

		list = append(list, f)
	}

	return list
}

func (s3 *S3) PresignedUrl(ctx context.Context, objectName string) (string, error) {
	reqParams := make(url.Values)
	reqParams.Set("response-content-disposition", "inline")

	url, err := s3.client.PresignedGetObject(ctx, types.Config.S3.Bucket, objectName, time.Hour, reqParams)
	if err != nil {
		logger.Error().Err(err).Send()
		return "", err
	}

	return url.String(), nil
}

func (s3 *S3) FindObject(ctx context.Context, objectPath string) string {
	// Try decoding the path if it might be URL-encoded
	decodedPath, decodeErr := url.QueryUnescape(objectPath)
	if decodeErr == nil && decodedPath != objectPath {
		logger.Debug().Msgf("Successfully decoded path: %s -> %s", objectPath, decodedPath)
	}

	if objectPath == "" {
		return ""
	}

	// 1. Try exact match first with original and decoded path
	objectList := s3.client.ListObjects(ctx, types.Config.S3.Bucket, minio.ListObjectsOptions{Recursive: true})
	for object := range objectList {
		if object.Err != nil {
			continue
		}

		// Try matching with the original path
		if object.Key == objectPath {
			logger.Debug().Msgf("Found exact match: %s", object.Key)
			return object.Key
		}

		// If we have a decoded path, try that too
		if decodeErr == nil && object.Key == decodedPath {
			logger.Debug().Msgf("Found exact match with decoded path: %s", object.Key)
			return object.Key
		}
	}

	// 2. Try case-insensitive match
	objectList = s3.client.ListObjects(ctx, types.Config.S3.Bucket, minio.ListObjectsOptions{Recursive: true})
	for object := range objectList {
		if object.Err != nil {
			continue
		}

		// Try with original path
		if strings.EqualFold(object.Key, objectPath) {
			logger.Info().Msgf("Found case-insensitive match: %s for request: %s", object.Key, objectPath)
			return object.Key
		}

		// If we have a decoded path, try that too
		if decodeErr == nil && strings.EqualFold(object.Key, decodedPath) {
			logger.Info().Msgf("Found case-insensitive match with decoded path: %s for request: %s",
				object.Key, decodedPath)
			return object.Key
		}
	}

	// 3. Try path comparison by components
	objectList = s3.client.ListObjects(ctx, types.Config.S3.Bucket, minio.ListObjectsOptions{Recursive: true})

	// Choose which path to use for component analysis
	analysisPath := objectPath
	if decodeErr == nil && decodedPath != objectPath {
		// If we successfully decoded a different path, use that for component analysis
		// as it might have special characters that make more sense for analysis
		analysisPath = decodedPath
	}

	// Split requested path into components
	requestParts := strings.Split(analysisPath, "/")
	if len(requestParts) < 2 {
		logger.Debug().Msgf("Object not found and path too simple for component matching: %s", analysisPath)
		return ""
	}

	// Extract filename from request (last component)
	requestedFilename := requestParts[len(requestParts)-1]

	// Find best match
	var bestMatch string
	var bestMatchScore int

	for object := range objectList {
		if object.Err != nil {
			continue
		}

		// Split S3 object key into components
		objectParts := strings.Split(object.Key, "/")
		if len(objectParts) < 2 {
			continue
		}

		// Get filename from object (last component)
		objectFilename := objectParts[len(objectParts)-1]

		// If filenames match exactly (case-insensitive)
		if strings.EqualFold(objectFilename, requestedFilename) {
			// Folder path similarity check - basic implementation
			// Score path component similarities
			score := 0

			// Compare folder components
			reqFolder := strings.ToLower(strings.Join(requestParts[:len(requestParts)-1], "/"))
			objFolder := strings.ToLower(strings.Join(objectParts[:len(objectParts)-1], "/"))

			// Calculate similarity (this is a simple algorithm - Levenshtein would be better)
			// But for basic purposes, this checks if one is contained in the other
			switch {
			case reqFolder == objFolder:
				score = 100
			case strings.Contains(objFolder, reqFolder) || strings.Contains(reqFolder, objFolder):
				score = 50
			default:
				similarChars := 0
				maxLen := len(reqFolder)
				maxLen = min(len(objFolder), maxLen)

				for i := range reqFolder[:maxLen] {
					if reqFolder[i] == objFolder[i] {
						similarChars++
					}
				}

				// Score based on character similarity percentage
				if maxLen > 0 {
					score = (similarChars * 40) / maxLen
				}
			}

			// Update best match if this is better
			if score > bestMatchScore {
				bestMatchScore = score
				bestMatch = object.Key
			}
		}
	}

	// If we found a reasonably good match, use it
	if bestMatchScore > 30 { // Threshold for accepting a match
		logger.Info().Msgf("Found fuzzy match: %s (score: %d) for request: %s", bestMatch, bestMatchScore, objectPath)
		return bestMatch
	}

	logger.Debug().Msgf("Object not found in bucket: %s", objectPath)
	return ""
}
