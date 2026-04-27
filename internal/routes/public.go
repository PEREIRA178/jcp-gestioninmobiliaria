package routes

import (
	"jcp-gestioninmobiliaria/internal/config"
	"jcp-gestioninmobiliaria/internal/handlers/web"
	"jcp-gestioninmobiliaria/internal/handlers/ws"
	"jcp-gestioninmobiliaria/internal/realtime"

	"github.com/gofiber/fiber/v2"
	gows "github.com/gofiber/websocket/v2"
	"github.com/pocketbase/pocketbase"
)

// RegisterPublic monta las rutas públicas del sitio y el WebSocket de realtime.
func RegisterPublic(app *fiber.App, cfg *config.Config, pb *pocketbase.PocketBase, hub *realtime.Hub) {
	app.Get("/", web.PageHandler(cfg, "propiedades"))
	app.Get("/propiedades.html", web.PageHandler(cfg, "propiedades"))
	app.Get("/noticias.html", web.PageHandler(cfg, "noticias"))
	app.Get("/noticias/:id", web.NoticiaHandler(cfg, pb))
	app.Get("/propiedades/:key", web.PropiedadHandler(cfg, pb))
	app.Get("/rss.xml", web.RSSFeed(cfg))
	app.Post("/webhook/whatsapp", web.WhatsAppWebhook(cfg))

	// WebSocket para updates en tiempo real (propiedades y noticias)
	app.Use("/ws", func(c *fiber.Ctx) error {
		if gows.IsWebSocketUpgrade(c) {
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})
	app.Get("/ws/web", gows.New(ws.WebSocket(hub)))
}
