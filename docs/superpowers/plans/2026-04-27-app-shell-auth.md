# App Shell + Google Auth Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Transform the public property site into a Zillow-like authenticated app: landing page at `/`, Google OAuth login wall at `/login`, and an app shell with dark sidebar at `/propiedades` and `/propiedades/:key`.

**Architecture:** The public landing page uses the existing `layouts.Base`. Authenticated app pages use a new `layouts.AppShell` that renders a dark sidebar (220 px) + scrollable main area. Visitor sessions use the existing JWT infrastructure with a new `jcp_visitor` cookie and a new `VisitorAuthRequired` middleware. Google OAuth is handled server-side via `golang.org/x/oauth2/google` — no database storage of visitor records needed; the JWT itself is the session.

**Tech Stack:** Go 1.23 · Fiber v2 · Templ v0.3 · PocketBase v0.25 (embedded) · golang.org/x/oauth2 · Leaflet.js · HTMX · Material Symbols Rounded

---

## File Map

| Action | Path | Purpose |
|--------|------|---------|
| Create | `internal/templates/layouts/types.go` | `AppUser` struct shared by app layout and web templates |
| Create | `internal/templates/layouts/app.templ` | App shell: dark sidebar + main content slot |
| Create | `internal/templates/web/landing.templ` | Marketing landing page (hero, stats, how-it-works, CTA) |
| Create | `internal/templates/web/login.templ` | Minimal login card with Google OAuth button |
| Create | `internal/handlers/web/auth.go` | `GoogleLogin`, `GoogleCallback`, `VisitorLogout` handlers |
| Create | `internal/middleware/visitor.go` | `VisitorAuthRequired` middleware |
| Modify | `internal/config/config.go` | Add `GoogleClientID`, `GoogleClientSecret`, `GoogleRedirectURL` |
| Modify | `go.mod` | Promote `golang.org/x/oauth2` from indirect to direct |
| Modify | `internal/routes/public.go` | New routes: landing, login, auth callbacks; protect propiedades group |
| Modify | `internal/handlers/web/handlers.go` | `LandingHandler`, update `PropiedadesHandler` + `PropiedadHandler` to inject user info |
| Modify | `internal/templates/web/propiedades.templ` | Use AppShell, add topbar + filter bar, remove hero + CTA |
| Modify | `internal/templates/web/propiedad.templ` | Use AppShell, new aesthetic, add schedule-visit calendar |
| Modify | `internal/templates/layouts/base.templ` | Update footer (2026, Digital Wellness → diwe.cl); update nav links |
| Modify | `internal/middleware/security.go` | Add `accounts.google.com` to CSP `connect-src` and `img-src` |
| Modify | `.env.example` | Add `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`, `GOOGLE_REDIRECT_URL` |

---

## Task 1: Promote oauth2 dep + add Google config vars

**Files:**
- Modify: `go.mod`
- Modify: `internal/config/config.go`
- Modify: `.env.example`

- [ ] **Step 1: Promote oauth2 to direct dependency**

```bash
cd /Users/matiaspereira/jcp-gestioninmobiliaria/jcp-gestioninmobiliaria
go get golang.org/x/oauth2@v0.25.0
go get golang.org/x/oauth2/google@v0.25.0
```

Expected: `go.mod` now lists `golang.org/x/oauth2 v0.25.0` in the `require` block (not under `// indirect`).

- [ ] **Step 2: Add Google OAuth fields to Config struct**

In `internal/config/config.go`, add three fields to the `Config` struct after `JWTExpiration`:

```go
// Google OAuth2
GoogleClientID     string
GoogleClientSecret string
GoogleRedirectURL  string
```

- [ ] **Step 3: Populate fields in Load()**

In the `Load()` function, after the JWT block, add:

```go
// Google OAuth2
GoogleClientID:     getEnv("GOOGLE_CLIENT_ID", ""),
GoogleClientSecret: getEnv("GOOGLE_CLIENT_SECRET", ""),
GoogleRedirectURL:  getEnv("GOOGLE_REDIRECT_URL", "http://localhost:3000/auth/google/callback"),
```

- [ ] **Step 4: Document in .env.example**

Read `.env.example` first, then append at the end:

```
# Google OAuth2 (create at console.cloud.google.com)
GOOGLE_CLIENT_ID=your-client-id.apps.googleusercontent.com
GOOGLE_CLIENT_SECRET=your-client-secret
GOOGLE_REDIRECT_URL=https://yoursite.cl/auth/google/callback
```

- [ ] **Step 5: Build to verify**

```bash
go build ./...
```

Expected: no output (clean build).

- [ ] **Step 6: Commit**

```bash
git add go.mod go.sum internal/config/config.go .env.example
git commit -m "feat: add Google OAuth2 config vars"
```

---

## Task 2: AppUser type + App Shell layout

**Files:**
- Create: `internal/templates/layouts/types.go`
- Create: `internal/templates/layouts/app.templ`

- [ ] **Step 1: Create `types.go` with AppUser**

```go
// internal/templates/layouts/types.go
package layouts

// AppUser holds the authenticated visitor's display info for the app shell.
type AppUser struct {
	Name  string
	Email string
}
```

- [ ] **Step 2: Create `app.templ`**

```go
// internal/templates/layouts/app.templ
package layouts

templ AppShell(title string, includeMaps bool, user AppUser) {
	<!DOCTYPE html>
	<html lang="es">
	<head>
		<meta charset="UTF-8"/>
		<meta name="viewport" content="width=device-width, initial-scale=1.0"/>
		<title>{ title }</title>
		<link rel="icon" type="image/svg+xml" href="/static/favicon.svg"/>
		<link rel="preconnect" href="https://fonts.googleapis.com"/>
		<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin/>
		<link href="https://fonts.googleapis.com/css2?family=Cormorant+Garamond:ital,wght@0,600;0,700;1,600&family=Outfit:wght@300;400;500;600;700&family=Material+Symbols+Rounded:opsz,wght,FILL,GRAD@20,400,0,0&display=swap" rel="stylesheet"/>
		if includeMaps {
			<link rel="stylesheet" href="https://unpkg.com/leaflet@1.9.4/dist/leaflet.css"/>
		}
		<script src="https://unpkg.com/htmx.org@1.9.10/dist/htmx.min.js" defer></script>
		if includeMaps {
			<script src="https://unpkg.com/leaflet@1.9.4/dist/leaflet.js"></script>
			<script src="/static/js/maps.js" defer></script>
		}
		<style>
			*,*::before,*::after{box-sizing:border-box;margin:0;padding:0}
			:root{
			  --blue:#1D4ED8;--blue-dark:#1E3A8A;--blue-light:#3B82F6;
			  --navy:#0F172A;--sb-bg:#111827;--sb-w:220px;
			  --white:#fff;--off:#F8FAFC;--gray:#64748B;--outline:#E2E8F0;
			  --on-surface:#111827;--on-surface-var:#4B5563;
			  --jcp-primary:#1D4ED8;--jcp-primary-dark:#1E3A8A;--jcp-primary-light:#3B82F6;
			  --jcp-primary-50:#EFF6FF;--jcp-primary-100:#DBEAFE;
			  --surface:#F8FAFC;--surface-bright:#fff;
			  --surface-container-low:#F5F5F3;--surface-container-high:#EEEEE9;
			  --outline-var:#E5E7EB;--inverse-surface:#0F1419;
			  --r-sm:8px;--r-md:12px;--r-lg:16px;--r-full:9999px;
			  --elev-4:0 8px 24px rgba(15,20,25,.14);
			  --ease-express:cubic-bezier(0.05,0.7,0.1,1.0);
			  --font-display:'Cormorant Garamond',Georgia,serif;
			  --font-body:'Outfit',system-ui,sans-serif;
			  --ms-font:'Material Symbols Rounded';
			  --max-w:1280px;
			}
			html{-webkit-font-smoothing:antialiased;scroll-behavior:smooth}
			body{font-family:var(--font-body);background:var(--off);color:var(--on-surface);
			  display:flex;height:100vh;overflow:hidden}
			.ms{font-family:var(--ms-font);font-variation-settings:'FILL' 0,'wght' 300,'GRAD' 0,'opsz' 20;
			  font-size:18px;line-height:1;display:inline-flex;align-items:center;user-select:none}
			.ms-fill{font-variation-settings:'FILL' 1,'wght' 400,'GRAD' 0,'opsz' 20}
			.ms-sm{font-size:16px}.ms-lg{font-size:24px}

			/* ── SIDEBAR ── */
			.sb{width:var(--sb-w);background:var(--sb-bg);display:flex;flex-direction:column;
			  flex-shrink:0;height:100vh;overflow:hidden;border-right:1px solid rgba(255,255,255,.05)}
			.sb-logo{display:flex;align-items:center;gap:10px;padding:18px 16px;
			  border-bottom:1px solid rgba(255,255,255,.06);text-decoration:none}
			.sb-logo-box{width:34px;height:34px;background:var(--blue);border-radius:8px;
			  display:flex;align-items:center;justify-content:center;color:#fff;flex-shrink:0}
			.sb-logo-name{font-family:var(--font-display);font-weight:700;font-size:17px;color:#fff;line-height:1.1}
			.sb-logo-sub{font-size:10px;color:rgba(255,255,255,.35);letter-spacing:.04em}
			.sb-nav{flex:1;padding:10px 8px;display:flex;flex-direction:column;gap:2px;overflow-y:auto}
			.sb-label{font-size:10px;font-weight:600;letter-spacing:.1em;text-transform:uppercase;
			  color:rgba(255,255,255,.28);padding:14px 10px 6px}
			.sb-item{display:flex;align-items:center;gap:10px;padding:9px 10px;border-radius:10px;
			  color:rgba(255,255,255,.55);text-decoration:none;font-size:13px;font-weight:500;
			  transition:all .18s;cursor:pointer;border:none;background:none;width:100%;font-family:inherit;
			  white-space:nowrap}
			.sb-item:hover{background:rgba(255,255,255,.07);color:rgba(255,255,255,.85)}
			.sb-item.active{background:rgba(29,78,216,.25);color:#93C5FD;font-weight:600}
			.sb-item.active .ms{font-variation-settings:'FILL' 1,'wght' 400,'GRAD' 0,'opsz' 20}
			.sb-user{padding:12px 10px;border-top:1px solid rgba(255,255,255,.06);
			  display:flex;align-items:center;gap:10px}
			.sb-avatar{width:32px;height:32px;border-radius:50%;
			  background:linear-gradient(135deg,#3B82F6,#8B5CF6);
			  display:flex;align-items:center;justify-content:center;
			  color:#fff;font-size:13px;font-weight:700;flex-shrink:0;text-transform:uppercase}
			.sb-user-info{flex:1;min-width:0}
			.sb-user-name{font-size:12px;font-weight:600;color:rgba(255,255,255,.85);
			  white-space:nowrap;overflow:hidden;text-overflow:ellipsis}
			.sb-user-email{font-size:10px;color:rgba(255,255,255,.35);
			  white-space:nowrap;overflow:hidden;text-overflow:ellipsis}
			.sb-logout{width:28px;height:28px;border-radius:7px;border:none;
			  background:rgba(255,255,255,.06);color:rgba(255,255,255,.4);cursor:pointer;
			  display:flex;align-items:center;justify-content:center;transition:all .18s;flex-shrink:0;
			  text-decoration:none}
			.sb-logout:hover{background:rgba(239,68,68,.2);color:#FCA5A5}

			/* ── MAIN ── */
			.app-main{flex:1;display:flex;flex-direction:column;overflow:hidden;min-width:0}

			/* Map popup */
			.leaflet-popup-content-wrapper{border-radius:12px!important;box-shadow:0 8px 24px rgba(0,0,0,.18)!important}
			.leaflet-popup-content{margin:14px 16px!important}
			.map-popup-type{font-size:11px;text-transform:uppercase;color:#1D4ED8;font-weight:700;letter-spacing:.05em;margin-bottom:4px}
			.map-popup-title{font-size:14px;font-weight:600;color:#111827;line-height:1.35;margin-bottom:4px}
			.map-popup-price{font-size:17px;font-weight:700;color:#111827;margin-bottom:4px}
			.map-popup-loc{font-size:12px;color:#64748B;display:flex;align-items:center;gap:3px}

			@media(max-width:900px){
			  .sb{display:none}
			  .app-main{width:100%}
			}
		</style>
	</head>
	<body>
		<aside class="sb">
			<a href="/propiedades" class="sb-logo">
				<div class="sb-logo-box"><span class="ms ms-fill">home</span></div>
				<div>
					<div class="sb-logo-name">JCP Gestión</div>
					<div class="sb-logo-sub">Inmobiliaria</div>
				</div>
			</a>
			<nav class="sb-nav">
				<a href="/propiedades" class="sb-item active">
					<span class="ms">search</span>
					Explorar
				</a>
				<a href="#" class="sb-item">
					<span class="ms">favorite</span>
					Guardadas
				</a>
				<a href="#" class="sb-item">
					<span class="ms">notifications</span>
					Alertas de precio
				</a>
				<div class="sb-label">Cuenta</div>
				<a href="#" class="sb-item">
					<span class="ms">person</span>
					Mi perfil
				</a>
				<a href="https://wa.me/56912345678" class="sb-item" target="_blank" rel="noopener">
					<span class="ms">chat</span>
					Contactar a JCP
				</a>
			</nav>
			<div class="sb-user">
				<div class="sb-avatar">{ initials(user.Name) }</div>
				<div class="sb-user-info">
					<div class="sb-user-name">{ user.Name }</div>
					<div class="sb-user-email">{ user.Email }</div>
				</div>
				<a href="/auth/logout" class="sb-logout" title="Cerrar sesión">
					<span class="ms">logout</span>
				</a>
			</div>
		</aside>
		<div class="app-main">
			{ children... }
		</div>
		<script>
			document.querySelectorAll('.sb-item').forEach(function(item){
			  if(item.href && item.href !== '#' && window.location.pathname.startsWith(new URL(item.href).pathname)){
			    document.querySelectorAll('.sb-item').forEach(function(i){i.classList.remove('active')});
			    item.classList.add('active');
			  }
			});
		</script>
	</body>
	</html>
}

func initials(name string) string {
	if name == "" {
		return "?"
	}
	runes := []rune(name)
	if len(runes) >= 2 {
		return string(runes[0:1])
	}
	return string(runes[0:1])
}
```

- [ ] **Step 3: Generate templ**

```bash
templ generate ./internal/templates/layouts/
```

Expected: `(✓) Complete`

- [ ] **Step 4: Build to verify**

```bash
go build ./...
```

Expected: no output.

- [ ] **Step 5: Commit**

```bash
git add internal/templates/layouts/
git commit -m "feat: add AppShell layout with dark sidebar"
```

---

## Task 3: Landing page

**Files:**
- Create: `internal/templates/web/landing.templ`
- Modify: `internal/handlers/web/handlers.go` (add LandingHandler)

- [ ] **Step 1: Create `landing.templ`**

```go
// internal/templates/web/landing.templ
package web

import "jcp-gestioninmobiliaria/internal/templates/layouts"

templ LandingPage() {
	@layouts.Base("JCP Gestión Inmobiliaria — Propiedades en Chile", false) {
		<style>
			.hero{min-height:100vh;position:relative;display:flex;flex-direction:column;justify-content:center;overflow:hidden}
			.hero-bg{position:absolute;inset:0;
			  background:linear-gradient(155deg,rgba(15,23,42,.88) 0%,rgba(30,58,138,.72) 55%,rgba(15,23,42,.60) 100%),
			    url('https://images.unsplash.com/photo-1512917774080-9991f1c4c750?auto=format&fit=crop&w=1920&q=80') center/cover}
			.hero-content{position:relative;z-index:1;max-width:1200px;margin:0 auto;padding:120px 48px 200px}
			.hero-eyebrow{display:inline-flex;align-items:center;gap:8px;padding:7px 18px;
			  background:rgba(29,78,216,.22);border:1px solid rgba(96,165,250,.35);
			  color:#93C5FD;font-size:11px;font-weight:700;letter-spacing:.12em;text-transform:uppercase;
			  border-radius:9999px;margin-bottom:28px}
			.hero-h1{font-family:'DM Serif Display',Georgia,serif;font-size:clamp(44px,8vw,88px);
			  font-weight:400;color:#fff;line-height:1.05;margin-bottom:22px;max-width:680px}
			.hero-h1 em{font-style:italic;color:#60A5FA}
			.hero-sub{font-size:clamp(15px,1.8vw,18px);color:rgba(255,255,255,.68);max-width:440px;
			  line-height:1.75;margin-bottom:44px;font-weight:300}
			.hero-actions{display:flex;gap:14px;flex-wrap:wrap}
			.btn-hero{display:inline-flex;align-items:center;gap:9px;padding:16px 32px;
			  border-radius:9999px;font-size:15px;font-weight:700;font-family:inherit;
			  text-decoration:none;transition:all .25s}
			.btn-hero-p{background:#1D4ED8;color:#fff;box-shadow:0 8px 28px rgba(29,78,216,.4)}
			.btn-hero-p:hover{background:#1e40af;transform:translateY(-2px)}
			.btn-hero-s{background:rgba(255,255,255,.1);backdrop-filter:blur(10px);
			  color:#fff;border:1.5px solid rgba(255,255,255,.22)}
			.btn-hero-s:hover{background:rgba(255,255,255,.18)}
			.stats-bar{position:absolute;bottom:0;left:0;right:0;z-index:1;
			  background:rgba(15,23,42,.78);backdrop-filter:blur(20px);
			  border-top:1px solid rgba(255,255,255,.07)}
			.stats-inner{max-width:1200px;margin:0 auto;padding:26px 48px;
			  display:flex;align-items:center;gap:40px}
			.stat-v{font-family:'DM Serif Display',serif;font-size:34px;color:#fff;line-height:1}
			.stat-l{font-size:11px;color:rgba(255,255,255,.45);margin-top:4px;letter-spacing:.04em}
			.stat-div{height:40px;width:1px;background:rgba(255,255,255,.1);flex-shrink:0}
			.how-section{padding:88px 48px;max-width:1200px;margin:0 auto}
			.how-eyebrow{font-size:11px;font-weight:700;letter-spacing:.14em;text-transform:uppercase;color:#1D4ED8;margin-bottom:12px}
			.how-title{font-family:'DM Serif Display',serif;font-size:clamp(28px,4.5vw,48px);
			  line-height:1.12;margin-bottom:14px}
			.how-sub{font-size:15px;color:#64748B;max-width:440px;line-height:1.75;margin-bottom:56px}
			.steps{display:grid;grid-template-columns:repeat(3,1fr);gap:24px}
			.step{background:#fff;border-radius:20px;padding:34px 30px;border:1px solid #E2E8F0;
			  position:relative;overflow:hidden;transition:all .3s}
			.step:hover{transform:translateY(-4px);box-shadow:0 20px 56px rgba(15,23,42,.1);border-color:#1D4ED8}
			.step-num{font-family:'DM Serif Display',serif;font-size:80px;
			  color:rgba(29,78,216,.06);position:absolute;top:8px;right:14px;line-height:1}
			.step-icon{width:50px;height:50px;background:#EFF6FF;border-radius:14px;
			  display:flex;align-items:center;justify-content:center;margin-bottom:20px;font-size:24px}
			.step h3{font-family:'DM Serif Display',serif;font-size:22px;margin-bottom:10px}
			.step p{font-size:14px;color:#64748B;line-height:1.65}
			.cta-wrap{padding:0 48px 88px;max-width:1200px;margin:0 auto}
			.cta-banner{background:linear-gradient(135deg,#0F172A 0%,#1E3A8A 100%);
			  border-radius:28px;padding:64px;display:flex;align-items:center;
			  justify-content:space-between;gap:32px;position:relative;overflow:hidden}
			.cta-banner::before{content:'';position:absolute;width:500px;height:500px;
			  background:radial-gradient(circle,rgba(29,78,216,.25),transparent 65%);
			  top:-150px;right:-100px;border-radius:50%}
			.cta-banner-text{position:relative;z-index:1}
			.cta-banner-text h2{font-family:'DM Serif Display',serif;font-size:clamp(24px,3.5vw,40px);
			  color:#fff;margin-bottom:10px;line-height:1.15}
			.cta-banner-text p{color:rgba(255,255,255,.58);font-size:15px}
			@media(max-width:900px){
			  .hero-content{padding:100px 24px 180px}
			  .stats-inner{padding:20px 24px;gap:20px;overflow-x:auto}
			  .steps{grid-template-columns:1fr}
			  .how-section{padding:60px 24px}
			  .cta-wrap{padding:0 24px 60px}
			  .cta-banner{flex-direction:column;padding:40px 28px}
			}
		</style>
		<section class="hero">
			<div class="hero-bg"></div>
			<div class="hero-content">
				<div class="hero-eyebrow">
					<span class="ms ms-sm">location_on</span>
					Copiapó · Santiago · Todo Chile
				</div>
				<h1 class="hero-h1">Tu próxima<br/><em>propiedad</em><br/>está aquí.</h1>
				<p class="hero-sub">Accedé al catálogo completo de JCP. Iniciá sesión con Google y explorá propiedades en venta y arriendo.</p>
				<div class="hero-actions">
					<a href="/propiedades" class="btn-hero btn-hero-p">
						<span class="ms ms-sm">search</span>
						Ver propiedades
					</a>
					<a href="https://wa.me/56912345678" class="btn-hero btn-hero-s" target="_blank" rel="noopener">
						<span class="ms ms-sm">chat</span>
						WhatsApp
					</a>
				</div>
			</div>
			<div class="stats-bar">
				<div class="stats-inner">
					<div><div class="stat-v">48+</div><div class="stat-l">Propiedades activas</div></div>
					<div class="stat-div"></div>
					<div><div class="stat-v">12</div><div class="stat-l">Años de experiencia</div></div>
					<div class="stat-div"></div>
					<div><div class="stat-v">Venta</div><div class="stat-l">y Arriendo</div></div>
					<div class="stat-div"></div>
					<div><div class="stat-v">100%</div><div class="stat-l">Atención personalizada</div></div>
				</div>
			</div>
		</section>
		<div class="how-section">
			<p class="how-eyebrow">Cómo funciona</p>
			<h2 class="how-title">Encontrá tu propiedad<br/>en 3 pasos simples</h2>
			<p class="how-sub">Sin burocracia. Solo iniciás sesión, explorás el mapa y nos contactás.</p>
			<div class="steps">
				<div class="step">
					<span class="step-num">1</span>
					<div class="step-icon">🔑</div>
					<h3>Iniciá sesión con Google</h3>
					<p>Un clic con tu cuenta de Gmail. Sin formularios ni contraseñas extra. Acceso inmediato a todo el catálogo.</p>
				</div>
				<div class="step">
					<span class="step-num">2</span>
					<div class="step-icon">🗺️</div>
					<h3>Explorá en el mapa</h3>
					<p>Filtrá por tipo, precio y operación. Cada propiedad tiene su pin en el mapa para que ubiques la zona fácilmente.</p>
				</div>
				<div class="step">
					<span class="step-num">3</span>
					<div class="step-icon">💬</div>
					<h3>Contactanos directamente</h3>
					<p>¿Encontraste algo que te gusta? Agendá una visita o escribinos por WhatsApp. Te asesoramos sin compromiso.</p>
				</div>
			</div>
		</div>
		<div class="cta-wrap">
			<div class="cta-banner">
				<div class="cta-banner-text">
					<h2>¿Listo para encontrar<br/>tu próxima propiedad?</h2>
					<p>Iniciá sesión y accedé al catálogo completo de JCP.</p>
				</div>
				<a href="/propiedades" class="btn-hero btn-hero-p" style="position:relative;z-index:1;white-space:nowrap">
					Explorar propiedades →
				</a>
			</div>
		</div>
	}
}
```

- [ ] **Step 2: Add `LandingHandler` to `handlers.go`**

In `internal/handlers/web/handlers.go`, add after the existing `PropiedadesHandler`:

```go
// LandingHandler renders the marketing landing page.
func LandingHandler(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		return renderTempl(c, webtmpl.LandingPage())
	}
}
```

- [ ] **Step 3: Generate + build**

```bash
templ generate ./internal/templates/web/
go build ./...
```

Expected: `(✓) Complete` then no output.

- [ ] **Step 4: Commit**

```bash
git add internal/templates/web/landing.templ internal/templates/web/landing_templ.go internal/handlers/web/handlers.go
git commit -m "feat: landing page with hero, stats, how-it-works"
```

---

## Task 4: Login page

**Files:**
- Create: `internal/templates/web/login.templ`
- Modify: `internal/handlers/web/handlers.go` (add LoginPageHandler)

- [ ] **Step 1: Create `login.templ`**

```go
// internal/templates/web/login.templ
package web

templ LoginPage(errorMsg string) {
	<!DOCTYPE html>
	<html lang="es">
	<head>
		<meta charset="UTF-8"/>
		<meta name="viewport" content="width=device-width, initial-scale=1.0"/>
		<title>Iniciar sesión — JCP Gestión Inmobiliaria</title>
		<link rel="icon" type="image/svg+xml" href="/static/favicon.svg"/>
		<link rel="preconnect" href="https://fonts.googleapis.com"/>
		<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin/>
		<link href="https://fonts.googleapis.com/css2?family=DM+Serif+Display&family=DM+Sans:wght@300;400;500;600&family=Material+Symbols+Rounded:opsz,wght,FILL,GRAD@24,400,1,0&display=swap" rel="stylesheet"/>
		<style>
			*,*::before,*::after{box-sizing:border-box;margin:0;padding:0}
			body{font-family:'DM Sans',sans-serif;min-height:100vh;display:flex;
			  flex-direction:column;align-items:center;justify-content:center;padding:24px;
			  background:linear-gradient(155deg,rgba(15,23,42,.92) 0%,rgba(30,58,138,.78) 100%),
			    url('https://images.unsplash.com/photo-1560518883-ce09059eeffa?auto=format&fit=crop&w=1920&q=80') center/cover no-repeat fixed}
			.ms{font-family:'Material Symbols Rounded';font-variation-settings:'FILL' 1,'wght' 400,'GRAD' 0,'opsz' 24;
			  font-size:20px;line-height:1;display:inline-flex;align-items:center}
			.back{position:fixed;top:20px;left:20px;display:flex;align-items:center;gap:6px;
			  color:rgba(255,255,255,.55);font-size:13px;text-decoration:none;z-index:10;transition:color .2s}
			.back:hover{color:#fff}
			.card{width:100%;max-width:420px;background:rgba(255,255,255,.97);
			  border-radius:28px;padding:46px 42px;
			  box-shadow:0 32px 80px rgba(0,0,0,.35);
			  display:flex;flex-direction:column;align-items:center;text-align:center}
			.logo-wrap{display:flex;align-items:center;gap:10px;margin-bottom:32px}
			.logo-box{width:44px;height:44px;background:#1D4ED8;border-radius:12px;
			  display:flex;align-items:center;justify-content:center;color:#fff}
			.logo-name{font-family:'DM Serif Display',serif;font-size:22px;color:#0F172A}
			.card h1{font-family:'DM Serif Display',serif;font-size:26px;color:#0F172A;
			  margin-bottom:10px;line-height:1.2}
			.card p{font-size:14px;color:#64748B;line-height:1.65;max-width:300px;margin-bottom:32px}
			.divider{width:44px;height:2px;background:#1D4ED8;border-radius:2px;margin:0 auto 32px}
			.error-msg{width:100%;background:#FEE2E2;color:#991B1B;border-radius:10px;
			  padding:10px 14px;font-size:13px;margin-bottom:20px;text-align:left}
			.btn-google{display:flex;align-items:center;justify-content:center;gap:12px;
			  width:100%;padding:15px 24px;border-radius:14px;background:#fff;
			  border:2px solid #E2E8F0;font-size:15px;font-weight:600;color:#0F172A;
			  cursor:pointer;transition:all .22s;font-family:inherit;text-decoration:none;
			  box-shadow:0 2px 8px rgba(0,0,0,.06)}
			.btn-google:hover{border-color:#1D4ED8;box-shadow:0 4px 20px rgba(29,78,216,.18);transform:translateY(-1px)}
			.google-svg{width:20px;height:20px;flex-shrink:0}
			.or{display:flex;align-items:center;gap:12px;width:100%;margin:22px 0;
			  color:#94A3B8;font-size:12px}
			.or::before,.or::after{content:'';flex:1;height:1px;background:#E2E8F0}
			.btn-wa{display:flex;align-items:center;justify-content:center;gap:10px;
			  width:100%;padding:13px 24px;border-radius:14px;
			  background:#F0FDF4;border:1.5px solid #86EFAC;
			  font-size:14px;font-weight:600;color:#15803D;
			  cursor:pointer;transition:all .22s;font-family:inherit;text-decoration:none}
			.btn-wa:hover{background:#DCFCE7}
			.note{margin-top:24px;font-size:12px;color:#94A3B8;line-height:1.65;max-width:300px}
		</style>
	</head>
	<body>
		<a href="/" class="back">
			<span class="ms" style="font-size:16px">arrow_back</span>
			Volver al inicio
		</a>
		<div class="card">
			<div class="logo-wrap">
				<div class="logo-box"><span class="ms">home</span></div>
				<span class="logo-name">JCP Gestión</span>
			</div>
			<h1>Accedé al catálogo completo</h1>
			<p>Iniciá sesión con tu cuenta de Google para ver todas las propiedades disponibles.</p>
			<div class="divider"></div>
			if errorMsg != "" {
				<div class="error-msg">{ errorMsg }</div>
			}
			<a href="/auth/google" class="btn-google">
				<svg class="google-svg" viewBox="0 0 24 24">
					<path fill="#4285F4" d="M23.745 12.27c0-.79-.07-1.54-.19-2.27h-11.3v4.51h6.47c-.29 1.48-1.14 2.73-2.4 3.58v3h3.86c2.26-2.09 3.56-5.17 3.56-8.82z"/>
					<path fill="#34A853" d="M12.255 24c3.24 0 5.95-1.08 7.93-2.91l-3.86-3c-1.08.72-2.45 1.16-4.07 1.16-3.13 0-5.78-2.11-6.73-4.96h-3.98v3.09C3.515 21.3 7.615 24 12.255 24z"/>
					<path fill="#FBBC05" d="M5.525 14.29c-.25-.72-.38-1.49-.38-2.29s.14-1.57.38-2.29V6.62h-3.98a11.86 11.86 0 0 0 0 10.76l3.98-3.09z"/>
					<path fill="#EA4335" d="M12.255 4.75c1.77 0 3.35.61 4.6 1.8l3.42-3.42C18.205 1.19 15.495 0 12.255 0c-4.64 0-8.74 2.7-10.71 6.62l3.98 3.09c.95-2.85 3.6-4.96 6.73-4.96z"/>
				</svg>
				Continuar con Google
			</a>
			<div class="or">o también</div>
			<a href="https://wa.me/56912345678" class="btn-wa" target="_blank" rel="noopener">
				<span class="ms" style="font-size:18px">chat</span>
				Consultar por WhatsApp
			</a>
			<p class="note">Al iniciar sesión aceptás que JCP Gestión Inmobiliaria almacene tu correo para personalizar tu experiencia.</p>
		</div>
	</body>
	</html>
}
```

- [ ] **Step 2: Add `LoginPageHandler` to `handlers.go`**

```go
// LoginPageHandler serves the Google OAuth login page.
func LoginPageHandler(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		errMsg := ""
		switch c.Query("error") {
		case "state":
			errMsg = "Error de seguridad. Intentá de nuevo."
		case "exchange":
			errMsg = "No se pudo completar el inicio de sesión. Intentá de nuevo."
		}
		return renderTempl(c, webtmpl.LoginPage(errMsg))
	}
}
```

- [ ] **Step 3: Generate + build**

```bash
templ generate ./internal/templates/web/
go build ./...
```

- [ ] **Step 4: Commit**

```bash
git add internal/templates/web/login.templ internal/templates/web/login_templ.go internal/handlers/web/handlers.go
git commit -m "feat: login page with Google OAuth button"
```

---

## Task 5: Google OAuth handlers

**Files:**
- Create: `internal/handlers/web/auth.go`

- [ ] **Step 1: Create `auth.go`**

```go
// internal/handlers/web/auth.go
package web

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"io"
	"time"

	"jcp-gestioninmobiliaria/internal/auth"
	"jcp-gestioninmobiliaria/internal/config"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type googleUserInfo struct {
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
}

func googleOAuthConfig(cfg *config.Config) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     cfg.GoogleClientID,
		ClientSecret: cfg.GoogleClientSecret,
		RedirectURL:  cfg.GoogleRedirectURL,
		Scopes:       []string{"openid", "email", "profile"},
		Endpoint:     google.Endpoint,
	}
}

func randomState() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

// GoogleLogin redirects the user to Google's OAuth2 consent screen.
func GoogleLogin(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		state := randomState()
		c.Cookie(&fiber.Cookie{
			Name:     "oauth_state",
			Value:    state,
			MaxAge:   600,
			HTTPOnly: true,
			SameSite: "Lax",
			Secure:   cfg.IsProd(),
		})
		conf := googleOAuthConfig(cfg)
		return c.Redirect(conf.AuthCodeURL(state, oauth2.AccessTypeOnline))
	}
}

// GoogleCallback handles the OAuth2 callback from Google.
func GoogleCallback(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if c.Query("state") != c.Cookies("oauth_state") {
			return c.Redirect("/login?error=state")
		}
		c.ClearCookie("oauth_state")

		conf := googleOAuthConfig(cfg)
		token, err := conf.Exchange(context.Background(), c.Query("code"))
		if err != nil {
			return c.Redirect("/login?error=exchange")
		}

		client := conf.Client(context.Background(), token)
		resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
		if err != nil || resp.StatusCode != 200 {
			return c.Redirect("/login?error=exchange")
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)

		var gu googleUserInfo
		if err := json.Unmarshal(body, &gu); err != nil || gu.Email == "" {
			return c.Redirect("/login?error=exchange")
		}

		name := gu.Name
		if name == "" {
			name = gu.Email
		}

		jwtToken, err := auth.GenerateToken(cfg, "", gu.Email, "visitor", name)
		if err != nil {
			return c.Redirect("/login?error=exchange")
		}

		c.Cookie(&fiber.Cookie{
			Name:     "jcp_visitor",
			Value:    jwtToken,
			MaxAge:   int(cfg.JWTExpiration / time.Second),
			HTTPOnly: true,
			SameSite: "Lax",
			Secure:   cfg.IsProd(),
		})

		next := c.Query("next", "/propiedades")
		return c.Redirect(next)
	}
}

// VisitorLogout clears the visitor session and redirects to the landing page.
func VisitorLogout(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Cookie(&fiber.Cookie{
			Name:    "jcp_visitor",
			Value:   "",
			MaxAge:  -1,
			Expires: time.Unix(0, 0),
		})
		return c.Redirect("/")
	}
}
```

- [ ] **Step 2: Build to verify**

```bash
go build ./...
```

Expected: no output.

- [ ] **Step 3: Commit**

```bash
git add internal/handlers/web/auth.go
git commit -m "feat: Google OAuth2 login/callback/logout handlers"
```

---

## Task 6: Visitor auth middleware

**Files:**
- Create: `internal/middleware/visitor.go`

- [ ] **Step 1: Create `visitor.go`**

```go
// internal/middleware/visitor.go
package middleware

import (
	"net/url"

	"jcp-gestioninmobiliaria/internal/auth"
	"jcp-gestioninmobiliaria/internal/config"

	"github.com/gofiber/fiber/v2"
)

// VisitorAuthRequired redirects unauthenticated visitors to /login.
func VisitorAuthRequired(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tokenStr := c.Cookies("jcp_visitor")
		if tokenStr == "" {
			return c.Redirect("/login?next=" + url.QueryEscape(c.Path()))
		}
		claims, err := auth.ValidateToken(cfg, tokenStr)
		if err != nil {
			c.ClearCookie("jcp_visitor")
			return c.Redirect("/login?next=" + url.QueryEscape(c.Path()))
		}
		c.Locals("visitor_email", claims.Email)
		c.Locals("visitor_name", claims.Nombre)
		return c.Next()
	}
}
```

- [ ] **Step 2: Build to verify**

```bash
go build ./...
```

- [ ] **Step 3: Commit**

```bash
git add internal/middleware/visitor.go
git commit -m "feat: VisitorAuthRequired middleware for visitor sessions"
```

---

## Task 7: Update routes

**Files:**
- Modify: `internal/routes/public.go`

- [ ] **Step 1: Replace `public.go` content**

```go
// internal/routes/public.go
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
	app.Get("/", web.LandingHandler(cfg))
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
	app.Get("/auth/google", web.GoogleLogin(cfg))
	app.Get("/auth/google/callback", web.GoogleCallback(cfg))
	app.Get("/auth/logout", web.VisitorLogout(cfg))

	// Protected visitor routes
	visitor := app.Group("", middleware.VisitorAuthRequired(cfg))
	visitor.Get("/propiedades", web.PropiedadesHandler(cfg))
	visitor.Get("/propiedades/:key", web.PropiedadHandler(cfg, pb))

	// WebSocket
	app.Use("/ws", func(c *fiber.Ctx) error {
		if gows.IsWebSocketUpgrade(c) {
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})
	app.Get("/ws/web", gows.New(ws.WebSocket(hub)))
}
```

- [ ] **Step 2: Build to verify**

```bash
go build ./...
```

Expected: no output.

- [ ] **Step 3: Commit**

```bash
git add internal/routes/public.go
git commit -m "feat: protect /propiedades routes with VisitorAuthRequired"
```

---

## Task 8: Redesign propiedades.templ (app shell)

**Files:**
- Modify: `internal/templates/web/propiedades.templ`
- Modify: `internal/handlers/web/handlers.go` (update PropiedadesHandler signature)

- [ ] **Step 1: Update `PropiedadesHandler` to inject user info**

In `internal/handlers/web/handlers.go`, replace the `PropiedadesHandler` function:

```go
func PropiedadesHandler(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		name, _ := c.Locals("visitor_name").(string)
		email, _ := c.Locals("visitor_email").(string)
		if name == "" {
			name = "Visitante"
		}
		return renderTempl(c, webtmpl.PropiedadesPage(name, email))
	}
}
```

- [ ] **Step 2: Rewrite `propiedades.templ`**

Replace the entire content of `internal/templates/web/propiedades.templ`:

```go
package web

import "jcp-gestioninmobiliaria/internal/templates/layouts"

templ PropiedadesPage(userName string, userEmail string) {
	@layouts.AppShell("Propiedades — JCP Gestión Inmobiliaria", true, layouts.AppUser{Name: userName, Email: userEmail}) {
		<style>
			.topbar{height:60px;background:#fff;border-bottom:1px solid #E2E8F0;
			  display:flex;align-items:center;gap:12px;padding:0 20px;flex-shrink:0}
			.tb-search{flex:1;max-width:480px;position:relative}
			.tb-search .ms{position:absolute;left:11px;top:50%;transform:translateY(-50%);
			  color:#64748B;font-size:18px;pointer-events:none}
			.tb-input{width:100%;padding:9px 12px 9px 36px;border:1.5px solid #E2E8F0;
			  border-radius:10px;font-size:13.5px;font-family:inherit;outline:none;
			  background:#F8FAFC;color:#111827;transition:border-color .2s}
			.tb-input:focus{border-color:#1D4ED8;background:#fff}
			.tb-chips{display:flex;gap:8px}
			.tb-chip{padding:7px 16px;border-radius:9999px;font-size:12.5px;font-weight:500;
			  border:1.5px solid #E2E8F0;background:#fff;color:#64748B;
			  cursor:pointer;transition:all .18s;font-family:inherit}
			.tb-chip:hover{border-color:#1D4ED8;color:#1D4ED8}
			.tb-chip.active{background:#1D4ED8;color:#fff;border-color:#1D4ED8}
			.tb-right{margin-left:auto;display:flex;align-items:center;gap:10px}
			.tb-count{font-size:12px;color:#64748B;white-space:nowrap}
			.filter-bar{background:#fff;border-bottom:1px solid #E2E8F0;
			  padding:10px 20px;display:flex;gap:8px;align-items:center;flex-shrink:0;overflow-x:auto}
			.fl{font-size:11px;font-weight:700;text-transform:uppercase;letter-spacing:.08em;
			  color:#64748B;white-space:nowrap;margin-right:4px}
			.fsel{padding:6px 12px;border:1.5px solid #E2E8F0;border-radius:8px;
			  font-size:12.5px;font-family:inherit;background:#fff;color:#111827;
			  cursor:pointer;outline:none;transition:border-color .18s}
			.fsel:focus{border-color:#1D4ED8}
			.fdiv{width:1px;height:22px;background:#E2E8F0;margin:0 4px;flex-shrink:0}
			.sort-wrap{margin-left:auto;display:flex;align-items:center;gap:6px;flex-shrink:0}
			.sort-label{font-size:11px;color:#64748B;white-space:nowrap}
			.content-split{flex:1;display:flex;overflow:hidden}
			.listings-panel{width:54%;min-width:0;overflow-y:auto;padding:20px}
			.listings-info{display:flex;align-items:center;justify-content:space-between;
			  margin-bottom:16px;padding:0 2px}
			.listings-info h2{font-family:'DM Serif Display',Georgia,serif;font-size:20px}
			.listings-info span{font-size:12px;color:#64748B}
			.prop-grid{display:grid;grid-template-columns:repeat(auto-fill,minmax(200px,1fr));gap:14px}
			.prop-card{background:#fff;border-radius:14px;overflow:hidden;
			  border:1px solid #E2E8F0;cursor:pointer;
			  transition:transform .25s,box-shadow .25s,border-color .2s;position:relative}
			.prop-card:hover,.prop-card.map-highlighted{transform:translateY(-3px);
			  box-shadow:0 12px 36px rgba(15,23,42,.1);border-color:#1D4ED8}
			.prop-card-link{text-decoration:none;color:inherit;display:block}
			.prop-media{aspect-ratio:4/3;overflow:hidden;background:#E2E8F0;position:relative}
			.prop-media img{width:100%;height:100%;object-fit:cover;transition:transform .4s}
			.prop-card:hover .prop-media img{transform:scale(1.04)}
			.prop-img-placeholder{display:flex;align-items:center;justify-content:center;
			  height:100%;color:#94A3B8;font-size:40px}
			.prop-badges{position:absolute;top:10px;left:10px;display:flex;gap:5px}
			.prop-badge{font-size:10px;font-weight:700;padding:4px 9px;border-radius:6px}
			.badge-v{background:rgba(29,78,216,.9);color:#fff}
			.badge-a{background:rgba(124,58,237,.9);color:#fff}
			.badge-dest{background:rgba(217,119,6,.9);color:#fff}
			.badge-deal{background:rgba(5,150,105,.9);color:#fff}
			.prop-body{padding:12px 14px 14px}
			.prop-price{font-family:'DM Serif Display',Georgia,serif;font-size:22px;
			  color:#111827;line-height:1.1;margin-bottom:4px}
			.prop-price-sub{font-size:11px;color:#94A3B8;font-weight:400}
			.prop-feats{display:flex;flex-wrap:wrap;gap:10px;margin:6px 0 8px;
			  color:#64748B;font-size:11.5px}
			.prop-feat{display:inline-flex;align-items:center;gap:3px}
			.prop-title{font-size:13px;font-weight:500;color:#111827;line-height:1.35;
			  display:-webkit-box;-webkit-line-clamp:2;-webkit-box-orient:vertical;overflow:hidden;margin-bottom:5px}
			.prop-loc{font-size:11.5px;color:#64748B;display:flex;align-items:center;gap:3px}
			.prop-loc .ms{color:#1D4ED8;font-size:13px}
			.loading-state{text-align:center;padding:60px 0;color:#94A3B8;font-size:14px}
			.map-panel{flex:1;border-left:1px solid #E2E8F0;position:relative}
			#prop-map{height:100%;width:100%}
			@media(max-width:900px){
			  .topbar{padding:0 12px}
			  .tb-chips{display:none}
			  .content-split{flex-direction:column}
			  .listings-panel{width:100%}
			  .map-panel{height:260px;border-left:none;border-top:1px solid #E2E8F0}
			}
		</style>
		<header class="topbar">
			<div class="tb-search">
				<span class="ms">search</span>
				<input class="tb-input" type="text" id="tb-q" placeholder="Comuna, dirección, barrio..."
					hx-get="/fragments/propiedades-page" hx-target="#prop-results"
					hx-trigger="input changed delay:400ms"
					hx-include="[name='operacion'],[name='tipo'],[name='dormitorios'],[name='sort']"
					name="q" autocomplete="off"/>
			</div>
			<div class="tb-chips">
				<button class="tb-chip active" type="button" data-op="">Todas</button>
				<button class="tb-chip" type="button" data-op="VENTA">Comprar</button>
				<button class="tb-chip" type="button" data-op="ARRIENDO">Arrendar</button>
			</div>
			<div class="tb-right">
				<span class="tb-count" id="tb-count">Cargando...</span>
			</div>
			<input type="hidden" name="operacion" id="op-hidden" value=""/>
		</header>
		<div class="filter-bar">
			<span class="fl">Filtrar</span>
			<select class="fsel" name="tipo"
				hx-get="/fragments/propiedades-page" hx-target="#prop-results"
				hx-trigger="change" hx-include="[name='operacion'],[name='q'],[name='dormitorios'],[name='sort']">
				<option value="">Todos los tipos</option>
				<option value="CASA">Casa</option>
				<option value="DEPARTAMENTO">Departamento</option>
				<option value="PARCELA">Parcela</option>
				<option value="TERRENO">Terreno</option>
				<option value="OFICINA">Oficina</option>
				<option value="LOCAL">Local comercial</option>
				<option value="BODEGA">Bodega</option>
			</select>
			<select class="fsel" name="dormitorios" id="dorm-sel"
				hx-get="/fragments/propiedades-page" hx-target="#prop-results"
				hx-trigger="change" hx-include="[name='operacion'],[name='tipo'],[name='q'],[name='sort']">
				<option value="">Dormitorios</option>
				<option value="1">1+</option>
				<option value="2">2+</option>
				<option value="3">3+</option>
				<option value="4">4+</option>
			</select>
			<div class="fdiv"></div>
			<div class="sort-wrap">
				<span class="sort-label">Ordenar:</span>
				<select class="fsel" name="sort" id="sort-select"
					hx-get="/fragments/propiedades-page" hx-target="#prop-results"
					hx-trigger="change"
					hx-include="[name='operacion'],[name='tipo'],[name='q'],[name='dormitorios']">
					<option value="recientes">Más recientes</option>
					<option value="precio_asc">Precio ↑</option>
					<option value="precio_desc">Precio ↓</option>
					<option value="superficie">Mayor superficie</option>
				</select>
			</div>
		</div>
		<div class="content-split">
			<div class="listings-panel">
				<div class="listings-info">
					<h2>Propiedades disponibles</h2>
					<span id="prop-count-label">Actualizado en tiempo real</span>
				</div>
				<div id="prop-results" class="prop-grid"
					hx-get="/fragments/propiedades-page"
					hx-trigger="load"
					hx-swap="innerHTML">
					<div class="loading-state">Cargando propiedades...</div>
				</div>
			</div>
			<div class="map-panel">
				<div id="prop-map" data-map="listing"></div>
			</div>
		</div>
		<script>
			(function(){
			  document.querySelectorAll('.tb-chip').forEach(function(btn){
			    btn.addEventListener('click',function(){
			      document.querySelectorAll('.tb-chip').forEach(function(b){b.classList.remove('active')});
			      btn.classList.add('active');
			      document.getElementById('op-hidden').value=btn.dataset.op||'';
			      htmx.ajax('GET','/fragments/propiedades-page',{
			        target:'#prop-results',swap:'innerHTML',
			        values:{operacion:btn.dataset.op||'',tipo:document.querySelector('[name=tipo]').value,
			          q:document.getElementById('tb-q').value,dormitorios:document.querySelector('[name=dormitorios]').value,
			          sort:document.getElementById('sort-select').value}});
			    });
			  });
			  var p=new URLSearchParams(location.search);
			  if(p.get('operacion')){
			    document.getElementById('op-hidden').value=p.get('operacion');
			    document.querySelectorAll('.tb-chip').forEach(function(b){b.classList.toggle('active',b.dataset.op===p.get('operacion'))});
			  }
			  if(p.get('tipo'))document.querySelector('[name=tipo]').value=p.get('tipo');
			  if(p.get('q'))document.getElementById('tb-q').value=p.get('q');
			}());
		</script>
	}
}
```

- [ ] **Step 3: Generate + build**

```bash
templ generate ./internal/templates/web/
go build ./...
```

- [ ] **Step 4: Commit**

```bash
git add internal/templates/web/propiedades.templ internal/templates/web/propiedades_templ.go internal/handlers/web/handlers.go
git commit -m "feat: propiedades app shell with sidebar, topbar, filter bar"
```

---

## Task 9: Redesign propiedad.templ (detail + calendar)

**Files:**
- Modify: `internal/templates/web/propiedad.templ`
- Modify: `internal/handlers/web/handlers.go` (update PropiedadHandler)

- [ ] **Step 1: Add user fields to `PropiedadData` struct**

In `internal/templates/web/propiedad.templ`, add two fields to `PropiedadData`:

```go
type PropiedadData struct {
	Titulo        string
	Direccion     string
	PriceHTML     template.HTML
	PriceSub      string
	ChipsHTML     template.HTML
	CoverHTML     template.HTML
	ThumbsHTML    template.HTML
	FeatsHTML     template.HTML
	BodyHTML      template.HTML
	AmenitiesHTML template.HTML
	WhatsappHTML  template.HTML
	WhatsappPhone string
	Lat           float64
	Lng           float64
	Comuna        string
	UserName      string
	UserEmail     string
}
```

- [ ] **Step 2: Change `PropiedadDetail` to use AppShell**

Replace the `templ PropiedadDetail(d PropiedadData)` function signature line:

```go
templ PropiedadDetail(d PropiedadData) {
	@layouts.AppShell(d.Titulo+" — JCP Gestión Inmobiliaria", true, layouts.AppUser{Name: d.UserName, Email: d.UserEmail}) {
```

(Keep the rest of the function body but update the styles and structure — see steps below.)

- [ ] **Step 3: Add calendar section CSS + HTML to `propiedad.templ`**

Inside the `PropiedadDetail` body, after the existing contact sidebar content and before the `</style>` closing tag, add this CSS:

```css
			.schedule-section{background:#fff;border-radius:16px;border:1px solid #E2E8F0;padding:28px;margin-top:24px}
			.schedule-title{font-family:'DM Serif Display',Georgia,serif;font-size:22px;margin-bottom:6px}
			.schedule-sub{font-size:13px;color:#64748B;margin-bottom:22px}
			.schedule-form{display:flex;flex-direction:column;gap:14px}
			.sched-row{display:grid;grid-template-columns:1fr 1fr;gap:12px}
			.sched-field{display:flex;flex-direction:column;gap:5px}
			.sched-field label{font-size:12px;font-weight:600;color:#374151;letter-spacing:.02em}
			.sched-input{padding:11px 14px;border:1.5px solid #E2E8F0;border-radius:10px;
			  font-size:14px;font-family:inherit;outline:none;color:#111827;transition:border-color .2s}
			.sched-input:focus{border-color:#1D4ED8}
			.sched-btn{display:flex;align-items:center;justify-content:center;gap:8px;
			  padding:14px 24px;background:#25D366;color:#fff;border:none;
			  border-radius:9999px;font-size:14px;font-weight:700;cursor:pointer;
			  font-family:inherit;transition:all .22s;margin-top:4px}
			.sched-btn:hover{background:#1DAB54;transform:translateY(-1px);box-shadow:0 4px 16px rgba(37,211,102,.35)}
```

Add the schedule HTML inside the detail main column, after the description section and before `if d.Lat != 0 && d.Lng != 0`:

```go
			<div class="schedule-section">
				<h3 class="schedule-title">Agendar una visita</h3>
				<p class="schedule-sub">Elegí fecha y horario y te confirmamos por WhatsApp.</p>
				<form class="schedule-form" id="schedule-form">
					<div class="sched-row">
						<div class="sched-field">
							<label>Fecha</label>
							<input class="sched-input" type="date" name="fecha" id="sched-date" required/>
						</div>
						<div class="sched-field">
							<label>Hora</label>
							<select class="sched-input" name="hora" id="sched-time">
								<option value="09:00">09:00 AM</option>
								<option value="10:00">10:00 AM</option>
								<option value="11:00">11:00 AM</option>
								<option value="12:00">12:00 PM</option>
								<option value="14:00">02:00 PM</option>
								<option value="15:00">03:00 PM</option>
								<option value="16:00">04:00 PM</option>
								<option value="17:00">05:00 PM</option>
							</select>
						</div>
					</div>
					<div class="sched-field">
						<label>Tu nombre</label>
						<input class="sched-input" type="text" name="nombre" placeholder="Juan Carlos" required/>
					</div>
					<div class="sched-field">
						<label>Tu teléfono</label>
						<input class="sched-input" type="tel" name="telefono" placeholder="+56 9 1234 5678"/>
					</div>
					<button type="submit" class="sched-btn">
						<span class="ms ms-sm ms-fill">calendar_month</span> Confirmar visita por WhatsApp
					</button>
				</form>
			</div>
```

Add this JS at the bottom of the `PropiedadDetail` template, before the closing `}`:

```go
		<script>
			(function(){
			  var df=document.getElementById('sched-date');
			  if(df){df.min=new Date().toISOString().slice(0,10);}
			  var sf=document.getElementById('schedule-form');
			  var propTitle={ d.Titulo };
			  var waPhone={ d.WhatsappPhone };
			  if(sf){
			    sf.addEventListener('submit',function(e){
			      e.preventDefault();
			      var fecha=sf.fecha.value,hora=sf.hora.value,nombre=sf.nombre.value.trim(),tel=sf.telefono.value.trim();
			      var text='Hola! Quiero agendar una visita para: '+propTitle+'\nFecha: '+fecha+'\nHora: '+hora;
			      if(nombre)text+='\nNombre: '+nombre;
			      if(tel)text+='\nTeléfono: '+tel;
			      var phone=waPhone||'56912345678';
			      window.open('https://wa.me/'+phone+'?text='+encodeURIComponent(text),'_blank');
			    });
			  }
			}());
		</script>
```

**Important:** Replace `{ d.Titulo }` and `{ d.WhatsappPhone }` with proper Templ JS string injection using `templ.JSExpression` or inline them with concatenation. The safe way in templ is:

```go
		<script>
			(function(){
			  var df=document.getElementById('sched-date');
			  if(df){df.min=new Date().toISOString().slice(0,10);}
			  var sf=document.getElementById('schedule-form');
			  if(sf){
			    sf.addEventListener('submit',function(e){
			      e.preventDefault();
			      var fecha=sf.fecha.value,hora=sf.hora.value;
			      var nombre=sf.nombre.value.trim(),tel=sf.telefono.value.trim();
			      var propTitle=sf.closest('[data-prop-title]')? sf.closest('[data-prop-title]').dataset.propTitle : document.title;
			      var waPhone=sf.dataset.waPhone||'56912345678';
			      var text='Hola! Quiero agendar una visita para: '+propTitle+'\nFecha: '+fecha+'\nHora: '+hora;
			      if(nombre)text+='\nNombre: '+nombre;
			      if(tel)text+='\nTeléfono: '+tel;
			      window.open('https://wa.me/'+waPhone+'?text='+encodeURIComponent(text),'_blank');
			    });
			  }
			}());
		</script>
```

And set data attributes on the schedule section element:
```go
			<div class="schedule-section" data-prop-title={ d.Titulo }>
			  <form class="schedule-form" id="schedule-form" data-wa-phone={ d.WhatsappPhone }>
```

- [ ] **Step 4: Update `PropiedadHandler` to inject user info**

In `internal/handlers/web/handlers.go`, in `PropiedadHandler`, add these two lines before `data := webtmpl.PropiedadData{`:

```go
		visitorName, _ := c.Locals("visitor_name").(string)
		visitorEmail, _ := c.Locals("visitor_email").(string)
		if visitorName == "" {
			visitorName = "Visitante"
		}
```

And add these fields to the `data` struct:
```go
		data := webtmpl.PropiedadData{
			// ... existing fields ...
			WhatsappPhone: onlyDigits(whatsapp),
			UserName:      visitorName,
			UserEmail:     visitorEmail,
		}
```

Also remove the `whatsappHTML` WhatsApp button construction that uses `onlyDigits(whatsapp)` as the destination (it's now directly in `WhatsappPhone`). The existing `WhatsappHTML` still has the full button HTML — keep it as-is for the sidebar.

- [ ] **Step 5: Generate + build**

```bash
templ generate ./internal/templates/web/
go build ./...
```

- [ ] **Step 6: Commit**

```bash
git add internal/templates/web/propiedad.templ internal/templates/web/propiedad_templ.go internal/handlers/web/handlers.go
git commit -m "feat: property detail with app shell + schedule visit calendar"
```

---

## Task 10: Update footer + base layout nav

**Files:**
- Modify: `internal/templates/layouts/base.templ`

- [ ] **Step 1: Update footer bottom bar**

In `base.templ`, replace the `.footer-bottom` content:

```go
			<div class="footer-bottom">
				<span>© 2026 JCP Gestión Inmobiliaria · Todos los derechos reservados</span>
				<span>Desarrollado por <a href="https://diwe.cl" target="_blank" rel="noopener" style="color:rgba(255,255,255,.55);text-decoration:none;transition:color .2s" onmouseover="this.style.color='#fff'" onmouseout="this.style.color='rgba(255,255,255,.55)'">Digital Wellness</a></span>
			</div>
```

- [ ] **Step 2: Update nav links to point to `/propiedades`**

Replace the three `<a>` tags inside `.nav-links`:

```go
					<nav class="nav-links">
						<a href="/propiedades?operacion=VENTA">Comprar</a>
						<a href="/propiedades?operacion=ARRIENDO">Arrendar</a>
						<a href="/propiedades">Todas las propiedades</a>
					</nav>
```

And replace the WhatsApp nav-cta with a login button:

```go
					<a href="/propiedades" class="nav-cta">
						<span class="ms ms-sm">search</span> Ver propiedades
					</a>
```

- [ ] **Step 3: Generate + build**

```bash
templ generate ./internal/templates/layouts/
go build ./...
```

- [ ] **Step 4: Commit**

```bash
git add internal/templates/layouts/base.templ internal/templates/layouts/base_templ.go
git commit -m "fix: footer 2026 + Digital Wellness credit + nav links to /propiedades"
```

---

## Task 11: CSP update for Google OAuth

**Files:**
- Modify: `internal/middleware/security.go`

- [ ] **Step 1: Update CSP `connect-src` and `img-src`**

In `internal/middleware/security.go`, replace the `Content-Security-Policy` value:

```go
		c.Set("Content-Security-Policy",
			"default-src 'self'; "+
				"script-src 'self' unpkg.com cdn.jsdelivr.net 'unsafe-inline'; "+
				"style-src 'self' 'unsafe-inline' fonts.googleapis.com unpkg.com; "+
				"img-src 'self' data: blob: https:; "+
				"connect-src 'self' accounts.google.com; "+
				"font-src 'self' data: fonts.googleapis.com fonts.gstatic.com;",
		)
```

- [ ] **Step 2: Build + run local smoke test**

```bash
go build ./...
go run . serve &
sleep 2
curl -I http://localhost:3000/ | grep -i "content-security"
kill %1
```

Expected: `Content-Security-Policy` header present with `accounts.google.com` in `connect-src`.

- [ ] **Step 3: Commit**

```bash
git add internal/middleware/security.go
git commit -m "fix: CSP allow accounts.google.com for OAuth2 flow"
```

---

## Task 12: Deploy + verify

- [ ] **Step 1: Final build**

```bash
go build ./...
```

Expected: no output.

- [ ] **Step 2: Push to origin/main**

```bash
git push origin main
```

- [ ] **Step 3: Verify Google OAuth env vars are set in Fly.io**

```bash
fly secrets list
```

Ensure `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`, `GOOGLE_REDIRECT_URL` are set. If not:

```bash
fly secrets set GOOGLE_CLIENT_ID="your-id.apps.googleusercontent.com" \
  GOOGLE_CLIENT_SECRET="your-secret" \
  GOOGLE_REDIRECT_URL="https://your-app.fly.dev/auth/google/callback"
```

**Note:** In Google Cloud Console, add `https://your-app.fly.dev/auth/google/callback` as an Authorized Redirect URI for the OAuth2 client.

- [ ] **Step 4: Smoke test the full flow**

1. Visit `/` → landing page with hero ✓
2. Visit `/propiedades` → redirects to `/login?next=%2Fpropiedades` ✓
3. Click "Continuar con Google" → Google consent ✓
4. After login → `/propiedades` with sidebar showing user name/email ✓
5. Click a property → detail page with sidebar + calendar ✓
6. Fill calendar form → WhatsApp opens with pre-filled message ✓
7. Click logout → cookie cleared, redirected to `/` ✓
