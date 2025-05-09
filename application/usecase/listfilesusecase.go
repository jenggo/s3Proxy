package usecase

import (
	"context"
	"html"
	"net/url"
	"s3proxy/application/dto"
	"s3proxy/domain/repository"
)

// ListFilesUseCase implements the use case for listing files
type ListFilesUseCase struct {
	s3Repository repository.S3Repository
}

// NewListFilesUseCase creates a new instance of ListFilesUseCase
func NewListFilesUseCase(s3Repository repository.S3Repository) *ListFilesUseCase {
	return &ListFilesUseCase{
		s3Repository: s3Repository,
	}
}

// Execute performs the list files operation
func (uc *ListFilesUseCase) Execute(ctx context.Context, baseURL string) (*dto.ListResponseDTO, error) {
	// Get objects from repository
	collection, err := uc.s3Repository.ListObjects(ctx)
	if err != nil {
		return &dto.ListResponseDTO{
			Error:   true,
			Message: "Failed to list objects: " + err.Error(),
		}, err
	}

	// Filter out folders and suspicious objects
	collection = collection.FilterFolders().FilterSuspicious()

	// Convert to DTOs
	var objects []dto.ObjectDTO
	for _, obj := range collection.Objects {
		objects = append(objects, dto.ObjectDTO{
			Name: obj.Key,
			Url:  baseURL + "/" + url.QueryEscape(obj.Key),
		})
	}

	return &dto.ListResponseDTO{
		Error:   false,
		Objects: objects,
	}, nil
}

// ExecuteForView prepares data for HTML view
func (uc *ListFilesUseCase) ExecuteForView(ctx context.Context, baseURL string) (*dto.ListViewModelDTO, error) {
	// Get objects from repository
	collection, err := uc.s3Repository.ListObjects(ctx)
	if err != nil {
		return nil, err
	}

	// Filter out folders and suspicious objects
	collection = collection.FilterFolders().FilterSuspicious()

	// Organize by directory
	organizedFiles := collection.OrganizeByDirectory()
	dirNames := collection.DirectoryNames()

	// Create view model
	viewModel := &dto.ListViewModelDTO{
		Directories: make([]dto.DirectoryDTO, 0, len(dirNames)),
	}

	// Build directory DTOs
	for _, dirName := range dirNames {
		files := organizedFiles[dirName]
		
		// Create file DTOs for this directory
		fileDTOs := make([]dto.ObjectDTO, 0, len(files))
		for _, file := range files {
			// Extract filename for display
			filename := file.GetFilename()
			
			fileDTOs = append(fileDTOs, dto.ObjectDTO{
				Name:        file.Key,
				Url:         baseURL + "/" + url.QueryEscape(file.Key),
				DisplayName: html.EscapeString(filename),
			})
		}

		// Format directory name for display
		displayDirName := dirName
		if dirName == "Root" {
			displayDirName = "Root Directory"
		} else {
			displayDirName = html.EscapeString(displayDirName)
		}

		// Add directory to view model
		viewModel.Directories = append(viewModel.Directories, dto.DirectoryDTO{
			Name:      displayDirName,
			FileCount: len(files),
			Files:     fileDTOs,
		})
	}

	return viewModel, nil
}