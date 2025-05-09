package handler

import (
	"s3proxy/application/dto"
	"s3proxy/application/usecase"
	"s3proxy/domain/repository"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog/log"
)

// Handler contains all HTTP handlers
type Handler struct {
	listFilesUseCase *usecase.ListFilesUseCase
	proxyFileUseCase *usecase.ProxyFileUseCase
}

// NewHandler creates a new instance of Handler
func NewHandler(s3Repository repository.S3Repository) *Handler {
	return &Handler{
		listFilesUseCase: usecase.NewListFilesUseCase(s3Repository),
		proxyFileUseCase: usecase.NewProxyFileUseCase(s3Repository),
	}
}

// ListFiles handles the file listing endpoint
func (h *Handler) ListFiles(ctx fiber.Ctx) error {
	// Check Accept header to determine whether to return HTML or JSON
	if strings.Contains(ctx.Get("Accept"), "text/html") {
		return h.renderHTMLList(ctx)
	}

	// Default to JSON response
	response, err := h.listFilesUseCase.Execute(ctx.Context(), ctx.BaseURL())
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponseDTO{
			Error:   true,
			Message: "Failed to list files: " + err.Error(),
		})
	}

	return ctx.JSON(response)
}

// renderHTMLList renders the HTML list view
func (h *Handler) renderHTMLList(ctx fiber.Ctx) error {
	viewModel, err := h.listFilesUseCase.ExecuteForView(ctx.Context(), ctx.BaseURL())
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponseDTO{
			Error:   true,
			Message: "Failed to prepare list view: " + err.Error(),
		})
	}

	// Render template with data
	return ctx.Render("list", fiber.Map{
		"Directories": viewModel.Directories,
	})
}

// ProxyFile handles the file proxy endpoint
func (h *Handler) ProxyFile(ctx fiber.Ctx) error {
	// Get the encoded object path from the URL
	encodedPath := ctx.Params("*")

	// Execute use case to get presigned URL
	presignedURL, err := h.proxyFileUseCase.Execute(ctx.Context(), encodedPath)
	if err != nil {
		switch err {
		case usecase.ErrObjectNotFound:
			return ctx.SendStatus(fiber.StatusNotFound)
		case usecase.ErrSuspiciousPath:
			return ctx.SendStatus(fiber.StatusBadRequest)
		case usecase.ErrInvalidHost:
			return ctx.SendStatus(fiber.StatusForbidden)
		default:
			log.Error().Err(err).Msg("Error handling proxy request")
			return ctx.SendStatus(fiber.StatusInternalServerError)
		}
	}

	// Use 302 (Found) status code for temporary redirects
	return ctx.Redirect().Status(302).To(presignedURL)
}

// ErrorHandler is a global error handler
func ErrorHandler(ctx fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	if fiberErr, ok := err.(*fiber.Error); ok {
		code = fiberErr.Code
	}

	ua := ctx.Get(fiber.HeaderUserAgent)
	ip := ctx.IP()
	method := ctx.Method()
	path := ctx.Path()

	if ua != "" && ip != "" && code != fiber.StatusNotFound && code != fiber.StatusMethodNotAllowed {
		log.Error().
			Str("UserAgent", ua).
			Str("IP", ip).
			Str("Method", method).
			Str("Path", path).
			Err(err).
			Msg("HTTP error")
	}

	return ctx.Status(code).JSON(dto.ErrorResponseDTO{
		Error:   true,
		Message: err.Error(),
	})
}