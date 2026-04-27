# Seguridad y Preparación para Producción — JCP Gestión Inmobiliaria

**Fecha:** 2026-04-26  
**Enfoque elegido:** Middleware centralizado (Enfoque 1)  
**Estado:** Aprobado

---

## Contexto

El proyecto corre en Fly.io con una sola instancia Go+Fiber+PocketBase. Actualmente le faltan controles de seguridad básicos para producción: la contraseña admin y el JWT secret tienen defaults hardcodeados en el código fuente, no hay rate limiting en el login, no hay security headers, ni graceful shutdown.

## Alcance

Solo se modifican 2 archivos existentes y se crean 2 archivos nuevos. No se toca ningún handler de negocio ni template.

---

## Archivos a modificar/crear

| Archivo | Acción |
|---|---|
| `internal/config/config.go` | Eliminar defaults inseguros; agregar `IsProd()` y `ValidateRequired()` |
| `cmd/server/main.go` | Graceful shutdown, `/health` endpoint, logger condicional, wiring de nuevo middleware |
| `internal/middleware/security.go` | **Nuevo** — rate limiter + security headers |
| `.env.example` | **Nuevo** — plantilla de variables de entorno |

---

## Sección 1: Config (`internal/config/config.go`)

- `AdminPassword`: sin default. En producción, si `ENV=production` y la var está vacía → `ValidateRequired()` llama `log.Fatal` con mensaje claro.
- `JWTSecret`: sin default. Mismo comportamiento.
- `IsProd() bool`: devuelve `cfg.Env == "production"`. Evita strings dispersos en el código.
- `ValidateRequired(*Config)`: función standalone que se llama al inicio de `main()`. Lista **todas** las vars faltantes en un solo mensaje antes de abortar.

En `ENV=development` los campos vacíos se permiten (arrancan con valores de dev que se definen en el `.env.example`).

---

## Sección 2: Security Middleware (`internal/middleware/security.go`)

### Rate Limiting (in-memory, `fiber/v2/middleware/limiter`)

| Endpoint | Límite | Ventana | Motivo |
|---|---|---|---|
| `POST /admin/login` | 5 requests | 15 min | Brute force en credenciales |
| `GET /fragments/*` | 60 requests | 1 min | Anti-scraping/bots |
| Global (todas las rutas) | 200 requests | 1 min | DoS básico |

- Key por IP (`c.IP()`).
- Respuesta en límite excedido: HTTP 429 con mensaje en español.
- Para HTMX (`HX-Request: true`): respuesta como `<div class="toast toast-error">Demasiadas solicitudes...</div>`.

### Security Headers (función `SecurityHeaders() fiber.Handler`)

Headers aplicados globalmente:

```
X-Frame-Options: DENY
X-Content-Type-Options: nosniff
X-XSS-Protection: 1; mode=block
Referrer-Policy: strict-origin-when-cross-origin
Permissions-Policy: camera=(), microphone=(), geolocation=()
Content-Security-Policy: default-src 'self'; script-src 'self' unpkg.com cdn.jsdelivr.net 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data: blob: *; connect-src 'self'; font-src 'self' data:
```

Header solo en producción:
```
Strict-Transport-Security: max-age=31536000; includeSubDomains
```

El CSP permite: HTMX (inline scripts), Leaflet.js (unpkg/jsdelivr), imágenes de R2/externas.

---

## Sección 3: Main (`cmd/server/main.go`)

### Graceful Shutdown

```
ctx, stop = signal.NotifyContext(background, SIGINT, SIGTERM)
→ go app.Listen(":port")
→ <-ctx.Done()
→ app.ShutdownWithTimeout(10s)
```

PocketBase se inicia como goroutine igual que hoy; el shutdown de Fiber cierra conexiones HTTP activas en 10s.

### Health Check `/health`

Ruta pública (sin auth), antes de todos los middlewares de negocio:

```json
{ "status": "ok", "env": "production" }
```

HTTP 200. Útil para Fly.io health probes y monitoreo externo.

### Logger condicional

- `ENV=development`: formato legible actual `[15:04:05] 200 GET /path (2ms)`
- `ENV=production`: formato JSON de una línea:
  `{"time":"...","status":200,"method":"GET","path":"/","latency":"2ms","ip":"x.x.x.x"}`

### CORS más estricto

- Development: `AllowOrigins = "*"` (sin cambio)
- Production: `AllowOrigins = cfg.CORSOrigins` — debe setearse explícitamente. Si está vacío en producción, `ValidateRequired` lo reporta como advertencia (no fatal).

---

## Sección 4: `.env.example`

Archivo nuevo, comiteado al repo, **sin valores reales**. Muestra todas las variables disponibles marcando cuáles son obligatorias en producción.

Variables críticas marcadas con `# REQUERIDA EN PRODUCCIÓN`:
- `JWT_SECRET`
- `ADMIN_PASSWORD`
- `ENV`

---

## Orden de implementación

1. `internal/config/config.go` — base que todo lo demás necesita
2. `internal/middleware/security.go` — nuevo archivo
3. `cmd/server/main.go` — integra todo + graceful shutdown + health
4. `.env.example` — documentación de vars
5. Verificar que compila: `go build ./cmd/server`

---

## Riesgos y mitigaciones

| Riesgo | Mitigación |
|---|---|
| CSP rompe Leaflet o HTMX | CSP permisivo con `unsafe-inline` para scripts; testar en dev antes de prod |
| Rate limiter bloquea el admin real | Límite de 5 intentos en 15 min es razonable; mensajes claros al usuario |
| `ValidateRequired` rompe deploy si falta una var | Solo falla en `ENV=production`; dev sigue funcionando |
| Graceful shutdown no espera PocketBase | PocketBase tiene su propio ciclo; el timeout de 10s cubre requests HTTP pendientes |
