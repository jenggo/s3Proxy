package entity

import (
	"sort"
	"strings"
)

// ObjectCollection represents a collection of S3 objects
type ObjectCollection struct {
	Objects []*S3Object
}

// NewObjectCollection creates a new empty collection
func NewObjectCollection() *ObjectCollection {
	return &ObjectCollection{
		Objects: make([]*S3Object, 0),
	}
}

// Add adds an object to the collection
func (c *ObjectCollection) Add(obj *S3Object) {
	c.Objects = append(c.Objects, obj)
}

// FilterFolders removes folder objects from the collection
func (c *ObjectCollection) FilterFolders() *ObjectCollection {
	result := NewObjectCollection()
	
	for _, obj := range c.Objects {
		if !obj.IsFolder() {
			result.Add(obj)
		}
	}
	
	return result
}

// FilterSuspicious removes objects with suspicious paths
func (c *ObjectCollection) FilterSuspicious() *ObjectCollection {
	result := NewObjectCollection()
	
	for _, obj := range c.Objects {
		if !obj.IsSuspicious() {
			result.Add(obj)
		}
	}
	
	return result
}

// FindByPath finds an object by exact path
func (c *ObjectCollection) FindByPath(path string) *S3Object {
	for _, obj := range c.Objects {
		if obj.Key == path {
			return obj
		}
	}
	return nil
}

// FindByCaseInsensitivePath finds an object by case-insensitive path
func (c *ObjectCollection) FindByCaseInsensitivePath(path string) *S3Object {
	for _, obj := range c.Objects {
		if strings.EqualFold(obj.Key, path) {
			return obj
		}
	}
	return nil
}

// FindByFilenameFuzzy finds an object by fuzzy filename matching
// Returns the best match and its score (0-100, with 100 being perfect match)
func (c *ObjectCollection) FindByFilenameFuzzy(path string) (*S3Object, int) {
	// Split requested path into components
	requestParts := strings.Split(path, "/")
	if len(requestParts) < 2 {
		return nil, 0
	}

	// Extract filename from request (last component)
	requestedFilename := requestParts[len(requestParts)-1]

	// Find best match
	var bestMatch *S3Object
	var bestMatchScore int

	for _, obj := range c.Objects {
		// Split S3 object key into components
		objectParts := strings.Split(obj.Key, "/")
		if len(objectParts) < 2 {
			continue
		}

		// Get filename from object (last component)
		objectFilename := objectParts[len(objectParts)-1]

		// If filenames match exactly (case-insensitive)
		if strings.EqualFold(objectFilename, requestedFilename) {
			// Folder path similarity check
			score := 0

			// Compare folder components
			reqFolder := strings.ToLower(strings.Join(requestParts[:len(requestParts)-1], "/"))
			objFolder := strings.ToLower(strings.Join(objectParts[:len(objectParts)-1], "/"))

			// Calculate similarity
			switch {
			case reqFolder == objFolder:
				score = 100
			case strings.Contains(objFolder, reqFolder) || strings.Contains(reqFolder, objFolder):
				score = 50
			default:
				similarChars := 0
				maxLen := min(len(reqFolder), len(objFolder))

				for i := 0; i < maxLen; i++ {
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
				bestMatch = obj
			}
		}
	}

	return bestMatch, bestMatchScore
}

// OrganizeByDirectory organizes objects into a map by directory path
func (c *ObjectCollection) OrganizeByDirectory() map[string][]*S3Object {
	organizedFiles := make(map[string][]*S3Object)

	for _, obj := range c.Objects {
		// Determine the directory
		dir := "Root"
		lastSlashIndex := strings.LastIndex(obj.Key, "/")
		if lastSlashIndex >= 0 {
			dir = obj.Key[:lastSlashIndex]
		}

		// Add file to its directory group
		organizedFiles[dir] = append(organizedFiles[dir], obj)
	}

	// Sort files within each directory
	for dir := range organizedFiles {
		sort.Slice(organizedFiles[dir], func(i, j int) bool {
			return strings.ToLower(organizedFiles[dir][i].Key) < strings.ToLower(organizedFiles[dir][j].Key)
		})
	}

	return organizedFiles
}

// DirectoryNames returns a sorted list of directory names
func (c *ObjectCollection) DirectoryNames() []string {
	// Get directories
	dirMap := make(map[string]bool)
	
	for _, obj := range c.Objects {
		dir := "Root"
		lastSlashIndex := strings.LastIndex(obj.Key, "/")
		if lastSlashIndex >= 0 {
			dir = obj.Key[:lastSlashIndex]
		}
		dirMap[dir] = true
	}
	
	// Convert to sorted slice
	var dirs []string
	for dir := range dirMap {
		dirs = append(dirs, dir)
	}
	sort.Strings(dirs)
	
	return dirs
}

// Size returns the number of objects in the collection
func (c *ObjectCollection) Size() int {
	return len(c.Objects)
}