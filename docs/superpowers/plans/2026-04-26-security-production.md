# Security & Production-Readiness Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add rate limiting, security headers, graceful shutdown, validated env vars, health check, and production-ready logging to the JCP Gestión Inmobiliaria Go+Fiber app.

**Architecture:** Enfoque 1 — Middleware centralizado. All new security concerns live in `internal/middleware/security.go`. Config gains `IsProd()` and `ValidateRequired()`. Main.go gains graceful shutdown and `/health`. No business logic changes.

**Tech Stack:** Go 1.23, Fiber v2, `fiber/v2/middleware/limiter` (already in go.mod), `os/signal` stdlib

---

## File Map

| File | Action | Responsibility |
|---|---|---|
| `internal/config/config.go` | Modify | Remove hardcoded secrets; add `IsProd()`, `ValidateRequired()` |
| `internal/middleware/security.go` | Create | Rate limiters (login, fragments, global) + security headers |
| `cmd/server/main.go` | Modify | Graceful shutdown, `/health`, logger mode, middleware wiring |
| `.env.example` | Create | Document all env vars |

---

## Task 1: Securizar `internal/config/config.go`

**Files:**
- Modify: `internal/config/config.go`

- [ ] **Step 1: Reemplazar el archivo completo con la versión segura**

```go
package config

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

// Config holds all application configuration
type Config struct {
	// Server
	Port                string
	Env                 string
	CORSOrigins         string
	StaticCacheDuration time.Duration

	// PocketBase
	PBUrl   string
	PBAdmin string
	PBPass  string

	// Admin credentials
	AdminEmail    string
	AdminPassword string

	// JWT
	JWTSecret     string
	JWTExpiration time.Duration

	// Cloudflare R2
	R2AccountID  string
	R2AccessKey  string
	R2SecretKey  string
	R2BucketName string
	R2Region     string
	R2PublicURL  string

	// WhatsApp (Twilio)
	TwilioAccountSID string
	TwilioAuthToken  string
	TwilioFromNumber string

	// Ollama
	OllamaURL   string
	OllamaModel string

	// Web
	BaseURL  string
	SiteName string
}

func Load() *Config {
	cfg := &Config{
		// Server
		Port:                getEnv("PORT", "3000"),
		Env:                 getEnv("ENV", "development"),
		CORSOrigins:         getEnv("CORS_ORIGINS", "*"),
		StaticCacheDuration: 24 * time.Hour,

		// PocketBase
		PBUrl:   getEnv("PB_URL", "http://127.0.0.1:8090"),
		PBAdmin: getEnv("PB_ADMIN_EMAIL", "admin@jcp.cl"),
		PBPass:  getEnv("PB_ADMIN_PASSWORD", ""),

		// Admin credentials — no hardcoded defaults for secrets
		AdminEmail:    getEnv("ADMIN_EMAIL", "admin@jcp-gestioninmobiliaria.cl"),
		AdminPassword: getEnv("ADMIN_PASSWORD", ""),

		// JWT — no hardcoded default
		JWTSecret:     getEnv("JWT_SECRET", ""),
		JWTExpiration: 72 * time.Hour,

		// Cloudflare R2
		R2AccountID:  getEnv("R2_ACCOUNT_ID", ""),
		R2AccessKey:  getEnv("R2_ACCESS_KEY_ID", ""),
		R2SecretKey:  getEnv("R2_SECRET_ACCESS_KEY", ""),
		R2BucketName: getEnv("R2_BUCKET_NAME", "jcp-media"),
		R2Region:     getEnv("R2_REGION", "auto"),
		R2PublicURL:  getEnv("R2_PUBLIC_URL", ""),

		// WhatsApp
		TwilioAccountSID: getEnv("TWILIO_ACCOUNT_SID", ""),
		TwilioAuthToken:  getEnv("TWILIO_AUTH_TOKEN", ""),
		TwilioFromNumber: getEnv("TWILIO_FROM_NUMBER", ""),

		// Ollama
		OllamaURL:   getEnv("OLLAMA_URL", "http://localhost:11434"),
		OllamaModel: getEnv("OLLAMA_MODEL", "llama3"),

		// Web
		BaseURL:  getEnv("BASE_URL", "http://localhost:3000"),
		SiteName: "JCP Gestión Inmobiliaria",
	}

	// In development only, provide safe placeholder defaults so the app
	// starts without any env file. These values must never reach production.
	if !cfg.IsProd() {
		if cfg.AdminPassword == "" {
			cfg.AdminPassword = "dev-only-change-me"
		}
		if cfg.JWTSecret == "" {
			cfg.JWTSecret = "dev-jwt-secret-change-me"
		}
	}

	return cfg
}

// IsProd returns true when running in the production environment.
func (c *Config) IsProd() bool {
	return c.Env == "production"
}

// ValidateRequired aborts startup if critical secrets are missing in production.
func ValidateRequired(cfg *Config) {
	if !cfg.IsProd() {
		return
	}
	var missing []string
	if cfg.JWTSecret == "" {
		missing = append(missing, "JWT_SECRET")
	}
	if cfg.AdminPassword == "" {
		missing = append(missing, "ADMIN_PASSWORD")
	}
	if len(missing) > 0 {
		log.Fatalf("FATAL: variables de entorno requeridas en producción no configuradas: %s\n"+
			"Configuralas con: fly secrets set %s=<valor>",
			strings.Join(missing, ", "),
			strings.Join(missing, "=<valor> "),
		)
	}
	if cfg.CORSOrigins == "*" {
		fmt.Println("ADVERTENCIA: CORS_ORIGINS es '*' en producción. Considerá restringirlo.")
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
```

- [ ] **Step 2: Verificar compilación**

```bash
cd /Users/matiaspereira/jcp-gestioninmobiliaria/jcp-gestioninmobiliaria
go build ./...
```

Expected: sin errores.

- [ ] **Step 3: Commit**

```bash
git add internal/config/config.go
git commit -m "security: remove hardcoded secrets, add IsProd() and ValidateRequired()"
```

---

## Task 2: Crear `internal/middleware/security.go`

**Files:**
- Create: `internal/middleware/security.go`

- [ ] **Step 1: Crear el archivo de middleware de seguridad**

```go
package middleware

import (
	"time"

	"jcp-gestioninmobiliaria/internal/config"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
)

// LoginRateLimiter allows 5 login attempts per IP per 15 minutes.
func LoginRateLimiter() fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        5,
		Expiration: 15 * time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			if c.Get("HX-Request") == "true" {
				return c.Status(fiber.StatusTooManyRequests).SendString(
					`<div class="toast toast-error">Demasiados intentos. Intentá de nuevo en 15 minutos.</div>`,
				)
			}
			return c.Status(fiber.StatusTooManyRequests).
				SendString("Demasiados intentos de login. Intentá de nuevo en 15 minutos.")
		},
	})
}

// FragmentsRateLimiter allows 60 fragment requests per IP per minute.
func FragmentsRateLimiter() fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        60,
		Expiration: 1 * time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).
				SendString("Límite de solicitudes alcanzado. Intentá en un momento.")
		},
	})
}

// GlobalRateLimiter allows 200 requests per IP per minute across all routes.
func GlobalRateLimiter() fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        200,
		Expiration: 1 * time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).
				SendString("Límite de solicitudes alcanzado.")
		},
	})
}

// SecurityHeaders adds standard security headers to every response.
// HSTS is only added in production (requires HTTPS).
func SecurityHeaders(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Set("X-Frame-Options", "DENY")
		c.Set("X-Content-Type-Options", "nosniff")
		c.Set("X-XSS-Protection", "1; mode=block")
		c.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		c.Set("Content-Security-Policy",
			"default-src 'self'; "+
				"script-src 'self' unpkg.com cdn.jsdelivr.net 'unsafe-inline'; "+
				"style-src 'self' 'unsafe-inline'; "+
				"img-src 'self' data: blob: *; "+
				"connect-src 'self'; "+
				"font-src 'self' data:",
		)
		if cfg.IsProd() {
			c.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		return c.Next()
	}
}
```

- [ ] **Step 2: Verificar compilación**

```bash
go build ./...
```

Expected: sin errores.

- [ ] **Step 3: Commit**

```bash
git add internal/middleware/security.go
git commit -m "security: add rate limiters and security headers middleware"
```

---

## Task 3: Actualizar `cmd/server/main.go`

**Files:**
- Modify: `cmd/server/main.go`

- [ ] **Step 1: Reemplazar el archivo completo con graceful shutdown, /health y middleware cableado**

```go
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
```

- [ ] **Step 2: Verificar compilación**

```bash
go build ./...
```

Expected: sin errores.

- [ ] **Step 3: Verificar que el health check funciona en local**

Arrancar el servidor (en otra terminal) y ejecutar:

```bash
curl -s http://localhost:3000/health
```

Expected:
```json
{"env":"development","status":"ok"}
```

- [ ] **Step 4: Verificar security headers**

```bash
curl -sI http://localhost:3000/health | grep -i -E "x-frame|x-content|referrer|permissions|content-security"
```

Expected: todos los headers presentes.

- [ ] **Step 5: Commit**

```bash
git add cmd/server/main.go
git commit -m "security: graceful shutdown, /health endpoint, security headers wiring, logger by env"
```

---

## Task 4: Crear `.env.example`

**Files:**
- Create: `.env.example`

- [ ] **Step 1: Crear el archivo**

```bash
# ============================================================
# JCP Gestión Inmobiliaria — Variables de entorno
# Copiá este archivo a .env y completá los valores
# NUNCA commitees el .env real al repositorio
# ============================================================

# ── SERVER ──────────────────────────────────────────────────
ENV=development                        # production | development
PORT=3000
BASE_URL=http://localhost:3000         # En prod: https://jcp-gestioninmobiliaria.fly.dev
CORS_ORIGINS=*                         # En prod: https://jcp-gestioninmobiliaria.fly.dev

# ── SECRETS (REQUERIDOS EN PRODUCCIÓN) ──────────────────────
# Generá el JWT_SECRET con: openssl rand -hex 32
JWT_SECRET=
ADMIN_PASSWORD=

# ── ADMIN ───────────────────────────────────────────────────
ADMIN_EMAIL=admin@jcp-gestioninmobiliaria.cl

# ── POCKETBASE ──────────────────────────────────────────────
PB_URL=http://127.0.0.1:8090
PB_ADMIN_EMAIL=admin@jcp.cl
PB_ADMIN_PASSWORD=

# ── CLOUDFLARE R2 (almacenamiento de imágenes) ───────────────
R2_ACCOUNT_ID=
R2_ACCESS_KEY_ID=
R2_SECRET_ACCESS_KEY=
R2_BUCKET_NAME=jcp-media
R2_REGION=auto
R2_PUBLIC_URL=

# ── WHATSAPP / TWILIO (opcional) ────────────────────────────
TWILIO_ACCOUNT_SID=
TWILIO_AUTH_TOKEN=
TWILIO_FROM_NUMBER=

# ── OLLAMA (IA local, opcional) ──────────────────────────────
OLLAMA_URL=http://localhost:11434
OLLAMA_MODEL=llama3
```

- [ ] **Step 2: Verificar que .env.example está en .gitignore pero NO el .env**

```bash
grep -n "\.env" .gitignore
```

Expected: `.env` en la lista (ignorado). `.env.example` NO debe estar ignorado.

- [ ] **Step 3: Commit**

```bash
git add .env.example
git commit -m "docs: add .env.example with all required variables documented"
```

---

## Self-Review Checklist

- [x] **config.go**: `AdminPassword` y `JWTSecret` sin defaults hardcodeados → Task 1 ✓
- [x] **ValidateRequired**: falla en producción si faltan secrets → Task 1 ✓
- [x] **IsProd()**: helper disponible para todos los módulos → Task 1 ✓
- [x] **Rate limiting login**: 5 req / 15 min en `POST /admin/login` → Task 2 + Task 3 ✓
- [x] **Rate limiting fragments**: 60 req / 1 min → Task 2 + Task 3 ✓
- [x] **Rate limiting global**: 200 req / 1 min → Task 2 + Task 3 ✓
- [x] **Security headers**: X-Frame-Options, nosniff, XSS, Referrer, Permissions, CSP → Task 2 ✓
- [x] **HSTS**: solo en producción → Task 2 ✓
- [x] **Graceful shutdown**: signal.NotifyContext + ShutdownWithTimeout(10s) → Task 3 ✓
- [x] **Health check /health**: público, JSON → Task 3 ✓
- [x] **Logger JSON en producción**: condicional por `cfg.IsProd()` → Task 3 ✓
- [x] **.env.example**: todas las vars documentadas → Task 4 ✓
- [x] **Sin cambios a handlers ni templates**: confirmado, solo 4 archivos tocados
