package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"jcp-gestioninmobiliaria/internal/auth"
	"jcp-gestioninmobiliaria/internal/config"
	apiHandlers "jcp-gestioninmobiliaria/internal/handlers/api"
	"jcp-gestioninmobiliaria/internal/handlers/admin"
	"jcp-gestioninmobiliaria/internal/handlers/fragments"
	"jcp-gestioninmobiliaria/internal/handlers/web"
	"jcp-gestioninmobiliaria/internal/handlers/ws"
	"jcp-gestioninmobiliaria/internal/middleware"
	"jcp-gestioninmobiliaria/internal/realtime"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	gows "github.com/gofiber/websocket/v2"
	"github.com/pocketbase/pocketbase"
)

func main() {
	cfg := config.Load()
	config.ValidateRequired(cfg)

	pb := pocketbase.New()
	auth.RegisterPBHooks(pb)
	realtime.RegisterPBHooks(pb)

	go func() {
		if err := pb.Start(); err != nil {
			log.Fatalf("PocketBase failed: %v", err)
		}
	}()

	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			log.Printf("ERROR [%d] %s %s: %v", code, c.Method(), c.Path(), err)
			if c.Get("HX-Request") == "true" {
				return c.Status(code).SendString(`<div class="toast toast-error">Error interno</div>`)
			}
			return c.Status(code).SendString("Error interno del servidor")
		},
		BodyLimit: 50 * 1024 * 1024,
	})

	// ── GLOBAL MIDDLEWARE ──
	app.Use(recover.New())
	app.Use(middleware.SecurityHeaders(cfg))
	app.Use(middleware.GlobalRateLimiter())

	if cfg.IsProd() {
		app.Use(logger.New(logger.Config{
			Format:     `{"time":"${time}","status":${status},"method":"${method}","path":"${path}","latency":"${latency}","ip":"${ip}"}` + "\n",
			TimeFormat: time.RFC3339,
		}))
	} else {
		app.Use(logger.New(logger.Config{
			Format:     "[${time}] ${status} ${method} ${path} (${latency})\n",
			TimeFormat: "15:04:05",
		}))
	}

	app.Use(cors.New(cors.Config{
		AllowOrigins: cfg.CORSOrigins,
		AllowHeaders: "Origin, Content-Type, Accept, Authorization, HX-Request, HX-Trigger",
	}))

	// ── HEALTH CHECK (público, sin auth) ──
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status": "ok",
			"env":    cfg.Env,
		})
	})

	app.Static("/static", "./web/static", fiber.Static{
		Compress:      true,
		CacheDuration: cfg.StaticCacheDuration,
	})

	hub := realtime.NewHub()
	go hub.Run()
	realtime.SetHubInstance(hub)

	// ── PUBLIC WEB ──
	app.Get("/", web.PageHandler(cfg, "propiedades"))
	app.Get("/propiedades.html", web.PageHandler(cfg, "propiedades"))
	app.Get("/noticias.html", web.PageHandler(cfg, "noticias"))

	// ── HTMX FRAGMENTS ──
	frag := app.Group("/fragments", middleware.FragmentsRateLimiter())
	frag.Get("/hero", fragments.HeroCarousel(cfg, pb))
	frag.Get("/eventos", fragments.Eventos(cfg, pb))
	frag.Get("/noticias", fragments.Noticias(cfg, pb))
	frag.Get("/comunicados", fragments.Comunicados(cfg, pb))
	frag.Get("/blog", fragments.Blog(cfg, pb))
	frag.Get("/noticias-page", fragments.NoticiasPage(cfg, pb))
	frag.Get("/propiedades-destacadas", fragments.PropiedadesDestacadas(cfg, pb))
	frag.Get("/propiedades-page", fragments.PropiedadesPage(cfg, pb))

	app.Get("/noticias/:id", web.NoticiaHandler(cfg, pb))
	app.Get("/propiedades/:key", web.PropiedadHandler(cfg, pb))
	app.Get("/rss.xml", web.RSSFeed(cfg))

	// ── PUBLIC API ──
	api := app.Group("/api")
	api.Get("/devices/:code/playlist", apiHandlers.DevicePlaylist(cfg, pb))
	api.Get("/events/upcoming", apiHandlers.UpcomingEvents(cfg, pb))

	// ── DEVICE / WS ──
	app.Get("/display/:code", web.DeviceDisplay(cfg))
	app.Get("/totem/:code", web.TotemDisplay(cfg))
	app.Use("/ws", func(c *fiber.Ctx) error {
		if gows.IsWebSocketUpgrade(c) {
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})
	app.Get("/ws/device/:code", gows.New(ws.DeviceSocket(hub)))
	app.Get("/ws/web", gows.New(ws.WebSocket(hub)))

	// ── ADMIN ──
	app.Get("/admin/login", admin.LoginPage(cfg))
	app.Post("/admin/login", middleware.LoginRateLimiter(), admin.LoginSubmit(cfg))
	app.Post("/admin/logout", admin.Logout())

	adm := app.Group("/admin", middleware.AuthRequired(cfg))

	adm.Get("/", admin.Dashboard(cfg))
	adm.Get("/dashboard", admin.Dashboard(cfg))
	adm.Get("/dashboard/stats", admin.DashboardStats(cfg, pb))

	// Multimedia
	adm.Get("/multimedia", admin.MultimediaList(cfg, pb))
	adm.Get("/multimedia/new", admin.MultimediaForm(cfg))
	adm.Post("/multimedia", admin.MultimediaCreate(cfg, pb))
	adm.Get("/multimedia/:id/edit", admin.MultimediaEdit(cfg, pb))
	adm.Put("/multimedia/:id", admin.MultimediaUpdate(cfg, pb))
	adm.Delete("/multimedia/:id", admin.MultimediaDelete(cfg, pb))

	// Events
	adm.Get("/events", admin.EventsList(cfg, pb))
	adm.Get("/events/new", admin.EventForm(cfg))
	adm.Post("/events", admin.EventCreate(cfg, pb))
	adm.Get("/events/:id/edit", admin.EventEdit(cfg, pb))
	adm.Put("/events/:id", admin.EventUpdate(cfg, pb))
	adm.Delete("/events/:id", admin.EventDelete(cfg, pb))
	adm.Post("/events/:id/publish", admin.EventPublish(cfg, pb))

	// News
	adm.Get("/news", admin.NewsList(cfg, pb))
	adm.Get("/news/new", admin.NewsForm(cfg))
	adm.Post("/news", admin.NewsCreate(cfg, pb))
	adm.Get("/news/:id/edit", admin.NewsEdit(cfg, pb))
	adm.Put("/news/:id", admin.NewsUpdate(cfg, pb))
	adm.Delete("/news/:id", admin.NewsDelete(cfg, pb))

	// Playlists
	adm.Get("/playlists", admin.PlaylistList(cfg, pb))
	adm.Get("/playlists/new", admin.PlaylistForm(cfg, pb))
	adm.Post("/playlists", admin.PlaylistCreate(cfg, pb))
	adm.Get("/playlists/:id/edit", admin.PlaylistEdit(cfg, pb))
	adm.Put("/playlists/:id", admin.PlaylistUpdate(cfg, pb))
	adm.Delete("/playlists/:id", admin.PlaylistDelete(cfg, pb))
	adm.Post("/playlists/:id/items/reorder", admin.PlaylistReorder(cfg, pb))

	// Devices
	adm.Get("/devices", admin.DeviceList(cfg, pb))
	adm.Get("/devices/new", admin.DeviceForm(cfg, pb))
	adm.Post("/devices", admin.DeviceCreate(cfg, pb))
	adm.Get("/devices/:id/edit", admin.DeviceEdit(cfg, pb))
	adm.Put("/devices/:id", admin.DeviceUpdate(cfg, pb))
	adm.Delete("/devices/:id", admin.DeviceDelete(cfg, pb))
	adm.Post("/devices/:id/assign-playlist", admin.DeviceAssignPlaylist(cfg, pb))

	// Users
	adm.Get("/users", middleware.RoleRequired("superadmin", "director"), admin.UserList(cfg))
	adm.Post("/users", middleware.RoleRequired("superadmin", "director"), admin.UserCreate(cfg))
	adm.Put("/users/:id", middleware.RoleRequired("superadmin", "director"), admin.UserUpdate(cfg))
	adm.Delete("/users/:id", middleware.RoleRequired("superadmin"), admin.UserDelete(cfg))

	adm.Get("/whatsapp-logs", admin.WhatsAppLogs(cfg))

	// Propiedades
	adm.Get("/propiedades", admin.PropiedadesList(cfg, pb))
	adm.Get("/propiedades/new", admin.PropiedadForm(cfg))
	adm.Post("/propiedades", admin.PropiedadCreate(cfg, pb))
	adm.Get("/propiedades/:id/edit", admin.PropiedadEdit(cfg, pb))
	adm.Put("/propiedades/:id", admin.PropiedadUpdate(cfg, pb))
	adm.Delete("/propiedades/:id", admin.PropiedadDelete(cfg, pb))
	adm.Post("/propiedades/:id/publish", admin.PropiedadToggleStatus(cfg, pb))

	app.Post("/webhook/whatsapp", web.WhatsAppWebhook(cfg))

	port := cfg.Port

	log.Printf("🏢 JCP Gestión Inmobiliaria en http://localhost:%s", port)
	log.Printf("📊 Dashboard: http://localhost:%s/admin", port)
	log.Printf("🔧 PocketBase Admin: http://localhost:8090/_/")

	// ── GRACEFUL SHUTDOWN ──
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		if err := app.Listen(":" + port); err != nil {
			log.Printf("Servidor detenido: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("Apagando servidor...")
	if err := app.ShutdownWithTimeout(10 * time.Second); err != nil {
		log.Printf("Error en shutdown: %v", err)
	}
	log.Println("Servidor detenido correctamente.")
}
