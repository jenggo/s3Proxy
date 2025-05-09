package server

import (
	"errors"
	"s3proxy/types"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/keyauth"
	"github.com/rs/zerolog/log"
)

func errHandler(c fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	var e *fiber.Error
	if errors.As(err, &e) {
		code = e.Code
	}

	ua := c.Get(fiber.HeaderUserAgent)
	ip := c.IP()
	method := c.Method()
	path := c.Path()

	if ua != "" && ip != "" && code != fiber.StatusNotFound && code != fiber.StatusMethodNotAllowed && err != keyauth.ErrMissingOrMalformedAPIKey {
		log.Error().Str("UserAgent", ua).Str("IP", ip).Str("Method", method).Str("Path", path).Err(err).Send()
	}

	return c.Status(code).JSON(types.Response{
		Error:   true,
		Message: err.Error(),
	})
}
