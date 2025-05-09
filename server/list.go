package server

import (
	"html"
	"s3proxy/pkg"
	"s3proxy/types"
	"sort"
	"strings"

	"github.com/gofiber/fiber/v3"
)

func list(ctx fiber.Ctx) error {
	// Use global S3 client
	list := pkg.Client.List(ctx.Context(), ctx.BaseURL())

	// Check Accept header to determine whether to return HTML or JSON
	if strings.Contains(ctx.Get("Accept"), "text/html") {
		return renderHTMLList(ctx, list) // Render HTML view
	}

	// Default to JSON response
	return ctx.JSON(types.Response{
		Error: false,
		List:  list,
	})
}

func renderHTMLList(ctx fiber.Ctx, files []types.List) error {
	// Create a map to organize files by directory
	organizedFiles := make(map[string][]types.FileInfo)

	for _, file := range files {
		// Determine the directory
		dir := "Root"
		lastSlashIndex := strings.LastIndex(file.Name, "/")
		if lastSlashIndex >= 0 {
			dir = file.Name[:lastSlashIndex]
		}

		// Extract just the filename from the path
		filename := file.Name
		if lastSlashIndex >= 0 {
			filename = file.Name[lastSlashIndex+1:]
		}

		// Create FileInfo with display name
		fileInfo := types.FileInfo{
			List:        file,
			DisplayName: html.EscapeString(filename),
		}

		// Add file to its directory group
		organizedFiles[dir] = append(organizedFiles[dir], fileInfo)
	}

	// Sort the directories for consistent display
	var dirs []string
	for dir := range organizedFiles {
		dirs = append(dirs, dir)
	}
	sort.Strings(dirs)

	// Create Directory objects for template
	var directories []types.Directory
	for _, dir := range dirs {
		files := organizedFiles[dir]

		// Sort files within this directory
		sort.Slice(files, func(i, j int) bool {
			return strings.ToLower(files[i].Name) < strings.ToLower(files[j].Name)
		})

		// Format directory name for display
		dirName := dir
		if dir == "Root" {
			dirName = "Root Directory"
		} else {
			dirName = html.EscapeString(dirName)
		}

		directories = append(directories, types.Directory{
			Name:      dirName,
			FileCount: len(files),
			Files:     files,
		})
	}

	// Render template with our data
	return ctx.Render("list", fiber.Map{
		"Directories": directories,
	})
}
