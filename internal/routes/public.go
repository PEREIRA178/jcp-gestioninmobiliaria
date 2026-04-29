package routes

import (
	"jcp-gestioninmobiliaria/internal/config"
	"jcp-gestioninmobiliaria/internal/handlers/web"
	"jcp-gestioninmobiliaria/internal/handlers/ws"
	"jcp-gestioninmobiliaria/internal/middleware"
	"jcp-gestioninmobiliaria/internal/realtime"

	"github.com/gofiber/fiber/v2"
	gows "github.com/gofiber/websocket/v2"
	"github.com/pocketbase/pocketbase"
)

func RegisterPublic(app *fiber.App, cfg *config.Config, pb *pocketbase.PocketBase, hub *realtime.Hub) {
	// Public marketing pages
	app.Get("/", web.LandingHandler(cfg, pb))
	app.Get("/noticias.html", web.PageHandler(cfg, "noticias"))
	app.Get("/noticias/:id", web.NoticiaHandler(cfg, pb))
	app.Get("/rss.xml", web.RSSFeed(cfg))
	app.Post("/webhook/whatsapp", web.WhatsAppWebhook(cfg))

	// Legacy redirect
	app.Get("/propiedades.html", func(c *fiber.Ctx) error {
		return c.Redirect("/propiedades", fiber.StatusMovedPermanently)
	})

	// Auth routes
	app.Get("/login", web.LoginPageHandler(cfg))
	app.Post("/login", middleware.LoginRateLimiter(), web.VisitorLoginSubmit(cfg, pb))
	app.Get("/register", web.RegisterPageHandler(cfg))
	app.Post("/register", middleware.LoginRateLimiter(), web.RegisterSubmit(cfg, pb))
	app.Get("/auth/google", web.GoogleLogin(cfg))
	app.Get("/auth/google/callback", web.GoogleCallback(cfg, pb))
	app.Get("/auth/logout", web.VisitorLogout(cfg))

	// Protected visitor routes
	visitor := app.Group("", middleware.VisitorAuthRequired(cfg))
	visitor.Get("/propiedades", web.PropiedadesHandler(cfg))
	visitor.Get("/propiedades/:key", web.PropiedadHandler(cfg, pb))
	visitor.Get("/guardadas", web.GuardasHandler(cfg))

	// WebSocket
	app.Use("/ws", func(c *fiber.Ctx) error {
		if gows.IsWebSocketUpgrade(c) {
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})
	app.Get("/ws/web", gows.New(ws.WebSocket(hub)))
}
