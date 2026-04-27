# Mapas Leaflet.js + Frontend Polish Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Agregar mapas Leaflet.js interactivos al listado y detalle de propiedades, migrar las vistas públicas a Templ, unificar paleta JCP azul, Material Symbols Rounded, y agregar hero de marca.

**Architecture:** Un único `maps.js` gestiona todos los mapas vía atributos `data-map`/`data-lat`/`data-lng`. Las páginas públicas pasan de archivos `.html` estáticos a componentes Templ compilados. Los handlers de Fiber llaman a `component.Render(c.Context(), c.Response().BodyWriter())`.

**Tech Stack:** Go 1.23, Fiber v2.52.5, Templ v0.3.x, PocketBase v0.25, HTMX 1.9.10, Leaflet.js 1.9.4 (CDN), Material Symbols Rounded (CDN), DM Serif Display + DM Sans (CDN).

---

## File Map

| Acción | Archivo |
|--------|---------|
| Crear | `web/static/js/maps.js` |
| Crear | `internal/templates/layouts/base.templ` + `base_templ.go` |
| Crear | `internal/templates/web/propiedades.templ` + `propiedades_templ.go` |
| Crear | `internal/templates/fragments/propcard.templ` + `propcard_templ.go` |
| Crear | `internal/templates/web/propiedad.templ` + `propiedad_templ.go` |
| Modificar | `internal/handlers/web/handlers.go` |
| Modificar | `internal/handlers/fragments/propiedades.go` |
| Modificar | `internal/routes/public.go` |
| Modificar | `internal/handlers/admin/handlers.go` |
| Modificar | `Dockerfile` |
| Eliminar | `web/propiedades.html` |
| Eliminar | `internal/templates/web/propiedad.html` |

---

## Task 1: Instalar Templ y actualizar Dockerfile

**Files:**
- Modify: `go.mod`, `go.sum`
- Modify: `Dockerfile`

- [ ] **Step 1: Instalar Templ en el módulo Go**

```bash
cd /Users/matiaspereira/jcp-gestioninmobiliaria/jcp-gestioninmobiliaria
go get github.com/a-h/templ@latest
```

Expected: `go.mod` agrega `github.com/a-h/templ vX.Y.Z` y `go.sum` se actualiza.

- [ ] **Step 2: Instalar el CLI de Templ**

```bash
go install github.com/a-h/templ/cmd/templ@latest
```

Expected: `templ version` devuelve algo como `v0.3.x`.

- [ ] **Step 3: Verificar que el CLI está disponible**

```bash
templ version
```

Expected: imprime la versión sin error.

- [ ] **Step 4: Actualizar Dockerfile para incluir templ generate**

Reemplazar el bloque de BUILD en `Dockerfile`:

```dockerfile
# ====================== BUILD STAGE ======================
FROM golang:1.23-bookworm AS builder

WORKDIR /usr/src/app

COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Install templ CLI for code generation
RUN go install github.com/a-h/templ/cmd/templ@latest

COPY . .

# Generate Templ components
RUN templ generate

RUN go build -v -o /run-app ./cmd/server

# ====================== RUNTIME STAGE ======================
FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y ca-certificates tzdata && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /run-app /app/run-app
COPY --from=builder /usr/src/app/web /app/web
COPY --from=builder /usr/src/app/internal/templates /app/internal/templates

RUN mkdir -p /app/web/static /app/pb_data

EXPOSE 3000

CMD ["/app/run-app", "serve"]
```

- [ ] **Step 5: Commit**

```bash
git add go.mod go.sum Dockerfile
git commit -m "build: add templ dependency and generate step in Dockerfile"
```

---

## Task 2: Crear maps.js

**Files:**
- Create: `web/static/js/maps.js`

- [ ] **Step 1: Crear directorio si no existe**

```bash
mkdir -p /Users/matiaspereira/jcp-gestioninmobiliaria/jcp-gestioninmobiliaria/web/static/js
```

- [ ] **Step 2: Crear `web/static/js/maps.js`**

```js
(function () {
  'use strict';

  var JCP_BLUE = '#1D4ED8';
  var listingMap = null;
  var pinLayer = null;

  function makePin(color, size) {
    size = size || 28;
    return L.divIcon({
      className: '',
      html:
        '<div style="width:' + size + 'px;height:' + size + 'px;' +
        'border-radius:50% 50% 50% 0;transform:rotate(-45deg);' +
        'background:' + color + ';border:2.5px solid white;' +
        'box-shadow:0 3px 10px rgba(0,0,0,.28)"></div>',
      iconSize: [size, size],
      iconAnchor: [size / 2, size],
      popupAnchor: [0, -size],
    });
  }

  function makeBigPin(color) {
    return L.divIcon({
      className: '',
      html:
        '<div style="width:40px;height:40px;' +
        'border-radius:50% 50% 50% 0;transform:rotate(-45deg);' +
        'background:' + color + ';border:3px solid white;' +
        'box-shadow:0 4px 14px rgba(29,78,216,.4)">' +
        '<div style="transform:rotate(45deg);width:12px;height:12px;' +
        'background:white;border-radius:50%;margin:auto;margin-top:10px"></div>' +
        '</div>',
      iconSize: [40, 40],
      iconAnchor: [20, 40],
      popupAnchor: [0, -44],
    });
  }

  function tileLayer() {
    return L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
      attribution: '© <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a>',
      maxZoom: 19,
    });
  }

  // ── Listing map ──────────────────────────────────────────────────
  function initListingMap() {
    var el = document.getElementById('prop-map');
    if (!el || listingMap) return;

    listingMap = L.map('prop-map', { zoomControl: true }).setView([-33.45, -70.65], 10);
    tileLayer().addTo(listingMap);
    pinLayer = L.layerGroup().addTo(listingMap);
    syncListingPins();
  }

  function syncListingPins() {
    if (!listingMap || !pinLayer) return;
    pinLayer.clearLayers();

    var cards = document.querySelectorAll('[data-lat][data-lng]');
    var bounds = [];

    cards.forEach(function (card) {
      var lat = parseFloat(card.dataset.lat);
      var lng = parseFloat(card.dataset.lng);
      if (!lat || !lng) return;

      var title = card.dataset.title || '';
      var price = card.dataset.price || '';
      var tipo  = card.dataset.tipo  || '';
      var slug  = card.dataset.slug  || card.dataset.id || '#';
      var commune = card.dataset.commune || '';

      var marker = L.marker([lat, lng], { icon: makePin(JCP_BLUE) });
      marker.bindPopup(
        '<div class="map-popup">' +
        '<div class="map-popup-type">' + tipo + '</div>' +
        '<div class="map-popup-title">' + title + '</div>' +
        '<div class="map-popup-price">' + price + '</div>' +
        (commune ? '<div class="map-popup-loc"><span class="ms ms-sm">location_on</span>' + commune + '</div>' : '') +
        '<a href="/propiedades/' + slug + '" class="map-popup-link">Ver detalle</a>' +
        '</div>',
        { maxWidth: 220 }
      );

      // Highlight card on popup open
      marker.on('popupopen', function () {
        document.querySelectorAll('.prop-card').forEach(function (c) { c.classList.remove('map-highlighted'); });
        card.classList.add('map-highlighted');
        card.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
      });
      marker.on('popupclose', function () {
        card.classList.remove('map-highlighted');
      });

      pinLayer.addLayer(marker);
      bounds.push([lat, lng]);
    });

    if (bounds.length > 0) {
      listingMap.fitBounds(bounds, { padding: [40, 40], maxZoom: 13 });
    }
  }

  // ── Detail map ───────────────────────────────────────────────────
  function initDetailMap() {
    var el = document.querySelector('[data-map="detail"]');
    if (!el) return;
    var lat = parseFloat(el.dataset.lat);
    var lng = parseFloat(el.dataset.lng);
    if (!lat || !lng) return;

    var map = L.map(el, { scrollWheelZoom: false }).setView([lat, lng], 14);
    tileLayer().addTo(map);
    L.marker([lat, lng], { icon: makeBigPin(JCP_BLUE) })
      .addTo(map)
      .bindPopup(el.dataset.title || 'Propiedad')
      .openPopup();
  }

  function initDetailMiniMap() {
    var el = document.querySelector('[data-map="detail-mini"]');
    if (!el) return;
    var lat = parseFloat(el.dataset.lat);
    var lng = parseFloat(el.dataset.lng);
    if (!lat || !lng) return;

    var map = L.map(el, {
      zoomControl: false,
      attributionControl: false,
      dragging: false,
      scrollWheelZoom: false,
    }).setView([lat, lng], 13);
    tileLayer().addTo(map);
    L.circle([lat, lng], {
      color: JCP_BLUE, fillColor: '#3B82F6', fillOpacity: 0.15, radius: 800, weight: 2,
    }).addTo(map);
    L.marker([lat, lng], { icon: makeBigPin(JCP_BLUE) }).addTo(map);
  }

  // ── Init ─────────────────────────────────────────────────────────
  function init() {
    initListingMap();
    initDetailMap();
    initDetailMiniMap();
  }

  document.addEventListener('DOMContentLoaded', init);

  // Re-sync listing pins after every HTMX swap (new cards loaded)
  document.addEventListener('htmx:afterSettle', function () {
    if (listingMap) syncListingPins();
  });

  // Expose for inline card hover (optional enhancement)
  window.JCPMaps = { syncListingPins: syncListingPins };
}());
```

- [ ] **Step 3: Verificar que el archivo existe**

```bash
ls -la /Users/matiaspereira/jcp-gestioninmobiliaria/jcp-gestioninmobiliaria/web/static/js/maps.js
```

Expected: archivo de ~120 líneas.

- [ ] **Step 4: Commit**

```bash
git add web/static/js/maps.js
git commit -m "feat: add maps.js for Leaflet listing/detail init via data-map attributes"
```

---

## Task 3: Crear base.templ (layout compartido)

**Files:**
- Create: `internal/templates/layouts/base.templ`
- Create: `internal/templates/layouts/base_templ.go` (generado por templ)

- [ ] **Step 1: Crear directorio**

```bash
mkdir -p /Users/matiaspereira/jcp-gestioninmobiliaria/jcp-gestioninmobiliaria/internal/templates/layouts
```

- [ ] **Step 2: Crear `internal/templates/layouts/base.templ`**

```templ
package layouts

templ Base(title string, includeMaps bool) {
	<!DOCTYPE html>
	<html lang="es">
	<head>
		<meta charset="UTF-8"/>
		<meta name="viewport" content="width=device-width, initial-scale=1.0"/>
		<title>{ title }</title>
		<link rel="icon" type="image/svg+xml" href="/static/favicon.svg"/>
		<link rel="preconnect" href="https://fonts.googleapis.com"/>
		<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin/>
		<link href="https://fonts.googleapis.com/css2?family=DM+Serif+Display:ital@0;1&family=DM+Sans:ital,opsz,wght@0,9..40,300;0,9..40,400;0,9..40,500;0,9..40,600;1,9..40,300&family=Material+Symbols+Rounded:opsz,wght,FILL,GRAD@24,400,1,0&display=swap" rel="stylesheet"/>
		if includeMaps {
			<link rel="stylesheet" href="https://unpkg.com/leaflet@1.9.4/dist/leaflet.css"/>
		}
		<script src="https://unpkg.com/htmx.org@1.9.10/dist/htmx.min.js" defer></script>
		if includeMaps {
			<script src="https://unpkg.com/leaflet@1.9.4/dist/leaflet.js"></script>
			<script src="/static/js/maps.js" defer></script>
		}
		<style>
			:root{
			  --jcp-primary:#1D4ED8;--jcp-primary-dark:#1E3A8A;--jcp-primary-light:#3B82F6;
			  --jcp-primary-50:#EFF6FF;--jcp-primary-100:#DBEAFE;
			  --surface:#FAFAF9;--surface-bright:#FFFFFF;
			  --surface-container-low:#F5F5F3;--surface-container-high:#EEEEE9;
			  --on-surface:#111827;--on-surface-var:#4B5563;
			  --outline:#9CA3AF;--outline-var:#E5E7EB;
			  --inverse-surface:#0F1419;
			  --r-sm:12px;--r-md:16px;--r-lg:24px;--r-xl:32px;--r-full:9999px;
			  --elev-2:0 2px 6px rgba(15,20,25,.08);
			  --elev-3:0 4px 12px rgba(15,20,25,.10);
			  --elev-4:0 8px 24px rgba(15,20,25,.14);
			  --ease-express:cubic-bezier(0.05,0.7,0.1,1.0);
			  --font-display:'DM Serif Display',Georgia,serif;
			  --font-body:'DM Sans',system-ui,sans-serif;
			  --nav-h:68px;--max-w:1280px;
			}
			*,*::before,*::after{box-sizing:border-box;margin:0;padding:0}
			html{scroll-behavior:smooth;-webkit-font-smoothing:antialiased}
			body{font-family:var(--font-body);background:var(--surface);color:var(--on-surface);line-height:1.6}
			.container{max-width:var(--max-w);margin:0 auto;padding:0 24px}
			/* Material Symbols */
			.ms{font-family:'Material Symbols Rounded';font-variation-settings:'FILL' 1,'wght' 400,'GRAD' 0,'opsz' 24;font-size:20px;line-height:1;display:inline-flex;align-items:center;user-select:none}
			.ms-sm{font-size:16px}.ms-lg{font-size:28px}.ms-xl{font-size:40px}
			.ms-outline{font-variation-settings:'FILL' 0,'wght' 300,'GRAD' 0,'opsz' 24}
			/* NAV */
			.nav{position:sticky;top:0;z-index:200;height:var(--nav-h);
			  background:rgba(250,250,249,.90);backdrop-filter:blur(20px);
			  -webkit-backdrop-filter:blur(20px);border-bottom:1px solid var(--outline-var)}
			.nav.scrolled{box-shadow:0 4px 24px rgba(15,20,25,.08)}
			.nav-inner{max-width:var(--max-w);margin:0 auto;padding:0 24px;height:100%;
			  display:flex;align-items:center;justify-content:space-between;gap:16px}
			.nav-brand{display:flex;align-items:center;gap:10px;text-decoration:none}
			.nav-logo-box{width:38px;height:38px;background:var(--jcp-primary);border-radius:10px;
			  display:flex;align-items:center;justify-content:center;color:white;flex-shrink:0}
			.nav-brand-name{font-family:var(--font-display);font-size:18px;color:var(--on-surface);line-height:1.1}
			.nav-brand-sub{font-size:10px;font-weight:500;letter-spacing:.12em;color:var(--outline);text-transform:uppercase}
			.nav-links{display:flex;align-items:center;gap:2px}
			.nav-links a{font-size:13.5px;font-weight:500;color:var(--on-surface-var);text-decoration:none;
			  padding:8px 14px;border-radius:var(--r-full);transition:background 200ms,color 200ms}
			.nav-links a:hover,.nav-links a.active{background:var(--jcp-primary-100);color:var(--jcp-primary-dark)}
			.nav-cta{font-size:13px;font-weight:600;color:#fff;background:var(--jcp-primary);
			  padding:10px 22px;border-radius:var(--r-full);text-decoration:none;
			  box-shadow:0 2px 8px rgba(29,78,216,.28);transition:all 200ms;
			  display:flex;align-items:center;gap:6px}
			.nav-cta:hover{background:var(--jcp-primary-dark);transform:translateY(-1px)}
			/* MAP POPUP */
			.leaflet-popup-content-wrapper{border-radius:12px!important;box-shadow:0 8px 30px rgba(0,0,0,.14)!important}
			.map-popup{padding:4px;min-width:170px}
			.map-popup-type{font-size:10px;font-weight:700;color:var(--jcp-primary);text-transform:uppercase;letter-spacing:.3px}
			.map-popup-title{font-size:13px;font-weight:700;color:var(--on-surface);margin:2px 0}
			.map-popup-price{font-size:14px;font-weight:800;color:var(--jcp-primary)}
			.map-popup-loc{font-size:11px;color:var(--on-surface-var);display:flex;align-items:center;gap:3px;margin-top:2px}
			.map-popup-link{display:block;margin-top:8px;padding:6px 12px;background:var(--jcp-primary);
			  color:white;border-radius:8px;text-align:center;text-decoration:none;font-size:12px;font-weight:600}
			/* FOOTER */
			footer{background:var(--inverse-surface);color:rgba(255,255,255,0.72);padding:64px 24px 32px;margin-top:64px}
			.footer-grid{display:grid;grid-template-columns:1.5fr 1fr 1fr 1fr;gap:40px;max-width:var(--max-w);margin:0 auto}
			.footer-brand{font-family:var(--font-display);font-size:20px;color:#fff;margin-bottom:10px}
			.footer-sub{font-size:13px;opacity:.6;line-height:1.65}
			.footer-col h4{font-size:12px;font-weight:600;letter-spacing:.1em;text-transform:uppercase;
			  color:rgba(255,255,255,.45);margin-bottom:16px}
			.footer-col a{display:block;font-size:13px;color:rgba(255,255,255,.7);text-decoration:none;margin-bottom:8px;transition:color 200ms}
			.footer-col a:hover{color:#fff}
			.footer-bottom{max-width:var(--max-w);margin:40px auto 0;padding-top:24px;border-top:1px solid rgba(255,255,255,.1);
			  display:flex;justify-content:space-between;font-size:12px;color:rgba(255,255,255,.4);flex-wrap:wrap;gap:8px}
			@media(max-width:768px){
			  :root{--nav-h:58px}
			  .nav-links{display:none}
			  .footer-grid{grid-template-columns:1fr}
			}
		</style>
	</head>
	<body>
		<nav class="nav" id="main-nav">
			<div class="nav-inner">
				<a href="/" class="nav-brand">
					<div class="nav-logo-box"><span class="ms" style="font-size:22px;">home</span></div>
					<div>
						<div class="nav-brand-name">JCP Gestión</div>
						<div class="nav-brand-sub">Inmobiliaria</div>
					</div>
				</a>
				<nav class="nav-links">
					<a href="/propiedades.html?operacion=VENTA">Comprar</a>
					<a href="/propiedades.html?operacion=ARRIENDO">Arrendar</a>
					<a href="/propiedades.html">Todas las propiedades</a>
				</nav>
				<a href="https://wa.me/56912345678" class="nav-cta" target="_blank" rel="noopener">
					<span class="ms ms-sm">chat</span> WhatsApp
				</a>
			</div>
		</nav>
		{ children... }
		<footer>
			<div class="footer-grid">
				<div>
					<div class="footer-brand">JCP Gestión Inmobiliaria</div>
					<div class="footer-sub">Compra, venta y arriendo de propiedades en Chile.<br/>Corredores con experiencia comprobada.</div>
				</div>
				<div class="footer-col">
					<h4>Operaciones</h4>
					<a href="/propiedades.html?operacion=VENTA">Comprar</a>
					<a href="/propiedades.html?operacion=ARRIENDO">Arrendar</a>
				</div>
				<div class="footer-col">
					<h4>Tipos</h4>
					<a href="/propiedades.html?tipo=CASA">Casas</a>
					<a href="/propiedades.html?tipo=DEPARTAMENTO">Departamentos</a>
					<a href="/propiedades.html?tipo=PARCELA">Parcelas</a>
					<a href="/propiedades.html?tipo=OFICINA">Oficinas</a>
				</div>
				<div class="footer-col">
					<h4>Contacto</h4>
					<a href="mailto:contacto@jcp-inmobiliaria.cl">contacto@jcp-inmobiliaria.cl</a>
					<a href="tel:+56912345678">+56 9 1234 5678</a>
					<a href="#">Santiago · Chile</a>
				</div>
			</div>
			<div class="footer-bottom">
				<span>© 2026 JCP Gestión Inmobiliaria · Todos los derechos reservados</span>
				<span>Go · PocketBase · Fiber · HTMX</span>
			</div>
		</footer>
		<script>
			window.addEventListener('scroll',function(){
			  document.getElementById('main-nav').classList.toggle('scrolled',window.scrollY>10);
			},{passive:true});
		</script>
	</body>
	</html>
}
```

- [ ] **Step 3: Generar el archivo Go**

```bash
cd /Users/matiaspereira/jcp-gestioninmobiliaria/jcp-gestioninmobiliaria
templ generate ./internal/templates/layouts/
```

Expected: crea `internal/templates/layouts/base_templ.go` sin errores.

- [ ] **Step 4: Verificar compilación**

```bash
go build ./...
```

Expected: sin errores.

- [ ] **Step 5: Commit**

```bash
git add internal/templates/layouts/
git commit -m "feat: add base.templ layout with JCP tokens, Material Symbols, Leaflet CDN"
```

---

## Task 4: Crear propcard.templ (card fragment)

**Files:**
- Create: `internal/templates/fragments/propcard.templ`
- Create: `internal/templates/fragments/propcard_templ.go` (generado)

- [ ] **Step 1: Crear directorio**

```bash
mkdir -p /Users/matiaspereira/jcp-gestioninmobiliaria/jcp-gestioninmobiliaria/internal/templates/fragments
```

- [ ] **Step 2: Crear `internal/templates/fragments/propcard.templ`**

```templ
package fragments

import "fmt"

type PropItem struct {
	ID         string
	Titulo     string
	Slug       string
	Operacion  string
	Tipo       string
	Comuna     string
	Region     string
	PrecioUF   float64
	PrecioCLP  float64
	Dormitorios int
	Banos       int
	Estac       int
	SupUtil    float64
	SupTotal   float64
	Destacada  bool
	Oportunidad bool
	CoverImage string
	Lat        float64
	Lng        float64
	PriceLabel string
}

func fmtCoord(f float64) string {
	if f == 0 {
		return ""
	}
	return fmt.Sprintf("%.6f", f)
}

templ PropCard(p PropItem, i int) {
	<article
		class={ "prop-card reveal visible", templ.KV("reveal-delay-1", i%4==1), templ.KV("reveal-delay-2", i%4==2), templ.KV("reveal-delay-3", i%4==3) }
		data-id={ p.ID }
		data-lat={ fmtCoord(p.Lat) }
		data-lng={ fmtCoord(p.Lng) }
		data-title={ p.Titulo }
		data-price={ p.PriceLabel }
		data-tipo={ p.Tipo + " · " + p.Operacion }
		data-slug={ func() string { if p.Slug != "" { return p.Slug }; return p.ID }() }
		data-commune={ p.Comuna }
	>
		<a href={ templ.SafeURL("/propiedades/" + func() string { if p.Slug != "" { return p.Slug }; return p.ID }()) } class="prop-card-link" aria-label={ "Ver " + p.Titulo }>
			<div class="prop-media">
				if p.CoverImage != "" {
					<img src={ p.CoverImage } alt={ p.Titulo } loading="lazy"/>
				} else {
					<div class="prop-img-placeholder">
						<span class="ms ms-xl ms-outline" style="color:var(--outline);">home</span>
					</div>
				}
				<div class="prop-badges">
					if p.Destacada {
						<span class="prop-badge prop-badge-featured"><span class="ms ms-sm">star</span> Destacada</span>
					}
					if p.Oportunidad {
						<span class="prop-badge prop-badge-deal">Oportunidad</span>
					}
					if p.Operacion == "VENTA" {
						<span class="prop-badge prop-badge-venta">En Venta</span>
					} else if p.Operacion == "ARRIENDO" {
						<span class="prop-badge prop-badge-arriendo">En Arriendo</span>
					}
				</div>
				<button class="prop-fav" type="button" aria-label="Guardar propiedad" onclick="event.preventDefault();event.stopPropagation();this.classList.toggle('is-active')">
					<span class="ms ms-sm ms-outline">favorite</span>
				</button>
			</div>
			<div class="prop-body">
				<div class="prop-price">@templ.Raw(p.PriceLabel)</div>
				<div class="prop-feats">
					if p.Dormitorios > 0 {
						<span class="prop-feat"><span class="ms ms-sm">bed</span> { fmt.Sprintf("%d dorm", p.Dormitorios) }</span>
					}
					if p.Banos > 0 {
						<span class="prop-feat"><span class="ms ms-sm">shower</span> { fmt.Sprintf("%d baño%s", p.Banos, func() string { if p.Banos == 1 { return "" }; return "s" }()) }</span>
					}
					if p.SupUtil > 0 {
						<span class="prop-feat"><span class="ms ms-sm">square_foot</span> { fmt.Sprintf("%g m²", p.SupUtil) }</span>
					} else if p.SupTotal > 0 {
						<span class="prop-feat"><span class="ms ms-sm">square_foot</span> { fmt.Sprintf("%g m²", p.SupTotal) }</span>
					}
					if p.Estac > 0 {
						<span class="prop-feat"><span class="ms ms-sm">directions_car</span> { fmt.Sprintf("%d", p.Estac) }</span>
					}
				</div>
				<h3 class="prop-title">{ p.Titulo }</h3>
				<p class="prop-loc">
					<span class="ms ms-sm">location_on</span>
					{ func() string {
						loc := p.Comuna
						if p.Region != "" && p.Comuna != "" { loc += " · " + p.Region }
						return loc
					}() }
				</p>
			</div>
		</a>
	</article>
}
```

- [ ] **Step 3: Generar**

```bash
cd /Users/matiaspereira/jcp-gestioninmobiliaria/jcp-gestioninmobiliaria
templ generate ./internal/templates/fragments/
```

Expected: crea `propcard_templ.go` sin errores.

- [ ] **Step 4: Verificar compilación**

```bash
go build ./...
```

Expected: sin errores.

- [ ] **Step 5: Commit**

```bash
git add internal/templates/fragments/
git commit -m "feat: add PropCard Templ component with data-lat/lng map attributes"
```

---

## Task 5: Actualizar propiedades.go para usar PropCard templ

**Files:**
- Modify: `internal/handlers/fragments/propiedades.go`

El fragmento handler sigue construyendo el HTML envolvente (grid, load-more button), pero delega cada card a `fragments.PropCard`.

- [ ] **Step 1: Reemplazar `renderPropCard` y actualizar `fetchPropiedades`**

En `internal/handlers/fragments/propiedades.go`, realizar estos cambios:

**a) Agregar imports necesarios** — en el bloque `import`:
```go
import (
    "bytes"
    "context"
    "fmt"
    "html/template"
    "net/url"
    "strings"

    "jcp-gestioninmobiliaria/internal/config"
    proptempl "jcp-gestioninmobiliaria/internal/templates/fragments"

    "github.com/gofiber/fiber/v2"
    "github.com/pocketbase/pocketbase"
)
```

**b) Cambiar el tipo `propiedad` para que exporte un `PriceLabel`** — agregar campo al struct:
```go
type propiedad struct {
    // ... campos existentes sin cambios ...
    PriceLabel string  // ← agregar al final
}
```

**c) En `fetchPropiedades`, poblar `PriceLabel` al construir cada item:**
```go
p := propiedad{
    // ... campos existentes ...
    PriceLabel: priceLabel(propiedad{ PrecioUF: r.GetFloat("precio_uf"), PrecioCLP: r.GetFloat("precio_clp") }),
}
```

En realidad es más limpio calcular el label después de construir la struct. Reemplazar el loop dentro de `fetchPropiedades`:
```go
item := propiedad{
    ID:          r.Id,
    Titulo:      r.GetString("titulo"),
    Slug:        r.GetString("slug"),
    Descripcion: r.GetString("descripcion"),
    Operacion:   r.GetString("operacion"),
    Tipo:        r.GetString("tipo"),
    Direccion:   r.GetString("direccion"),
    Comuna:      r.GetString("comuna"),
    Region:      r.GetString("region"),
    PrecioUF:    r.GetFloat("precio_uf"),
    PrecioCLP:   r.GetFloat("precio_clp"),
    Dormitorios: r.GetInt("dormitorios"),
    Banos:       r.GetInt("banos"),
    Estac:       r.GetInt("estacionamientos"),
    SupUtil:     r.GetFloat("superficie_util"),
    SupTotal:    r.GetFloat("superficie_total"),
    Ano:         r.GetInt("ano_construccion"),
    Estado:      r.GetString("estado_propiedad"),
    Amenidades:  amen,
    Destacada:   r.GetBool("destacada"),
    Oportunidad: r.GetBool("oportunidad"),
    CoverImage:  r.GetString("cover_image"),
    Gallery:     gal,
    TourURL:     r.GetString("tour_url"),
    Lat:         r.GetFloat("lat"),
    Lng:         r.GetFloat("lng"),
    FechaPublicado: date,
    Whatsapp:    r.GetString("contacto_whatsapp"),
}
item.PriceLabel = priceLabel(item)
out = append(out, item)
```

**d) Agregar función `toPropItem` que convierte `propiedad` → `proptempl.PropItem`:**
```go
func toPropItem(p propiedad) proptempl.PropItem {
    return proptempl.PropItem{
        ID:          p.ID,
        Titulo:      p.Titulo,
        Slug:        p.Slug,
        Operacion:   p.Operacion,
        Tipo:        p.Tipo,
        Comuna:      p.Comuna,
        Region:      p.Region,
        PrecioUF:    p.PrecioUF,
        PrecioCLP:   p.PrecioCLP,
        Dormitorios: p.Dormitorios,
        Banos:       p.Banos,
        Estac:       p.Estac,
        SupUtil:     p.SupUtil,
        SupTotal:    p.SupTotal,
        Destacada:   p.Destacada,
        Oportunidad: p.Oportunidad,
        CoverImage:  p.CoverImage,
        Lat:         p.Lat,
        Lng:         p.Lng,
        PriceLabel:  p.PriceLabel,
    }
}
```

**e) Agregar función `renderCard` que usa Templ:**
```go
func renderCard(p propiedad, i int) string {
    var buf bytes.Buffer
    if err := proptempl.PropCard(toPropItem(p), i).Render(context.Background(), &buf); err != nil {
        return ""
    }
    return buf.String()
}
```

**f) Eliminar la función `renderPropCard` completa** (ya no se usa).

**g) En `PropiedadesDestacadas` y `PropiedadesPage`, reemplazar toda llamada a `renderPropCard(p, i)` por `renderCard(p, i)`.**

- [ ] **Step 2: Verificar compilación**

```bash
cd /Users/matiaspereira/jcp-gestioninmobiliaria/jcp-gestioninmobiliaria
go build ./...
```

Expected: sin errores. Si hay `undefined: renderPropCard` es porque faltó reemplazar alguna llamada en `PropiedadesDestacadas`.

- [ ] **Step 3: Prueba rápida**

```bash
go run ./cmd/server serve &
sleep 3
curl -s http://localhost:3000/fragments/propiedades-page | grep -c "data-lat"
kill %1
```

Expected: imprime un número > 0 (las cards tienen `data-lat`).

- [ ] **Step 4: Commit**

```bash
git add internal/handlers/fragments/propiedades.go
git commit -m "feat: propiedades fragment delegates card render to PropCard templ component"
```

---

## Task 6: Crear propiedades.templ (página de listado)

**Files:**
- Create: `internal/templates/web/propiedades.templ`
- Create: `internal/templates/web/propiedades_templ.go` (generado)

- [ ] **Step 1: Crear `internal/templates/web/propiedades.templ`**

```templ
package web

import "jcp-gestioninmobiliaria/internal/templates/layouts"

templ PropiedadesPage() {
	@layouts.Base("Propiedades — JCP Gestión Inmobiliaria", true) {
		<!-- HERO DE MARCA -->
		<div class="brand-hero">
			<div class="brand-hero-inner">
				<div class="brand-hero-logo">
					<span class="ms" style="font-size:32px;color:white;">home</span>
				</div>
				<div class="brand-hero-text">
					<h1 class="brand-hero-title">JCP Gestión Inmobiliaria</h1>
					<p class="brand-hero-desc">Compra, arrienda y vende propiedades en todo Chile con corredores de confianza.</p>
				</div>
				<div class="brand-hero-ctas">
					<a href="https://wa.me/56912345678" class="brand-cta-wa" target="_blank" rel="noopener">
						<span class="ms ms-sm">chat</span> Consultar por WhatsApp
					</a>
					<a href="#prop-results" class="brand-cta-secondary">
						<span class="ms ms-sm">search</span> Ver propiedades
					</a>
				</div>
			</div>
		</div>
		<!-- BUSCADOR -->
		<div class="search-wrap">
			<div class="container">
				<form class="search-bar"
					hx-get="/fragments/propiedades-page"
					hx-target="#prop-results"
					hx-trigger="submit, change from:select"
					hx-swap="innerHTML"
					hx-push-url="true">
					<div class="tabs" role="tablist">
						<button type="button" class="tab active" data-op="">Todas</button>
						<button type="button" class="tab" data-op="VENTA">Comprar</button>
						<button type="button" class="tab" data-op="ARRIENDO">Arrendar</button>
					</div>
					<div class="search-input-wrap">
						<span class="ms ms-sm search-icon">search</span>
						<input type="search" name="q" class="search-input" placeholder="Comuna, dirección, barrio..."
							hx-get="/fragments/propiedades-page" hx-target="#prop-results"
							hx-trigger="input changed delay:400ms" hx-include="[name='operacion'],[name='tipo']" autocomplete="off"/>
					</div>
					<select name="tipo">
						<option value="">Todos los tipos</option>
						<option value="CASA">Casa</option>
						<option value="DEPARTAMENTO">Departamento</option>
						<option value="PARCELA">Parcela</option>
						<option value="TERRENO">Terreno</option>
						<option value="OFICINA">Oficina</option>
						<option value="LOCAL">Local comercial</option>
						<option value="BODEGA">Bodega</option>
					</select>
					<input type="hidden" name="operacion" id="op-hidden" value=""/>
					<button type="submit" class="btn-search"><span class="ms ms-sm">search</span> Buscar</button>
				</form>
			</div>
		</div>
		<!-- FILTROS -->
		<div class="filters">
			<div class="container">
				<div class="filter-row">
					<span class="filter-label">Dormitorios</span>
					<button class="prop-chip" type="button" data-filter="dormitorios" data-value="">Cualquiera</button>
					<button class="prop-chip" type="button" data-filter="dormitorios" data-value="1">1+</button>
					<button class="prop-chip" type="button" data-filter="dormitorios" data-value="2">2+</button>
					<button class="prop-chip" type="button" data-filter="dormitorios" data-value="3">3+</button>
					<button class="prop-chip" type="button" data-filter="dormitorios" data-value="4">4+</button>
					<div class="sort-wrap">
						<label for="sort-select">Ordenar:</label>
						<select id="sort-select" name="sort"
							hx-get="/fragments/propiedades-page"
							hx-target="#prop-results"
							hx-trigger="change"
							hx-include="[name='operacion'],[name='tipo'],[name='q'],[name='dormitorios']">
							<option value="recientes">Más recientes</option>
							<option value="precio_asc">Precio ↑</option>
							<option value="precio_desc">Precio ↓</option>
							<option value="superficie">Mayor superficie</option>
						</select>
					</div>
				</div>
			</div>
		</div>
		<!-- LAYOUT: MAPA + RESULTADOS -->
		<div class="listings-layout">
			<div class="listings-col">
				<div class="listings-head container">
					<h2 class="listings-heading">Propiedades disponibles</h2>
					<span class="listings-count">Actualizado en tiempo real</span>
				</div>
				<input type="hidden" name="dormitorios" id="dorm-hidden" value=""/>
				<div id="prop-results" class="container"
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
		<style>
			/* HERO DE MARCA */
			.brand-hero{background:linear-gradient(135deg,#0F172A 0%,#1E3A8A 100%);
			  padding:clamp(28px,5vw,48px) 24px;color:white;position:relative;overflow:hidden}
			.brand-hero::before{content:'';position:absolute;inset:0;
			  background:radial-gradient(ellipse at top right,rgba(59,130,246,.2),transparent 60%);pointer-events:none}
			.brand-hero-inner{max-width:var(--max-w);margin:0 auto;position:relative;z-index:1;
			  display:flex;align-items:center;gap:20px;flex-wrap:wrap}
			.brand-hero-logo{width:56px;height:56px;background:rgba(255,255,255,.12);
			  border:1px solid rgba(255,255,255,.2);border-radius:16px;
			  display:flex;align-items:center;justify-content:center;flex-shrink:0;
			  backdrop-filter:blur(8px)}
			.brand-hero-text{flex:1;min-width:200px}
			.brand-hero-title{font-family:var(--font-display);font-size:clamp(22px,3vw,30px);
			  line-height:1.15;margin-bottom:4px}
			.brand-hero-desc{font-size:14px;color:rgba(255,255,255,.72);max-width:500px}
			.brand-hero-ctas{display:flex;gap:10px;flex-wrap:wrap}
			.brand-cta-wa{display:flex;align-items:center;gap:6px;padding:10px 20px;
			  background:#25D366;color:white;border-radius:var(--r-full);
			  font-size:13px;font-weight:700;text-decoration:none;transition:all 200ms}
			.brand-cta-wa:hover{background:#1DAB54;transform:translateY(-1px)}
			.brand-cta-secondary{display:flex;align-items:center;gap:6px;padding:10px 20px;
			  background:rgba(255,255,255,.12);backdrop-filter:blur(8px);
			  color:white;border:1px solid rgba(255,255,255,.2);border-radius:var(--r-full);
			  font-size:13px;font-weight:600;text-decoration:none;transition:all 200ms}
			.brand-cta-secondary:hover{background:rgba(255,255,255,.2)}
			@media(max-width:600px){
			  .brand-hero-inner{flex-direction:column;align-items:flex-start;gap:14px}
			  .brand-hero-ctas{width:100%}
			  .brand-cta-wa,.brand-cta-secondary{flex:1;justify-content:center}
			}
			/* BUSCADOR */
			.search-wrap{background:var(--surface-bright);border-bottom:1px solid var(--outline-var);
			  padding:16px 0;position:sticky;top:var(--nav-h);z-index:150;
			  box-shadow:0 2px 8px rgba(15,20,25,.06)}
			.search-bar{display:grid;grid-template-columns:auto 1fr auto auto;
			  gap:8px;align-items:center;background:var(--surface-bright)}
			.search-bar .tabs{display:flex;gap:4px;padding:4px;background:var(--surface-container-low);border-radius:var(--r-full)}
			.search-bar .tab{font-size:12.5px;font-weight:500;padding:7px 16px;border:none;
			  background:transparent;color:var(--on-surface-var);border-radius:var(--r-full);cursor:pointer;font-family:inherit;transition:all 180ms}
			.search-bar .tab.active{background:var(--jcp-primary);color:#fff}
			.search-input-wrap{position:relative}
			.search-icon{position:absolute;left:12px;top:50%;transform:translateY(-50%);color:var(--outline)}
			.search-input{width:100%;padding:11px 12px 11px 36px;border:none;outline:none;
			  font-size:14px;font-family:inherit;color:var(--on-surface);background:transparent}
			.search-bar select{border:1px solid var(--outline-var);outline:none;padding:11px 12px;
			  font-size:13px;background:var(--surface-container-low);border-radius:var(--r-md);
			  cursor:pointer;font-family:inherit}
			.btn-search{background:var(--jcp-primary);color:#fff;border:none;border-radius:var(--r-full);
			  padding:11px 22px;font-size:13px;font-weight:600;cursor:pointer;font-family:inherit;
			  transition:all 180ms;display:flex;align-items:center;gap:6px}
			.btn-search:hover{background:var(--jcp-primary-dark)}
			@media(max-width:900px){.search-bar{grid-template-columns:1fr}}
			/* FILTROS */
			.filters{padding:14px 0;border-bottom:1px solid var(--outline-var);
			  background:var(--surface-bright);position:sticky;
			  top:calc(var(--nav-h) + 60px);z-index:140}
			.filter-row{display:flex;gap:8px;overflow-x:auto;padding:2px 0;align-items:center;flex-wrap:wrap}
			.filter-label{font-size:11px;font-weight:600;letter-spacing:.1em;text-transform:uppercase;
			  color:var(--outline);margin-right:4px;white-space:nowrap}
			.prop-chip{font-size:13px;font-weight:500;padding:7px 16px;border-radius:var(--r-full);
			  border:1.5px solid var(--outline-var);background:var(--surface-bright);
			  color:var(--on-surface-var);cursor:pointer;white-space:nowrap;
			  transition:all 200ms var(--ease-express);font-family:var(--font-body)}
			.prop-chip:hover{border-color:var(--jcp-primary);color:var(--jcp-primary)}
			.prop-chip.active{background:var(--jcp-primary);color:#fff;border-color:var(--jcp-primary)}
			.sort-wrap{margin-left:auto;display:flex;align-items:center;gap:8px;font-size:13px;color:var(--outline)}
			.sort-wrap select{padding:7px 12px;border:1.5px solid var(--outline-var);
			  border-radius:var(--r-full);font-size:13px;font-family:inherit;
			  background:var(--surface-bright);cursor:pointer;outline:none}
			/* LISTINGS LAYOUT — split with map */
			.listings-layout{display:flex;height:calc(100vh - var(--nav-h) - 118px);min-height:600px}
			.listings-col{width:54%;overflow-y:auto;display:flex;flex-direction:column}
			.listings-head{display:flex;justify-content:space-between;align-items:baseline;
			  padding:20px 0 12px}
			.listings-heading{font-family:var(--font-display);font-size:22px}
			.listings-count{font-size:13px;color:var(--outline)}
			.map-panel{width:46%;border-left:1px solid var(--outline-var);position:relative}
			#prop-map{height:100%;width:100%}
			.loading-state{text-align:center;padding:60px 0;color:var(--outline);font-size:14px}
			/* Card CSS */
			.prop-grid{display:grid;grid-template-columns:repeat(auto-fill,minmax(280px,1fr));gap:20px;padding-bottom:32px}
			.prop-card{background:var(--surface-bright);border-radius:var(--r-lg);overflow:hidden;
			  border:1px solid var(--outline-var);
			  transition:transform 300ms var(--ease-express),box-shadow 300ms var(--ease-express),border-color 200ms;
			  position:relative}
			.prop-card:hover,.prop-card.map-highlighted{transform:translateY(-4px);box-shadow:var(--elev-4);border-color:var(--jcp-primary)}
			.prop-card-link{text-decoration:none;color:inherit;display:block}
			.prop-media{position:relative;aspect-ratio:4/3;overflow:hidden;background:var(--surface-container-high)}
			.prop-media img{width:100%;height:100%;object-fit:cover;transition:transform 400ms var(--ease-express)}
			.prop-card:hover .prop-media img{transform:scale(1.04)}
			.prop-img-placeholder{display:flex;align-items:center;justify-content:center;height:100%;color:var(--outline)}
			.prop-badges{position:absolute;top:12px;left:12px;display:flex;gap:6px;flex-wrap:wrap}
			.prop-badge{font-size:11px;font-weight:600;padding:5px 10px;border-radius:var(--r-sm);
			  letter-spacing:.02em;backdrop-filter:blur(8px);display:flex;align-items:center;gap:4px}
			.prop-badge-featured{background:rgba(217,119,6,.95);color:#fff}
			.prop-badge-deal{background:rgba(5,150,105,.95);color:#fff}
			.prop-badge-venta{background:rgba(29,78,216,.95);color:#fff}
			.prop-badge-arriendo{background:rgba(124,58,237,.95);color:#fff}
			.prop-fav{position:absolute;top:12px;right:12px;width:34px;height:34px;
			  background:rgba(255,255,255,.95);border:none;border-radius:50%;
			  display:flex;align-items:center;justify-content:center;cursor:pointer;
			  color:var(--on-surface-var);box-shadow:0 2px 6px rgba(0,0,0,.15);transition:all 200ms}
			.prop-fav:hover{color:var(--jcp-primary)}
			.prop-fav.is-active .ms{font-variation-settings:'FILL' 1,'wght' 400,'GRAD' 0,'opsz' 24;color:var(--jcp-primary)}
			.prop-body{padding:14px 16px 16px}
			.prop-price{font-family:var(--font-display);font-size:22px;color:var(--on-surface);line-height:1.1;margin-bottom:4px}
			.prop-price-clp{font-family:var(--font-body);font-size:12px;color:var(--outline);font-weight:400}
			.prop-feats{display:flex;flex-wrap:wrap;gap:10px;margin:8px 0 10px;
			  color:var(--on-surface-var);font-size:12.5px}
			.prop-feat{display:inline-flex;align-items:center;gap:4px;white-space:nowrap}
			.prop-feat .ms{color:var(--outline)}
			.prop-title{font-size:14px;font-weight:500;color:var(--on-surface);line-height:1.35;
			  display:-webkit-box;-webkit-line-clamp:2;-webkit-box-orient:vertical;overflow:hidden;margin-bottom:6px}
			.prop-loc{font-size:12px;color:var(--outline);display:inline-flex;align-items:center;gap:4px}
			.prop-loc .ms{color:var(--jcp-primary);font-size:14px}
			.reveal{opacity:0;transform:translateY(12px);transition:opacity 500ms,transform 500ms}
			.reveal.visible{opacity:1;transform:none}
			.reveal-delay-1{transition-delay:60ms}
			.reveal-delay-2{transition-delay:120ms}
			.reveal-delay-3{transition-delay:180ms}
			@media(max-width:900px){
			  .listings-layout{flex-direction:column;height:auto}
			  .listings-col{width:100%;overflow-y:visible}
			  .map-panel{width:100%;border-left:none;border-top:1px solid var(--outline-var);height:240px;order:-1}
			  .prop-grid{grid-template-columns:1fr}
			}
		</style>
		<script>
			(function(){
			  // Operation tabs
			  document.querySelectorAll('.search-bar .tab').forEach(function(btn){
			    btn.addEventListener('click',function(){
			      document.querySelectorAll('.search-bar .tab').forEach(function(b){b.classList.remove('active')});
			      btn.classList.add('active');
			      document.getElementById('op-hidden').value=btn.dataset.op||'';
			      htmx.ajax('GET','/fragments/propiedades-page',{target:'#prop-results',swap:'innerHTML',values:{operacion:btn.dataset.op||''}});
			    });
			  });
			  // Dormitorios chips
			  document.querySelectorAll('.prop-chip[data-filter="dormitorios"]').forEach(function(btn){
			    btn.addEventListener('click',function(){
			      document.querySelectorAll('.prop-chip[data-filter="dormitorios"]').forEach(function(b){b.classList.remove('active')});
			      btn.classList.add('active');
			      document.getElementById('dorm-hidden').value=btn.dataset.value||'';
			      htmx.ajax('GET','/fragments/propiedades-page',{target:'#prop-results',swap:'innerHTML',values:{
			        operacion:document.getElementById('op-hidden').value,
			        tipo:document.querySelector('select[name=tipo]').value,
			        q:document.querySelector('input[name=q]').value,
			        dormitorios:document.getElementById('dorm-hidden').value,
			        sort:document.getElementById('sort-select').value
			      }});
			    });
			  });
			  // Deep link sync
			  var p=new URLSearchParams(location.search);
			  if(p.get('operacion')){
			    document.getElementById('op-hidden').value=p.get('operacion');
			    document.querySelectorAll('.search-bar .tab').forEach(function(b){b.classList.toggle('active',b.dataset.op===p.get('operacion'))});
			  }
			  if(p.get('tipo'))document.querySelector('select[name=tipo]').value=p.get('tipo');
			  if(p.get('q'))document.querySelector('input[name=q]').value=p.get('q');
			  // Reveal on scroll
			  document.body.addEventListener('htmx:afterSwap',function(){
			    document.querySelectorAll('.reveal:not(.visible)').forEach(function(el){el.classList.add('visible')});
			  });
			  window.addEventListener('scroll',function(){
			    document.querySelectorAll('.reveal:not(.visible)').forEach(function(el){
			      var r=el.getBoundingClientRect();
			      if(r.top<window.innerHeight-50)el.classList.add('visible');
			    });
			  },{passive:true});
			}());
		</script>
	}
}
```

- [ ] **Step 2: Generar**

```bash
cd /Users/matiaspereira/jcp-gestioninmobiliaria/jcp-gestioninmobiliaria
templ generate ./internal/templates/web/
```

Expected: `propiedades_templ.go` creado.

- [ ] **Step 3: Verificar compilación**

```bash
go build ./...
```

Expected: sin errores.

- [ ] **Step 4: Commit**

```bash
git add internal/templates/web/propiedades.templ internal/templates/web/propiedades_templ.go
git commit -m "feat: add PropiedadesPage templ with hero strip, split map layout, JCP design system"
```

---

## Task 7: Actualizar handler web para servir propiedades.templ

**Files:**
- Modify: `internal/handlers/web/handlers.go`
- Modify: `internal/routes/public.go`

- [ ] **Step 1: Agregar helper `renderTempl` y actualizar imports en `handlers.go`**

Agregar al bloque `import` de `internal/handlers/web/handlers.go`:
```go
import (
    "context"
    "fmt"
    "html/template"
    "strings"
    "time"

    "jcp-gestioninmobiliaria/internal/config"
    webtmpl "jcp-gestioninmobiliaria/internal/templates/web"

    "github.com/gofiber/fiber/v2"
    "github.com/pocketbase/pocketbase"
    "github.com/pocketbase/pocketbase/core"
)
```

Agregar función `renderTempl` al final del archivo:
```go
func renderTempl(c *fiber.Ctx, component interface{ Render(context.Context, io.Writer) error }) error {
    c.Set("Content-Type", "text/html; charset=utf-8")
    return component.Render(c.Context(), c.Response().BodyWriter())
}
```

También agregar `"io"` al import si no está.

- [ ] **Step 2: Reemplazar `PageHandler` para propiedades**

En `internal/routes/public.go`, cambiar la línea:
```go
app.Get("/propiedades.html", web.PageHandler(cfg, "propiedades"))
```
por:
```go
app.Get("/propiedades.html", web.PropiedadesHandler(cfg))
```

- [ ] **Step 3: Agregar `PropiedadesHandler` en `handlers.go`**

```go
// PropiedadesHandler renders the Templ-based propiedades listing page.
func PropiedadesHandler(cfg *config.Config) fiber.Handler {
    return func(c *fiber.Ctx) error {
        return renderTempl(c, webtmpl.PropiedadesPage())
    }
}
```

- [ ] **Step 4: Compilar**

```bash
go build ./...
```

Expected: sin errores.

- [ ] **Step 5: Probar en el navegador**

```bash
go run ./cmd/server serve &
sleep 3
curl -s http://localhost:3000/propiedades.html | grep -c "brand-hero"
kill %1
```

Expected: devuelve `1` (el hero de marca está presente).

- [ ] **Step 6: Eliminar el archivo HTML estático que reemplazamos**

```bash
rm /Users/matiaspereira/jcp-gestioninmobiliaria/jcp-gestioninmobiliaria/web/propiedades.html
```

- [ ] **Step 7: Commit**

```bash
git add internal/handlers/web/handlers.go internal/routes/public.go
git rm web/propiedades.html
git commit -m "feat: serve propiedades.html via PropiedadesPage Templ handler"
```

---

## Task 8: Agregar Lat/Lng a propiedadData y PropiedadHandler

**Files:**
- Modify: `internal/handlers/web/handlers.go`

- [ ] **Step 1: Agregar `Lat, Lng float64` a `propiedadData`**

En el struct `propiedadData` dentro de `handlers.go`, agregar:
```go
type propiedadData struct {
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
    Lat           float64  // ← nuevo
    Lng           float64  // ← nuevo
    Comuna        string   // ← nuevo (para el overlay del mapa)
}
```

- [ ] **Step 2: Poblar `Lat`, `Lng`, `Comuna` en `PropiedadHandler`**

En la función `PropiedadHandler`, después de leer los campos existentes del record, agregar:
```go
lat := r.GetFloat("lat")
lng := r.GetFloat("lng")
```

Y al construir `propiedadData`:
```go
data := propiedadData{
    // ... campos existentes ...
    Lat:    lat,
    Lng:    lng,
    Comuna: comuna,
}
```

- [ ] **Step 3: Compilar**

```bash
go build ./...
```

Expected: sin errores.

- [ ] **Step 4: Commit**

```bash
git add internal/handlers/web/handlers.go
git commit -m "feat: add Lat/Lng/Comuna to propiedadData for detail map rendering"
```

---

## Task 9: Crear propiedad.templ (página de detalle)

**Files:**
- Create: `internal/templates/web/propiedad.templ`
- Create: `internal/templates/web/propiedad_templ.go` (generado)

- [ ] **Step 1: Crear `internal/templates/web/propiedad.templ`**

```templ
package web

import (
	"fmt"
	"html/template"
	"jcp-gestioninmobiliaria/internal/templates/layouts"
)

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
	Lat           float64
	Lng           float64
	Comuna        string
}

templ PropiedadDetail(d PropiedadData) {
	@layouts.Base(d.Titulo + " — JCP Gestión Inmobiliaria", true) {
		<style>
			.detail-wrap{max-width:1200px;margin:28px auto 96px;padding:0 24px}
			.detail-back{display:inline-flex;align-items:center;gap:6px;font-size:13px;font-weight:500;
			  color:var(--on-surface-var);text-decoration:none;margin-bottom:16px;
			  padding:7px 14px;border-radius:var(--r-full);border:1px solid var(--outline-var);
			  transition:all 200ms}
			.detail-back:hover{color:var(--jcp-primary);border-color:var(--jcp-primary)}
			.gallery{display:grid;grid-template-columns:2fr 1fr 1fr;grid-template-rows:1fr 1fr;
			  gap:8px;border-radius:var(--r-lg);overflow:hidden;aspect-ratio:16/9;margin-bottom:32px;
			  background:var(--surface-container-low)}
			.g-main{grid-column:1;grid-row:1/span 2;background:var(--jcp-primary-100);
			  display:flex;align-items:center;justify-content:center;overflow:hidden}
			.g-main img,.g-cell img{width:100%;height:100%;object-fit:cover}
			.g-cell{background:var(--surface-container-high);overflow:hidden}
			.gallery-placeholder{font-size:80px;color:rgba(29,78,216,.15)}
			.price-bar{background:var(--jcp-primary-dark);color:white;padding:16px 24px;
			  display:flex;align-items:center;justify-content:space-between;flex-wrap:wrap;gap:12px;
			  margin-bottom:0}
			.price-main{font-family:var(--font-display);font-size:32px;line-height:1.1}
			.price-sub{font-size:13px;opacity:.75;margin-top:3px;display:flex;align-items:center;gap:6px}
			.pb-ctas{display:flex;gap:10px}
			.pb-btn{padding:10px 20px;border-radius:var(--r-md);border:none;font-size:13px;font-weight:700;
			  cursor:pointer;display:flex;align-items:center;gap:6px;font-family:inherit}
			.pb-wa{background:#25D366;color:white}
			.pb-tel{background:rgba(255,255,255,.15);backdrop-filter:blur(8px);color:white;border:1px solid rgba(255,255,255,.25)}
			.detail-grid{display:grid;grid-template-columns:1fr 380px;gap:48px;margin-top:32px}
			.detail-header h1{font-family:var(--font-display);font-size:clamp(26px,4vw,38px);line-height:1.15;margin-bottom:8px}
			.detail-loc{font-size:14px;color:var(--on-surface-var);margin-bottom:12px;display:flex;align-items:center;gap:6px}
			.op-chips{display:flex;gap:8px;margin-bottom:16px;flex-wrap:wrap}
			.feats-grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(110px,1fr));gap:14px;margin:24px 0 28px}
			.feat-box{padding:16px;border:1px solid var(--outline-var);border-radius:var(--r-md);background:var(--surface-bright);text-align:center}
			.feat-icon{color:var(--jcp-primary);margin-bottom:6px}
			.feat-val{font-family:var(--font-display);font-size:22px;color:var(--on-surface)}
			.feat-label{font-size:11px;color:var(--outline);text-transform:uppercase;letter-spacing:.06em}
			.section-block{margin-bottom:32px}
			.section-block h2{font-family:var(--font-display);font-size:22px;margin-bottom:12px;
			  display:flex;align-items:center;gap:10px}
			.section-icon{width:32px;height:32px;background:var(--jcp-primary-50);border-radius:8px;
			  display:flex;align-items:center;justify-content:center;color:var(--jcp-primary);flex-shrink:0}
			.desc{font-size:15px;line-height:1.85;color:var(--on-surface-var)}
			.prestaciones{display:flex;flex-wrap:wrap;gap:8px;margin-top:12px}
			.prest-pill{font-size:12.5px;padding:7px 14px;background:var(--jcp-primary-50);
			  border:1px solid var(--jcp-primary-100);border-radius:var(--r-full);
			  color:var(--jcp-primary);display:flex;align-items:center;gap:6px}
			/* MAP */
			.map-box{height:300px;border-radius:var(--r-md);overflow:hidden;
			  border:1px solid var(--outline-var);position:relative;margin-bottom:8px}
			.map-overlay-badge{position:absolute;bottom:10px;left:10px;z-index:500;
			  background:white;border-radius:8px;padding:7px 12px;
			  box-shadow:var(--elev-3);font-size:12px;font-weight:600;color:var(--on-surface);
			  border:1px solid var(--outline-var);display:flex;align-items:center;gap:5px}
			.map-hint{font-size:11px;color:var(--outline);display:flex;align-items:center;gap:4px;margin-top:6px}
			.zone-pill{display:inline-flex;align-items:center;gap:5px;
			  background:var(--jcp-primary-50);color:var(--jcp-primary);
			  padding:5px 12px;border-radius:var(--r-full);font-size:12px;font-weight:600;
			  border:1px solid var(--jcp-primary-100);margin:0 6px 8px 0}
			/* SIDEBAR */
			.contact-card{position:sticky;top:88px;background:var(--surface-bright);
			  border:1px solid var(--outline-var);border-radius:var(--r-lg);padding:24px;
			  box-shadow:var(--elev-2);height:fit-content}
			.price-detail-row{display:flex;justify-content:space-between;align-items:center;
			  padding:9px 0;border-bottom:1px solid var(--outline-var);font-size:14px}
			.price-detail-row:last-of-type{border:none}
			.pr-label{color:var(--on-surface-var)}
			.pr-val{font-weight:700;color:var(--on-surface)}
			.pr-val.big{color:var(--jcp-primary);font-size:20px}
			.sb-cta{display:flex;align-items:center;justify-content:center;gap:8px;
			  width:100%;padding:13px;border-radius:11px;border:none;font-size:14px;font-weight:700;
			  cursor:pointer;margin-bottom:10px;text-decoration:none;font-family:inherit}
			.sb-wa{background:#25D366;color:white}
			.sb-tel{background:var(--jcp-primary);color:white}
			.sb-form{background:var(--surface-container-low);color:var(--on-surface);border:1.5px solid var(--outline-var)}
			.sidebar-map-box{height:180px;border-radius:10px;overflow:hidden;border:1px solid var(--outline-var)}
			@media(max-width:900px){
			  .detail-grid{grid-template-columns:1fr}
			  .gallery{aspect-ratio:4/3;grid-template-columns:1fr 1fr;grid-template-rows:2fr 1fr}
			  .g-main{grid-column:1/span 2;grid-row:1}
			}
		</style>
		<main>
			<div class="detail-wrap">
				<a href="/propiedades.html" class="detail-back">
					<span class="ms ms-sm">arrow_back</span> Volver al listado
				</a>
				<!-- GALLERY -->
				<div class="gallery">
					<div class="g-main">@templ.Raw(string(d.CoverHTML))</div>
					@templ.Raw(string(d.ThumbsHTML))
				</div>
				<!-- PRICE BAR -->
				<div class="price-bar">
					<div>
						<div class="price-main">@templ.Raw(string(d.PriceHTML))</div>
						<div class="price-sub">
							<span class="ms ms-sm" style="opacity:.7;">location_on</span>
							{ d.PriceSub }
						</div>
					</div>
					<div class="pb-ctas">
						if d.WhatsappHTML != "" {
							@templ.Raw(string(d.WhatsappHTML))
						} else {
							<a href="https://wa.me/56912345678" class="pb-btn pb-wa" target="_blank" rel="noopener">
								<span class="ms ms-sm">chat</span> WhatsApp
							</a>
						}
						<button class="pb-btn pb-tel"><span class="ms ms-sm">phone</span> Llamar</button>
					</div>
				</div>
				<!-- DETAIL GRID -->
				<div class="detail-grid">
					<div>
						<!-- HEADER -->
						<div class="detail-header" style="margin-bottom:20px">
							<div class="op-chips">@templ.Raw(string(d.ChipsHTML))</div>
							<h1>{ d.Titulo }</h1>
							<p class="detail-loc">
								<span class="ms ms-sm" style="color:var(--jcp-primary)">location_on</span>
								{ d.Direccion }
							</p>
						</div>
						<!-- STATS -->
						<div class="feats-grid">@templ.Raw(string(d.FeatsHTML))</div>
						<!-- DESCRIPCIÓN -->
						<div class="section-block">
							<h2>
								<div class="section-icon"><span class="ms ms-sm">description</span></div>
								Descripción
							</h2>
							<div class="desc">@templ.Raw(string(d.BodyHTML))</div>
						</div>
						<!-- UBICACIÓN + MAPA -->
						if d.Lat != 0 && d.Lng != 0 {
							<div class="section-block">
								<h2>
									<div class="section-icon"><span class="ms ms-sm">map</span></div>
									Ubicación
								</h2>
								<div style="margin-bottom:12px">
									if d.Comuna != "" {
										<span class="zone-pill"><span class="ms ms-sm">location_on</span> { d.Comuna }</span>
									}
								</div>
								<div
									class="map-box"
									data-map="detail"
									data-lat={ fmt.Sprintf("%.6f", d.Lat) }
									data-lng={ fmt.Sprintf("%.6f", d.Lng) }
									data-title={ d.Titulo }
								>
									<div class="map-overlay-badge">
										<span class="ms ms-sm" style="color:var(--jcp-primary)">location_on</span>
										{ d.Comuna }
									</div>
								</div>
								<p class="map-hint">
									<span class="ms ms-sm">lock</span>
									Dirección exacta disponible al contactar al corredor
								</p>
							</div>
						}
						<!-- PRESTACIONES -->
						if d.AmenitiesHTML != "" {
							<div class="section-block">
								<h2>
									<div class="section-icon"><span class="ms ms-sm">auto_awesome</span></div>
									Prestaciones
								</h2>
								<div class="prestaciones">@templ.Raw(string(d.AmenitiesHTML))</div>
							</div>
						}
					</div>
					<!-- SIDEBAR -->
					<aside class="contact-card" id="contactar">
						<div style="font-size:15px;font-weight:800;color:var(--on-surface);margin-bottom:14px">Precio y condiciones</div>
						<div class="price-detail-row">
							<span class="pr-label">Precio</span>
							<span class="pr-val big">@templ.Raw(string(d.PriceHTML))</span>
						</div>
						<div style="margin-top:16px">
							if d.WhatsappHTML != "" {
								@templ.Raw(string(d.WhatsappHTML))
							} else {
								<a href="https://wa.me/56912345678" class="sb-cta sb-wa" target="_blank" rel="noopener">
									<span class="ms ms-sm">chat</span> Consultar por WhatsApp
								</a>
							}
							<button class="sb-cta sb-tel"><span class="ms ms-sm">phone</span> Llamar al corredor</button>
							<button class="sb-cta sb-form"><span class="ms ms-sm">mail</span> Enviar consulta</button>
						</div>
						if d.Lat != 0 && d.Lng != 0 {
							<div style="margin-top:20px;padding-top:20px;border-top:1px solid var(--outline-var)">
								<div style="font-size:13px;font-weight:700;color:var(--on-surface);margin-bottom:8px;display:flex;align-items:center;gap:5px">
									<span class="ms ms-sm" style="color:var(--jcp-primary)">map</span> Zona de la propiedad
								</div>
								<div
									class="sidebar-map-box"
									data-map="detail-mini"
									data-lat={ fmt.Sprintf("%.6f", d.Lat) }
									data-lng={ fmt.Sprintf("%.6f", d.Lng) }
								></div>
								<p class="map-hint" style="justify-content:center;margin-top:6px">
									<span class="ms ms-sm">info</span> Ubicación aproximada
								</p>
							</div>
						}
					</aside>
				</div>
			</div>
		</main>
	}
}
```

- [ ] **Step 2: Generar**

```bash
cd /Users/matiaspereira/jcp-gestioninmobiliaria/jcp-gestioninmobiliaria
templ generate ./internal/templates/web/
```

Expected: `propiedad_templ.go` creado.

- [ ] **Step 3: Compilar**

```bash
go build ./...
```

Expected: sin errores.

- [ ] **Step 4: Commit**

```bash
git add internal/templates/web/propiedad.templ internal/templates/web/propiedad_templ.go
git commit -m "feat: add PropiedadDetail templ with map sections and Prestaciones"
```

---

## Task 10: Migrar PropiedadHandler a usar propiedad.templ

**Files:**
- Modify: `internal/handlers/web/handlers.go`

El `PropiedadHandler` actual usa `html/template` + `propiedad.html`. Lo reemplazamos para que use `PropiedadDetail` templ, y adaptamos `propiedadData` → `webtmpl.PropiedadData`.

- [ ] **Step 1: Eliminar `propiedadData` del handlers.go**

El struct `propiedadData` en `handlers.go` queda reemplazado por `webtmpl.PropiedadData`. Eliminar el struct local y usar el del paquete templ.

- [ ] **Step 2: Actualizar `PropiedadHandler` para llamar a `PropiedadDetail`**

Reemplazar el bloque final de `PropiedadHandler` (desde la construcción de `data` hasta la ejecución de la template) por:

```go
data := webtmpl.PropiedadData{
    Titulo:        titulo,
    Direccion:     locLine,
    PriceHTML:     priceHTML,
    PriceSub:      priceSub,
    ChipsHTML:     chipsHTML,
    CoverHTML:     coverHTML,
    ThumbsHTML:    thumbsHTML,
    FeatsHTML:     featsHTML,
    BodyHTML:      bodyHTML,
    AmenitiesHTML: amenitiesHTML,
    WhatsappHTML:  whatsappHTML,
    Lat:           r.GetFloat("lat"),
    Lng:           r.GetFloat("lng"),
    Comuna:        comuna,
}
return renderTempl(c, webtmpl.PropiedadDetail(data))
```

Donde los campos `priceHTML`, `priceSub`, `chipsHTML`, etc. ya se construyen en el handler con el mismo código que antes — solo cambia el struct de destino y el render final.

- [ ] **Step 3: Compilar**

```bash
go build ./...
```

Expected: sin errores.

- [ ] **Step 4: Prueba**

```bash
go run ./cmd/server serve &
sleep 3
curl -s http://localhost:3000/propiedades/casa-mediterranea-la-dehesa | grep -c "data-map"
kill %1
```

Expected: devuelve `2` (hay `data-map="detail"` y `data-map="detail-mini"`).

- [ ] **Step 5: Eliminar el HTML estático obsoleto**

```bash
rm /Users/matiaspereira/jcp-gestioninmobiliaria/jcp-gestioninmobiliaria/internal/templates/web/propiedad.html
```

- [ ] **Step 6: Commit**

```bash
git add internal/handlers/web/handlers.go
git rm internal/templates/web/propiedad.html
git commit -m "feat: PropiedadHandler now renders via PropiedadDetail Templ with interactive maps"
```

---

## Task 11: Agregar lat/lng al formulario admin

**Files:**
- Modify: `internal/handlers/admin/handlers.go`

- [ ] **Step 1: Agregar lectura de lat/lng en `setPropiedadFields`**

En la función `setPropiedadFields` (línea ~1390), agregar al final:

```go
if v := c.FormValue("lat"); v != "" {
    var f float64
    fmt.Sscanf(v, "%f", &f)
    r.Set("lat", f)
}
if v := c.FormValue("lng"); v != "" {
    var f float64
    fmt.Sscanf(v, "%f", &f)
    r.Set("lng", f)
}
```

- [ ] **Step 2: Agregar `lat, lng float64` como parámetros a `propiedadFormHTML`**

La firma actual:
```go
func propiedadFormHTML(id, titulo, slug, operacion, tipo, descripcion, direccion, comuna, region, coverImage string,
    precioUF, precioCLP, dormitorios, banos, estacionamientos, superficieUtil float64,
    estadoPropiedad, status string, destacada, oportunidad bool, amenidades, whatsapp, gallery string,
) string {
```

Nueva firma (agregar `lat, lng float64` al final):
```go
func propiedadFormHTML(id, titulo, slug, operacion, tipo, descripcion, direccion, comuna, region, coverImage string,
    precioUF, precioCLP, dormitorios, banos, estacionamientos, superficieUtil float64,
    estadoPropiedad, status string, destacada, oportunidad bool, amenidades, whatsapp, gallery string,
    lat, lng float64,
) string {
```

- [ ] **Step 3: Agregar los campos al HTML del form dentro de `propiedadFormHTML`**

Dentro de la función, antes del cierre del `</form>`, agregar:

```go
sb.WriteString(fmt.Sprintf(`
<div class="form-field">
  <label>Latitud (zona, no dirección exacta)</label>
  <input type="number" name="lat" step="0.000001" value="%s" class="form-input" placeholder="-33.4513"/>
</div>
<div class="form-field">
  <label>Longitud (zona, no dirección exacta)</label>
  <input type="number" name="lng" step="0.000001" value="%s" class="form-input" placeholder="-70.6653"/>
</div>`,
    fmtFloatOrEmpty(lat),
    fmtFloatOrEmpty(lng),
))
```

Agregar helper:
```go
func fmtFloatOrEmpty(f float64) string {
    if f == 0 { return "" }
    return fmt.Sprintf("%.6f", f)
}
```

- [ ] **Step 4: Actualizar las dos llamadas a `propiedadFormHTML`**

**En `PropiedadForm` (create):**
```go
html := propiedadFormHTML("", "", "", "VENTA", "CASA", "", "", "", "", "", 0, 0, 0, 0, 0, 0, "usada", "publicado", false, false, "", "", "", 0, 0)
```

**En `PropiedadEdit`:**
```go
html := propiedadFormHTML(
    r.Id,
    r.GetString("titulo"),
    r.GetString("slug"),
    r.GetString("operacion"),
    r.GetString("tipo"),
    r.GetString("descripcion"),
    r.GetString("direccion"),
    r.GetString("comuna"),
    r.GetString("region"),
    r.GetString("cover_image"),
    r.GetFloat("precio_uf"),
    r.GetFloat("precio_clp"),
    float64(r.GetInt("dormitorios")),
    float64(r.GetInt("banos")),
    float64(r.GetInt("estacionamientos")),
    r.GetFloat("superficie_util"),
    r.GetString("estado_propiedad"),
    r.GetString("status"),
    r.GetBool("destacada"),
    r.GetBool("oportunidad"),
    r.GetString("amenidades"),
    r.GetString("contacto_whatsapp"),
    r.GetString("gallery"),
    r.GetFloat("lat"),   // ← nuevo
    r.GetFloat("lng"),   // ← nuevo
)
```

- [ ] **Step 5: Compilar**

```bash
go build ./...
```

Expected: sin errores.

- [ ] **Step 6: Commit**

```bash
git add internal/handlers/admin/handlers.go
git commit -m "feat: add lat/lng fields to propiedades admin form"
```

---

## Task 12: Prueba de integración completa y commit final

- [ ] **Step 1: Arrancar el servidor**

```bash
cd /Users/matiaspereira/jcp-gestioninmobiliaria/jcp-gestioninmobiliaria
go run ./cmd/server serve
```

Expected: arranca sin errores, PocketBase seeds propiedades.

- [ ] **Step 2: Verificar listado**

Abrir http://localhost:3000/propiedades.html

Verificar:
- Hero de marca visible con fondo azul oscuro y gradiente
- Buscador sticky debajo del nav
- Panel de mapa a la derecha (desktop) o arriba (mobile)
- Cards con iconos Material Symbols (no emojis)
- Al cargar las cards, el mapa muestra pins azules

- [ ] **Step 3: Verificar detalle**

Abrir http://localhost:3000/propiedades/casa-mediterranea-la-dehesa (o cualquier slug publicado)

Verificar:
- Galería + price bar azul oscuro
- Sección "Prestaciones" (no "Amenidades")
- Si lat/lng ≠ 0: mapa de zona con pin + mini mapa en sidebar
- Si lat/lng = 0: las secciones de mapa no aparecen

- [ ] **Step 4: Verificar admin**

Abrir http://localhost:3000/admin → Propiedades → editar una propiedad
Verificar que el formulario tiene campos "Latitud" y "Longitud".
Ingresar `-33.364` y `-70.541` → guardar → ver detalle público → mapa debe aparecer.

- [ ] **Step 5: Verificar build de producción**

```bash
go build -v -o /tmp/jcp-test ./cmd/server
```

Expected: compila sin errores.

- [ ] **Step 6: Commit de cierre**

```bash
git add -A
git commit -m "feat: maps Leaflet + Templ migration + JCP frontend polish complete"
```

---

## Notas de mantenimiento

**Para agregar un nuevo componente Templ:**
1. Crear el `.templ` en `internal/templates/<paquete>/`
2. Correr `templ generate ./<ruta>/`
3. El `*_templ.go` generado debe commitearse junto al `.templ`

**Para actualizar el mapa del listado:**
- Editar solo `web/static/js/maps.js` — no hay lógica de mapa en los templates
- `syncListingPins()` se llama automáticamente tras cada swap HTMX

**Para agregar coordenadas a una propiedad:**
- Admin → Propiedades → editar → campos Latitud/Longitud
- Usar coordenadas del centro de la zona/barrio, no la dirección exacta
