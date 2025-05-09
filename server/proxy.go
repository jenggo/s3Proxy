package server

import (
	"net/url"
	"s3proxy/pkg"
	"s3proxy/types"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog/log"
)

func proxy(ctx fiber.Ctx) error {
	// Get the encoded object path from the URL
	encodedPath := ctx.Params("*")

	// Basic security check: reject paths with suspicious patterns
	if strings.Contains(encodedPath, "..") || strings.Contains(encodedPath, "//") {
		log.Warn().Msgf("Rejected suspicious path: %s", encodedPath)
		return ctx.SendStatus(400)
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
		return ctx.SendStatus(400)
	}

	if objectPath == "" || encodedPath == "" {
		return ctx.SendStatus(404)
	}

	log.Debug().Msgf("request for %s (decoded from %s)", objectPath, encodedPath)

	// Use global S3 client
	// Verify object exists first
	objectKey := pkg.Client.FindObject(ctx.Context(), objectPath)
	if objectKey == "" {
		log.Warn().Msgf("object not found: %s", objectPath)
		return ctx.SendStatus(404)
	}

	// Generate presigned URL for the verified object
	presignedURL, err := pkg.Client.PresignedUrl(ctx.Context(), objectKey)
	if err != nil {
		log.Error().Err(err).Send()
		return ctx.SendStatus(404)
	}

	// Verify the generated URL isn't redirecting to an unexpected domain
	parsedURL, err := url.Parse(presignedURL)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to parse presigned URL: %s", presignedURL)
		return ctx.SendStatus(500)
	}

	// Ensure the URL is pointing to the configured S3 endpoint or its proxy
	s3Hostname := strings.Split(types.Config.S3.Endpoint, ":")[0] // Remove port if present
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
		return ctx.SendStatus(403)
	}

	// Use 302 (Found) status code for temporary redirects
	return ctx.Redirect().Status(302).To(presignedURL)
}
