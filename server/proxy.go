package server

import (
	"io"
	"net/url"
	"s3proxy/pkg"
	"s3proxy/types"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/ilyakaznacheev/cleanenv"
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

	// When bucket is not defined in config, then assume bucket is in url
	if types.IsEmptyBucket {
		parts := strings.SplitN(objectPath, "/", 2)

		types.Config.S3.Bucket = parts[0]
		if err := cleanenv.UpdateEnv(&types.Config); err != nil {
			log.Warn().Err(err).Msg("failed to update environment variables")
		}

		objectPath = ""
		if len(parts) == 2 {
			objectPath = parts[1]
		}

		log.Debug().Str("bucket", types.Config.S3.Bucket).Str("object", objectPath).Msg("bucket set from url")
	}

	// Use global S3 client
	// Verify object exists first
	objectKey := pkg.Client.FindObject(ctx.Context(), objectPath)
	if objectKey == "" {
		log.Warn().Msgf("object not found: %s", objectPath)
		return ctx.SendStatus(404)
	}

	// Get object info first to set headers
	log.Debug().Msgf("getting object info for: %s", objectKey)
	objectInfo, err := pkg.Client.StatObject(ctx.Context(), objectKey)
	if err != nil {
		log.Error().Err(err).Msgf("failed to get object info for: %s", objectKey)
		return ctx.SendStatus(404)
	}

	// Get object from S3 and stream it
	log.Debug().Msgf("getting object stream for: %s (size: %d)", objectKey, objectInfo.Size)
	object, err := pkg.Client.GetObject(ctx.Context(), objectKey)
	if err != nil {
		log.Error().Err(err).Msgf("failed to get object from S3: %s", objectKey)
		return ctx.SendStatus(404)
	}
	defer object.Close()

	// Set status and headers
	log.Debug().Msgf("streaming object: %s (type: %s, size: %d)", objectKey, objectInfo.ContentType, objectInfo.Size)
	ctx.Status(200)

	// Set content type - if empty, detect from file extension
	contentType := objectInfo.ContentType
	if contentType == "" || contentType == "application/octet-stream" {
		// Try to detect content type from object key
		lowerKey := strings.ToLower(objectKey)
		switch {
		case strings.HasSuffix(lowerKey, ".jpg"), strings.HasSuffix(lowerKey, ".jpeg"):
			contentType = "image/jpeg"
		case strings.HasSuffix(lowerKey, ".png"):
			contentType = "image/png"
		case strings.HasSuffix(lowerKey, ".gif"):
			contentType = "image/gif"
		case strings.HasSuffix(lowerKey, ".webp"):
			contentType = "image/webp"
		case strings.HasSuffix(lowerKey, ".svg"):
			contentType = "image/svg+xml"
		case strings.HasSuffix(lowerKey, ".pdf"):
			contentType = "application/pdf"
		}
	}

	// Extract filename from object key for Content-Disposition
	filename := objectKey
	if lastSlash := strings.LastIndex(objectKey, "/"); lastSlash >= 0 {
		filename = objectKey[lastSlash+1:]
	}

	ctx.Set(fiber.HeaderContentType, contentType)
	ctx.Set(fiber.HeaderContentDisposition, `inline; filename="`+filename+`"`)
	ctx.Set(fiber.HeaderContentLength, strconv.FormatInt(objectInfo.Size, 10))

	// Copy the object stream to the response
	_, err = io.Copy(ctx.Response().BodyWriter(), object)
	if err != nil {
		log.Error().Err(err).Msgf("failed to stream object: %s", objectKey)
		return err
	}

	return nil
}
