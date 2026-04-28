# Mobile Fixes & New Features — JCP Gestión Inmobiliaria

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Corregir 5 problemas críticos: navbar superior en mobile, tarjetas cortadas por footer, Guardadas roto, formulario de contacto con WhatsApp, y registro de visitantes Google en el admin.

**Architecture:** Cambios en Templ (layouts + páginas web + admin handlers) y Go (auth handler). Sin nuevas dependencias; usa Fiber + PocketBase + vanilla JS ya existentes.

**Tech Stack:** Go + Fiber, Templ, HTMX, PocketBase, DOMParser, vanilla JS, CSS

**Nota de seguridad:** Toda inserción de HTML en el DOM usa DOMParser o HTMX (que sanitiza internamente), nunca innerHTML directo con contenido no confiable. El contenido viene siempre de nuestro propio servidor Go/Templ con HTMLEscapeString aplicado.

---

## Mapa de archivos

| Archivo | Cambio |
|---------|--------|
| `internal/templates/layouts/base.templ` | Hamburger + mobile menu drawer |
| `internal/templates/layouts/app.templ` | Mobile top header + drawer + fix padding + Guardadas en sidebar |
| `internal/templates/web/guardadas.templ` | Fix htmx.ajax timing → outerHTML swap seguro |
| `internal/handlers/fragments/propiedades.go` | GuardasFragment retorna wrapper div para outerHTML |
| `internal/templates/web/propiedad.templ` | Modal de contacto + WhatsApp |
| `internal/handlers/web/auth.go` | Aceptar pb, loguear visitante |
| `internal/routes/public.go` | Pasar pb a GoogleCallback |
| `internal/handlers/admin/handlers.go` | Handler VisitorLogs |
| `internal/routes/admin.go` | Ruta /admin/visitors |
| `internal/templates/admin/pages/visitors.html` | Crear (nuevo archivo) |
| `internal/templates/admin/pages/dashboard.html` | Agregar link "Visitantes" al sidebar |

---

## Task 1: Hamburger mobile en `base.templ`

**Files:**
- Modify: `internal/templates/layouts/base.templ`

La landing page usa `base.templ`. En mobile (<768px) se ocultan los `.nav-links` pero no hay hamburguesa. Se agrega: botón `.nav-ham`, menú `.mobile-menu` animado, JS de toggle.

- [ ] **Step 1: Agregar CSS para hamburger y mobile menu**

Dentro del bloque `<style>` de `base.templ`, ANTES de la línea `@media(max-width:768px){`, agregar:

```css
.nav-ham{display:none;width:38px;height:38px;border-radius:var(--r-sm);
  background:transparent;border:1px solid var(--outline-var);
  cursor:pointer;align-items:center;justify-content:center;
  color:var(--on-surface);transition:all .18s;flex-shrink:0}
.nav-ham:hover{background:var(--surface-container-low)}
.mobile-menu{display:none;position:fixed;top:var(--nav-h);left:0;right:0;z-index:199;
  background:rgba(250,250,249,.97);backdrop-filter:blur(20px);-webkit-backdrop-filter:blur(20px);
  border-bottom:1px solid var(--outline-var);
  padding:14px 20px 20px;flex-direction:column;gap:2px;
  box-shadow:0 8px 32px rgba(15,20,25,.10)}
.mobile-menu-anim{animation:slideDown .22s var(--ease-express)}
@keyframes slideDown{from{opacity:0;transform:translateY(-8px)}to{opacity:1;transform:translateY(0)}}
.mobile-menu.open{display:flex}
.mobile-menu a{font-size:15px;font-weight:500;color:var(--on-surface);
  text-decoration:none;padding:12px 16px;border-radius:var(--r-sm);
  transition:background .18s;display:flex;align-items:center;gap:10px}
.mobile-menu a:hover{background:var(--jcp-primary-50)}
.mob-menu-divider{height:1px;background:var(--outline-var);margin:6px 0}
.mob-menu-cta{background:var(--jcp-primary)!important;color:#fff!important;
  border-radius:var(--r-full)!important;font-weight:600!important;
  justify-content:center;margin-top:4px}
.mob-menu-cta:hover{background:var(--jcp-primary-dark)!important}
```

Luego reemplazar el bloque `@media(max-width:768px){` existente con:

```css
@media(max-width:768px){
  :root{--nav-h:58px}
  .nav-links{display:none}
  .nav-cta{display:none}
  .nav-ham{display:flex}
  .footer-grid{grid-template-columns:1fr}
}
```

- [ ] **Step 2: Agregar botón hamburger al nav-inner**

En `base.templ`, reemplazar el bloque `<div class="nav-inner">...</div>` con:

```templ
<div class="nav-inner">
    <a href="/" class="nav-brand">
        <div class="nav-logo-box"><span class="ms" style="font-size:22px;">home</span></div>
        <div>
            <div class="nav-brand-name">JCP Gestión</div>
            <div class="nav-brand-sub">Inmobiliaria</div>
        </div>
    </a>
    <nav class="nav-links">
        <a href="/propiedades?operacion=VENTA">Comprar</a>
        <a href="/propiedades?operacion=ARRIENDO">Arrendar</a>
        <a href="/propiedades">Todas las propiedades</a>
    </nav>
    <a href="/propiedades" class="nav-cta">
        <span class="ms ms-sm">search</span> Ver propiedades
    </a>
    <button class="nav-ham" id="nav-ham" aria-label="Menú" aria-expanded="false">
        <span class="ms">menu</span>
    </button>
</div>
```

- [ ] **Step 3: Agregar mobile menu drawer HTML**

Después del cierre de `<nav class="nav" id="main-nav">...</nav>` y ANTES de `{ children... }`, agregar:

```templ
<div class="mobile-menu" id="mobile-menu" role="dialog" aria-label="Menú móvil">
    <a href="/propiedades?operacion=VENTA"><span class="ms ms-sm">sell</span> Comprar</a>
    <a href="/propiedades?operacion=ARRIENDO"><span class="ms ms-sm">key</span> Arrendar</a>
    <a href="/propiedades"><span class="ms ms-sm">home_work</span> Todas las propiedades</a>
    <div class="mob-menu-divider"></div>
    <a href="/propiedades" class="mob-menu-cta"><span class="ms ms-sm">search</span> Ver propiedades</a>
</div>
```

- [ ] **Step 4: Agregar JS para toggle del hamburger**

En el `<script>` existente de `base.templ` (el que maneja el scroll-nav), agregar al FINAL del bloque, antes del cierre `</script>`:

```javascript
(function(){
  var ham=document.getElementById('nav-ham');
  var menu=document.getElementById('mobile-menu');
  if(!ham||!menu)return;
  function openMenu(){
    menu.classList.add('open','mobile-menu-anim');
    ham.setAttribute('aria-expanded','true');
    ham.querySelector('.ms').textContent='close';
  }
  function closeMenu(){
    menu.classList.remove('open');
    ham.setAttribute('aria-expanded','false');
    ham.querySelector('.ms').textContent='menu';
  }
  ham.addEventListener('click',function(e){
    e.stopPropagation();
    menu.classList.contains('open')?closeMenu():openMenu();
  });
  document.addEventListener('click',function(e){
    if(menu.classList.contains('open')&&!menu.contains(e.target)&&e.target!==ham){
      closeMenu();
    }
  });
  menu.querySelectorAll('a').forEach(function(a){
    a.addEventListener('click',function(){closeMenu()});
  });
}());
```

- [ ] **Step 5: Compilar y verificar en mobile**

```bash
cd /Users/matiaspereira/jcp-gestioninmobiliaria/jcp-gestioninmobiliaria && templ generate
```
Abrir landing page en Chrome DevTools mobile (iPhone 14). Expected: hamburger aparece, CTA y links se ocultan. Al clickear hamburger aparece menú animado con links. Clic fuera cierra.

- [ ] **Step 6: Commit**

```bash
cd /Users/matiaspereira/jcp-gestioninmobiliaria/jcp-gestioninmobiliaria
git add internal/templates/layouts/base.templ
git commit -m "feat: add mobile hamburger menu to landing/login base layout"
```

---

## Task 2: Mobile top header + drawer en `app.templ`

**Files:**
- Modify: `internal/templates/layouts/app.templ`

El app shell (propiedades, guardadas, detalle) no tiene barra superior en mobile. Se agrega: header sticky con logo+hamburguesa, drawer lateral oscuro con nav completo, y "Guardadas" al sidebar desktop.

- [ ] **Step 1: Agregar CSS del mobile header y drawer**

En `app.templ`, ANTES del cierre `</style>`, agregar:

```css
/* Mobile top header */
.mobile-header{display:none;align-items:center;justify-content:space-between;
  height:56px;padding:0 16px;background:#fff;border-bottom:1px solid var(--outline);
  flex-shrink:0}
.mob-logo-link{display:flex;align-items:center;gap:8px;text-decoration:none}
.mob-logo-box{width:32px;height:32px;background:var(--blue);border-radius:8px;
  display:flex;align-items:center;justify-content:center;color:#fff}
.mob-logo-name{font-family:var(--font-display);font-size:16px;color:#0F172A;font-weight:600}
.mob-ham-btn{width:38px;height:38px;border-radius:9px;border:1px solid var(--outline);
  background:#fff;cursor:pointer;display:flex;align-items:center;justify-content:center;
  color:#111827;transition:background .18s;font-family:inherit}
.mob-ham-btn:hover{background:#F8FAFC}
/* Drawer */
.mob-drawer{position:fixed;top:0;left:-100%;width:min(280px,85vw);height:100vh;
  background:#111827;z-index:300;transition:left .28s cubic-bezier(0.05,0.7,0.1,1);
  display:flex;flex-direction:column;overflow-y:auto}
.mob-drawer.open{left:0}
.mob-drawer-backdrop{position:fixed;inset:0;background:rgba(0,0,0,.55);z-index:299;
  cursor:pointer;opacity:0;pointer-events:none;transition:opacity .28s}
.mob-drawer-backdrop.open{opacity:1;pointer-events:auto}
.mob-drawer-top{padding:18px 16px;border-bottom:1px solid rgba(255,255,255,.07);
  display:flex;align-items:center;justify-content:space-between;flex-shrink:0}
.mob-drawer-close{width:32px;height:32px;border-radius:8px;border:none;
  background:rgba(255,255,255,.08);color:rgba(255,255,255,.6);cursor:pointer;
  display:flex;align-items:center;justify-content:center;transition:background .18s}
.mob-drawer-close:hover{background:rgba(255,255,255,.15)}
.mob-drawer-nav{flex:1;padding:12px 10px;display:flex;flex-direction:column;gap:2px}
.mob-drawer-item{display:flex;align-items:center;gap:10px;padding:11px 12px;
  border-radius:10px;color:rgba(255,255,255,.7);text-decoration:none;font-size:14px;
  font-weight:500;transition:all .18s;font-family:var(--font-body)}
.mob-drawer-item:hover{background:rgba(255,255,255,.08);color:#fff}
.mob-drawer-item.active{background:rgba(29,78,216,.3);color:#93C5FD}
.mob-drawer-divider{height:1px;background:rgba(255,255,255,.08);margin:8px 10px}
.mob-drawer-user{padding:14px 16px;border-top:1px solid rgba(255,255,255,.07);
  display:flex;align-items:center;gap:10px;flex-shrink:0}
.save-badge-pill{margin-left:auto;background:rgba(29,78,216,.35);color:#93C5FD;
  font-size:10px;font-weight:700;border-radius:9999px;padding:2px 7px;display:none}
```

Reemplazar el `@media(max-width:900px)` existente con:

```css
@media(max-width:900px){
  .sb{display:none}
  .app-main{width:100%}
  .mobile-nav{display:flex;position:fixed;bottom:0;left:0;right:0;height:60px;
    background:#fff;border-top:1px solid #E2E8F0;z-index:200;align-items:stretch}
  .content-split,.detail-scroll,.guard-scroll{padding-bottom:80px}
  .mobile-header{display:flex}
}
```

- [ ] **Step 2: Agregar "Guardadas" al sidebar desktop**

En `app.templ`, en el bloque `.sb-nav`, después de `<a href="/propiedades" class="sb-item active">...Explorar</a>`, agregar:

```templ
<a href="/guardadas" class="sb-item">
    <span class="ms">favorite</span>
    Guardadas
    <span id="sb-saved-count" class="save-badge-pill"></span>
</a>
```

- [ ] **Step 3: Agregar mobile-header dentro de app-main**

En `app.templ`, dentro de `<div class="app-main">`, ANTES de `{ children... }`, agregar:

```templ
<header class="mobile-header" id="mobile-header">
    <a href="/propiedades" class="mob-logo-link">
        <div class="mob-logo-box"><span class="ms ms-fill" style="font-size:18px">home</span></div>
        <span class="mob-logo-name">JCP Gestión</span>
    </a>
    <button class="mob-ham-btn" id="mob-ham-btn" aria-label="Abrir menú">
        <span class="ms">menu</span>
    </button>
</header>
```

- [ ] **Step 4: Agregar drawer y backdrop después de mn-overlay**

Después de `<div class="mn-overlay" id="mn-overlay">...</div>` y ANTES del `<script>`, agregar:

```templ
<div class="mob-drawer-backdrop" id="mob-drawer-backdrop"></div>
<div class="mob-drawer" id="mob-drawer" role="dialog" aria-label="Menú principal">
    <div class="mob-drawer-top">
        <div style="display:flex;align-items:center;gap:10px">
            <div class="sb-logo-box"><span class="ms ms-fill">home</span></div>
            <div>
                <div class="sb-logo-name">JCP Gestión</div>
                <div class="sb-logo-sub">Inmobiliaria</div>
            </div>
        </div>
        <button class="mob-drawer-close" id="mob-drawer-close" aria-label="Cerrar menú">
            <span class="ms">close</span>
        </button>
    </div>
    <nav class="mob-drawer-nav">
        <a href="/propiedades" class="mob-drawer-item">
            <span class="ms">search</span> Explorar
        </a>
        <a href="/guardadas" class="mob-drawer-item">
            <span class="ms">favorite</span> Guardadas
            <span id="md-saved-count" class="save-badge-pill"></span>
        </a>
        <a href="https://wa.me/56912345678" class="mob-drawer-item" target="_blank" rel="noopener">
            <span class="ms">chat</span> Contactar a JCP
        </a>
        <div class="mob-drawer-divider"></div>
        <a href="/auth/logout" class="mob-drawer-item" style="color:#FCA5A5">
            <span class="ms">logout</span> Cerrar sesión
        </a>
    </nav>
    <div class="mob-drawer-user">
        <div class="sb-avatar">{ initials(user.Name) }</div>
        <div class="sb-user-info">
            <div class="sb-user-name">{ user.Name }</div>
            <div class="sb-user-email">{ user.Email }</div>
        </div>
    </div>
</div>
```

- [ ] **Step 5: Agregar JS del drawer al bloque script existente**

En el `<script>` de `app.templ`, dentro del IIFE `(function(){...}())`, ANTES del cierre `}());`, agregar:

```javascript
// Mobile drawer
(function(){
  var hamBtn=document.getElementById('mob-ham-btn');
  var drawer=document.getElementById('mob-drawer');
  var backdrop=document.getElementById('mob-drawer-backdrop');
  var closeBtn=document.getElementById('mob-drawer-close');
  function openDrawer(){
    if(!drawer)return;
    drawer.classList.add('open');
    backdrop.classList.add('open');
    document.body.style.overflow='hidden';
  }
  function closeDrawer(){
    if(!drawer)return;
    drawer.classList.remove('open');
    backdrop.classList.remove('open');
    document.body.style.overflow='';
  }
  if(hamBtn)hamBtn.addEventListener('click',openDrawer);
  if(backdrop)backdrop.addEventListener('click',closeDrawer);
  if(closeBtn)closeBtn.addEventListener('click',closeDrawer);
  // Marcar item activo en drawer
  var curPath=window.location.pathname;
  document.querySelectorAll('.mob-drawer-item[href]').forEach(function(item){
    var href=item.getAttribute('href');
    if(!href||!href.startsWith('/'))return;
    if(curPath===href||curPath.startsWith(href+'/'))item.classList.add('active');
  });
  // Mostrar contador de guardadas
  var jcpSv=JSON.parse(localStorage.getItem('jcp_saved')||'[]');
  if(jcpSv.length>0){
    ['sb-saved-count','md-saved-count'].forEach(function(id){
      var el=document.getElementById(id);
      if(el){el.textContent=jcpSv.length;el.style.display='inline'}
    });
  }
}());
```

- [ ] **Step 6: Compilar y verificar**

```bash
cd /Users/matiaspereira/jcp-gestioninmobiliaria/jcp-gestioninmobiliaria && templ generate
```
En mobile (≤900px): header con logo+hamburguesa arriba, drawer abre desde izquierda con Explorar/Guardadas/Contacto/Cerrar sesión. En desktop: sidebar muestra "Guardadas" con ícono heart.

- [ ] **Step 7: Commit**

```bash
git add internal/templates/layouts/app.templ
git commit -m "feat: add mobile top header, slide drawer, and Guardadas sidebar link"
```

---

## Task 3: Fix corte de tarjetas por navbar inferior en mobile

Cubierto en Task 2 Step 1. El media query ahora incluye `.guard-scroll` junto a `.content-split` y `.detail-scroll` con `padding-bottom:80px`.

- [ ] **Step 1: Verificar que `.guard-scroll` tiene padding en mobile**

Confirmar en `app.templ` que el media query tiene:
```css
.content-split,.detail-scroll,.guard-scroll{padding-bottom:80px}
```

- [ ] **Step 2: Arrancar servidor y verificar en mobile**

```bash
go run ./cmd/server/main.go
```
Abrir `/propiedades` en mobile → scroll al final → última tarjeta completamente visible sobre el nav inferior. Abrir `/guardadas` → mismo resultado.

- [ ] **Step 3: Commit** (si no se incluyó en Task 2)

Ya incluido en el commit de Task 2.

---

## Task 4: Arreglar Guardadas — htmx timing

**Files:**
- Modify: `internal/templates/web/guardadas.templ`
- Modify: `internal/handlers/fragments/propiedades.go`

Bug: `htmx.ajax()` falla porque htmx se carga con `defer` (async) y el script inline corre antes. Fix: usar HTMX `hx-get` + `hx-trigger` actualizados vía JS con `htmx.process()` después de que htmx esté disponible. El servidor retorna `<div id="guard-results">` completo para outerHTML swap.

- [ ] **Step 1: Actualizar GuardasFragment para retornar wrapper div**

En `internal/handlers/fragments/propiedades.go`, en la función `GuardasFragment`, cambiar la línea:

```go
sb.WriteString(`<div class="prop-grid">`)
```
a:
```go
sb.WriteString(`<div id="guard-results"><div class="prop-grid">`)
```

Y cambiar el cierre:
```go
sb.WriteString(`</div>`)
```
a:
```go
sb.WriteString(`</div></div>`)
```

También cambiar el caso de items vacíos para que también incluya el wrapper:
```go
// Caso sin IDs
return c.SendString(`<div id="guard-results"><div style="text-align:center;padding:48px;color:#94A3B8;font-size:14px">Sin propiedades guardadas.</div></div>`)

// Caso sin resultados encontrados
return c.SendString(`<div id="guard-results"><div style="text-align:center;padding:48px;color:#94A3B8;font-size:14px">No se encontraron las propiedades guardadas.</div></div>`)
```

- [ ] **Step 2: Actualizar script en `guardadas.templ`**

En `guardadas.templ`, reemplazar el bloque `<script>` completo con:

```templ
<script>
(function(){
  var saved=JSON.parse(localStorage.getItem('jcp_saved')||'[]');
  var results=document.getElementById('guard-results');
  var empty=document.getElementById('guard-empty');

  if(saved.length===0){
    if(results)results.style.display='none';
    if(empty)empty.style.display='block';
    return;
  }

  // Wait for htmx (loaded with defer) then trigger via hx-get + htmx.process
  function doLoad(){
    if(!window.htmx){setTimeout(doLoad,50);return;}
    if(!results)return;
    var url='/fragments/guardadas?ids='+encodeURIComponent(saved.join(','));
    results.setAttribute('hx-get',url);
    results.setAttribute('hx-trigger','guardadas-load');
    results.setAttribute('hx-swap','outerHTML');
    htmx.process(results);
    htmx.trigger(results,'guardadas-load');
    document.body.addEventListener('htmx:afterSwap',function(e){
      if(e.detail&&e.detail.target&&e.detail.target.id==='guard-results'){
        if(window.jcpInitSaved)window.jcpInitSaved();
      }
    },{once:true});
  }
  doLoad();
}());
</script>
```

- [ ] **Step 3: Compilar y probar**

```bash
templ generate && go run ./cmd/server/main.go
```
Guardar 2-3 propiedades desde el listado (clickear corazón). Navegar a `/guardadas`. Expected: las tarjetas se cargan correctamente sin errores en consola.

- [ ] **Step 4: Commit**

```bash
git add internal/templates/web/guardadas.templ internal/handlers/fragments/propiedades.go
git commit -m "fix: resolve htmx timing in guardadas using process+trigger pattern"
```

---

## Task 5: Modal de contacto con WhatsApp en `propiedad.templ`

**Files:**
- Modify: `internal/templates/web/propiedad.templ`

El botón "Enviar consulta" es inerte. Se agrega modal con form (nombre, teléfono, consulta) → al enviar abre WhatsApp con mensaje pre-formateado.

- [ ] **Step 1: Agregar CSS del modal**

En `propiedad.templ`, dentro de `<style>`, ANTES del cierre `</style>`, agregar:

```css
/* Contact modal */
.contact-modal{display:none;position:fixed;inset:0;z-index:500;
  align-items:center;justify-content:center;padding:16px}
.contact-modal.open{display:flex}
.cm-backdrop{position:absolute;inset:0;background:rgba(15,23,42,.6);cursor:pointer}
.cm-box{position:relative;background:#fff;border-radius:20px;padding:28px;
  width:min(480px,100%);max-height:90vh;overflow-y:auto;
  box-shadow:0 24px 60px rgba(0,0,0,.22);
  animation:cmIn .22s cubic-bezier(0.05,0.7,0.1,1)}
@keyframes cmIn{from{opacity:0;transform:translateY(14px) scale(.97)}to{opacity:1;transform:none}}
.cm-header{display:flex;align-items:center;justify-content:space-between;margin-bottom:6px}
.cm-title{font-family:'Cormorant Garamond',Georgia,serif;font-size:22px;font-weight:700;color:#111827}
.cm-close-btn{width:32px;height:32px;border-radius:8px;border:1px solid #E2E8F0;
  background:#F8FAFC;cursor:pointer;display:flex;align-items:center;justify-content:center;
  color:#64748B;transition:all .18s}
.cm-close-btn:hover{background:#FEE2E2;border-color:#FCA5A5;color:#EF4444}
.cm-sub{font-size:13px;color:#64748B;margin-bottom:16px;padding-bottom:16px;
  border-bottom:1px solid #F1F5F9}
.cm-prop-pill{font-size:12px;font-weight:600;color:#1D4ED8;background:#EFF6FF;
  border:1px solid #DBEAFE;border-radius:7px;padding:5px 10px;
  display:inline-block;margin-bottom:18px;max-width:100%;overflow:hidden;
  text-overflow:ellipsis;white-space:nowrap}
.cm-field{display:flex;flex-direction:column;gap:5px;margin-bottom:14px}
.cm-label{font-size:11px;font-weight:600;color:#374151;letter-spacing:.05em;text-transform:uppercase}
.cm-input,.cm-textarea{padding:11px 14px;border:1.5px solid #E2E8F0;border-radius:10px;
  font-size:14px;font-family:'Outfit',system-ui,sans-serif;outline:none;
  color:#111827;transition:border-color .2s;width:100%}
.cm-input:focus,.cm-textarea:focus{border-color:#1D4ED8;box-shadow:0 0 0 3px rgba(29,78,216,.08)}
.cm-textarea{resize:vertical;min-height:90px}
.cm-submit{width:100%;padding:14px;background:#25D366;color:#fff;border:none;
  border-radius:12px;font-size:14px;font-weight:700;cursor:pointer;
  display:flex;align-items:center;justify-content:center;gap:8px;
  font-family:'Outfit',system-ui,sans-serif;margin-top:6px;transition:all .22s}
.cm-submit:hover{background:#1DAB54;transform:translateY(-1px);
  box-shadow:0 6px 20px rgba(37,211,102,.35)}
```

- [ ] **Step 2: Agregar modal HTML**

En `propiedad.templ`, después del cierre `</div>` del `<div class="detail-scroll">` y ANTES de `<link rel="stylesheet" href=".../flatpickr...">`, agregar:

```templ
<div class="contact-modal" id="contact-modal" role="dialog" aria-modal="true" aria-labelledby="cm-modal-title">
    <div class="cm-backdrop" id="cm-backdrop"></div>
    <div class="cm-box">
        <div class="cm-header">
            <h3 class="cm-title" id="cm-modal-title">Enviar consulta</h3>
            <button class="cm-close-btn" id="cm-close" type="button" aria-label="Cerrar">
                <span class="ms ms-sm">close</span>
            </button>
        </div>
        <p class="cm-sub">Te responderemos por WhatsApp a la brevedad.</p>
        <div class="cm-prop-pill">{ d.Titulo }</div>
        <form id="cm-form" data-wa-phone={ d.WhatsappPhone } data-prop-title={ d.Titulo }>
            <div class="cm-field">
                <label class="cm-label" for="cm-nombre">Tu nombre *</label>
                <input class="cm-input" id="cm-nombre" name="nombre" type="text"
                    placeholder="Juan Pérez" required/>
            </div>
            <div class="cm-field">
                <label class="cm-label" for="cm-telefono">Teléfono (WhatsApp)</label>
                <input class="cm-input" id="cm-telefono" name="telefono" type="tel"
                    placeholder="+56 9 1234 5678"/>
            </div>
            <div class="cm-field">
                <label class="cm-label" for="cm-mensaje">Tu consulta *</label>
                <textarea class="cm-textarea" id="cm-mensaje" name="mensaje" rows="3"
                    placeholder="Hola, me interesa esta propiedad y quisiera más información..." required></textarea>
            </div>
            <button type="submit" class="cm-submit">
                <span class="ms ms-sm ms-fill">chat</span> Enviar por WhatsApp
            </button>
        </form>
    </div>
</div>
```

- [ ] **Step 3: Conectar el botón "Enviar consulta" al modal**

En `propiedad.templ`, cambiar:

```templ
<button class="sb-cta sb-form"><span class="ms ms-sm">mail</span> Enviar consulta</button>
```

por:

```templ
<button class="sb-cta sb-form" id="open-contact-modal" type="button">
    <span class="ms ms-sm">mail</span> Enviar consulta
</button>
```

- [ ] **Step 4: Agregar JS del modal al script existente**

En el `<script>` de `propiedad.templ`, ANTES del cierre `}());`, agregar:

```javascript
// Contact modal
(function(){
  var openBtn=document.getElementById('open-contact-modal');
  var modal=document.getElementById('contact-modal');
  var backdrop=document.getElementById('cm-backdrop');
  var closeBtn=document.getElementById('cm-close');
  var form=document.getElementById('cm-form');
  function openModal(){
    if(modal){modal.classList.add('open');document.body.style.overflow='hidden';}
  }
  function closeModal(){
    if(modal){modal.classList.remove('open');document.body.style.overflow='';}
  }
  if(openBtn)openBtn.addEventListener('click',openModal);
  if(backdrop)backdrop.addEventListener('click',closeModal);
  if(closeBtn)closeBtn.addEventListener('click',closeModal);
  document.addEventListener('keydown',function(e){if(e.key==='Escape')closeModal();});
  if(form){
    form.addEventListener('submit',function(e){
      e.preventDefault();
      var nombre=(form.nombre&&form.nombre.value||'').trim();
      var telefono=(form.telefono&&form.telefono.value||'').trim();
      var mensaje=(form.mensaje&&form.mensaje.value||'').trim();
      var propTitle=form.dataset.propTitle||'la propiedad';
      var waPhone=(form.dataset.waPhone&&form.dataset.waPhone.length>4)
        ?form.dataset.waPhone:'56912345678';
      var parts=['Hola! Consulto sobre: '+propTitle];
      if(nombre)parts.push('Nombre: '+nombre);
      if(telefono)parts.push('Teléfono: '+telefono);
      if(mensaje)parts.push('\n'+mensaje);
      var text=parts.join('\n');
      window.open('https://wa.me/'+waPhone+'?text='+encodeURIComponent(text),'_blank');
      closeModal();
    });
  }
}());
```

- [ ] **Step 5: Compilar y probar**

```bash
templ generate && go run ./cmd/server/main.go
```
Ir a cualquier propiedad. Clickear "Enviar consulta" → modal aparece animado. Llenar nombre + consulta → Enviar por WhatsApp → se abre wa.me con mensaje pre-rellenado con título de propiedad. Tecla Escape cierra.

- [ ] **Step 6: Commit**

```bash
git add internal/templates/web/propiedad.templ
git commit -m "feat: add WhatsApp contact form modal to property detail"
```

---

## Task 6: Registro de visitantes Google en Admin

**Files:**
- Modify: `internal/handlers/web/auth.go`
- Modify: `internal/routes/public.go`
- Modify: `internal/handlers/admin/handlers.go`
- Modify: `internal/routes/admin.go`
- Create: `internal/templates/admin/pages/visitors.html`
- Modify: `internal/templates/admin/pages/dashboard.html`

Al hacer Google OAuth, loguear email/nombre/foto/IP en PocketBase. Admin ve el CRM en `/admin/visitors`.

**Prerequisito (manual):** Crear la colección `visitor_logs` en PocketBase admin (`/pb/_/`) con campos: `email` (Text), `name` (Text), `picture` (Text), `ip` (Text), `user_agent` (Text).

- [ ] **Step 1: Actualizar `auth.go` para aceptar pb y loguear visitantes**

Reemplazar `internal/handlers/web/auth.go` completo:

```go
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
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
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
func GoogleCallback(cfg *config.Config, pb *pocketbase.PocketBase) fiber.Handler {
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

		// Log visitor asynchronously — never blocks the auth flow
		go logVisitor(pb, gu, c.IP(), c.Get("User-Agent"))

		next := c.Query("next", "/propiedades")
		return c.Redirect(next)
	}
}

// logVisitor saves visitor login data to visitor_logs.
// Silently swallows errors so auth is never affected.
func logVisitor(pb *pocketbase.PocketBase, u googleUserInfo, ip, ua string) {
	collection, err := pb.FindCollectionByNameOrId("visitor_logs")
	if err != nil {
		return // collection not created yet
	}
	record := core.NewRecord(collection)
	record.Set("email", u.Email)
	record.Set("name", u.Name)
	record.Set("picture", u.Picture)
	record.Set("ip", ip)
	record.Set("user_agent", ua)
	_ = pb.Save(record)
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

- [ ] **Step 2: Actualizar `public.go` — pasar pb a GoogleCallback**

En `internal/routes/public.go`, cambiar:
```go
app.Get("/auth/google/callback", web.GoogleCallback(cfg))
```
por:
```go
app.Get("/auth/google/callback", web.GoogleCallback(cfg, pb))
```

- [ ] **Step 3: Agregar VisitorLogs a `handlers/admin/handlers.go`**

Al final de `internal/handlers/admin/handlers.go`, agregar:

```go
// VisitorLogs muestra el CRM de visitantes que ingresaron con Google.
func VisitorLogs(cfg *config.Config, pb *pocketbase.PocketBase) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if c.Query("fragment") != "rows" {
			return c.SendFile("./internal/templates/admin/pages/visitors.html")
		}
		records, err := pb.FindRecordsByFilter("visitor_logs", "", "-created", 100, 0)
		var sb strings.Builder
		if err != nil || len(records) == 0 {
			sb.WriteString(`<tr><td colspan="4" style="text-align:center;padding:40px;color:var(--md-outline)">Sin visitantes registrados aún.</td></tr>`)
		} else {
			for _, r := range records {
				created := "—"
				if dt := r.GetDateTime("created"); !dt.IsZero() {
					created = dt.Time().Format("02/01/2006 15:04")
				}
				picture := r.GetString("picture")
				name := r.GetString("name")
				initial := "?"
				if runes := []rune(name); len(runes) > 0 {
					initial = string(runes[:1])
				}
				var avatarTag string
				if picture != "" {
					avatarTag = fmt.Sprintf(
						`<img src="%s" style="width:36px;height:36px;border-radius:50%%;object-fit:cover;flex-shrink:0" loading="lazy"/>`,
						template.HTMLEscapeString(picture),
					)
				} else {
					avatarTag = fmt.Sprintf(
						`<div style="width:36px;height:36px;border-radius:50%%;background:linear-gradient(135deg,#3B82F6,#8B5CF6);display:flex;align-items:center;justify-content:center;color:#fff;font-size:14px;font-weight:700;flex-shrink:0">%s</div>`,
						template.HTMLEscapeString(initial),
					)
				}
				sb.WriteString(fmt.Sprintf(`<tr>
<td style="padding:14px 24px">
  <div style="display:flex;align-items:center;gap:10px">
    %s
    <div>
      <div style="font-weight:600;font-size:14px">%s</div>
      <div style="font-size:12px;color:var(--md-outline)">%s</div>
    </div>
  </div>
</td>
<td style="padding:14px 24px;font-size:13px">%s</td>
<td style="padding:14px 24px;font-size:13px;color:var(--md-on-surface-variant)">%s</td>
<td style="padding:14px 24px;font-size:12px;color:var(--md-outline);max-width:260px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap">%s</td>
</tr>`,
					avatarTag,
					template.HTMLEscapeString(name),
					template.HTMLEscapeString(r.GetString("email")),
					template.HTMLEscapeString(created),
					template.HTMLEscapeString(r.GetString("ip")),
					template.HTMLEscapeString(r.GetString("user_agent")),
				))
			}
		}
		c.Set("Content-Type", "text/html; charset=utf-8")
		return c.SendString(sb.String())
	}
}
```

- [ ] **Step 4: Agregar ruta en `routes/admin.go`**

En `internal/routes/admin.go`, después de `adm.Get("/whatsapp-logs", ...)`:

```go
adm.Get("/visitors", admin.VisitorLogs(cfg, pb))
```

- [ ] **Step 5: Crear `visitors.html`**

Crear `internal/templates/admin/pages/visitors.html` con el layout del admin (replicar estructura de dashboard.html pero adaptada para la tabla de visitantes). El tbody se carga con un script usando fetch + DOMParser:

```html
<!DOCTYPE html>
<html lang="es">
<head>
<meta charset="UTF-8"/>
<meta name="viewport" content="width=device-width, initial-scale=1.0"/>
<title>Visitantes — JCP Admin</title>
<link rel="icon" type="image/svg+xml" href="/static/favicon.svg"/>
<link href="https://fonts.googleapis.com/css2?family=DM+Serif+Display:ital@0;1&family=DM+Sans:ital,opsz,wght@0,9..40,300;0,9..40,400;0,9..40,500;0,9..40,600;1,9..40,300&display=swap" rel="stylesheet"/>
<link href="https://fonts.googleapis.com/icon?family=Material+Symbols+Outlined" rel="stylesheet"/>
<style>
:root{
  --md-primary:#3D3DEF;--md-on-primary:#fff;
  --md-primary-container:#E0E0FF;--md-on-primary-container:#0000AC;
  --md-primary-dark:#2828CC;
  --md-surface:#FAFAFA;--md-surface-dim:#E8E8F5;
  --md-surface-container-low:#F7F7FF;--md-surface-container:#F2F2FB;
  --md-surface-container-high:#EBEBF5;--md-surface-bright:#FFFFFF;
  --md-on-surface:#1C1B1F;--md-on-surface-variant:#49454F;
  --md-outline:#7C7B7F;--md-outline-variant:#CAC4D0;
  --r-sm:12px;--r-md:16px;--r-lg:24px;--r-full:9999px;
  --font-display:'DM Serif Display',Georgia,serif;
  --font-body:'DM Sans',system-ui,sans-serif;
  --sidebar-w:260px;--topbar-h:64px;
}
*,*::before,*::after{box-sizing:border-box;margin:0;padding:0}
html{-webkit-font-smoothing:antialiased}
body{font-family:var(--font-body);background:var(--md-surface-dim);color:var(--md-on-surface);display:flex;min-height:100vh}
.sidebar{width:var(--sidebar-w);min-height:100vh;position:fixed;left:0;top:0;z-index:100;
  background:var(--md-surface-bright);border-right:1px solid var(--md-outline-variant);
  display:flex;flex-direction:column;padding:20px 12px}
.sb-brand{display:flex;align-items:center;gap:10px;padding:0 8px 20px;
  border-bottom:1px solid var(--md-outline-variant);margin-bottom:12px}
.sb-icon{width:40px;height:40px;border-radius:10px;background:var(--md-primary);
  display:flex;align-items:center;justify-content:center;color:#fff;
  font-family:var(--font-display);font-size:18px}
.sb-name{font-family:var(--font-display);font-size:15px;color:var(--md-on-surface);line-height:1.1}
.sb-sub{font-size:10px;letter-spacing:.1em;text-transform:uppercase;color:var(--md-outline);font-weight:500}
.sb-section{margin-bottom:8px}
.sb-section-title{font-size:10px;font-weight:600;letter-spacing:.12em;text-transform:uppercase;
  color:var(--md-outline);padding:12px 12px 6px}
.sb-link{display:flex;align-items:center;gap:10px;padding:10px 12px;border-radius:var(--r-sm);
  font-size:13.5px;color:var(--md-on-surface-variant);text-decoration:none;transition:background 150ms,color 150ms}
.sb-link:hover{background:var(--md-surface-container-high);color:var(--md-on-surface)}
.sb-link.active{background:var(--md-primary-container);color:var(--md-on-primary-container);font-weight:500}
.sb-link .material-symbols-outlined{font-size:20px}
.sb-spacer{flex:1}
.sb-user{padding:12px;border-top:1px solid var(--md-outline-variant);margin-top:8px;
  display:flex;align-items:center;gap:10px}
.sb-avatar{width:36px;height:36px;border-radius:var(--r-full);background:var(--md-primary-container);
  display:flex;align-items:center;justify-content:center;font-size:14px;font-weight:500;
  color:var(--md-on-primary-container)}
.sb-user-name{font-size:13px;font-weight:500;color:var(--md-on-surface)}
.sb-user-role{font-size:11px;color:var(--md-outline)}
.main{margin-left:var(--sidebar-w);flex:1;min-height:100vh}
.topbar{height:var(--topbar-h);background:rgba(250,250,255,.72);backdrop-filter:blur(20px) saturate(180%);
  border-bottom:1px solid rgba(61,61,239,.10);display:flex;align-items:center;
  justify-content:space-between;padding:0 32px;position:sticky;top:0;z-index:50}
.topbar-title{font-family:var(--font-display);font-size:20px}
.tb-btn{display:flex;align-items:center;gap:6px;padding:8px 16px;border-radius:var(--r-full);
  font-size:13px;font-weight:500;border:none;cursor:pointer;font-family:var(--font-body);
  background:transparent;color:var(--md-on-surface-variant);border:1px solid var(--md-outline-variant);
  transition:background 150ms}
.tb-btn:hover{background:var(--md-surface-container-high)}
.content{padding:32px}
.card{background:var(--md-surface-bright);border-radius:var(--r-lg);
  border:1px solid var(--md-outline-variant);overflow:hidden;margin-bottom:24px}
.card-header{padding:20px 24px;border-bottom:1px solid var(--md-outline-variant);
  display:flex;align-items:center;justify-content:space-between;flex-wrap:wrap;gap:12px}
.card-title{font-family:var(--font-display);font-size:18px}
.card-sub{font-size:12px;color:var(--md-outline)}
table{width:100%;border-collapse:collapse;table-layout:auto}
th{text-align:left;padding:12px 24px;font-size:11px;font-weight:600;letter-spacing:.08em;
  text-transform:uppercase;color:var(--md-outline);background:var(--md-surface-container-low);
  border-bottom:1px solid var(--md-outline-variant)}
tr:last-child td{border:none}
tr:hover td{background:var(--md-surface-container-low)}
@keyframes spin{to{transform:rotate(360deg)}}
@media(max-width:768px){.sidebar{display:none}.main{margin-left:0}.content{padding:20px 16px}}
</style>
</head>
<body>

<aside class="sidebar">
  <div class="sb-brand">
    <div class="sb-icon">JCP</div>
    <div>
      <div class="sb-name">JCP Gestión</div>
      <div class="sb-sub">Inmobiliaria · Admin</div>
    </div>
  </div>
  <div class="sb-section">
    <div class="sb-section-title">General</div>
    <a href="/admin/dashboard" class="sb-link"><span class="material-symbols-outlined">dashboard</span> Dashboard</a>
  </div>
  <div class="sb-section">
    <div class="sb-section-title">Inmobiliaria</div>
    <a href="/admin/propiedades" class="sb-link"><span class="material-symbols-outlined">home_work</span> Propiedades</a>
  </div>
  <div class="sb-section">
    <div class="sb-section-title">CRM</div>
    <a href="/admin/visitors" class="sb-link active"><span class="material-symbols-outlined">person_search</span> Visitantes Google</a>
  </div>
  <div class="sb-section">
    <div class="sb-section-title">Sistema</div>
    <a href="/admin/users" class="sb-link"><span class="material-symbols-outlined">group</span> Usuarios</a>
  </div>
  <div class="sb-spacer"></div>
  <div class="sb-user">
    <div class="sb-avatar">AD</div>
    <div>
      <div class="sb-user-name">Administrador</div>
      <div class="sb-user-role">superadmin</div>
    </div>
  </div>
</aside>

<div class="main">
  <header class="topbar">
    <h1 class="topbar-title">Visitantes Google</h1>
    <div style="display:flex;gap:12px">
      <button class="tb-btn" id="refresh-btn">
        <span class="material-symbols-outlined" style="font-size:16px">refresh</span> Actualizar
      </button>
      <a href="/" target="_blank" class="tb-btn">
        <span class="material-symbols-outlined" style="font-size:16px">open_in_new</span> Ver web
      </a>
    </div>
  </header>
  <div class="content">
    <p style="font-size:13px;color:var(--md-outline);margin-bottom:24px;max-width:640px">
      Registro de personas que iniciaron sesión con Google. Usalo para identificar leads
      y comenzar conversaciones de venta proactivas.
    </p>
    <div class="card">
      <div class="card-header">
        <div>
          <h2 class="card-title">Visitantes registrados</h2>
          <p class="card-sub">Últimos 100 · ordenados por fecha de ingreso</p>
        </div>
      </div>
      <div style="overflow-x:auto">
        <table>
          <thead>
            <tr>
              <th>Visitante</th>
              <th>Fecha</th>
              <th>IP</th>
              <th>Navegador</th>
            </tr>
          </thead>
          <tbody id="visitors-tbody">
            <tr id="loading-row">
              <td colspan="4" style="text-align:center;padding:40px;color:var(--md-outline)">
                <div style="width:24px;height:24px;border:2px solid var(--md-primary);
                  border-top-color:transparent;border-radius:50%;
                  animation:spin 1s linear infinite;margin:0 auto 8px"></div>
                Cargando visitantes...
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</div>

<script>
function loadVisitors() {
  fetch('/admin/visitors?fragment=rows')
    .then(function(r) { return r.text(); })
    .then(function(html) {
      var tbody = document.getElementById('visitors-tbody');
      if (!tbody) return;
      // Parse server HTML safely via DOMParser (no direct JS DOM injection of untrusted strings)
      var parser = new DOMParser();
      var doc = parser.parseFromString('<table><tbody>' + html + '</tbody></table>', 'text/html');
      var parsedTbody = doc.querySelector('tbody');
      tbody.textContent = '';
      if (parsedTbody) {
        Array.from(parsedTbody.childNodes).forEach(function(node) {
          tbody.appendChild(node.cloneNode(true));
        });
      }
    })
    .catch(function() {
      var tbody = document.getElementById('visitors-tbody');
      if (tbody) tbody.textContent = 'Error al cargar visitantes.';
    });
}

loadVisitors();

var refreshBtn = document.getElementById('refresh-btn');
if (refreshBtn) refreshBtn.addEventListener('click', loadVisitors);
</script>
</body>
</html>
```

- [ ] **Step 6: Agregar "Visitantes Google" al sidebar de dashboard.html**

En `internal/templates/admin/pages/dashboard.html`, dentro del sidebar, agregar después de la sección "Inmobiliaria" y antes de "Sistema":

```html
<div class="sidebar-section">
  <div class="sidebar-section-title">CRM</div>
  <a href="/admin/visitors" class="sidebar-link">
    <span class="material-symbols-outlined">person_search</span> Visitantes Google
  </a>
</div>
```

- [ ] **Step 7: Compilar Go y verificar imports**

```bash
go build ./...
```
Expected: sin errores. Si falta import `"github.com/pocketbase/pocketbase/core"` en auth.go, agregarlo a los imports.

- [ ] **Step 8: Crear colección en PocketBase y probar flujo**

1. Ir a `/pb/_/` → Collections → New → nombre `visitor_logs`
2. Agregar campos: `email` (Text), `name` (Text), `picture` (Text), `ip` (Text), `user_agent` (Text) → Save
3. Logout de visitante → `/login` → Google OAuth
4. Ir a `/admin/visitors` → debe aparecer el visitante con foto, nombre, email, IP, fecha

- [ ] **Step 9: Commit**

```bash
git add internal/handlers/web/auth.go internal/routes/public.go \
    internal/handlers/admin/handlers.go internal/routes/admin.go \
    internal/templates/admin/pages/visitors.html \
    internal/templates/admin/pages/dashboard.html
git commit -m "feat: log Google OAuth visitors to PocketBase and add admin CRM view"
```

---

## Verificación final: spec vs plan

| Requisito del usuario | Tarea | Archivos clave |
|----------------------|-------|----------------|
| Navbar mobile con logo + hamburguesa (landing/login) | Task 1 | `base.templ` |
| Navbar mobile con logo + hamburguesa (app) | Task 2 | `app.templ` |
| Tarjetas no cortadas por footer nav mobile | Task 3 | `app.templ` media query |
| Guardadas funcional en sidebar + page | Task 4 | `guardadas.templ`, `propiedades.go` |
| Contacto abre formulario + envía por WhatsApp | Task 5 | `propiedad.templ` |
| Admin registra y muestra visitantes de Google | Task 6 | `auth.go`, `handlers.go`, `visitors.html` |
