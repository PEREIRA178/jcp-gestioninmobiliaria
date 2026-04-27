# Routes Refactor & Cleanup Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Separar las rutas de `cmd/server/main.go` en tres archivos dedicados (`internal/routes/`), eliminar todo lo relacionado con devices/playlists/colegio, y enfocnar el realtime exclusivamente en propiedades y noticias.

**Architecture:** Paquete `internal/routes/` con tres funciones `Register*()` que encapsulan las rutas por audiencia. El hub de realtime se simplifica para solo hookear `propiedades` y `content_blocks`. `collections.go` elimina colecciones y seeds del colegio. `main.go` queda como bootstrap puro de ~60 líneas.

**Tech Stack:** Go 1.23, Fiber v2, PocketBase v0.25, gofiber/websocket/v2

---

## File Map

| Archivo | Acción | Responsabilidad |
|---|---|---|
| `internal/routes/public.go` | Crear | Web pública + WebSocket /ws/web |
| `internal/routes/fragments.go` | Crear | HTMX fragments propiedades + noticias |
| `internal/routes/admin.go` | Crear | Panel admin sin devices ni playlists |
| `internal/realtime/hub.go` | Reemplazar | Hooks solo propiedades/noticias, Client simplificado |
| `internal/handlers/ws/websocket.go` | Reemplazar | Solo WebSocket (eliminar DeviceSocket) |
| `internal/auth/collections.go` | Reemplazar | Sin devices/playlists/seeds escolares |
| `cmd/server/main.go` | Reemplazar | Bootstrap puro ~60 líneas |

---

## Task 1: Crear `internal/routes/public.go`

**Files:**
- Create: `internal/routes/public.go`

- [ ] **Step 1: Crear el directorio y el archivo**

```go
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
```

Guardar en: `internal/routes/public.go`

- [ ] **Step 2: Verificar compilación parcial**

```bash
cd /Users/matiaspereira/jcp-gestioninmobiliaria/jcp-gestioninmobiliaria
go build ./internal/routes/... 2>&1
```

Expected: error de paquete no encontrado para `ws` — normal, aún no modificamos websocket.go. Si el error es solo de ese import, continuar.

---

## Task 2: Crear `internal/routes/fragments.go`

**Files:**
- Create: `internal/routes/fragments.go`

- [ ] **Step 1: Crear el archivo**

```go
package routes

import (
	"jcp-gestioninmobiliaria/internal/config"
	"jcp-gestioninmobiliaria/internal/handlers/fragments"
	"jcp-gestioninmobiliaria/internal/middleware"

	"github.com/gofiber/fiber/v2"
	"github.com/pocketbase/pocketbase"
)

// RegisterFragments monta los endpoints HTMX de propiedades y noticias.
func RegisterFragments(app *fiber.App, cfg *config.Config, pb *pocketbase.PocketBase) {
	frag := app.Group("/fragments", middleware.FragmentsRateLimiter())
	frag.Get("/hero", fragments.HeroCarousel(cfg, pb))
	frag.Get("/propiedades-destacadas", fragments.PropiedadesDestacadas(cfg, pb))
	frag.Get("/propiedades-page", fragments.PropiedadesPage(cfg, pb))
	frag.Get("/noticias", fragments.Noticias(cfg, pb))
	frag.Get("/noticias-page", fragments.NoticiasPage(cfg, pb))
}
```

Guardar en: `internal/routes/fragments.go`

---

## Task 3: Crear `internal/routes/admin.go`

**Files:**
- Create: `internal/routes/admin.go`

- [ ] **Step 1: Crear el archivo**

```go
package routes

import (
	"jcp-gestioninmobiliaria/internal/config"
	"jcp-gestioninmobiliaria/internal/handlers/admin"
	"jcp-gestioninmobiliaria/internal/middleware"

	"github.com/gofiber/fiber/v2"
	"github.com/pocketbase/pocketbase"
)

// RegisterAdmin monta el panel de administración de JCP Gestión Inmobiliaria.
func RegisterAdmin(app *fiber.App, cfg *config.Config, pb *pocketbase.PocketBase) {
	// Rutas públicas del admin (sin auth)
	app.Get("/admin/login", admin.LoginPage(cfg))
	app.Post("/admin/login", middleware.LoginRateLimiter(), admin.LoginSubmit(cfg))
	app.Post("/admin/logout", admin.Logout())

	adm := app.Group("/admin", middleware.AuthRequired(cfg))

	adm.Get("/", admin.Dashboard(cfg))
	adm.Get("/dashboard", admin.Dashboard(cfg))
	adm.Get("/dashboard/stats", admin.DashboardStats(cfg, pb))

	// Multimedia (gestión de imágenes y archivos)
	adm.Get("/multimedia", admin.MultimediaList(cfg, pb))
	adm.Get("/multimedia/new", admin.MultimediaForm(cfg))
	adm.Post("/multimedia", admin.MultimediaCreate(cfg, pb))
	adm.Get("/multimedia/:id/edit", admin.MultimediaEdit(cfg, pb))
	adm.Put("/multimedia/:id", admin.MultimediaUpdate(cfg, pb))
	adm.Delete("/multimedia/:id", admin.MultimediaDelete(cfg, pb))

	// Noticias
	adm.Get("/news", admin.NewsList(cfg, pb))
	adm.Get("/news/new", admin.NewsForm(cfg))
	adm.Post("/news", admin.NewsCreate(cfg, pb))
	adm.Get("/news/:id/edit", admin.NewsEdit(cfg, pb))
	adm.Put("/news/:id", admin.NewsUpdate(cfg, pb))
	adm.Delete("/news/:id", admin.NewsDelete(cfg, pb))

	// Usuarios
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
}
```

Guardar en: `internal/routes/admin.go`

- [ ] **Step 2: Commit de los 3 archivos de rutas**

```bash
git add internal/routes/
git commit -m "refactor: extract routes into internal/routes package (public, fragments, admin)"
```

---

## Task 4: Simplificar `internal/realtime/hub.go`

**Files:**
- Modify: `internal/realtime/hub.go`

El archivo actual (284 líneas) mezcla tipos de devices con web clients, y hookea colecciones del colegio. Se reemplaza completo.

- [ ] **Step 1: Reemplazar el archivo completo**

```go
package realtime

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/gofiber/websocket/v2"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// ══════════════════════════════════════════════════════
//  MESSAGE TYPES
// ══════════════════════════════════════════════════════

type MessageType string

const (
	MsgPropiedadesUpdate MessageType = "propiedades_update"
	MsgNoticiasUpdate    MessageType = "noticias_update"
	MsgRefreshWeb        MessageType = "refresh_web"
	MsgRefreshAll        MessageType = "refresh_all"
)

type Message struct {
	Type    MessageType     `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

// ══════════════════════════════════════════════════════
//  CLIENT
// ══════════════════════════════════════════════════════

type Client struct {
	Conn *websocket.Conn
	Send chan []byte
}

// ══════════════════════════════════════════════════════
//  HUB — Central WebSocket broadcaster
// ══════════════════════════════════════════════════════

type Hub struct {
	clients    map[*Client]bool
	mu         sync.RWMutex
	broadcast  chan Message
	register   chan *Client
	unregister chan *Client
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan Message, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Printf("🔌 WS client connected (total: %d)", len(h.clients))

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.Send)
			}
			h.mu.Unlock()
			log.Printf("🔌 WS client disconnected")

		case msg := <-h.broadcast:
			data, err := json.Marshal(msg)
			if err != nil {
				log.Printf("❌ WS marshal error: %v", err)
				continue
			}
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.Send <- data:
				default:
					go func(c *Client) { h.unregister <- c }(client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

func (h *Hub) Register(c *Client)   { h.register <- c }
func (h *Hub) Unregister(c *Client) { h.unregister <- c }
func (h *Hub) Broadcast(msg Message) { h.broadcast <- msg }

// BroadcastWeb envía un mensaje de actualización a todos los clientes web conectados.
func (h *Hub) BroadcastWeb(msgType MessageType) {
	h.Broadcast(Message{Type: msgType})
}

// ══════════════════════════════════════════════════════
//  POCKETBASE HOOKS — propiedades y noticias únicamente
// ══════════════════════════════════════════════════════

var hubInstance *Hub

func RegisterPBHooks(pb *pocketbase.PocketBase) {
	pb.OnServe().BindFunc(func(se *core.ServeEvent) error {
		// Propiedades → actualiza listados públicos
		se.App.OnRecordAfterCreateSuccess("propiedades").BindFunc(func(e *core.RecordEvent) error {
			if hubInstance != nil {
				hubInstance.BroadcastWeb(MsgPropiedadesUpdate)
			}
			return e.Next()
		})
		se.App.OnRecordAfterUpdateSuccess("propiedades").BindFunc(func(e *core.RecordEvent) error {
			if hubInstance != nil {
				hubInstance.BroadcastWeb(MsgPropiedadesUpdate)
			}
			return e.Next()
		})
		se.App.OnRecordAfterDeleteSuccess("propiedades").BindFunc(func(e *core.RecordEvent) error {
			if hubInstance != nil {
				hubInstance.BroadcastWeb(MsgPropiedadesUpdate)
			}
			return e.Next()
		})

		// Content_blocks (noticias) → actualiza listados públicos
		se.App.OnRecordAfterCreateSuccess("content_blocks").BindFunc(func(e *core.RecordEvent) error {
			if hubInstance != nil {
				hubInstance.BroadcastWeb(MsgNoticiasUpdate)
			}
			return e.Next()
		})
		se.App.OnRecordAfterUpdateSuccess("content_blocks").BindFunc(func(e *core.RecordEvent) error {
			if hubInstance != nil {
				hubInstance.BroadcastWeb(MsgNoticiasUpdate)
			}
			return e.Next()
		})
		se.App.OnRecordAfterDeleteSuccess("content_blocks").BindFunc(func(e *core.RecordEvent) error {
			if hubInstance != nil {
				hubInstance.BroadcastWeb(MsgNoticiasUpdate)
			}
			return e.Next()
		})

		return se.Next()
	})
}

// SetHubInstance almacena la referencia del hub para los PB hooks.
func SetHubInstance(h *Hub) {
	hubInstance = h
}
```

- [ ] **Step 2: Verificar compilación**

```bash
go build ./internal/realtime/... 2>&1
```

Expected: errores en `ws/websocket.go` porque usa `ClientDevice` y `DeviceCode` eliminados — normal, se arregla en Task 5.

- [ ] **Step 3: Commit**

```bash
git add internal/realtime/hub.go
git commit -m "refactor: simplify hub to propiedades/noticias realtime only, remove device types"
```

---

## Task 5: Simplificar `internal/handlers/ws/websocket.go`

**Files:**
- Modify: `internal/handlers/ws/websocket.go`

El archivo actual tiene `DeviceSocket` (63 líneas) y `WebSocket` (28 líneas). Se elimina `DeviceSocket` y se adapta `WebSocket` al nuevo `Client` simplificado.

- [ ] **Step 1: Reemplazar el archivo completo**

```go
package ws

import (
	"jcp-gestioninmobiliaria/internal/realtime"

	"github.com/gofiber/websocket/v2"
)

// WebSocket maneja conexiones WebSocket de clientes web para recibir
// actualizaciones en tiempo real de propiedades y noticias.
func WebSocket(hub *realtime.Hub) func(*websocket.Conn) {
	return func(c *websocket.Conn) {
		client := &realtime.Client{
			Conn: c,
			Send: make(chan []byte, 64),
		}

		hub.Register(client)
		defer hub.Unregister(client)

		// Writer goroutine: reenvía mensajes del hub al cliente
		go func() {
			for msg := range client.Send {
				if err := c.WriteMessage(websocket.TextMessage, msg); err != nil {
					return
				}
			}
		}()

		// Reader loop: mantiene la conexión viva
		for {
			if _, _, err := c.ReadMessage(); err != nil {
				break
			}
		}
	}
}
```

- [ ] **Step 2: Verificar compilación**

```bash
go build ./internal/... 2>&1
```

Expected: errores en `auth/collections.go` por referencia a `SeedDevicesAndPlaylists` y colecciones eliminadas — normal, se arregla en Task 6.

- [ ] **Step 3: Commit**

```bash
git add internal/handlers/ws/websocket.go
git commit -m "refactor: remove DeviceSocket, keep only WebSocket for web realtime"
```

---

## Task 6: Limpiar `internal/auth/collections.go`

**Files:**
- Modify: `internal/auth/collections.go`

El archivo actual (816 líneas) crea colecciones de devices/playlists, seeds del colegio, y tiene el password del superadmin hardcodeado. Se reemplaza con versión limpia para inmobiliaria.

**Cambios clave:**
- `RegisterPBHooks(pb, cfg)` recibe `*config.Config` para el seed del superadmin
- `ensureCollections(app, cfg)` pasa `cfg.AdminPassword` al crear el superadmin
- Eliminar colecciones: `playlists`, `playlist_items`, `devices`
- Eliminar: `SeedDevicesAndPlaylists`, `seedContentBlocks`, `seedBlock` type
- Eliminar migrations: `migrateDevicesCurrentView`, `migratePlaylistItemsContentBlockID`, `migrateMultimediaStartTime`

- [ ] **Step 1: Reemplazar el archivo completo**

```go
package auth

import (
	"log"

	"jcp-gestioninmobiliaria/internal/config"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

// RegisterPBHooks configura las colecciones de PocketBase al arrancar.
func RegisterPBHooks(pb *pocketbase.PocketBase, cfg *config.Config) {
	pb.OnServe().BindFunc(func(se *core.ServeEvent) error {
		log.Println("📦 PocketBase: Verificando colecciones...")
		if err := ensureCollections(se.App, cfg); err != nil {
			log.Printf("⚠️  Error creando colecciones: %v", err)
		}
		return se.Next()
	})
}

func ensureCollections(app core.App, cfg *config.Config) error {
	// ── 1. USERS ──
	if _, err := app.FindCollectionByNameOrId("users"); err != nil {
		col := core.NewAuthCollection("users")
		col.Fields.Add(
			&core.TextField{Name: "role", Required: true},
			&core.TextField{Name: "nombre"},
			&core.TextField{Name: "telefono"},
			&core.TextField{Name: "rut"},
			&core.BoolField{Name: "activo"},
		)
		col.AuthToken.Duration = 259200
		col.PasswordAuth.Enabled = true
		if err := app.Save(col); err != nil {
			return err
		}
		log.Println("  ✅ Collection 'users' created")
	}

	// ── 2. MEDIA (biblioteca multimedia → R2) ──
	if _, err := app.FindCollectionByNameOrId("media"); err != nil {
		col := core.NewBaseCollection("media")
		col.Fields.Add(
			&core.TextField{Name: "filename", Required: true},
			&core.URLField{Name: "url_r2"},
			&core.TextField{Name: "type", Required: true},
			&core.NumberField{Name: "size"},
			&core.TextField{Name: "uploaded_by"},
			&core.TextField{Name: "status"},
			&core.TextField{Name: "description"},
			&core.NumberField{Name: "duration_seconds"},
			&core.FileField{Name: "thumbnail", MaxSelect: 1, MaxSize: 5242880},
		)
		if err := app.Save(col); err != nil {
			return err
		}
		log.Println("  ✅ Collection 'media' created")
	}

	// ── 3. CONTENT_BLOCKS (noticias) ──
	if _, err := app.FindCollectionByNameOrId("content_blocks"); err != nil {
		col := core.NewBaseCollection("content_blocks")
		col.Fields.Add(
			&core.TextField{Name: "title", Required: true},
			&core.EditorField{Name: "description"},
			&core.TextField{Name: "category"},
			&core.BoolField{Name: "urgency"},
			&core.DateField{Name: "date"},
			&core.BoolField{Name: "featured"},
			&core.TextField{Name: "status"},
			&core.TextField{Name: "media_ids"},
		)
		if err := app.Save(col); err != nil {
			return err
		}
		log.Println("  ✅ Collection 'content_blocks' created")
	}

	// ── 4. MULTIMEDIA (gestión de imágenes del admin) ──
	if _, err := app.FindCollectionByNameOrId("multimedia"); err != nil {
		col := core.NewBaseCollection("multimedia")
		col.Fields.Add(
			&core.TextField{Name: "filename", Required: true},
			&core.URLField{Name: "url_r2"},
			&core.TextField{Name: "type", Required: true},
			&core.NumberField{Name: "size"},
			&core.TextField{Name: "uploaded_by"},
			&core.TextField{Name: "estado"},
			&core.TextField{Name: "descripcion"},
			&core.NumberField{Name: "duracion_segundos"},
			&core.FileField{Name: "thumbnail", MaxSelect: 1, MaxSize: 5242880},
		)
		if err := app.Save(col); err != nil {
			return err
		}
		log.Println("  ✅ Collection 'multimedia' created")
	}

	// ── 5. WHATSAPP_LOGS ──
	if _, err := app.FindCollectionByNameOrId("whatsapp_logs"); err != nil {
		col := core.NewBaseCollection("whatsapp_logs")
		col.Fields.Add(
			&core.TextField{Name: "event_id"},
			&core.TextField{Name: "phone"},
			&core.TextField{Name: "message_sid"},
			&core.TextField{Name: "status"},
			&core.TextField{Name: "direction"},
			&core.TextField{Name: "body"},
			&core.TextField{Name: "error_message"},
		)
		if err := app.Save(col); err != nil {
			return err
		}
		log.Println("  ✅ Collection 'whatsapp_logs' created")
	}

	// ── 6. PROPIEDADES (JCP Gestión Inmobiliaria) ──
	if _, err := app.FindCollectionByNameOrId("propiedades"); err != nil {
		col := core.NewBaseCollection("propiedades")
		col.Fields.Add(
			&core.TextField{Name: "titulo", Required: true},
			&core.TextField{Name: "slug"},
			&core.EditorField{Name: "descripcion"},
			&core.TextField{Name: "operacion", Required: true}, // VENTA | ARRIENDO
			&core.TextField{Name: "tipo", Required: true},      // CASA | DEPARTAMENTO | TERRENO | PARCELA | LOCAL | OFICINA | BODEGA
			&core.TextField{Name: "direccion"},
			&core.TextField{Name: "comuna"},
			&core.TextField{Name: "region"},
			&core.NumberField{Name: "precio_uf"},
			&core.NumberField{Name: "precio_clp"},
			&core.NumberField{Name: "dormitorios"},
			&core.NumberField{Name: "banos"},
			&core.NumberField{Name: "estacionamientos"},
			&core.NumberField{Name: "superficie_util"},
			&core.NumberField{Name: "superficie_total"},
			&core.NumberField{Name: "ano_construccion"},
			&core.TextField{Name: "estado_propiedad"},
			&core.TextField{Name: "amenidades"},
			&core.TextField{Name: "status"}, // borrador|publicado|reservada|vendida|arrendada
			&core.BoolField{Name: "destacada"},
			&core.BoolField{Name: "oportunidad"},
			&core.TextField{Name: "cover_image"},
			&core.TextField{Name: "gallery"},
			&core.TextField{Name: "tour_url"},
			&core.NumberField{Name: "lat"},
			&core.NumberField{Name: "lng"},
			&core.DateField{Name: "publicado_en"},
			&core.TextField{Name: "corredor_id"},
			&core.TextField{Name: "contacto_whatsapp"},
		)
		if err := app.Save(col); err != nil {
			return err
		}
		log.Println("  ✅ Collection 'propiedades' created")
	}

	// ── Superadmin por defecto ──
	users, _ := app.FindCollectionByNameOrId("users")
	if users != nil {
		records, err := app.FindRecordsByFilter(users, "role = 'superadmin'", "", 1, 0)
		if err != nil || len(records) == 0 {
			record := core.NewRecord(users)
			record.Set("email", cfg.AdminEmail)
			record.Set("password", cfg.AdminPassword)
			record.Set("passwordConfirm", cfg.AdminPassword)
			record.Set("nombre", "Administrador")
			record.Set("role", "superadmin")
			record.Set("activo", true)
			record.Set("verified", true)
			if err := app.Save(record); err != nil {
				log.Printf("⚠️  Error creating superadmin: %v", err)
			} else {
				log.Println("  ✅ Default superadmin created")
			}
		}
	}

	// ── Seed de propiedades demo ──
	if err := seedPropiedades(app); err != nil {
		log.Printf("⚠️  Error seeding propiedades: %v", err)
	}

	// ── Migrations de content_blocks ──
	migrateContentBlocks(app)
	migrateUrgencyFromCategory(app)
	migrateContentBlocksTemplate(app)

	return nil
}

// migrateContentBlocks agrega campos introducidos después de la creación inicial.
func migrateContentBlocks(app core.App) {
	col, err := app.FindCollectionByNameOrId("content_blocks")
	if err != nil {
		return
	}
	changed := false
	for _, name := range []string{"pdf_url", "image_url", "body"} {
		if col.Fields.GetByName(name) == nil {
			col.Fields.Add(&core.TextField{Name: name})
			changed = true
			log.Printf("  ✅ content_blocks: added field '%s'", name)
		}
	}
	if changed {
		if err := app.Save(col); err != nil {
			log.Printf("⚠️  content_blocks migration error: %v", err)
		}
	}
}

// migrateUrgencyFromCategory setea urgency=true para todos los records EMERGENCIA.
func migrateUrgencyFromCategory(app core.App) {
	records, err := app.FindRecordsByFilter("content_blocks",
		"category = 'EMERGENCIA' && urgency = false", "", 1000, 0)
	if err != nil || len(records) == 0 {
		return
	}
	for _, r := range records {
		r.Set("urgency", true)
		if err := app.Save(r); err != nil {
			log.Printf("⚠️  urgency migration error for %s: %v", r.Id, err)
		}
	}
	log.Printf("  ✅ Migrated urgency=true for %d EMERGENCIA records", len(records))
}

// migrateContentBlocksTemplate agrega el campo 'template' si no existe.
func migrateContentBlocksTemplate(app core.App) {
	col, err := app.FindCollectionByNameOrId("content_blocks")
	if err != nil || col.Fields.GetByName("template") != nil {
		return
	}
	col.Fields.Add(&core.TextField{Name: "template"})
	if err := app.Save(col); err != nil {
		log.Printf("⚠️  content_blocks template migration: %v", err)
		return
	}
	log.Println("  ✅ content_blocks: added field 'template'")
}

// seedPropiedades inserta propiedades demo si la colección está vacía.
func seedPropiedades(app core.App) error {
	col, err := app.FindCollectionByNameOrId("propiedades")
	if err != nil {
		return err
	}
	existing, _ := app.FindRecordsByFilter(col, "status = 'publicado'", "", 1, 0)
	if len(existing) > 0 {
		return nil
	}

	type propSeed struct {
		titulo, slug, descripcion              string
		operacion, tipo                        string
		direccion, comuna, region              string
		precioUF, precioCLP                   float64
		dorm, banos, estac                    int
		supUtil, supTotal                     float64
		ano                                   int
		estado, amenidades                    string
		destacada, oportunidad                bool
	}

	now := types.NowDateTime()

	seeds := []propSeed{
		{
			titulo: "Casa mediterránea con jardín — La Dehesa", slug: "casa-mediterranea-la-dehesa",
			descripcion: "Amplia casa de 3 pisos con jardín, piscina y vista a la cordillera.",
			operacion: "VENTA", tipo: "CASA",
			direccion: "Av. La Dehesa 1234", comuna: "Lo Barnechea", region: "Metropolitana",
			precioUF: 18500, precioCLP: 720000000,
			dorm: 4, banos: 3, estac: 2, supUtil: 220, supTotal: 450, ano: 2016,
			estado: "usada", amenidades: "piscina,jardin,quincho,bodega,seguridad_24h",
			destacada: true,
		},
		{
			titulo: "Departamento moderno con vista al parque — Ñuñoa", slug: "depto-moderno-nunoa",
			descripcion: "Departamento de 2 dormitorios y 2 baños con vista panorámica al Parque Bustamante.",
			operacion: "VENTA", tipo: "DEPARTAMENTO",
			direccion: "Av. Irarrázaval 3400, Torre B, Dep. 1204", comuna: "Ñuñoa", region: "Metropolitana",
			precioUF: 6400, precioCLP: 250000000,
			dorm: 2, banos: 2, estac: 1, supUtil: 72, supTotal: 85, ano: 2022,
			estado: "a_estrenar", amenidades: "gimnasio,piscina,sala_multiuso,conserjeria",
			destacada: true, oportunidad: true,
		},
		{
			titulo: "Parcela 5.000 m² con vista al valle — Olmué", slug: "parcela-olmue-vista-valle",
			descripcion: "Terreno de 5.000 m² con árboles nativos, acceso a agua de pozo y rol propio.",
			operacion: "VENTA", tipo: "PARCELA",
			direccion: "Camino Los Olmos Km 4", comuna: "Olmué", region: "Valparaíso",
			precioUF: 3200, precioCLP: 124000000,
			supTotal: 5000, ano: 0, estado: "usada",
			amenidades: "agua_pozo,electricidad,acceso_pavimentado",
			oportunidad: true,
		},
		{
			titulo: "Arriendo departamento amoblado — Providencia", slug: "arriendo-depto-amoblado-providencia",
			descripcion: "Acogedor departamento de 1 dormitorio, completamente amoblado y equipado.",
			operacion: "ARRIENDO", tipo: "DEPARTAMENTO",
			direccion: "Av. Providencia 2015, Dep. 803", comuna: "Providencia", region: "Metropolitana",
			precioUF: 21, precioCLP: 820000,
			dorm: 1, banos: 1, estac: 1, supUtil: 48, supTotal: 55, ano: 2019,
			estado: "usada", amenidades: "amoblado,equipado,gimnasio,conserjeria",
		},
		{
			titulo: "Oficina Class A — Las Condes", slug: "oficina-class-a-las-condes",
			descripcion: "Oficina de 180 m² en edificio corporativo Class A. Piso 22 con vista 360°.",
			operacion: "ARRIENDO", tipo: "OFICINA",
			direccion: "Av. Apoquindo 4500, Piso 22", comuna: "Las Condes", region: "Metropolitana",
			precioUF: 75, precioCLP: 2900000,
			banos: 2, estac: 4, supUtil: 180, supTotal: 180, ano: 2018,
			estado: "usada", amenidades: "seguridad_24h,aire_acondicionado,gimnasio",
			destacada: true,
		},
		{
			titulo: "Local comercial esquina — Centro Viña", slug: "local-comercial-vina-centro",
			descripcion: "Local comercial de 120 m² en esquina con alto flujo peatonal.",
			operacion: "ARRIENDO", tipo: "LOCAL",
			direccion: "Esq. Arlegui con Quillota", comuna: "Viña del Mar", region: "Valparaíso",
			precioUF: 38, precioCLP: 1480000,
			banos: 1, supUtil: 120, supTotal: 120, ano: 2005,
			estado: "usada", amenidades: "dos_accesos,vitrina_esquina,alto_flujo",
			oportunidad: true,
		},
	}

	for _, s := range seeds {
		r := core.NewRecord(col)
		r.Set("titulo", s.titulo)
		r.Set("slug", s.slug)
		r.Set("descripcion", s.descripcion)
		r.Set("operacion", s.operacion)
		r.Set("tipo", s.tipo)
		r.Set("direccion", s.direccion)
		r.Set("comuna", s.comuna)
		r.Set("region", s.region)
		r.Set("precio_uf", s.precioUF)
		r.Set("precio_clp", s.precioCLP)
		r.Set("dormitorios", s.dorm)
		r.Set("banos", s.banos)
		r.Set("estacionamientos", s.estac)
		r.Set("superficie_util", s.supUtil)
		r.Set("superficie_total", s.supTotal)
		r.Set("ano_construccion", s.ano)
		r.Set("estado_propiedad", s.estado)
		r.Set("amenidades", s.amenidades)
		r.Set("status", "publicado")
		r.Set("destacada", s.destacada)
		r.Set("oportunidad", s.oportunidad)
		r.Set("publicado_en", now)
		if err := app.Save(r); err != nil {
			log.Printf("⚠️  seed propiedad %s: %v", s.titulo, err)
		}
	}
	log.Printf("  ✅ seedPropiedades: %d listings", len(seeds))
	return nil
}
```

- [ ] **Step 2: Verificar compilación**

```bash
go build ./internal/... 2>&1
```

Expected: sin errores (o solo errores en `cmd/server/main.go` por importaciones no actualizadas).

- [ ] **Step 3: Commit**

```bash
git add internal/auth/collections.go
git commit -m "refactor: remove device/playlist collections and school seeds from collections.go"
```

---

## Task 7: Actualizar `cmd/server/main.go`

**Files:**
- Modify: `cmd/server/main.go`

Reemplazar con versión limpia que delega rutas a `routes.Register*()` y llama `auth.RegisterPBHooks(pb, cfg)`.

- [ ] **Step 1: Reemplazar el archivo completo**

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
```

- [ ] **Step 2: Verificar compilación completa**

```bash
go build ./... 2>&1
```

Expected: sin errores.

- [ ] **Step 3: Verificar que el servidor arranca**

En una terminal separada:
```bash
go run ./cmd/server serve 2>&1 | head -20
```

Expected: logs de PocketBase + "🏢 JCP Gestión Inmobiliaria en http://localhost:3000"

- [ ] **Step 4: Verificar /health**

```bash
curl -s http://localhost:3000/health
```

Expected: `{"env":"development","status":"ok"}`

- [ ] **Step 5: Commit final**

```bash
git add cmd/server/main.go
git commit -m "refactor: main.go delegates to routes.Register*(), calls auth.RegisterPBHooks with cfg"
```

---

## Self-Review

**Spec coverage:**
- [x] `routes/public.go` — web pública + `/ws/web` → Task 1 ✓
- [x] `routes/fragments.go` — fragments propiedades + noticias → Task 2 ✓
- [x] `routes/admin.go` — panel sin devices/playlists → Task 3 ✓
- [x] `hub.go` — tipos simplificados, hooks propiedades/noticias → Task 4 ✓
- [x] `websocket.go` — solo WebSocket, sin DeviceSocket → Task 5 ✓
- [x] `collections.go` — sin devices/playlists/seeds escolares → Task 6 ✓
- [x] `collections.go` — superadmin usa `cfg.AdminPassword` → Task 6 ✓
- [x] `main.go` — ~60 líneas, llama Register*() → Task 7 ✓
- [x] `auth.RegisterPBHooks(pb, cfg)` firma consistente en Tasks 6 y 7 ✓
- [x] `Client` struct simplificado consistente en Tasks 4, 5, 7 ✓
- [x] Realtime para propiedades Y noticias → Task 4 ✓
