package server

import (
	"s3proxy/types"
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

func Start() (app *fiber.App, err error) {
	appCfg := fiber.Config{
		AppName:      types.AppName,
		JSONEncoder:  json.Marshal,
		JSONDecoder:  json.Unmarshal,
		ErrorHandler: errHandler,
		ProxyHeader:  "Cf-Connecting-Ip",
		ReadTimeout:  10 * time.Second,
		GETOnly:      true,
		Views:        html.New("./views", ".html"),
	}

	if !types.Config.App.Cloudflare {
		appCfg.ProxyHeader = "X-Real-Ip"
	}

	app = fiber.New(appCfg)

	if types.Config.App.PPROF != "" {
		log.Log().Msgf("» pprof enabled: %s", types.Config.App.PPROF)
		app.Use(pprof.New(pprof.Config{Prefix: types.Config.App.PPROF}))
	}

	app.Use(cors.New())
	app.Use(favicon.New())
	app.Use(helmet.New())
	app.Use(earlydata.New())
	app.Use(recover.New(recover.Config{EnableStackTrace: true}))

	if types.Config.App.EnableList {
		app.Get("/list", list)
	}

	app.Get("/*", proxy)

	go func() {
		log.Log().Msgf("» %s %s listen: %s", types.AppName, types.AppVersion, types.Config.App.Listen)

		if err := app.Listen(types.Config.App.Listen, fiber.ListenConfig{DisableStartupMessage: true}); err != nil {
			log.Error().Caller().Err(err).Send()
		}
	}()

	return
}
