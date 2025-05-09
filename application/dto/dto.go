package dto

// ObjectDTO represents data for a single S3 object
type ObjectDTO struct {
	// Name is the object's key or path in S3
	Name string `json:"name"`

	// Url is the path or presigned URL to access the object
	Url string `json:"url"`

	// DisplayName is a user-friendly name for display (usually the filename)
	DisplayName string `json:"display_name,omitempty"`
}

// ListResponseDTO represents the response for a list objects operation
type ListResponseDTO struct {
	Error   bool        `json:"error"`
	Message string      `json:"message,omitempty"`
	Objects []ObjectDTO `json:"list,omitempty"`
}

// ErrorResponseDTO represents an error response
type ErrorResponseDTO struct {
	Error   bool   `json:"error"`
	Message string `json:"message"`
}

// DirectoryDTO represents a directory with its files for the view
type DirectoryDTO struct {
	// Name is the display name of the directory
	Name string

	// FileCount is the number of files in this directory
	FileCount int

	// Files is the list of files in this directory
	Files []ObjectDTO
}

// ListViewModelDTO is the view model for the list HTML template
type ListViewModelDTO struct {
	Directories []DirectoryDTO
}