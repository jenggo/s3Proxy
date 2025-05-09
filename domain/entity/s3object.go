package entity

import (
	"net/url"
	"time"
)

// S3Object represents a file stored in S3
type S3Object struct {
	// Key is the object's full path in S3
	Key string

	// Size is the object's size in bytes
	Size int64

	// LastModified is when the object was last changed
	LastModified time.Time

	// ETag is the object's entity tag
	ETag string

	// ContentType is the MIME type of the object
	ContentType string
}

// IsFolder determines if this object represents a folder
func (o *S3Object) IsFolder() bool {
	// S3 represents folders as objects ending with '/'
	// or as zero-byte objects without file extensions
	if len(o.Key) == 0 {
		return false
	}

	// If it ends with slash, it's definitely a folder
	if o.Key[len(o.Key)-1] == '/' {
		return true
	}

	// If it's a zero-size object without an extension, it's likely a folder marker
	if o.Size == 0 {
		// Extract filename portion (after last slash)
		filename := o.Key
		lastSlashIndex := -1
		for i := len(o.Key) - 1; i >= 0; i-- {
			if o.Key[i] == '/' {
				lastSlashIndex = i
				break
			}
		}

		if lastSlashIndex >= 0 {
			filename = o.Key[lastSlashIndex+1:]
		}

		// If filename has no extension and is empty or doesn't contain a dot,
		// it's likely a folder marker
		return filename == "" || !containsRune(filename, '.')
	}

	return false
}

// GetFilename returns just the filename portion of the object key
func (o *S3Object) GetFilename() string {
	if o.Key == "" {
		return ""
	}

	// Find the last slash
	lastSlashIndex := -1
	for i := len(o.Key) - 1; i >= 0; i-- {
		if o.Key[i] == '/' {
			lastSlashIndex = i
			break
		}
	}

	// If no slash found, the entire key is the filename
	if lastSlashIndex == -1 {
		return o.Key
	}

	// Return everything after the last slash
	return o.Key[lastSlashIndex+1:]
}

// GetDisplayURL returns a URL-encoded version for display and linking
func (o *S3Object) GetDisplayURL(baseURL string) string {
	if o.Key == "" {
		return ""
	}
	
	return baseURL + "/" + url.QueryEscape(o.Key)
}

// IsSuspicious checks if the object key contains suspicious patterns
func (o *S3Object) IsSuspicious() bool {
	// Check for path traversal attempts
	return containsSubstring(o.Key, "..") || 
	       containsSubstring(o.Key, "//")
}

// Helper functions
func containsRune(s string, r rune) bool {
	for _, c := range s {
		if c == r {
			return true
		}
	}
	return false
}

func containsSubstring(s, substr string) bool {
	if len(s) < len(substr) {
		return false
	}
	
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}