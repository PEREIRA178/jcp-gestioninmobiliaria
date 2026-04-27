# Mapas Leaflet.js + Pulido Frontend — Design Spec

## Objetivo

Agregar mapas interactivos con Leaflet.js a la página de listado y detalle de propiedades, migrar las vistas públicas a Templ, y unificar el sistema de diseño visual (paleta JCP azul, Material Symbols Rounded, hero de marca).

## Arquitectura

### Integración de mapas: `data-map` attributes + `maps.js` (Opción A)

Un único archivo `/static/js/maps.js` (~100 líneas) maneja toda la lógica Leaflet. Los componentes Templ solo emiten `<div>` con atributos `data-map`, `data-lat`, `data-lng`, `data-zoom`. El JS escanea el DOM al cargar y escucha `htmx:afterSettle` para re-sincronizar pines después de cada swap HTMX.

Leaflet se carga por CDN en el layout base una sola vez.

### Modos de mapa

| `data-map`    | Dónde            | Comportamiento                                          |
|---------------|------------------|---------------------------------------------------------|
| `listing`     | Listado          | Un mapa con todos los pines; re-sincroniza post-HTMX   |
| `detail`      | Detalle (main)   | Pin único, `scrollWheelZoom: false`                     |
| `detail-mini` | Detalle (sidebar)| Círculo de zona, `dragging: false`, sin controles       |

Si una propiedad no tiene `lat != 0 && lng != 0`, el componente Templ no renderiza el div del mapa.

### Migración a Templ

Las vistas públicas dejan de ser archivos `.html` estáticos servidos con `SendFile` y pasan a ser componentes Templ compilados. El admin sigue en `html/template` (fuera de scope).

---

## Archivos

### Nuevos
| Archivo | Responsabilidad |
|---|---|
| `web/static/js/maps.js` | Init Leaflet, gestión de pines, listener `htmx:afterSettle` |
| `internal/templates/layouts/base.templ` | Layout compartido: `<head>` con CDNs, CSS tokens globales, nav, footer |
| `internal/templates/web/propiedades.templ` | Página pública de listado: hero, buscador, chips, panel mapa + grid |
| `internal/templates/web/propiedad.templ` | Página de detalle: galería, price bar, stats, mapa, prestaciones, sidebar |
| `internal/templates/fragments/propiedades.templ` | Cards del grid (fragment HTMX); card emite `data-lat`, `data-lng` |

### Modificados
| Archivo | Cambio |
|---|---|
| `internal/handlers/web/handlers.go` | `propiedadData` agrega `Lat, Lng float64`; `PropiedadHandler` lee `lat`/`lng` del record; handlers llaman a `.Render()` de Templ en lugar de `SendFile` |
| `internal/handlers/fragments/propiedades.go` | Delega render de cards al nuevo `.templ`; elimina string-building |
| `internal/templates/admin/pages/propiedades.html` | Agrega campos numéricos `lat` y `lng` al formulario de alta/edición |

### Eliminados
| Archivo | Razón |
|---|---|
| `web/propiedades.html` | Reemplazado por `propiedades.templ` |
| `internal/templates/web/propiedad.html` | Reemplazado por `propiedad.templ` |

---

## Sistema de diseño

### Paleta unificada (definida una sola vez en `base.templ`)

```css
--jcp-primary:       #1D4ED8;
--jcp-primary-dark:  #1E3A8A;
--jcp-primary-light: #3B82F6;
--jcp-primary-50:    #EFF6FF;
--jcp-primary-100:   #DBEAFE;
--surface:           #FAFAF9;
--surface-bright:    #FFFFFF;
--on-surface:        #111827;
--on-surface-var:    #4B5563;
--outline:           #9CA3AF;
--outline-var:       #E5E7EB;
--inverse-surface:   #0F1419;
--r-sm: 12px; --r-md: 16px; --r-lg: 24px; --r-full: 9999px;
--ease-express: cubic-bezier(0.05, 0.7, 0.1, 1.0);
--font-display: 'DM Serif Display', Georgia, serif;
--font-body:    'DM Sans', system-ui, sans-serif;
--nav-h: 68px;
```

Reemplaza las dos paletas actuales (indigo en `propiedades.html`, rojo en `propiedad.html`).

### Iconos: Material Symbols Rounded

CDN cargado en `base.templ`:
```html
<link rel="stylesheet"
  href="https://fonts.googleapis.com/css2?family=Material+Symbols+Rounded:opsz,wght,FILL,GRAD@24,400,1,0"/>
```

Clase base:
```css
.ms { font-family:'Material Symbols Rounded'; font-variation-settings:'FILL' 1,'wght' 400,'GRAD' 0,'opsz' 24; }
```

Todos los SVG inline de cards y detalle se reemplazan por `<span class="ms">nombre_icono</span>`.

Mapeo de iconos principales:
- Dormitorios → `bed`
- Baños → `shower`
- Estacionamientos → `directions_car`
- Superficie → `square_foot`
- Ubicación → `location_on`
- Mapa → `map`
- Piscina → `pool`
- Jardín → `yard`
- Quincho → `outdoor_grill`
- Bodega → `inventory_2`
- Seguridad → `security`
- WhatsApp/Chat → `chat`
- Teléfono → `phone`
- Correo → `mail`
- Volver → `arrow_back`
- Foto → `photo_camera`
- Estrella → `star`
- Candado → `lock`
- Info → `info`

### Hero de marca (listado)

Strip compacto sobre el buscador existente. Solo branding, no duplica búsqueda.

```
┌─────────────────────────────────────────────────────────┐
│  [ícono home]  JCP Gestión Inmobiliaria                  │
│  Compra, arrienda y vende propiedades en Chile.           │
│  [WhatsApp ▸]  [Ver propiedades ▸]                       │
│                          fondo: gradient dark azul       │
└─────────────────────────────────────────────────────────┘
```

- Height: `clamp(130px, 18vw, 200px)`
- Background: `linear-gradient(135deg, #0F172A 0%, #1E3A8A 100%)`
- Texto blanco, dos CTAs pill
- Mobile: layout vertical, texto centrado

---

## Detalle de componentes Templ

### `base.templ` — Layout base

```
templ Base(title string, includeMaps bool) {
  <!DOCTYPE html>
  <html>
    <head>
      <!-- fonts, Material Symbols, Leaflet (solo si includeMaps=true) -->
      <!-- CSS tokens globales, nav, footer -->
    </head>
    <body>
      { children... }
    </body>
  </html>
}
```

`includeMaps bool` evita cargar Leaflet en páginas que no lo necesitan (noticias, etc).

### `propiedades.templ`

```
templ PropiedadesPage() {
  @Base("Propiedades — JCP", true) {
    @HeroMarca()           <!-- strip branding -->
    @SearchBar()           <!-- form HTMX existente -->
    @FilterChips()         <!-- chips operacion/tipo/dormitorios -->
    <div class="listings-layout">
      <div id="prop-map" data-map="listing"></div>   <!-- panel mapa -->
      <div id="prop-results" hx-get="..." hx-trigger="load">...</div>
    </div>
  }
}
```

### `fragments/propiedades.templ`

```
templ PropCard(p Propiedad, i int) {
  <article class="prop-card"
    data-id={ p.ID }
    data-lat={ fmt(p.Lat) }
    data-lng={ fmt(p.Lng) }>
    ...
  </article>
}
```

Los `data-lat`/`data-lng` con valor `0` son ignorados por `maps.js` al crear pines.

### `propiedad.templ`

```
templ PropiedadDetail(d PropiedadData) {
  @Base(d.Titulo + " — JCP", true) {
    @Gallery(d)
    @PriceBar(d)
    <div class="detail-grid">
      <div>
        @Stats(d)
        @Descripcion(d)
        @MapaUbicacion(d)        <!-- data-map="detail" -->
        @Prestaciones(d)         <!-- antes "amenidades" -->
      </div>
      <aside>
        @ContactSidebar(d)
        @MapaMini(d)             <!-- data-map="detail-mini" -->
      </aside>
    </div>
  }
}
```

---

## maps.js — comportamiento

```js
// Al cargar la página
document.addEventListener('DOMContentLoaded', initMaps);

// Después de cada swap HTMX (re-sync pines del listado)
document.addEventListener('htmx:afterSettle', syncListingPins);

function initMaps() {
  // Modo listing: crea mapa vacío, espera htmx:afterSettle para pines
  // Modo detail: crea mapa con pin JCP + popup
  // Modo detail-mini: crea mapa con círculo de zona, no draggable
}

function syncListingPins(map) {
  // Lee todos [article[data-lat]][data-lat!="0"]
  // clearLayers() del layer group de pines
  // Re-agrega un marcador por propiedad
  // Ajusta bounds al conjunto de pines
}
```

Pin JCP: teardrop azul `#1D4ED8` con borde blanco — mismo diseño del mockup.

Popup al click de pin:
```
[ Tipo · Operacion ]
[ Título truncado  ]
[ UF XX.XXX        ]
[ 📍 Comuna        ]
[ Ver detalle →    ]
```

---

## Datos: lat/lng en el admin

El admin requiere tres cambios coordinados en `internal/handlers/admin/handlers.go`:

**1. `setPropiedadFields`** — agregar lectura de `lat` y `lng` del form:
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

**2. `propiedadFormHTML`** — agregar parámetros `lat, lng float64` y renderizar los inputs:
```html
<div class="form-field">
  <label>Latitud (aprox. zona)</label>
  <input type="number" name="lat" step="0.000001" value="{{lat}}" placeholder="-33.4513"/>
</div>
<div class="form-field">
  <label>Longitud (aprox. zona)</label>
  <input type="number" name="lng" step="0.000001" value="{{lng}}" placeholder="-70.6653"/>
</div>
```

**3. `PropiedadEdit`** — pasar `r.GetFloat("lat")` y `r.GetFloat("lng")` al llamar a `propiedadFormHTML`.

El `PropiedadCreate` no requiere cambios adicionales ya que llama a `setPropiedadFields` que incluirá los nuevos campos.

Ubicación aproximada (privacidad): el mapa público muestra el pin centrado en la **comuna**, no en la dirección exacta. El corredor provee coordenadas de la zona general. La dirección exacta se entrega al contactar.

---

## Exclusiones de scope

- Noticias (`web/noticias.html`, `noticia.html`) — no se tocan
- Admin templates — solo se agrega lat/lng al form de propiedades
- Geocodificación automática — las coords se ingresan manualmente
- Tour virtual (campo `tour_url`) — no se expone en esta iteración
- RSS feed — no se toca
