package usecase

import (
	"context"
	"errors"
	"net/url"
	"s3proxy/domain/repository"
	"strings"

	"github.com/rs/zerolog/log"
)

// ProxyFileUseCase implements the use case for proxying files
type ProxyFileUseCase struct {
	s3Repository repository.S3Repository
	expiryMinutes int
}

// NewProxyFileUseCase creates a new instance of ProxyFileUseCase
func NewProxyFileUseCase(s3Repository repository.S3Repository) *ProxyFileUseCase {
	return &ProxyFileUseCase{
		s3Repository: s3Repository,
		expiryMinutes: 60, // Default 1 hour expiry
	}
}

// SetExpiryMinutes configures the expiry time for presigned URLs
func (uc *ProxyFileUseCase) SetExpiryMinutes(minutes int) {
	if minutes > 0 {
		uc.expiryMinutes = minutes
	}
}

var (
	ErrObjectNotFound    = errors.New("object not found")
	ErrSuspiciousPath    = errors.New("suspicious path detected")
	ErrInvalidHost       = errors.New("invalid host in presigned URL")
)

// Execute performs the proxy file operation
func (uc *ProxyFileUseCase) Execute(ctx context.Context, encodedPath string) (string, error) {
	// Basic security check: reject paths with suspicious patterns
	if strings.Contains(encodedPath, "..") || strings.Contains(encodedPath, "//") {
		log.Warn().Msgf("Rejected suspicious path: %s", encodedPath)
		return "", ErrSuspiciousPath
	}

	// Decode the URL-encoded path
	objectPath, err := url.QueryUnescape(encodedPath)
	if err != nil {
		log.Warn().Err(err).Msgf("Failed to decode URL: %s", encodedPath)
		// If decoding fails, use the original encoded path
		objectPath = encodedPath
	}

	// Additional security check after decoding
	if strings.Contains(objectPath, "..") || strings.Contains(objectPath, "//") {
		log.Warn().Msgf("Rejected suspicious decoded path: %s", objectPath)
		return "", ErrSuspiciousPath
	}

	if objectPath == "" || encodedPath == "" {
		return "", ErrObjectNotFound
	}

	log.Debug().Msgf("Request for %s (decoded from %s)", objectPath, encodedPath)

	// Verify object exists first
	objectKey, err := uc.s3Repository.FindObject(ctx, objectPath)
	if err != nil {
		log.Error().Err(err).Msgf("Error finding object: %s", objectPath)
		return "", err
	}
	
	if objectKey == "" {
		log.Warn().Msgf("Object not found: %s", objectPath)
		return "", ErrObjectNotFound
	}

	// Generate presigned URL for the verified object
	presignedURL, err := uc.s3Repository.GetPresignedURL(ctx, objectKey, uc.expiryMinutes)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to generate presigned URL for: %s", objectKey)
		return "", err
	}

	// Verify the generated URL isn't redirecting to an unexpected domain
	parsedURL, err := url.Parse(presignedURL)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to parse presigned URL: %s", presignedURL)
		return "", err
	}

	// Ensure the URL is pointing to the configured S3 endpoint or its proxy
	s3Hostname := strings.Split(uc.s3Repository.GetEndpoint(), ":")[0] // Remove port if present
	allowedHosts := []string{s3Hostname}

	// Check if URL host matches any allowed host
	validHost := false
	for _, host := range allowedHosts {
		if parsedURL.Hostname() == host || strings.HasSuffix(parsedURL.Hostname(), "."+host) {
			validHost = true
			break
		}
	}

	if !validHost {
		log.Error().Msgf("URL host not allowed: %s", parsedURL.Hostname())
		return "", ErrInvalidHost
	}

	return presignedURL, nil
}