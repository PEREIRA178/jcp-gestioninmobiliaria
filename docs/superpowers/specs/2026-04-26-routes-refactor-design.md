# Refactor de Rutas y Limpieza — JCP Gestión Inmobiliaria

**Fecha:** 2026-04-26  
**Enfoque elegido:** Paquete `internal/routes/` con 3 archivos  
**Estado:** Aprobado

---

## Contexto

`cmd/server/main.go` tiene 196 líneas mezclando bootstrap del servidor, middleware y ~60 rutas de negocio. Además contiene rutas y lógica heredadas del proyecto anterior (colegio San Lorenzo): devices, totems, playlists, WebSocket de dispositivos, fragmentos escolares. El proyecto es exclusivamente JCP Gestión Inmobiliaria y debe limpiarse.

El realtime se mantiene pero se reenfoca en `propiedades` y `content_blocks` (noticias).

---

## Alcance

7 archivos afectados. Sin cambios a handlers de propiedades, templates ni lógica de negocio.

| Archivo | Acción | Contenido |
|---|---|---|
| `internal/routes/public.go` | Crear | Web pública + WebSocket /ws/web |
| `internal/routes/fragments.go` | Crear | HTMX fragments propiedades + noticias |
| `internal/routes/admin.go` | Crear | Panel admin sin devices ni playlists |
| `internal/realtime/hub.go` | Modificar | Hooks solo para propiedades y noticias |
| `internal/handlers/ws/websocket.go` | Modificar | Eliminar DeviceSocket |
| `internal/auth/collections.go` | Modificar | Eliminar colecciones y seeds del colegio |
| `cmd/server/main.go` | Modificar | ~60 líneas, llama a routes.Register*() |

---

## Sección 1: `internal/routes/public.go`

Función exportada: `RegisterPublic(app *fiber.App, cfg *config.Config, pb *pocketbase.PocketBase, hub *realtime.Hub)`

Rutas incluidas:
```
GET  /                           → web.PageHandler(cfg, "propiedades")
GET  /propiedades.html           → web.PageHandler(cfg, "propiedades")
GET  /noticias.html              → web.PageHandler(cfg, "noticias")
GET  /noticias/:id               → web.NoticiaHandler(cfg, pb)
GET  /propiedades/:key           → web.PropiedadHandler(cfg, pb)
GET  /rss.xml                    → web.RSSFeed(cfg)
POST /webhook/whatsapp           → web.WhatsAppWebhook(cfg)
```

WebSocket (realtime para web clients):
```
Use  /ws   → upgrade check middleware
GET  /ws/web → gows.New(ws.WebSocket(hub))
```

**Rutas eliminadas de este archivo:** /display/:code, /totem/:code, /ws/device/:code, /api/*

---

## Sección 2: `internal/routes/fragments.go`

Función exportada: `RegisterFragments(app *fiber.App, cfg *config.Config, pb *pocketbase.PocketBase)`

Grupo `/fragments` con `middleware.FragmentsRateLimiter()`:
```
GET /fragments/hero                   → fragments.HeroCarousel(cfg, pb)
GET /fragments/propiedades-destacadas → fragments.PropiedadesDestacadas(cfg, pb)
GET /fragments/propiedades-page       → fragments.PropiedadesPage(cfg, pb)
GET /fragments/noticias               → fragments.Noticias(cfg, pb)
GET /fragments/noticias-page          → fragments.NoticiasPage(cfg, pb)
```

**Rutas eliminadas:** /fragments/eventos, /fragments/comunicados, /fragments/blog

---

## Sección 3: `internal/routes/admin.go`

Función exportada: `RegisterAdmin(app *fiber.App, cfg *config.Config, pb *pocketbase.PocketBase)`

Rutas públicas del admin (sin auth):
```
GET  /admin/login
POST /admin/login  → + middleware.LoginRateLimiter()
POST /admin/logout
```

Grupo `/admin` con `middleware.AuthRequired(cfg)`:
```
GET  /admin, /admin/dashboard, /admin/dashboard/stats
/admin/multimedia/*   → CRUD completo (6 rutas)
/admin/news/*         → CRUD completo (6 rutas)
/admin/users/*        → CRUD con RoleRequired (4 rutas)
/admin/whatsapp-logs
/admin/propiedades/*  → CRUD + toggle publish (7 rutas)
```

**Rutas eliminadas:** /admin/playlists/*, /admin/devices/*, /admin/events/*

---

## Sección 4: `internal/realtime/hub.go`

### Message types — reemplazar por:
```go
MsgPropiedadesUpdate MessageType = "propiedades_update"
MsgNoticiasUpdate    MessageType = "noticias_update"
MsgRefreshWeb        MessageType = "refresh_web"   // mantener
MsgRefreshAll        MessageType = "refresh_all"   // mantener
```

Eliminar: `MsgPlaylistUpdate`, `MsgMultimediaUpdate`, `MsgEventUpdate`, `MsgDeviceHeartbeat`

### Client types — simplificar:
Eliminar `ClientDevice` type. Solo `ClientWeb` existe ahora.
El campo `DeviceCode` en `Client` se puede eliminar.

### Hub.Run() — simplificar routing:
El `switch msg.Target` ya no necesita manejar device codes. Solo `"web"` y `"all"`.

### Eliminar métodos:
- `BroadcastToDevice()` — solo servía a devices

### PB Hooks — reemplazar todo RegisterPBHooks por:
```go
// propiedades: create/update/delete → broadcast a web clients
OnRecordAfterCreateSuccess("propiedades") → hub.BroadcastWeb con MsgPropiedadesUpdate
OnRecordAfterUpdateSuccess("propiedades") → hub.BroadcastWeb con MsgPropiedadesUpdate  
OnRecordAfterDeleteSuccess("propiedades") → hub.BroadcastWeb con MsgPropiedadesUpdate

// content_blocks (noticias): create/update/delete → broadcast a web clients
OnRecordAfterCreateSuccess("content_blocks") → hub.BroadcastWeb con MsgNoticiasUpdate
OnRecordAfterUpdateSuccess("content_blocks") → hub.BroadcastWeb con MsgNoticiasUpdate
OnRecordAfterDeleteSuccess("content_blocks") → hub.BroadcastWeb con MsgNoticiasUpdate
```

Eliminar hooks de: multimedia, playlists, playlist_items, devices

### BroadcastWeb actualizado:
```go
func (h *Hub) BroadcastWeb(msgType MessageType) {
    h.Broadcast(Message{Type: msgType, Target: "web"})
}
```

---

## Sección 5: `internal/handlers/ws/websocket.go`

Eliminar `DeviceSocket()` completamente.  
Mantener solo `WebSocket(hub *realtime.Hub)` — maneja clientes web que se suscriben a updates de propiedades/noticias.

---

## Sección 6: `internal/auth/collections.go`

### Colecciones a NO crear (eliminar código de creación):
- `playlists` (colección 5)
- `playlist_items` (colección 6)
- `devices` (colección 7)

Las colecciones `users`, `media`, `content_blocks`, `multimedia`, `form_responses`, `whatsapp_logs`, `propiedades` se mantienen.

### Migrations a eliminar:
- `migrateDevicesCurrentView()`
- `migratePlaylistItemsContentBlockID()`
- `migrateMultimediaStartTime()` ← también es de playlist/device

### Seeds a eliminar:
- `SeedDevicesAndPlaylists()` — crea devices + playlists + playlist_items del colegio
- El seed de `content_blocks` escolar (simulacros, reuniones de apoderados, etc.)

### Mantener:
- `seedPropiedades()` — 6 propiedades demo de la inmobiliaria
- Creación del superadmin default
- Migrations de `content_blocks` (pdf_url, image_url, body, template)

### Seguridad: el password hardcodeado del superadmin
`collections.go:238` contiene `"jcp2026admin!"` hardcodeado en el seed del superadmin. Cambiar para leer de la config:
```go
record.Set("password", cfg.AdminPassword)
record.Set("passwordConfirm", cfg.AdminPassword)
```
Esto requiere pasar `cfg` a `ensureCollections()`.

---

## Sección 7: `cmd/server/main.go` resultante (~60 líneas)

```go
func main() {
    cfg := config.Load()
    config.ValidateRequired(cfg)

    pb := pocketbase.New()
    auth.RegisterPBHooks(pb, cfg)  // pasar cfg para superadmin seed
    realtime.RegisterPBHooks(pb)

    go pb.Start()

    app := fiber.New(/* ErrorHandler, BodyLimit */)

    // Middleware stack (sin cambios de Opción A)
    app.Use(recover, SecurityHeaders, GlobalRateLimiter, logger, cors)

    app.Get("/health", ...)
    app.Static("/static", ...)

    hub := realtime.NewHub()
    go hub.Run()
    realtime.SetHubInstance(hub)

    routes.RegisterPublic(app, cfg, pb, hub)
    routes.RegisterFragments(app, cfg, pb)
    routes.RegisterAdmin(app, cfg, pb)

    // Graceful shutdown (sin cambios de Opción A)
}
```

---

## Orden de implementación

1. `internal/routes/` — crear los 3 archivos nuevos (public, fragments, admin)
2. `internal/realtime/hub.go` — simplificar tipos, métodos y hooks
3. `internal/handlers/ws/websocket.go` — eliminar DeviceSocket
4. `internal/auth/collections.go` — eliminar device/playlist collections y seeds
5. `cmd/server/main.go` — ensamblar todo con las 3 llamadas a routes.Register*()
6. `go build ./...` — verificar compilación
7. Commit único final

---

## Riesgos y mitigaciones

| Riesgo | Mitigación |
|---|---|
| Eliminar código de colecciones rompe referencias en handlers | Los handlers de devices/playlists también se dejan de usar (no hay rutas que los llamen) |
| Seed del superadmin usa password hardcodeado | Migrar a `cfg.AdminPassword` en este mismo paso |
| `BroadcastWeb()` cambia firma | Actualizar los 2 call sites en hub.go simultáneamente |
| `auth.RegisterPBHooks` recibe nuevo parámetro `cfg` | Actualizar call en main.go en el mismo paso |
