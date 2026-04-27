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
	"jcp-gestioninmobiliaria/internal/middleware"
	"jcp-gestioninmobiliaria/internal/realtime"
	"jcp-gestioninmobiliaria/internal/routes"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/pocketbase/pocketbase"
)

func main() {
	cfg := config.Load()
	config.ValidateRequired(cfg)

	pb := pocketbase.New()
	auth.RegisterPBHooks(pb, cfg)
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

	// ── HEALTH CHECK ──
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok", "env": cfg.Env})
	})

	app.Static("/static", "./web/static", fiber.Static{
		Compress:      true,
		CacheDuration: cfg.StaticCacheDuration,
	})

	// ── REALTIME HUB ──
	hub := realtime.NewHub()
	go hub.Run()
	realtime.SetHubInstance(hub)

	// ── ROUTES ──
	routes.RegisterPublic(app, cfg, pb, hub)
	routes.RegisterFragments(app, cfg, pb)
	routes.RegisterAdmin(app, cfg, pb)

	// ── GRACEFUL SHUTDOWN ──
	port := cfg.Port
	log.Printf("🏢 JCP Gestión Inmobiliaria en http://localhost:%s", port)
	log.Printf("📊 Dashboard: http://localhost:%s/admin", port)
	log.Printf("🔧 PocketBase Admin: http://localhost:8090/_/")

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
