package http

import (
	"s3proxy/domain/repository"
	"s3proxy/infrastructure/config"
	"s3proxy/interfaces/http/handler"
	"time"

	"github.com/goccy/go-json"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/earlydata"
	"github.com/gofiber/fiber/v3/middleware/favicon"
	"github.com/gofiber/fiber/v3/middleware/helmet"
	"github.com/gofiber/fiber/v3/middleware/pprof"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/template/html/v2"
	"github.com/rs/zerolog/log"
)

// Server represents the HTTP server
type Server struct {
	app     *fiber.App
	handler *handler.Handler
}

// NewServer creates a new HTTP server
func NewServer(s3Repository repository.S3Repository) *Server {
	h := handler.NewHandler(s3Repository)
	
	appCfg := fiber.Config{
		AppName:      config.AppName,
		JSONEncoder:  json.Marshal,
		JSONDecoder:  json.Unmarshal,
		ErrorHandler: handler.ErrorHandler,
		ReadTimeout:  10 * time.Second,
		GETOnly:      true,
		Views:        html.New("./views", ".html"),
	}

	// Configure proxy header based on environment
	if config.IsCloudflareEnabled() {
		appCfg.ProxyHeader = "Cf-Connecting-Ip"
	} else {
		appCfg.ProxyHeader = "X-Real-Ip"
	}

	app := fiber.New(appCfg)

	// Configure pprof if enabled
	if pprofPath := config.GetPPROFPath(); pprofPath != "" {
		log.Log().Msgf("» pprof enabled: %s", pprofPath)
		app.Use(pprof.New(pprof.Config{Prefix: pprofPath}))
	}

	// Register middleware
	app.Use(cors.New())
	app.Use(favicon.New())
	app.Use(helmet.New())
	app.Use(earlydata.New())
	app.Use(recover.New(recover.Config{EnableStackTrace: true}))

	// Create server
	server := &Server{
		app:     app,
		handler: h,
	}
	
	// Register routes
	server.registerRoutes()
	
	return server
}

// registerRoutes sets up all HTTP routes
func (s *Server) registerRoutes() {
	// Register list endpoint if enabled
	if config.IsListEnabled() {
		s.app.Get("/list", s.handler.ListFiles)
	}

	// Register proxy endpoint
	s.app.Get("/*", s.handler.ProxyFile)
}

// Start starts the HTTP server
func (s *Server) Start() error {
	listenAddr := config.GetListenAddress()
	
	log.Log().Msgf("» %s %s listen: %s", config.AppName, config.AppVersion, listenAddr)
	
	return s.app.Listen(listenAddr, fiber.ListenConfig{
		DisableStartupMessage: true,
	})
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown() error {
	return s.app.Shutdown()
}

// ShutdownWithTimeout gracefully shuts down the server with a timeout
func (s *Server) ShutdownWithTimeout(timeout time.Duration) error {
	return s.app.ShutdownWithTimeout(timeout)
}