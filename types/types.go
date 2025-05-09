package types

type Response struct {
	Error   bool   `json:"error"`
	Message string `json:"message,omitempty"`
	List    []List `json:"list,omitempty"`
}

type List struct {
	Name string `json:"name"`
	Url  string `json:"url"`
}

// Directory represents a directory in the file listing
type Directory struct {
	Name      string
	FileCount int
	Files     []FileInfo
}

// FileInfo extends the types.List with a display name for the template
type FileInfo struct {
	List
	DisplayName string
}
