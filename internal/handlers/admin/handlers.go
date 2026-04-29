package admin

import (
	"jcp-gestioninmobiliaria/internal/auth"
	"jcp-gestioninmobiliaria/internal/config"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// ── LOGIN ──

func LoginPage(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.SendFile("./internal/templates/admin/pages/login.html")
	}
}

func LoginSubmit(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		email := c.FormValue("email")
		password := c.FormValue("password")
		remember := c.FormValue("remember") == "on"

		if email == "" || password == "" {
			return c.Status(fiber.StatusBadRequest).SendString(
				`<div class="toast toast-error">Email y contraseña requeridos</div>`,
			)
		}

		if email == cfg.AdminEmail && password == cfg.AdminPassword {
			token, err := auth.GenerateToken(cfg, "admin-id", email, "superadmin", "Administrador")
			if err != nil {
				return c.Status(500).SendString(`<div class="toast toast-error">Error generando sesión</div>`)
			}
			expiry := 24 * time.Hour
			if remember {
				expiry = 72 * time.Hour
			}
			c.Cookie(&fiber.Cookie{
				Name:     "csl_token",
				Value:    token,
				Expires:  time.Now().Add(expiry),
				HTTPOnly: true,
				Secure:   cfg.Env == "production",
				SameSite: "Lax",
				Path:     "/",
			})
			c.Set("HX-Redirect", "/admin/dashboard")
			return c.SendString("")
		}

		return c.Status(fiber.StatusUnauthorized).SendString(
			`<div class="toast toast-error">Credenciales incorrectas</div>`,
		)
	}
}

func Logout() fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Cookie(&fiber.Cookie{
			Name:    "csl_token",
			Value:   "",
			Expires: time.Now().Add(-time.Hour),
			Path:    "/",
		})
		c.Set("HX-Redirect", "/admin/login")
		return c.SendString("")
	}
}

// ── DASHBOARD ──

func Dashboard(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.SendFile("./internal/templates/admin/pages/dashboard.html")
	}
}

// DashboardStats renders the 4 stat cards for the dashboard (count of propiedades)
func DashboardStats(cfg *config.Config, pb *pocketbase.PocketBase) fiber.Handler {
	return func(c *fiber.Ctx) error {
		all, _ := pb.FindRecordsByFilter("propiedades", "", "", 1000, 0)
		total := len(all)
		publicadas, destacadas, ventas, arriendos := 0, 0, 0, 0
		for _, r := range all {
			if r.GetString("status") == "publicado" {
				publicadas++
			}
			if r.GetBool("destacada") {
				destacadas++
			}
			switch r.GetString("operacion") {
			case "VENTA":
				ventas++
			case "ARRIENDO":
				arriendos++
			}
		}
		html := fmt.Sprintf(`<div class="stat-card accent">
  <div class="stat-card-label">Propiedades totales</div>
  <div class="stat-card-value">%d</div>
  <div class="stat-card-delta">%d publicadas · %d borradores</div>
</div>
<div class="stat-card">
  <div class="stat-card-label">Destacadas</div>
  <div class="stat-card-value">%d</div>
  <div class="stat-card-delta">en portada</div>
</div>
<div class="stat-card">
  <div class="stat-card-label">En venta</div>
  <div class="stat-card-value">%d</div>
  <div class="stat-card-delta">ventas activas</div>
</div>
<div class="stat-card">
  <div class="stat-card-label">En arriendo</div>
  <div class="stat-card-value">%d</div>
  <div class="stat-card-delta">arriendos activos</div>
</div>`, total, publicadas, total-publicadas, destacadas, ventas, arriendos)
		c.Set("Content-Type", "text/html; charset=utf-8")
		return c.SendString(html)
	}
}

// ── MULTIMEDIA CRUD ──

func MultimediaList(cfg *config.Config, pb *pocketbase.PocketBase) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if c.Query("fragment") != "rows" {
			return c.SendFile("./internal/templates/admin/pages/multimedia.html")
		}
		records, err := pb.FindRecordsByFilter("multimedia", "", "filename", 100, 0)
		var sb strings.Builder
		if err != nil || len(records) == 0 {
			sb.WriteString(`<tr><td colspan="5" style="text-align:center;padding:32px;color:var(--md-outline)">Sin archivos multimedia</td></tr>`)
		} else {
			for _, r := range records {
				estado := r.GetString("estado")
				badgeClass := "badge-info"
				if estado == "publicado" {
					badgeClass = "badge-success"
				} else if estado == "archivado" {
					badgeClass = "badge-warning"
				}
				dur := r.GetFloat("duracion_segundos")
				durStr := "—"
				if dur > 0 {
					durStr = fmt.Sprintf("%.0fs", dur)
				}
				sb.WriteString(fmt.Sprintf(`<tr>
          <td>%s</td><td>%s</td><td>%s</td>
          <td><span class="badge %s">%s</span></td>
          <td>
            <button class="topbar-btn topbar-btn-outline" style="padding:4px 10px;font-size:12px"
              hx-get="/admin/multimedia/%s/edit" hx-target="#modal-container" hx-swap="innerHTML">Editar</button>
            <button class="topbar-btn" style="padding:4px 10px;font-size:12px;background:#FDECEA;color:#B71C1C;border:none;cursor:pointer"
              hx-delete="/admin/multimedia/%s" hx-confirm="¿Eliminar?" hx-target="closest tr" hx-swap="outerHTML swap:300ms">Eliminar</button>
          </td></tr>`,
					template.HTMLEscapeString(r.GetString("filename")),
					template.HTMLEscapeString(r.GetString("type")),
					durStr, badgeClass,
					template.HTMLEscapeString(estado),
					r.Id, r.Id,
				))
			}
		}
		c.Set("Content-Type", "text/html; charset=utf-8")
		return c.SendString(sb.String())
	}
}

func MultimediaForm(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		html := `<div class="modal-overlay" onclick="this.remove()">
  <div class="modal-card" onclick="event.stopPropagation()">
    <div class="modal-header"><h3>Nuevo Archivo Multimedia</h3>
      <button onclick="document.getElementById('modal-container').innerHTML=''" style="background:none;border:none;cursor:pointer;font-size:20px;color:var(--md-outline)">✕</button>
    </div>
    <form hx-post="/admin/multimedia" hx-encoding="multipart/form-data" hx-target="#toast-area" hx-swap="innerHTML">
      <div class="form-field"><label>Nombre</label><input type="text" name="filename" required class="form-input" placeholder="ej: imagen-bienvenida.jpg"/></div>
      <div class="form-field"><label>Tipo</label>
        <select name="type" class="form-input">
          <option value="imagen">Imagen</option><option value="video">Video</option>
          <option value="youtube">YouTube</option><option value="vimeo">Vimeo</option>
        </select>
      </div>
      <div class="form-field"><label>URL del archivo</label><input type="url" name="url_r2" class="form-input" placeholder="https://... URL completa del video o imagen"/></div>
      <div class="form-field"><label>Estado</label>
        <select name="estado" class="form-input">
          <option value="borrador">Borrador</option><option value="publicado">Publicado</option>
        </select>
      </div>
      <div class="modal-actions">
        <button type="button" onclick="document.getElementById('modal-container').innerHTML=''" class="topbar-btn topbar-btn-outline">Cancelar</button>
        <button type="submit" class="topbar-btn topbar-btn-primary">Guardar</button>
      </div>
    </form>
  </div>
</div>`
		c.Set("Content-Type", "text/html; charset=utf-8")
		return c.SendString(html)
	}
}

func MultimediaCreate(cfg *config.Config, pb *pocketbase.PocketBase) fiber.Handler {
	return func(c *fiber.Ctx) error {
		col, err := pb.FindCollectionByNameOrId("multimedia")
		if err != nil {
			return c.Status(500).SendString(`<div class="toast toast-error">Error de BD</div>`)
		}
		r := core.NewRecord(col)
		r.Set("filename", c.FormValue("filename"))
		r.Set("type", c.FormValue("type"))
		r.Set("url_r2", c.FormValue("url_r2"))
		r.Set("duracion_segundos", c.FormValue("duracion_segundos"))
		r.Set("estado", c.FormValue("estado"))
		if err := pb.Save(r); err != nil {
			return c.SendString(`<div class="toast toast-error">Error guardando</div>`)
		}
		return c.SendString(`<div class="toast toast-success" id="toast-area">✅ Archivo guardado
<script>
  document.getElementById('modal-container').innerHTML='';
  htmx.ajax('GET','/admin/multimedia?fragment=rows',{target:'#media-tbody',swap:'innerHTML'});
  setTimeout(function(){var t=document.getElementById('toast-area');if(t)t.innerHTML=''},2000);
</script></div>`)
	}
}

func MultimediaEdit(cfg *config.Config, pb *pocketbase.PocketBase) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id := c.Params("id")
		r, err := pb.FindRecordById("multimedia", id)
		if err != nil {
			return c.Status(404).SendString(`<div class="toast toast-error">No encontrado</div>`)
		}
		html := fmt.Sprintf(`<div class="modal-overlay" onclick="this.remove()">
  <div class="modal-card" onclick="event.stopPropagation()">
    <div class="modal-header"><h3>Editar Multimedia</h3>
      <button onclick="document.getElementById('modal-container').innerHTML=''" style="background:none;border:none;cursor:pointer;font-size:20px;color:var(--md-outline)">✕</button>
    </div>
    <form hx-put="/admin/multimedia/%s" hx-target="#toast-area" hx-swap="innerHTML">
      <div class="form-field"><label>Nombre</label><input type="text" name="filename" value="%s" required class="form-input"/></div>
      <div class="form-field"><label>Tipo</label>
        <select name="type" class="form-input">
          <option value="imagen"%s>Imagen</option><option value="video"%s>Video</option>
          <option value="youtube"%s>YouTube</option><option value="vimeo"%s>Vimeo</option>
        </select>
      </div>
      <div class="form-field"><label>URL (R2)</label><input type="url" name="url_r2" value="%s" class="form-input"/></div>
      <div class="form-field"><label>Estado</label>
        <select name="estado" class="form-input">
          <option value="borrador"%s>Borrador</option><option value="publicado"%s>Publicado</option>
        </select>
      </div>
      <div class="modal-actions">
        <button type="button" onclick="document.getElementById('modal-container').innerHTML=''" class="topbar-btn topbar-btn-outline">Cancelar</button>
        <button type="submit" class="topbar-btn topbar-btn-primary">Actualizar</button>
      </div>
    </form>
  </div>
</div>`,
			r.Id,
			template.HTMLEscapeString(r.GetString("filename")),
			sel(r.GetString("type"), "imagen"),
			sel(r.GetString("type"), "video"),
			sel(r.GetString("type"), "youtube"),
			sel(r.GetString("type"), "vimeo"),
			template.HTMLEscapeString(r.GetString("url_r2")),
			sel(r.GetString("estado"), "borrador"),
			sel(r.GetString("estado"), "publicado"),
		)
		c.Set("Content-Type", "text/html; charset=utf-8")
		return c.SendString(html)
	}
}

func MultimediaUpdate(cfg *config.Config, pb *pocketbase.PocketBase) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id := c.Params("id")
		r, err := pb.FindRecordById("multimedia", id)
		if err != nil {
			return c.SendString(`<div class="toast toast-error">No encontrado</div>`)
		}
		r.Set("filename", c.FormValue("filename"))
		r.Set("type", c.FormValue("type"))
		r.Set("url_r2", c.FormValue("url_r2"))
		r.Set("estado", c.FormValue("estado"))
		if err := pb.Save(r); err != nil {
			return c.SendString(`<div class="toast toast-error">Error actualizando</div>`)
		}
		c.Set("HX-Trigger", "mediaUpdated")
		return c.SendString(`<div class="toast toast-success" id="toast-area">✅ Actualizado
<script>setTimeout(()=>{document.getElementById('modal-container').innerHTML='';var _ta=document.getElementById('toast-area');if(_ta)_ta.innerHTML=''},1500)</script></div>`)
	}
}

func MultimediaDelete(cfg *config.Config, pb *pocketbase.PocketBase) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id := c.Params("id")
		r, err := pb.FindRecordById("multimedia", id)
		if err != nil {
			return c.Status(404).SendString(`<div class="toast toast-error">No encontrado</div>`)
		}
		if err := pb.Delete(r); err != nil {
			return c.SendString(`<div class="toast toast-error">Error eliminando</div>`)
		}
		c.Set("HX-Trigger", "mediaDeleted")
		return c.SendString("")
	}
}

// ── EVENTS CRUD (content_blocks) ──

func EventsList(cfg *config.Config, pb *pocketbase.PocketBase) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if c.Query("fragment") != "rows" {
			// Direct browser access (F5 / bookmark): serve full SPA shell.
			// HTMX navigation sets HX-Request; the JS in dashboard auto-loads this panel.
			if c.Get("HX-Request") != "true" {
				return c.SendFile("./internal/templates/admin/pages/dashboard.html")
			}
			return c.SendFile("./internal/templates/admin/pages/events.html")
		}
		records, err := pb.FindRecordsByFilter("content_blocks",
			"category != 'NOTICIA'", "-date", 100, 0)

		var sb strings.Builder
		if err != nil || len(records) == 0 {
			sb.WriteString(`<tr><td colspan="6" style="text-align:center;padding:32px;color:var(--md-outline)">Sin eventos — agrega uno con el botón de arriba</td></tr>`)
		} else {
			for _, r := range records {
				status := r.GetString("status")
				badgeClass := "badge-warning"
				if status == "publicado" {
					badgeClass = "badge-success"
				}
				urgIcon := ""
				if r.GetBool("urgency") {
					urgIcon = `<span class="badge badge-danger">URGENTE</span>`
				}
				dateStr := "—"
				if dt := r.GetDateTime("date"); !dt.IsZero() {
					dateStr = dt.Time().Format("2 Jan 2006")
				}
				sb.WriteString(fmt.Sprintf(`<tr>
          <td>%s</td><td>%s</td><td>%s</td><td>%s</td>
          <td><span class="badge %s">%s</span></td>
          <td>
            <button class="topbar-btn topbar-btn-outline" style="padding:4px 10px;font-size:12px"
              hx-get="/admin/events/%s/edit" hx-target="#modal-container" hx-swap="innerHTML">Editar</button>
            <button class="topbar-btn" style="padding:4px 10px;font-size:12px;background:#FDECEA;color:#B71C1C;border:none;cursor:pointer"
              hx-delete="/admin/events/%s" hx-confirm="¿Eliminar este evento?" hx-target="closest tr" hx-swap="outerHTML swap:300ms">Eliminar</button>
          </td></tr>`,
					template.HTMLEscapeString(r.GetString("title")),
					template.HTMLEscapeString(r.GetString("category")),
					urgIcon, dateStr,
					badgeClass, template.HTMLEscapeString(status),
					r.Id, r.Id,
				))
			}
		}
		c.Set("Content-Type", "text/html; charset=utf-8")
		return c.SendString(sb.String())
	}
}

func EventForm(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		html := eventFormHTML("", "", "", "", "", "", "")
		c.Set("Content-Type", "text/html; charset=utf-8")
		return c.SendString(html)
	}
}

func EventCreate(cfg *config.Config, pb *pocketbase.PocketBase) fiber.Handler {
	return func(c *fiber.Ctx) error {
		title := c.FormValue("title")
		description := c.FormValue("description")
		if len([]rune(title)) > 120 {
			return c.SendString(`<div class="toast toast-error">El título no puede superar 120 caracteres</div>`)
		}
		if len([]rune(description)) > 600 {
			return c.SendString(`<div class="toast toast-error">La descripción no puede superar 600 caracteres</div>`)
		}
		col, err := pb.FindCollectionByNameOrId("content_blocks")
		if err != nil {
			return c.Status(500).SendString(`<div class="toast toast-error">Error de BD</div>`)
		}
		r := core.NewRecord(col)
		r.Set("title", title)
		r.Set("description", description)
		r.Set("category", c.FormValue("category"))
		r.Set("urgency", c.FormValue("category") == "EMERGENCIA")
		if ds := c.FormValue("date"); ds != "" {
			if t, err2 := time.Parse("2006-01-02T15:04", ds); err2 == nil {
				r.Set("date", t.UTC())
			}
		}
		r.Set("featured", c.FormValue("featured") == "on")
		r.Set("status", c.FormValue("status"))
		r.Set("pdf_url", c.FormValue("pdf_url"))
		r.Set("image_url", c.FormValue("image_url"))
		r.Set("body", c.FormValue("body"))
		if err := pb.Save(r); err != nil {
			return c.SendString(`<div class="toast toast-error">Error guardando</div>`)
		}
		c.Set("HX-Trigger", "eventCreated")
		return c.SendString(`<div class="toast toast-success" id="toast-area">✅ Evento creado
<script>setTimeout(()=>{document.getElementById('modal-container').innerHTML='';var _ta=document.getElementById('toast-area');if(_ta)_ta.innerHTML=''},1500)</script></div>`)
	}
}

func EventEdit(cfg *config.Config, pb *pocketbase.PocketBase) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id := c.Params("id")
		r, err := pb.FindRecordById("content_blocks", id)
		if err != nil {
			return c.Status(404).SendString(`<div class="toast toast-error">No encontrado</div>`)
		}
		dateStr := ""
		if dt := r.GetDateTime("date"); !dt.IsZero() {
			dateStr = dt.Time().Format("2006-01-02T15:04")
		}
		var html string
		if r.GetString("category") == "NOTICIA" {
			html = newsFormHTML(r.Id, r.GetString("title"), r.GetString("description"),
				r.GetString("status"), dateStr, r.GetString("image_url"), r.GetString("body"))
		} else {
			html = eventFormHTML(r.Id, r.GetString("title"), r.GetString("description"),
				r.GetString("category"), r.GetString("status"), dateStr,
				r.GetString("pdf_url"))
		}
		c.Set("Content-Type", "text/html; charset=utf-8")
		return c.SendString(html)
	}
}

func EventUpdate(cfg *config.Config, pb *pocketbase.PocketBase) fiber.Handler {
	return func(c *fiber.Ctx) error {
		title := c.FormValue("title")
		description := c.FormValue("description")
		if len([]rune(title)) > 120 {
			return c.SendString(`<div class="toast toast-error">El título no puede superar 120 caracteres</div>`)
		}
		if len([]rune(description)) > 600 {
			return c.SendString(`<div class="toast toast-error">La descripción no puede superar 600 caracteres</div>`)
		}
		id := c.Params("id")
		r, err := pb.FindRecordById("content_blocks", id)
		if err != nil {
			return c.SendString(`<div class="toast toast-error">No encontrado</div>`)
		}
		r.Set("title", title)
		r.Set("description", description)
		r.Set("category", c.FormValue("category"))
		r.Set("urgency", c.FormValue("category") == "EMERGENCIA")
		if ds := c.FormValue("date"); ds != "" {
			if t, err2 := time.Parse("2006-01-02T15:04", ds); err2 == nil {
				r.Set("date", t.UTC())
			}
		}
		r.Set("featured", c.FormValue("featured") == "on")
		r.Set("status", c.FormValue("status"))
		r.Set("pdf_url", c.FormValue("pdf_url"))
		r.Set("image_url", c.FormValue("image_url"))
		r.Set("body", c.FormValue("body"))
		if err := pb.Save(r); err != nil {
			return c.SendString(`<div class="toast toast-error">Error actualizando</div>`)
		}
		c.Set("HX-Trigger", "eventUpdated")
		return c.SendString(`<div class="toast toast-success" id="toast-area">✅ Evento actualizado
<script>setTimeout(()=>{document.getElementById('modal-container').innerHTML='';var _ta=document.getElementById('toast-area');if(_ta)_ta.innerHTML=''},1500)</script></div>`)
	}
}

func EventDelete(cfg *config.Config, pb *pocketbase.PocketBase) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id := c.Params("id")
		r, err := pb.FindRecordById("content_blocks", id)
		if err != nil {
			return c.Status(404).SendString("")
		}
		if err := pb.Delete(r); err != nil {
			return c.SendString(`<div class="toast toast-error">Error eliminando</div>`)
		}
		c.Set("HX-Trigger", "eventDeleted")
		return c.SendString("")
	}
}

func EventPublish(cfg *config.Config, pb *pocketbase.PocketBase) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id := c.Params("id")
		r, err := pb.FindRecordById("content_blocks", id)
		if err != nil {
			return c.Status(404).SendString("")
		}
		r.Set("status", "publicado")
		if err := pb.Save(r); err != nil {
			return c.SendString(`<div class="toast toast-error">Error publicando</div>`)
		}
		c.Set("HX-Trigger", "eventUpdated")
		return c.SendString(`<div class="toast toast-success">✅ Publicado</div>`)
	}
}

// eventFormHTML builds the create/edit modal form for events (all categories except NOTICIA).
// Urgency is auto-derived from category=EMERGENCIA — no checkbox needed.
func eventFormHTML(id, title, description, category, status, date, pdfUrl string) string {
	method := `hx-post="/admin/events"`
	if id != "" {
		method = fmt.Sprintf(`hx-put="/admin/events/%s"`, id)
	}

	cats := []string{"EMERGENCIA", "REUNIÓN", "INFORMACIÓN", "ACADÉMICO", "EVENTO", "DEPORTIVO"}
	var catOpts strings.Builder
	for _, cat := range cats {
		selected := ""
		if cat == category {
			selected = " selected"
		}
		catOpts.WriteString(fmt.Sprintf(`<option value="%s"%s>%s</option>`, cat, selected, cat))
	}

	return fmt.Sprintf(`<div class="modal-overlay" onclick="this.remove()">
  <div class="modal-card" onclick="event.stopPropagation()">
    <div class="modal-header">
      <h3>%s</h3>
      <button onclick="document.getElementById('modal-container').innerHTML=''" style="background:none;border:none;cursor:pointer;font-size:20px;color:var(--md-outline)">✕</button>
    </div>
    <form %s hx-target="#toast-area" hx-swap="innerHTML">
      <div class="form-field">
        <label>Título <small id="ev-title-cnt" style="float:right;font-size:11px;color:var(--md-outline)">0/120</small></label>
        <input type="text" name="title" value="%s" required class="form-input" maxlength="120" placeholder="Título del comunicado"
          oninput="(function(el){var c=document.getElementById('ev-title-cnt');if(c){c.textContent=el.value.length+'/120';c.style.color=el.value.length>100?'#B71C1C':''}})(this)"
          onchange="(function(el){var c=document.getElementById('ev-title-cnt');if(c)c.textContent=el.value.length+'/120'})(this)"/>
      </div>
      <div class="form-field">
        <label>Descripción <small id="ev-desc-cnt" style="float:right;font-size:11px;color:var(--md-outline)">0/600</small></label>
        <textarea name="description" class="form-input" rows="3" maxlength="600" placeholder="Descripción breve..."
          oninput="(function(el){var c=document.getElementById('ev-desc-cnt');if(c){c.textContent=el.value.length+'/600';c.style.color=el.value.length>500?'#B71C1C':''}})( this)">%s</textarea>
      </div>
      <div class="form-field"><label>Link PDF (opcional)</label><input type="url" name="pdf_url" value="%s" class="form-input" placeholder="https://... enlace al comunicado PDF"/></div>
      <div class="form-row">
        <div class="form-field"><label>Categoría</label><select name="category" class="form-input">%s</select></div>
        <div class="form-field"><label>Estado</label>
          <select name="status" class="form-input">
            <option value="borrador"%s>Borrador</option>
            <option value="publicado"%s>Publicado</option>
            <option value="archivado"%s>Archivado</option>
          </select>
        </div>
      </div>
      <div class="form-field"><label>Fecha</label><input type="datetime-local" name="date" value="%s" class="form-input"/></div>
      <div class="modal-actions">
        <button type="button" onclick="document.getElementById('modal-container').innerHTML=''" class="topbar-btn topbar-btn-outline">Cancelar</button>
        <button type="submit" class="topbar-btn topbar-btn-primary">Guardar</button>
      </div>
    </form>
  </div>
</div>`,
		map[bool]string{true: "Editar Evento", false: "Nuevo Evento"}[id != ""],
		method,
		template.HTMLEscapeString(title),
		template.HTMLEscapeString(description),
		template.HTMLEscapeString(pdfUrl),
		catOpts.String(),
		sel(status, "borrador"),
		sel(status, "publicado"),
		sel(status, "archivado"),
		date,
	)
}

// newsFormHTML builds the create/edit modal form for noticias (category=NOTICIA only)
func newsFormHTML(id, title, description, status, date, imageUrl, body string) string {
	method := `hx-post="/admin/news"`
	if id != "" {
		method = fmt.Sprintf(`hx-put="/admin/news/%s"`, id)
	}
	return fmt.Sprintf(`<div class="modal-overlay" onclick="this.remove()">
  <div class="modal-card" onclick="event.stopPropagation()">
    <div class="modal-header">
      <h3>%s</h3>
      <button onclick="document.getElementById('modal-container').innerHTML=''" style="background:none;border:none;cursor:pointer;font-size:20px;color:var(--md-outline)">✕</button>
    </div>
    <form %s hx-target="#toast-area" hx-swap="innerHTML">
      <input type="hidden" name="category" value="NOTICIA"/>
      <div class="form-field">
        <label>Título <small id="nw-title-cnt" style="float:right;font-size:11px;color:var(--md-outline)">0/120</small></label>
        <input type="text" name="title" value="%s" required class="form-input" maxlength="120" placeholder="Título de la noticia"
          oninput="(function(el){var c=document.getElementById('nw-title-cnt');if(c){c.textContent=el.value.length+'/120';c.style.color=el.value.length>100?'#B71C1C':''}})(this)"/>
      </div>
      <div class="form-field">
        <label>Resumen <small id="nw-desc-cnt" style="float:right;font-size:11px;color:var(--md-outline)">0/600</small></label>
        <textarea name="description" class="form-input" rows="2" maxlength="600" placeholder="Resumen breve..."
          oninput="(function(el){var c=document.getElementById('nw-desc-cnt');if(c){c.textContent=el.value.length+'/600';c.style.color=el.value.length>500?'#B71C1C':''}})( this)">%s</textarea>
      </div>
      <div class="form-field"><label>Contenido completo</label><textarea name="body" class="form-input" rows="6" placeholder="Escribe el artículo completo aquí...">%s</textarea></div>
      <div class="form-field"><label>URL Foto principal</label><input type="url" name="image_url" value="%s" class="form-input" placeholder="https://... enlace a la imagen"/></div>
      <div class="form-row">
        <div class="form-field"><label>Fecha</label><input type="datetime-local" name="date" value="%s" class="form-input"/></div>
        <div class="form-field"><label>Estado</label>
          <select name="status" class="form-input">
            <option value="borrador"%s>Borrador</option>
            <option value="publicado"%s>Publicado</option>
            <option value="archivado"%s>Archivado</option>
          </select>
        </div>
      </div>
      <div class="modal-actions">
        <button type="button" onclick="document.getElementById('modal-container').innerHTML=''" class="topbar-btn topbar-btn-outline">Cancelar</button>
        <button type="submit" class="topbar-btn topbar-btn-primary">Guardar</button>
      </div>
    </form>
  </div>
</div>`,
		map[bool]string{true: "Editar Noticia", false: "Nueva Noticia"}[id != ""],
		method,
		template.HTMLEscapeString(title),
		template.HTMLEscapeString(description),
		template.HTMLEscapeString(body),
		template.HTMLEscapeString(imageUrl),
		date,
		sel(status, "borrador"),
		sel(status, "publicado"),
		sel(status, "archivado"),
	)
}

// ── NEWS (delegates to content_blocks with category=NOTICIA) ──

func NewsList(cfg *config.Config, pb *pocketbase.PocketBase) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if c.Query("fragment") != "rows" {
			return c.SendFile("./internal/templates/admin/pages/news.html")
		}
		records, err := pb.FindRecordsByFilter("content_blocks",
			"category = 'NOTICIA'", "-date", 50, 0)

		var sb strings.Builder
		if err != nil || len(records) == 0 {
			sb.WriteString(`<tr><td colspan="5" style="text-align:center;padding:32px;color:var(--md-outline)">Sin noticias</td></tr>`)
		} else {
			for _, r := range records {
				status := r.GetString("status")
				badgeClass := "badge-warning"
				if status == "publicado" {
					badgeClass = "badge-success"
				}
				feat := ""
				if r.GetBool("featured") {
					feat = `<span class="badge badge-info">⭐</span>`
				}
				dateStr := "—"
				if dt := r.GetDateTime("date"); !dt.IsZero() {
					dateStr = dt.Time().Format("2 Jan 2006")
				}
				sb.WriteString(fmt.Sprintf(`<tr>
          <td>%s</td><td>%s</td>
          <td><span class="badge %s">%s</span></td><td>%s</td>
          <td>
            <button class="topbar-btn topbar-btn-outline" style="padding:4px 10px;font-size:12px"
              hx-get="/admin/events/%s/edit" hx-target="#modal-container" hx-swap="innerHTML">Editar</button>
            <button class="topbar-btn" style="padding:4px 10px;font-size:12px;background:#FDECEA;color:#B71C1C;border:none;cursor:pointer"
              hx-delete="/admin/events/%s" hx-confirm="¿Eliminar?" hx-target="closest tr" hx-swap="outerHTML swap:300ms">Eliminar</button>
          </td></tr>`,
					template.HTMLEscapeString(r.GetString("title")),
					dateStr, badgeClass,
					template.HTMLEscapeString(status),
					feat, r.Id, r.Id,
				))
			}
		}
		c.Set("Content-Type", "text/html; charset=utf-8")
		return c.SendString(sb.String())
	}
}

func NewsForm(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		html := newsFormHTML("", "", "", "borrador", "", "", "")
		c.Set("Content-Type", "text/html; charset=utf-8")
		return c.SendString(html)
	}
}

func NewsCreate(cfg *config.Config, pb *pocketbase.PocketBase) fiber.Handler {
	return EventCreate(cfg, pb)
}
func NewsEdit(cfg *config.Config, pb *pocketbase.PocketBase) fiber.Handler {
	return EventEdit(cfg, pb)
}
func NewsUpdate(cfg *config.Config, pb *pocketbase.PocketBase) fiber.Handler {
	return EventUpdate(cfg, pb)
}
func NewsDelete(cfg *config.Config, pb *pocketbase.PocketBase) fiber.Handler {
	return EventDelete(cfg, pb)
}

// ── PLAYLISTS ──

func PlaylistList(cfg *config.Config, pb *pocketbase.PocketBase) fiber.Handler {
	return func(c *fiber.Ctx) error {
		switch c.Query("fragment") {
		case "list":
			records, _ := pb.FindRecordsByFilter("playlists", "", "name", 100, 0)
			var sb strings.Builder
			if len(records) == 0 {
				sb.WriteString(`<p class="pl-empty-small">Sin playlists.<br/>Crea una con el botón Nueva.</p>`)
			} else {
				for _, r := range records {
					items, _ := pb.FindRecordsByFilter("playlist_items", "playlist_id='"+r.Id+"'", "", 100, 0)
					status := r.GetString("status")
					sb.WriteString(fmt.Sprintf(
						`<div class="pl-list-item" data-plid="%s"`+
							` hx-get="/admin/playlists/%s/edit" hx-target="#pl-center" hx-swap="innerHTML"`+
							` onclick="window._plSetActive('%s')">`+
							`<span class="pl-list-icon material-symbols-outlined">playlist_play</span>`+
							`<div class="pl-list-body"><div class="pl-list-name">%s</div><div class="pl-list-meta">%d items · %s</div></div>`+
							`</div>`,
						r.Id, r.Id, r.Id,
						template.HTMLEscapeString(r.GetString("name")),
						len(items),
						template.HTMLEscapeString(status),
					))
				}
			}
			c.Set("Content-Type", "text/html; charset=utf-8")
			return c.SendString(sb.String())

		case "content":
			return buildContentPool(c, pb)

		default:
			if c.Get("HX-Request") != "true" {
				return c.SendFile("./internal/templates/admin/pages/dashboard.html")
			}
			return c.SendFile("./internal/templates/admin/pages/playlists.html")
		}
	}
}

func PlaylistForm(cfg *config.Config, pb *pocketbase.PocketBase) fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Set("Content-Type", "text/html; charset=utf-8")
		return c.SendString(playlistEditorHTML("", "", "[]"))
	}
}

func PlaylistCreate(cfg *config.Config, pb *pocketbase.PocketBase) fiber.Handler {
	return func(c *fiber.Ctx) error {
		name := strings.TrimSpace(c.FormValue("name"))
		if name == "" {
			return c.SendString(`<div class="toast toast-error">El nombre es requerido</div>`)
		}
		var items []plItemInput
		if ij := c.FormValue("items_json"); ij != "" {
			if err := json.Unmarshal([]byte(ij), &items); err != nil {
				log.Printf("⚠️  PlaylistCreate items_json parse: %v", err)
			}
		}
		plCol, err := pb.FindCollectionByNameOrId("playlists")
		if err != nil {
			return c.Status(500).SendString(`<div class="toast toast-error">Error de BD</div>`)
		}
		pl := core.NewRecord(plCol)
		pl.Set("name", name)
		pl.Set("status", "activa")
		if err := pb.Save(pl); err != nil {
			return c.SendString(`<div class="toast toast-error">Error guardando playlist</div>`)
		}
		savePlItems(pb, pl.Id, items)
		c.Set("HX-Trigger", "playlistCreated")
		return c.SendString(`<div class="toast toast-success">✅ Playlist creada
<script>if(window._plReset)window._plReset();htmx.trigger(document.body,'playlistCreated');setTimeout(function(){var t=document.getElementById('toast-area');if(t)t.innerHTML=''},2000)</script></div>`)
	}
}

func PlaylistEdit(cfg *config.Config, pb *pocketbase.PocketBase) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id := c.Params("id")
		r, err := pb.FindRecordById("playlists", id)
		if err != nil {
			return c.Status(404).SendString(`<p style="padding:24px;color:var(--md-outline)">Playlist no encontrada</p>`)
		}
		items, _ := pb.FindRecordsByFilter("playlist_items", "playlist_id='"+id+"'", "orden", 100, 0)
		type existItem struct {
			Tipo     string `json:"tipo"`
			RefID    string `json:"ref_id"`
			Orden    int    `json:"orden"`
			Duracion int    `json:"duracion"`
			Name     string `json:"name"`
			Thumb    string `json:"thumb"`
		}
		var existing []existItem
		for _, it := range items {
			d := existItem{
				Tipo:     it.GetString("tipo"),
				Orden:    it.GetInt("orden"),
				Duracion: it.GetInt("duracion_segundos"),
			}
			switch d.Tipo {
			case "content_block":
				d.RefID = it.GetString("content_block_id")
				if cb, e := pb.FindRecordById("content_blocks", d.RefID); e == nil {
					d.Name = cb.GetString("title")
				}
			default:
				d.RefID = it.GetString("multimedia_id")
				if mm, e := pb.FindRecordById("multimedia", d.RefID); e == nil {
					d.Name = mm.GetString("filename")
					d.Thumb = mm.GetString("url_r2")
				}
			}
			existing = append(existing, d)
		}
		existJSON, _ := json.Marshal(existing)
		c.Set("Content-Type", "text/html; charset=utf-8")
		return c.SendString(playlistEditorHTML(r.Id, r.GetString("name"), string(existJSON)))
	}
}

func PlaylistUpdate(cfg *config.Config, pb *pocketbase.PocketBase) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id := c.Params("id")
		r, err := pb.FindRecordById("playlists", id)
		if err != nil {
			return c.Status(404).SendString(`<div class="toast toast-error">No encontrada</div>`)
		}
		if name := strings.TrimSpace(c.FormValue("name")); name != "" {
			r.Set("name", name)
		}
		if err := pb.Save(r); err != nil {
			return c.SendString(`<div class="toast toast-error">Error actualizando</div>`)
		}
		// Replace items
		old, _ := pb.FindRecordsByFilter("playlist_items", "playlist_id='"+id+"'", "", 100, 0)
		for _, it := range old {
			pb.Delete(it)
		}
		var items []plItemInput
		if ij := c.FormValue("items_json"); ij != "" {
			json.Unmarshal([]byte(ij), &items)
		}
		savePlItems(pb, id, items)
		c.Set("HX-Trigger", "playlistUpdated")
		return c.SendString(`<div class="toast toast-success">✅ Playlist actualizada
<script>if(window._plReset)window._plReset();htmx.trigger(document.body,'playlistUpdated');setTimeout(function(){var t=document.getElementById('toast-area');if(t)t.innerHTML=''},2000)</script></div>`)
	}
}

func PlaylistDelete(cfg *config.Config, pb *pocketbase.PocketBase) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id := c.Params("id")
		r, err := pb.FindRecordById("playlists", id)
		if err != nil {
			return c.Status(404).SendString("")
		}
		items, _ := pb.FindRecordsByFilter("playlist_items", "playlist_id='"+id+"'", "", 100, 0)
		for _, it := range items {
			pb.Delete(it)
		}
		if err := pb.Delete(r); err != nil {
			return c.SendString(`<div class="toast toast-error">Error eliminando</div>`)
		}
		c.Set("HX-Trigger", "playlistDeleted")
		return c.SendString(`<div class="pl-placeholder"><span class="material-symbols-outlined">playlist_add</span><p>Playlist eliminada.<br/>Selecciona otra o crea una nueva.</p></div>`)
	}
}

func PlaylistReorder(cfg *config.Config, pb *pocketbase.PocketBase) fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.SendString(`<div class="toast toast-success">Orden guardado</div>`)
	}
}

// ── DEVICES ──

func DeviceList(cfg *config.Config, pb *pocketbase.PocketBase) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if c.Query("fragment") != "table" {
			if c.Get("HX-Request") != "true" {
				return c.SendFile("./internal/templates/admin/pages/dashboard.html")
			}
			return c.SendFile("./internal/templates/admin/pages/devices.html")
		}
		// Build playlist lookup map
		plRecs, _ := pb.FindRecordsByFilter("playlists", "", "name", 100, 0)
		plMap := make(map[string]string)
		for _, pl := range plRecs {
			plMap[pl.Id] = pl.GetString("name")
		}
		records, err := pb.FindRecordsByFilter("devices", "", "name", 100, 0)
		var sb strings.Builder
		sb.WriteString(`<table><thead><tr>`)
		sb.WriteString(`<th>Nombre</th><th>Tipo</th><th>Código</th><th>Ubicación</th><th>Playlist</th><th>Estado</th><th>Vista actual</th><th>Acciones</th>`)
		sb.WriteString(`</tr></thead><tbody>`)
		if err != nil || len(records) == 0 {
			sb.WriteString(`<tr><td colspan="8" style="text-align:center;padding:32px;color:var(--md-outline)">Sin dispositivos registrados</td></tr>`)
		} else {
			for _, r := range records {
				dtype := r.GetString("type")
				code := r.GetString("code")
				plName := plMap[r.GetString("playlist_id")]
				if plName == "" {
					plName = "—"
				}
				status := r.GetString("status")
				badgeClass := "badge-warning"
				if status == "activo" {
					badgeClass = "badge-success"
				}
				typeLabel := map[string]string{
					"web_hero":   "Web Hero",
					"vertical":   "Totem",
					"horizontal": "Pantalla",
				}[dtype]
				if typeLabel == "" {
					typeLabel = dtype
				}
				displayLink := ""
				if code != "" && dtype != "web_hero" {
					displayLink = fmt.Sprintf(`<a href="/display/%s" target="_blank" class="topbar-btn topbar-btn-outline" style="padding:3px 8px;font-size:11px;text-decoration:none">Ver pantalla</a>`,
						template.HTMLEscapeString(code))
				}
				currentView := r.GetString("current_view")
				if currentView == "" {
					currentView = "—"
				}
				sb.WriteString(fmt.Sprintf(`<tr>
					<td style="font-weight:500">%s</td>
					<td><span class="badge badge-info" style="font-size:11px">%s</span></td>
					<td><code style="background:var(--md-surface-container-high);padding:2px 6px;border-radius:4px;font-size:12px">%s</code></td>
					<td style="color:var(--md-on-surface-variant);font-size:13px">%s</td>
					<td style="font-size:13px">%s</td>
					<td><span class="badge %s">%s</span></td>
					<td style="font-size:12px;color:var(--md-outline)">%s</td>
					<td><div style="display:flex;gap:6px;align-items:center;flex-wrap:wrap">
						%s
						<button class="topbar-btn topbar-btn-outline" style="padding:3px 8px;font-size:12px"
							hx-get="/admin/devices/%s/edit" hx-target="#modal-container" hx-swap="innerHTML">Editar</button>
						<button style="padding:3px 8px;font-size:12px;background:#FDECEA;color:#B71C1C;border:none;cursor:pointer;border-radius:9999px;font-family:inherit"
							hx-delete="/admin/devices/%s" hx-confirm="¿Eliminar dispositivo?" hx-target="closest tr" hx-swap="outerHTML swap:300ms">Eliminar</button>
					</div></td>
				</tr>`,
					template.HTMLEscapeString(r.GetString("name")),
					template.HTMLEscapeString(typeLabel),
					template.HTMLEscapeString(code),
					template.HTMLEscapeString(r.GetString("ubicacion")),
					template.HTMLEscapeString(plName),
					badgeClass, template.HTMLEscapeString(status),
					template.HTMLEscapeString(currentView),
					displayLink,
					r.Id, r.Id,
				))
			}
		}
		sb.WriteString(`</tbody></table>`)
		c.Set("Content-Type", "text/html; charset=utf-8")
		return c.SendString(sb.String())
	}
}

func DeviceForm(cfg *config.Config, pb *pocketbase.PocketBase) fiber.Handler {
	return func(c *fiber.Ctx) error {
		opts := playlistOptsHTML(pb, "")
		html := fmt.Sprintf(`<div class="modal-overlay" onclick="this.remove()">
  <div class="modal-card" onclick="event.stopPropagation()">
    <div class="modal-header"><h3>Registrar Dispositivo</h3>
      <button onclick="document.getElementById('modal-container').innerHTML=''" style="background:none;border:none;cursor:pointer;font-size:20px;color:var(--md-outline)">✕</button>
    </div>
    <form hx-post="/admin/devices" hx-target="#toast-area" hx-swap="innerHTML">
      <div class="form-field"><label>Nombre</label><input type="text" name="name" required class="form-input" placeholder="ej: Totem Entrada Principal"/></div>
      <div class="form-row">
        <div class="form-field"><label>Tipo</label>
          <select name="type" class="form-input">
            <option value="horizontal">Pantalla Horizontal</option>
            <option value="vertical">Totem Vertical</option>
            <option value="web_hero">Web Hero</option>
          </select>
        </div>
        <div class="form-field"><label>Código único</label><input type="text" name="code" class="form-input" placeholder="ej: T003"/></div>
      </div>
      <div class="form-field"><label>Ubicación</label><input type="text" name="ubicacion" class="form-input" placeholder="ej: Gimnasio"/></div>
      <div class="form-field"><label>Playlist asignada</label>
        <select name="playlist_id" class="form-input">
          <option value="">— Sin playlist —</option>%s
        </select>
      </div>
      <div class="modal-actions">
        <button type="button" onclick="document.getElementById('modal-container').innerHTML=''" class="topbar-btn topbar-btn-outline">Cancelar</button>
        <button type="submit" class="topbar-btn topbar-btn-primary">Registrar</button>
      </div>
    </form>
  </div>
</div>`, opts)
		c.Set("Content-Type", "text/html; charset=utf-8")
		return c.SendString(html)
	}
}

func DeviceEdit(cfg *config.Config, pb *pocketbase.PocketBase) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id := c.Params("id")
		r, err := pb.FindRecordById("devices", id)
		if err != nil {
			return c.Status(404).SendString(`<div class="toast toast-error">Dispositivo no encontrado</div>`)
		}
		dtype := r.GetString("type")
		opts := playlistOptsHTML(pb, r.GetString("playlist_id"))
		html := fmt.Sprintf(`<div class="modal-overlay" onclick="this.remove()">
  <div class="modal-card" onclick="event.stopPropagation()">
    <div class="modal-header"><h3>Editar Dispositivo</h3>
      <button onclick="document.getElementById('modal-container').innerHTML=''" style="background:none;border:none;cursor:pointer;font-size:20px;color:var(--md-outline)">✕</button>
    </div>
    <form hx-put="/admin/devices/%s" hx-target="#toast-area" hx-swap="innerHTML">
      <div class="form-field"><label>Nombre</label><input type="text" name="name" value="%s" required class="form-input"/></div>
      <div class="form-row">
        <div class="form-field"><label>Tipo</label>
          <select name="type" class="form-input">
            <option value="horizontal"%s>Pantalla Horizontal</option>
            <option value="vertical"%s>Totem Vertical</option>
            <option value="web_hero"%s>Web Hero</option>
          </select>
        </div>
        <div class="form-field"><label>Código único</label><input type="text" name="code" value="%s" class="form-input"/></div>
      </div>
      <div class="form-field"><label>Ubicación</label><input type="text" name="ubicacion" value="%s" class="form-input"/></div>
      <div class="form-field"><label>Playlist asignada</label>
        <select name="playlist_id" class="form-input">
          <option value="">— Sin playlist —</option>%s
        </select>
      </div>
      <div class="form-field"><label>Estado</label>
        <select name="status" class="form-input">
          <option value="activo"%s>Activo</option>
          <option value="inactivo"%s>Inactivo</option>
        </select>
      </div>
      <div class="modal-actions">
        <button type="button" onclick="document.getElementById('modal-container').innerHTML=''" class="topbar-btn topbar-btn-outline">Cancelar</button>
        <button type="submit" class="topbar-btn topbar-btn-primary">Guardar cambios</button>
      </div>
    </form>
  </div>
</div>`,
			r.Id,
			template.HTMLEscapeString(r.GetString("name")),
			sel(dtype, "horizontal"), sel(dtype, "vertical"), sel(dtype, "web_hero"),
			template.HTMLEscapeString(r.GetString("code")),
			template.HTMLEscapeString(r.GetString("ubicacion")),
			opts,
			sel(r.GetString("status"), "activo"),
			sel(r.GetString("status"), "inactivo"),
		)
		c.Set("Content-Type", "text/html; charset=utf-8")
		return c.SendString(html)
	}
}

func DeviceCreate(cfg *config.Config, pb *pocketbase.PocketBase) fiber.Handler {
	return func(c *fiber.Ctx) error {
		col, err := pb.FindCollectionByNameOrId("devices")
		if err != nil {
			return c.Status(500).SendString(`<div class="toast toast-error">Error de BD</div>`)
		}
		r := core.NewRecord(col)
		r.Set("name", c.FormValue("name"))
		r.Set("type", c.FormValue("type"))
		r.Set("code", c.FormValue("code"))
		r.Set("ubicacion", c.FormValue("ubicacion"))
		r.Set("playlist_id", c.FormValue("playlist_id"))
		r.Set("status", "activo")
		if err := pb.Save(r); err != nil {
			return c.SendString(`<div class="toast toast-error">Error guardando</div>`)
		}
		c.Set("HX-Trigger", "deviceCreated")
		return c.SendString(`<div class="toast toast-success" id="toast-area">✅ Dispositivo registrado
<script>document.getElementById('modal-container').innerHTML='';htmx.ajax('GET','/admin/devices?fragment=table',{target:'#devices-table-wrap',swap:'innerHTML'});setTimeout(function(){var t=document.getElementById('toast-area');if(t)t.innerHTML=''},2000)</script></div>`)
	}
}

func DeviceUpdate(cfg *config.Config, pb *pocketbase.PocketBase) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id := c.Params("id")
		r, err := pb.FindRecordById("devices", id)
		if err != nil {
			return c.Status(404).SendString(`<div class="toast toast-error">No encontrado</div>`)
		}
		r.Set("name", c.FormValue("name"))
		r.Set("type", c.FormValue("type"))
		r.Set("code", c.FormValue("code"))
		r.Set("ubicacion", c.FormValue("ubicacion"))
		r.Set("playlist_id", c.FormValue("playlist_id"))
		r.Set("status", c.FormValue("status"))
		if err := pb.Save(r); err != nil {
			return c.SendString(`<div class="toast toast-error">Error actualizando</div>`)
		}
		c.Set("HX-Trigger", "deviceUpdated")
		return c.SendString(`<div class="toast toast-success" id="toast-area">✅ Dispositivo actualizado
<script>document.getElementById('modal-container').innerHTML='';htmx.ajax('GET','/admin/devices?fragment=table',{target:'#devices-table-wrap',swap:'innerHTML'});setTimeout(function(){var t=document.getElementById('toast-area');if(t)t.innerHTML=''},2000)</script></div>`)
	}
}

func DeviceDelete(cfg *config.Config, pb *pocketbase.PocketBase) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id := c.Params("id")
		r, err := pb.FindRecordById("devices", id)
		if err != nil {
			return c.Status(404).SendString("")
		}
		if err := pb.Delete(r); err != nil {
			return c.SendString(`<div class="toast toast-error">Error eliminando</div>`)
		}
		c.Set("HX-Trigger", "deviceDeleted")
		return c.SendString("")
	}
}

func DeviceAssignPlaylist(cfg *config.Config, pb *pocketbase.PocketBase) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id := c.Params("id")
		r, err := pb.FindRecordById("devices", id)
		if err != nil {
			return c.Status(404).SendString(`<div class="toast toast-error">Dispositivo no encontrado</div>`)
		}
		r.Set("playlist_id", c.FormValue("playlist_id"))
		if err := pb.Save(r); err != nil {
			return c.SendString(`<div class="toast toast-error">Error asignando playlist</div>`)
		}
		c.Set("HX-Trigger", "deviceUpdated")
		return c.SendString(`<div class="toast toast-success">✅ Playlist asignada</div>`)
	}
}

// ── USERS ──

func UserList(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if c.Query("fragment") != "table" {
			return c.SendFile("./internal/templates/admin/pages/users.html")
		}
		row := fmt.Sprintf(`<tr>
          <td>%s</td>
          <td>%s</td>
          <td><span class="badge badge-info">superadmin</span></td>
          <td>—</td>
          <td><span class="badge badge-success">Activo</span></td>
          <td><span style="font-size:12px;color:var(--md-outline)">Config del sistema</span></td>
        </tr>`,
			template.HTMLEscapeString("Administrador"),
			template.HTMLEscapeString(cfg.AdminEmail),
		)
		c.Set("Content-Type", "text/html; charset=utf-8")
		return c.SendString(row)
	}
}
func UserCreate(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.SendString(`<div class="toast toast-success">Usuario creado</div>`)
	}
}
func UserUpdate(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.SendString(`<div class="toast toast-success">Usuario actualizado</div>`)
	}
}
func UserDelete(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.SendString(`<div class="toast toast-success">Usuario eliminado</div>`)
	}
}

// ── WHATSAPP LOGS ──

func WhatsAppLogs(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.SendFile("./internal/templates/admin/pages/whatsapp-logs.html")
	}
}

// ── PROPIEDADES CRUD ──

func PropiedadesList(cfg *config.Config, pb *pocketbase.PocketBase) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if c.Query("fragment") != "rows" {
			return c.SendFile("./internal/templates/admin/pages/propiedades.html")
		}
		records, err := pb.FindRecordsByFilter("propiedades", "id != ''", "-publicado_en", 200, 0)
		var sb strings.Builder
		if err != nil {
			log.Printf("⚠️  PropiedadesList query error: %v", err)
			sb.WriteString(fmt.Sprintf(`<tr><td colspan="7" style="text-align:center;padding:32px;color:#B71C1C">Error cargando propiedades: %s</td></tr>`, template.HTMLEscapeString(err.Error())))
		} else if len(records) == 0 {
			sb.WriteString(`<tr class="empty-row"><td colspan="7" style="text-align:center;padding:32px;color:var(--md-outline)">Sin propiedades — agrega una con el botón de arriba</td></tr>`)
		} else {
			for _, r := range records {
				status := r.GetString("status")
				badgeClass := "badge-warning"
				switch status {
				case "publicado":
					badgeClass = "badge-success"
				case "vendida", "arrendada":
					badgeClass = "badge-danger"
				case "reservada":
					badgeClass = "badge-info"
				}
				op := r.GetString("operacion")
				opLabel := op
				if op == "VENTA" {
					opLabel = "Venta"
				} else if op == "ARRIENDO" {
					opLabel = "Arriendo"
				}
				precio := ""
				if uf := r.GetFloat("precio_uf"); uf > 0 {
					if uf == float64(int64(uf)) {
						precio = fmt.Sprintf("UF %d", int64(uf))
					} else {
						precio = fmt.Sprintf("UF %.1f", uf)
					}
				} else if clp := r.GetFloat("precio_clp"); clp > 0 {
					precio = fmt.Sprintf("$ %d", int64(clp))
				}
				dest := ""
				if r.GetBool("destacada") {
					dest = `<span class="badge badge-info" style="font-size:10px">⭐</span>`
				}
				sb.WriteString(fmt.Sprintf(`<tr>
          <td>%s %s</td>
          <td>%s</td>
          <td>%s</td>
          <td>%s</td>
          <td>%s</td>
          <td><span class="badge %s">%s</span></td>
          <td style="white-space:nowrap">
            <button class="topbar-btn topbar-btn-outline" style="padding:4px 10px;font-size:12px"
              hx-get="/admin/propiedades/%s/edit" hx-target="#modal-container" hx-swap="innerHTML">Editar</button>
            <button class="topbar-btn" style="padding:4px 10px;font-size:12px;background:#E8F5E9;color:#1B5E20;border:none;cursor:pointer"
              hx-post="/admin/propiedades/%s/publish" hx-target="#toast-area" hx-swap="innerHTML">%s</button>
            <button class="topbar-btn" style="padding:4px 10px;font-size:12px;background:#FDECEA;color:#B71C1C;border:none;cursor:pointer"
              hx-delete="/admin/propiedades/%s" hx-confirm="¿Eliminar esta propiedad?" hx-target="closest tr" hx-swap="outerHTML swap:300ms">Eliminar</button>
          </td></tr>`,
					template.HTMLEscapeString(r.GetString("titulo")), dest,
					template.HTMLEscapeString(r.GetString("tipo")),
					template.HTMLEscapeString(opLabel),
					template.HTMLEscapeString(r.GetString("comuna")),
					template.HTMLEscapeString(precio),
					badgeClass, template.HTMLEscapeString(status),
					r.Id, r.Id,
					map[bool]string{true: "Despublicar", false: "Publicar"}[status == "publicado"],
					r.Id,
				))
			}
		}
		c.Set("Content-Type", "text/html; charset=utf-8")
		return c.SendString(sb.String())
	}
}

func PropiedadForm(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		html := propiedadFormHTML("", "", "", "VENTA", "CASA", "", "", "", "", "", 0, 0, 0, 0, 0, 0, 0, "usada", "publicado", false, false, "", "", "", 0, 0)
		c.Set("Content-Type", "text/html; charset=utf-8")
		return c.SendString(html)
	}
}

func PropiedadCreate(cfg *config.Config, pb *pocketbase.PocketBase) fiber.Handler {
	return func(c *fiber.Ctx) error {
		titulo := strings.TrimSpace(c.FormValue("titulo"))
		if titulo == "" {
			return c.SendString(`<div class="toast toast-error">El título es requerido</div>`)
		}
		col, err := pb.FindCollectionByNameOrId("propiedades")
		if err != nil {
			return c.Status(500).SendString(`<div class="toast toast-error">Error de BD</div>`)
		}
		r := core.NewRecord(col)
		setPropiedadFields(r, c)
		// Always stamp publicado_en for new listings so they sort correctly
		r.Set("publicado_en", time.Now().UTC())
		if err := pb.Save(r); err != nil {
			log.Printf("⚠️  PropiedadCreate save error: %v", err)
			return c.SendString(`<div class="toast toast-error">Error guardando: ` + template.HTMLEscapeString(err.Error()) + `</div>`)
		}
		return c.SendString(`<div class="toast toast-success" id="toast-area">✅ Propiedad creada
<script>
  document.getElementById('modal-container').innerHTML='';
  htmx.ajax('GET','/admin/propiedades?fragment=rows',{target:'#prop-tbody',swap:'innerHTML'});
  var stats=document.querySelector('.stats-row');
  if(stats)htmx.ajax('GET','/admin/dashboard/stats',{target:'.stats-row',swap:'innerHTML'});
  setTimeout(function(){var t=document.getElementById('toast-area');if(t)t.innerHTML=''},2000);
</script></div>`)
	}
}

func PropiedadEdit(cfg *config.Config, pb *pocketbase.PocketBase) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id := c.Params("id")
		r, err := pb.FindRecordById("propiedades", id)
		if err != nil {
			return c.Status(404).SendString(`<div class="toast toast-error">No encontrado</div>`)
		}
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
			r.GetFloat("superficie_total"),
			r.GetString("estado_propiedad"),
			r.GetString("status"),
			r.GetBool("destacada"),
			r.GetBool("oportunidad"),
			r.GetString("amenidades"),
			r.GetString("contacto_whatsapp"),
			r.GetString("gallery"),
			r.GetFloat("lat"),
			r.GetFloat("lng"),
		)
		c.Set("Content-Type", "text/html; charset=utf-8")
		return c.SendString(html)
	}
}

func PropiedadUpdate(cfg *config.Config, pb *pocketbase.PocketBase) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id := c.Params("id")
		r, err := pb.FindRecordById("propiedades", id)
		if err != nil {
			return c.SendString(`<div class="toast toast-error">No encontrado</div>`)
		}
		setPropiedadFields(r, c)
		if err := pb.Save(r); err != nil {
			return c.SendString(`<div class="toast toast-error">Error actualizando</div>`)
		}
		return c.SendString(`<div class="toast toast-success" id="toast-area">✅ Propiedad actualizada
<script>
  document.getElementById('modal-container').innerHTML='';
  htmx.ajax('GET','/admin/propiedades?fragment=rows',{target:'#prop-tbody',swap:'innerHTML'});
  setTimeout(function(){var t=document.getElementById('toast-area');if(t)t.innerHTML=''},2000);
</script></div>`)
	}
}

func PropiedadDelete(cfg *config.Config, pb *pocketbase.PocketBase) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id := c.Params("id")
		r, err := pb.FindRecordById("propiedades", id)
		if err != nil {
			return c.Status(404).SendString("")
		}
		if err := pb.Delete(r); err != nil {
			return c.SendString(`<div class="toast toast-error">Error eliminando</div>`)
		}
		return c.SendString("")
	}
}

func PropiedadToggleStatus(cfg *config.Config, pb *pocketbase.PocketBase) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id := c.Params("id")
		r, err := pb.FindRecordById("propiedades", id)
		if err != nil {
			return c.Status(404).SendString("")
		}
		newStatus := "publicado"
		if r.GetString("status") == "publicado" {
			newStatus = "borrador"
		}
		r.Set("status", newStatus)
		if err := pb.Save(r); err != nil {
			return c.SendString(`<div class="toast toast-error">Error actualizando estado</div>`)
		}
		label := map[string]string{"publicado": "✅ Publicada", "borrador": "📝 Volvió a borrador"}[newStatus]
		return c.SendString(fmt.Sprintf(`<div class="toast toast-success" id="toast-area">%s
<script>
  htmx.ajax('GET','/admin/propiedades?fragment=rows',{target:'#prop-tbody',swap:'innerHTML'});
  setTimeout(function(){var t=document.getElementById('toast-area');if(t)t.innerHTML=''},2000);
</script></div>`, label))
	}
}

// setPropiedadFields maps form values onto a propiedades record.
func setPropiedadFields(r *core.Record, c *fiber.Ctx) {
	r.Set("titulo", strings.TrimSpace(c.FormValue("titulo")))
	r.Set("slug", strings.TrimSpace(c.FormValue("slug")))
	r.Set("descripcion", c.FormValue("descripcion"))
	r.Set("operacion", c.FormValue("operacion"))
	r.Set("tipo", c.FormValue("tipo"))
	r.Set("direccion", c.FormValue("direccion"))
	r.Set("comuna", c.FormValue("comuna"))
	r.Set("region", c.FormValue("region"))
	r.Set("cover_image", c.FormValue("cover_image"))
	r.Set("gallery", c.FormValue("gallery"))
	r.Set("amenidades", c.FormValue("amenidades"))
	r.Set("contacto_whatsapp", c.FormValue("contacto_whatsapp"))
	r.Set("estado_propiedad", c.FormValue("estado_propiedad"))
	r.Set("status", c.FormValue("status"))
	r.Set("destacada", c.FormValue("destacada") == "on")
	r.Set("oportunidad", c.FormValue("oportunidad") == "on")
	if v := c.FormValue("precio_uf"); v != "" {
		var f float64
		fmt.Sscanf(v, "%f", &f)
		r.Set("precio_uf", f)
	}
	if v := c.FormValue("precio_clp"); v != "" {
		var f float64
		fmt.Sscanf(v, "%f", &f)
		r.Set("precio_clp", f)
	}
	if v := c.FormValue("dormitorios"); v != "" {
		var n int
		fmt.Sscanf(v, "%d", &n)
		r.Set("dormitorios", n)
	}
	if v := c.FormValue("banos"); v != "" {
		var n int
		fmt.Sscanf(v, "%d", &n)
		r.Set("banos", n)
	}
	if v := c.FormValue("estacionamientos"); v != "" {
		var n int
		fmt.Sscanf(v, "%d", &n)
		r.Set("estacionamientos", n)
	}
	if v := c.FormValue("superficie_util"); v != "" {
		var f float64
		fmt.Sscanf(v, "%f", &f)
		r.Set("superficie_util", f)
	}
	if v := c.FormValue("superficie_total"); v != "" {
		var f float64
		fmt.Sscanf(v, "%f", &f)
		r.Set("superficie_total", f)
	}
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
}

// propiedadFormHTML builds the create/edit modal for propiedades.
func fmtFloatOrEmpty(f float64) string {
	if f == 0 {
		return ""
	}
	return fmt.Sprintf("%.6f", f)
}

func propiedadFormHTML(id, titulo, slug, operacion, tipo, descripcion, direccion, comuna, region, coverImage string,
	precioUF, precioCLP, dormitorios, banos, estacionamientos, superficieUtil, superficieTotal float64,
	estadoPropiedad, status string, destacada, oportunidad bool, amenidades, whatsapp, gallery string,
	lat, lng float64,
) string {
	method := `hx-post="/admin/propiedades"`
	if id != "" {
		method = fmt.Sprintf(`hx-put="/admin/propiedades/%s"`, id)
	}
	title := map[bool]string{true: "Editar Propiedad", false: "Nueva Propiedad"}[id != ""]

	fmtNum := func(f float64) string {
		if f == 0 {
			return ""
		}
		if f == float64(int64(f)) {
			return fmt.Sprintf("%d", int64(f))
		}
		return fmt.Sprintf("%g", f)
	}

	chk := func(b bool) string {
		if b {
			return " checked"
		}
		return ""
	}

	ops := []struct{ v, l string }{{"VENTA", "Venta"}, {"ARRIENDO", "Arriendo"}}
	var opOpts strings.Builder
	for _, o := range ops {
		s := ""
		if o.v == operacion {
			s = " selected"
		}
		opOpts.WriteString(fmt.Sprintf(`<option value="%s"%s>%s</option>`, o.v, s, o.l))
	}

	tipos := []string{"CASA", "DEPARTAMENTO", "TERRENO", "PARCELA", "LOCAL", "OFICINA", "BODEGA"}
	var tipoOpts strings.Builder
	for _, t := range tipos {
		s := ""
		if t == tipo {
			s = " selected"
		}
		tipoOpts.WriteString(fmt.Sprintf(`<option value="%s"%s>%s</option>`, t, s, strings.Title(strings.ToLower(t))))
	}

	estados := []struct{ v, l string }{{"nueva", "Nueva"}, {"usada", "Usada"}, {"a_estrenar", "A estrenar"}}
	var estOpts strings.Builder
	for _, e := range estados {
		s := ""
		if e.v == estadoPropiedad {
			s = " selected"
		}
		estOpts.WriteString(fmt.Sprintf(`<option value="%s"%s>%s</option>`, e.v, s, e.l))
	}

	statuses := []struct{ v, l string }{{"borrador", "Borrador"}, {"publicado", "Publicado"}, {"reservada", "Reservada"}, {"vendida", "Vendida"}, {"arrendada", "Arrendada"}}
	var statusOpts strings.Builder
	for _, st := range statuses {
		s := ""
		if st.v == status {
			s = " selected"
		}
		statusOpts.WriteString(fmt.Sprintf(`<option value="%s"%s>%s</option>`, st.v, s, st.l))
	}

	return fmt.Sprintf(`<div class="modal-overlay" onclick="this.remove()">
  <div class="modal-card" onclick="event.stopPropagation()" style="max-width:680px">
    <div class="modal-header">
      <h3>%s</h3>
      <button onclick="document.getElementById('modal-container').innerHTML=''" style="background:none;border:none;cursor:pointer;font-size:20px;color:var(--md-outline)">✕</button>
    </div>
    <form %s hx-target="#toast-area" hx-swap="innerHTML">
      <div class="form-row">
        <div class="form-field" style="grid-column:span 2">
          <label>Título *</label>
          <input type="text" name="titulo" value="%s" required class="form-input" placeholder="Casa en venta en..."/>
        </div>
      </div>
      <div class="form-row">
        <div class="form-field"><label>Operación</label><select name="operacion" class="form-input">%s</select></div>
        <div class="form-field"><label>Tipo</label><select name="tipo" class="form-input">%s</select></div>
      </div>
      <div class="form-row">
        <div class="form-field"><label>Precio UF</label><input type="number" name="precio_uf" value="%s" class="form-input" placeholder="0" step="0.01" min="0"/></div>
        <div class="form-field"><label>Precio CLP</label><input type="number" name="precio_clp" value="%s" class="form-input" placeholder="0" step="1" min="0"/></div>
      </div>
      <div class="form-row">
        <div class="form-field"><label>Dormitorios</label><input type="number" name="dormitorios" value="%s" class="form-input" placeholder="0" min="0"/></div>
        <div class="form-field"><label>Baños</label><input type="number" name="banos" value="%s" class="form-input" placeholder="0" min="0"/></div>
      </div>
      <div class="form-row">
        <div class="form-field"><label>Superficie útil (m²)</label><input type="number" name="superficie_util" value="%s" class="form-input" placeholder="0" step="0.01" min="0"/></div>
        <div class="form-field"><label>Superficie total (m²)</label><input type="number" name="superficie_total" value="%s" class="form-input" placeholder="0" step="0.01" min="0"/></div>
      </div>
      <div class="form-row">
        <div class="form-field"><label>Estacionamientos</label><input type="number" name="estacionamientos" value="%s" class="form-input" placeholder="0" min="0"/></div>
        <div class="form-field"></div>
      </div>
      <div class="form-row">
        <div class="form-field"><label>Dirección</label><input type="text" name="direccion" value="%s" class="form-input" placeholder="Av. Principal 123"/></div>
        <div class="form-field"><label>Comuna</label><input type="text" name="comuna" value="%s" class="form-input" placeholder="Santiago"/></div>
      </div>
      <div class="form-row">
        <div class="form-field"><label>Región</label><input type="text" name="region" value="%s" class="form-input" placeholder="Región Metropolitana"/></div>
        <div class="form-field"><label>WhatsApp contacto</label><input type="text" name="contacto_whatsapp" value="%s" class="form-input" placeholder="+56912345678"/></div>
      </div>
      <div class="form-field"><label>URL imagen principal</label><input type="url" name="cover_image" value="%s" class="form-input" placeholder="https://..."/></div>
      <div class="form-field"><label>Galería (URLs separadas por coma)</label><input type="text" name="gallery" value="%s" class="form-input" placeholder="https://img1.jpg, https://img2.jpg"/></div>
      <div class="form-field"><label>Amenidades (separadas por coma)</label><input type="text" name="amenidades" value="%s" class="form-input" placeholder="piscina, quincho, jardín"/></div>
      <div class="form-field"><label>Descripción</label><textarea name="descripcion" class="form-input" rows="3" placeholder="Descripción de la propiedad...">%s</textarea></div>
      <div class="form-row">
        <div class="form-field"><label>Estado construcción</label><select name="estado_propiedad" class="form-input">%s</select></div>
        <div class="form-field"><label>Estado publicación</label><select name="status" class="form-input">%s</select></div>
      </div>
      <div style="display:flex;gap:24px;margin-bottom:16px">
        <label style="display:flex;align-items:center;gap:8px;font-size:14px;cursor:pointer">
          <input type="checkbox" name="destacada"%s style="width:16px;height:16px;accent-color:var(--md-primary)"/> Destacada
        </label>
        <label style="display:flex;align-items:center;gap:8px;font-size:14px;cursor:pointer">
          <input type="checkbox" name="oportunidad"%s style="width:16px;height:16px;accent-color:var(--md-primary)"/> Oportunidad
        </label>
      </div>
      <div class="form-row">
        <div class="form-field"><label>Latitud (zona, no dirección exacta)</label><input type="number" name="lat" step="0.000001" value="%s" class="form-input" placeholder="-33.4513"/></div>
        <div class="form-field"><label>Longitud (zona, no dirección exacta)</label><input type="number" name="lng" step="0.000001" value="%s" class="form-input" placeholder="-70.6653"/></div>
      </div>
      <div class="modal-actions">
        <button type="button" onclick="document.getElementById('modal-container').innerHTML=''" class="topbar-btn topbar-btn-outline">Cancelar</button>
        <button type="submit" class="topbar-btn topbar-btn-primary">Guardar</button>
      </div>
    </form>
  </div>
</div>`,
		title, method,
		template.HTMLEscapeString(titulo),
		opOpts.String(), tipoOpts.String(),
		fmtNum(precioUF), fmtNum(precioCLP),
		fmtNum(dormitorios), fmtNum(banos),
		fmtNum(superficieUtil), fmtNum(superficieTotal), fmtNum(estacionamientos),
		template.HTMLEscapeString(direccion),
		template.HTMLEscapeString(comuna),
		template.HTMLEscapeString(region),
		template.HTMLEscapeString(whatsapp),
		template.HTMLEscapeString(coverImage),
		template.HTMLEscapeString(gallery),
		template.HTMLEscapeString(amenidades),
		template.HTMLEscapeString(descripcion),
		estOpts.String(), statusOpts.String(),
		chk(destacada), chk(oportunidad),
		fmtFloatOrEmpty(lat), fmtFloatOrEmpty(lng),
	)
}

// sel returns " selected" if val == target
func sel(val, target string) string {
	if val == target {
		return " selected"
	}
	return ""
}

// ── PLAYLIST HELPERS ──────────────────────────────────────────────────────────

// plItemInput is the JSON shape sent from the playlist editor frontend.
type plItemInput struct {
	Tipo     string `json:"tipo"`
	RefID    string `json:"ref_id"`
	Orden    int    `json:"orden"`
	Duracion int    `json:"duracion"`
	Name     string `json:"name"`
}

// playlistOptsHTML returns <option> elements for every playlist in PocketBase.
func playlistOptsHTML(pb *pocketbase.PocketBase, selectedID string) string {
	records, _ := pb.FindRecordsByFilter("playlists", "", "name", 100, 0)
	var sb strings.Builder
	for _, r := range records {
		s := ""
		if r.Id == selectedID {
			s = " selected"
		}
		sb.WriteString(fmt.Sprintf(`<option value="%s"%s>%s</option>`,
			r.Id, s, template.HTMLEscapeString(r.GetString("name"))))
	}
	return sb.String()
}

// savePlItems creates playlist_item records in PocketBase for the given playlist.
func savePlItems(pb *pocketbase.PocketBase, playlistID string, items []plItemInput) {
	col, err := pb.FindCollectionByNameOrId("playlist_items")
	if err != nil {
		log.Printf("⚠️  savePlItems: %v", err)
		return
	}
	for i, it := range items {
		r := core.NewRecord(col)
		r.Set("playlist_id", playlistID)
		r.Set("tipo", it.Tipo)
		r.Set("orden", i+1)
		r.Set("duracion_segundos", it.Duracion)
		switch it.Tipo {
		case "content_block":
			r.Set("content_block_id", it.RefID)
		default:
			r.Set("multimedia_id", it.RefID)
		}
		if err := pb.Save(r); err != nil {
			log.Printf("⚠️  savePlItems[%d]: %v", i, err)
		}
	}
}

// buildContentPool returns a vertical list of content items for the library sidebar.
func buildContentPool(c *fiber.Ctx, pb *pocketbase.PocketBase) error {
	var sb strings.Builder
	sb.WriteString(`<div class="content-list">`)

	cbs, _ := pb.FindRecordsByFilter("content_blocks", "", "-date", 100, 0)
	for _, r := range cbs {
		cat := r.GetString("category")
		dtype := "eventos"
		if cat == "NOTICIA" {
			dtype = "noticias"
		}
		icon := "event"
		if r.GetBool("urgency") {
			icon = "campaign"
		}
		if dtype == "noticias" {
			icon = "newspaper"
		}
		sb.WriteString(fmt.Sprintf(
			`<div class="content-item" data-id="%s" data-tipo="content_block" data-type="%s" data-name="%s" data-thumb="" onclick="addCardToPlaylist(this)">`+
				`<div class="content-item-thumb"><span class="material-symbols-outlined">%s</span></div>`+
				`<div class="content-item-body"><div class="content-item-name">%s</div><div class="content-item-meta">%s</div></div>`+
				`</div>`,
			r.Id, dtype,
			template.HTMLEscapeString(r.GetString("title")),
			icon,
			template.HTMLEscapeString(r.GetString("title")),
			template.HTMLEscapeString(cat),
		))
	}

	mms, _ := pb.FindRecordsByFilter("multimedia", "", "filename", 100, 0)
	for _, r := range mms {
		mtype := r.GetString("type")
		url := r.GetString("url_r2")
		tipo, dtype := "image", "imagen"
		if mtype == "video" {
			tipo, dtype = "video", "video"
		}
		var thumbHTML string
		if url != "" && tipo == "image" {
			thumbHTML = fmt.Sprintf(`<img src="%s" alt="" loading="lazy" onerror="this.parentElement.innerHTML='<span class=\'material-symbols-outlined\'>broken_image</span>'"/>`, template.HTMLEscapeString(url))
		} else if url != "" && tipo == "video" {
			thumbHTML = `<span class="material-symbols-outlined">movie</span>`
		} else {
			ic := "image"
			if tipo == "video" {
				ic = "movie"
			}
			thumbHTML = fmt.Sprintf(`<span class="material-symbols-outlined">%s</span>`, ic)
		}
		sb.WriteString(fmt.Sprintf(
			`<div class="content-item" data-id="%s" data-tipo="%s" data-type="%s" data-name="%s" data-thumb="%s" onclick="addCardToPlaylist(this)">`+
				`<div class="content-item-thumb">%s</div>`+
				`<div class="content-item-body"><div class="content-item-name">%s</div><div class="content-item-meta">%s</div></div>`+
				`</div>`,
			r.Id, tipo, dtype,
			template.HTMLEscapeString(r.GetString("filename")),
			template.HTMLEscapeString(url),
			thumbHTML,
			template.HTMLEscapeString(r.GetString("filename")),
			template.HTMLEscapeString(mtype),
		))
	}

	if len(cbs) == 0 && len(mms) == 0 {
		sb.WriteString(`<p class="pl-empty-small">Sin contenido. Sube imágenes, videos o crea eventos/noticias.</p>`)
	}

	sb.WriteString(`</div>`)
	c.Set("Content-Type", "text/html; charset=utf-8")
	return c.SendString(sb.String())
}

// playlistEditorHTML builds just the center-panel editor form.
// The surrounding 3-column layout and library live in playlists.html.
// Pass id="" and name="" for create mode; populated for edit mode.
func playlistEditorHTML(id, name, existingItemsJSON string) string {
	formAction := `hx-post="/admin/playlists"`
	deleteBtn := ""
	if id != "" {
		formAction = fmt.Sprintf(`hx-put="/admin/playlists/%s"`, id)
		deleteBtn = fmt.Sprintf(
			`<button type="button" class="pl-btn-delete" hx-delete="/admin/playlists/%s" hx-confirm="¿Eliminar esta playlist?" hx-target="#pl-center" hx-swap="innerHTML">Eliminar</button>`,
			id)
	}

	return fmt.Sprintf(`<form id="pl-form" %s hx-target="#toast-area" hx-swap="innerHTML" style="display:flex;flex-direction:column;height:100%%;min-height:0">
<input type="hidden" id="pl-items-json" name="items_json" value="[]"/>
<div class="pl-main-header">
  <input type="text" name="name" class="pl-name-input" value="%s" required placeholder="Nombre de la playlist"/>
  %s
  <button type="button" class="pl-btn-cancel" onclick="window._plReset&&window._plReset()">Cancelar</button>
  <button type="submit" class="pl-btn-save" onclick="plPrepareSubmit()">Guardar</button>
</div>
<div class="pl-table-wrap">
  <table class="pl-items-tbl">
    <thead>
      <tr>
        <th style="width:20px"></th>
        <th style="width:26px">#</th>
        <th style="width:54px"></th>
        <th>Título</th>
        <th style="width:80px">Tipo</th>
        <th style="width:90px">Duración</th>
        <th style="width:38px"></th>
      </tr>
    </thead>
    <tbody id="pl-tbody"></tbody>
  </table>
  <div id="pl-empty" class="pl-empty-state">
    <span class="material-symbols-outlined">queue_play_next</span>
    Haz clic en el contenido<br/>para agregarlo a la playlist
  </div>
</div>
</form>
<script>
(function(){
  var plItems = [];
  var EXISTING = %s;
  if(EXISTING && EXISTING.length){
    EXISTING.forEach(function(it){
      plItems.push({
        tipo: it.tipo,
        refID: it.ref_id||'',
        name: it.name||it.ref_id||'',
        thumb: it.thumb||'',
        duracion: it.duracion||15
      });
    });
  }
  function iconFor(t){ if(t==='video') return 'movie'; if(t==='content_block') return 'article'; return 'image'; }
  function labelFor(t){ if(t==='video') return 'Video'; if(t==='image') return 'Imagen'; if(t==='content_block') return 'Contenido'; return t; }
  function makeIcon(name){
    var s = document.createElement('span');
    s.className = 'material-symbols-outlined';
    s.textContent = name;
    return s;
  }
  function renderPl(){
    var tbody = document.getElementById('pl-tbody');
    var empty = document.getElementById('pl-empty');
    if(!tbody) return;
    tbody.innerHTML = '';
    if(!plItems.length){
      if(empty) empty.style.display = '';
      return;
    }
    if(empty) empty.style.display = 'none';
    plItems.forEach(function(it,idx){
      var tr = document.createElement('tr');
      tr.dataset.idx = idx;

      var tdH = document.createElement('td');
      tdH.className = 'pl-row-handle';
      var handle = document.createElement('span');
      handle.className = 'material-symbols-outlined';
      handle.textContent = 'drag_indicator';
      handle.style.fontSize = '16px';
      tdH.appendChild(handle);

      var tdIdx = document.createElement('td');
      tdIdx.className = 'pl-row-idx';
      tdIdx.textContent = (idx+1);

      var tdThumb = document.createElement('td');
      tdThumb.className = 'pl-row-thumb';
      var thumbWrap = document.createElement('div');
      thumbWrap.className = 'pl-row-thumb-inner';
      if(it.thumb && it.tipo==='image'){
        var im = document.createElement('img');
        im.src = it.thumb; im.alt = '';
        im.onerror = function(){ this.replaceWith(makeIcon(iconFor(it.tipo))); };
        thumbWrap.appendChild(im);
      } else if(it.thumb && it.tipo==='video'){
        var v = document.createElement('video');
        v.src = it.thumb; v.muted = true; v.playsInline = true; v.preload='metadata';
        thumbWrap.appendChild(v);
      } else {
        thumbWrap.appendChild(makeIcon(iconFor(it.tipo)));
      }
      tdThumb.appendChild(thumbWrap);

      var tdName = document.createElement('td');
      var nm = document.createElement('div');
      nm.className = 'pl-row-name';
      nm.textContent = it.name;
      nm.title = it.name;
      tdName.appendChild(nm);

      var tdType = document.createElement('td');
      tdType.className = 'pl-row-type';
      tdType.textContent = labelFor(it.tipo);

      var tdDur = document.createElement('td');
      var durWrap = document.createElement('div');
      durWrap.style.cssText = 'display:flex;align-items:center;gap:4px';
      var dur = document.createElement('input');
      dur.type = 'number';
      dur.className = 'pl-row-dur-input';
      dur.value = it.duracion||15;
      dur.min = 5; dur.max = 600;
      dur.addEventListener('change',(function(i){ return function(){ plItems[i].duracion=parseInt(this.value)||15; }; })(idx));
      var sLbl = document.createElement('span');
      sLbl.style.cssText = 'font-size:11px;color:var(--md-outline)';
      sLbl.textContent = 'seg';
      durWrap.appendChild(dur); durWrap.appendChild(sLbl);
      tdDur.appendChild(durWrap);

      var tdRm = document.createElement('td');
      var rm = document.createElement('button');
      rm.type = 'button';
      rm.className = 'pl-row-rm';
      rm.title = 'Quitar';
      var rmI = document.createElement('span');
      rmI.className = 'material-symbols-outlined';
      rmI.style.fontSize = '16px';
      rmI.textContent = 'close';
      rm.appendChild(rmI);
      rm.addEventListener('click',(function(i){ return function(){ plItems.splice(i,1); renderPl(); }; })(idx));
      tdRm.appendChild(rm);

      tr.appendChild(tdH); tr.appendChild(tdIdx); tr.appendChild(tdThumb);
      tr.appendChild(tdName); tr.appendChild(tdType); tr.appendChild(tdDur); tr.appendChild(tdRm);
      tbody.appendChild(tr);
    });
    if(window.Sortable){
      if(tbody._sortable){ tbody._sortable.destroy(); }
      tbody._sortable = new Sortable(tbody,{
        animation:180,handle:'.pl-row-handle',ghostClass:'sortable-ghost',
        onEnd:function(evt){
          var moved = plItems.splice(evt.oldIndex,1)[0];
          plItems.splice(evt.newIndex,0,moved);
          renderPl();
        }
      });
    }
  }
  window._plCurrentEditor = {
    add: function(card){
      var id = card.dataset.id;
      for(var i=0;i<plItems.length;i++){ if(plItems[i].refID===id){ return; } }
      plItems.push({
        tipo: card.dataset.tipo,
        refID: id,
        name: card.dataset.name||id,
        thumb: card.dataset.thumb||'',
        duracion: 15
      });
      renderPl();
    }
  };
  window.plPrepareSubmit = function(){
    var data = plItems.map(function(it,i){
      return {tipo:it.tipo,ref_id:it.refID,orden:i+1,duracion:it.duracion||15,name:it.name};
    });
    var inp = document.getElementById('pl-items-json');
    if(inp){ inp.value = JSON.stringify(data); }
  };
  renderPl();
})();
</script>`,
		formAction, template.HTMLEscapeString(name), deleteBtn, existingItemsJSON)
}

// VisitorLogs muestra el CRM de visitantes que ingresaron con Google OAuth.
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

// ── FUNNEL ──

func FunnelPage(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.SendFile("./internal/templates/admin/pages/funnel.html")
	}
}

func FunnelStats(cfg *config.Config, pb *pocketbase.PocketBase) fiber.Handler {
	return func(c *fiber.Ctx) error {
		visitors, err := pb.FindRecordsByFilter("visitor_logs", "", "", 10000, 0)
		if err != nil {
			log.Printf("⚠️  FunnelStats visitor_logs: %v", err)
		}
		propiedades, err := pb.FindRecordsByFilter("propiedades", "status='publicado'", "", 10000, 0)
		if err != nil {
			log.Printf("⚠️  FunnelStats propiedades: %v", err)
		}
		whatsappLogs, err := pb.FindRecordsByFilter("whatsapp_logs", "", "", 10000, 0)
		if err != nil {
			log.Printf("⚠️  FunnelStats whatsapp_logs: %v", err)
		}

		totalVisitantes := len(visitors)
		totalConsultas := len(whatsappLogs)
		totalPublicadas := len(propiedades)

		html := fmt.Sprintf(`
<div class="stat-card accent">
  <div class="stat-card-label">Visitantes registrados</div>
  <div class="stat-card-value">%d</div>
  <div class="stat-card-delta">Total acumulado</div>
</div>
<div class="stat-card">
  <div class="stat-card-label">Consultas WhatsApp</div>
  <div class="stat-card-value">%d</div>
  <div class="stat-card-delta">mensajes recibidos</div>
</div>
<div class="stat-card">
  <div class="stat-card-label">Propiedades publicadas</div>
  <div class="stat-card-value">%d</div>
  <div class="stat-card-delta">actualmente activas</div>
</div>
<div class="stat-card">
  <div class="stat-card-label">Tasa de conversión</div>
  <div class="stat-card-value">%s%%</div>
  <div class="stat-card-delta">consultas / visitantes</div>
</div>`,
			totalVisitantes, totalConsultas, totalPublicadas,
			funnelRate(totalConsultas, totalVisitantes),
		)
		c.Set("Content-Type", "text/html; charset=utf-8")
		return c.SendString(html)
	}
}

func SettingsLogoURL(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		s := loadSettings()
		c.Set("Content-Type", "text/plain; charset=utf-8")
		c.Set("Cache-Control", "no-store")
		return c.SendString(s.LogoURL)
	}
}

func funnelRate(num, denom int) string {
	if denom == 0 {
		return "0"
	}
	rate := float64(num) / float64(denom) * 100
	return fmt.Sprintf("%.1f", rate)
}

// ── SETTINGS ──

const settingsPath = "./pb_data/app_settings.json"

type AppSettings struct {
	LogoURL string `json:"logo_url"`
}

func loadSettings() AppSettings {
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return AppSettings{}
	}
	var s AppSettings
	if err := json.Unmarshal(data, &s); err != nil {
		log.Printf("⚠️  loadSettings: corrupt JSON in %s: %v", settingsPath, err)
	}
	return s
}

func saveSettings(s AppSettings) error {
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(settingsPath, data, 0644)
}

func SettingsPage(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.SendFile("./internal/templates/admin/pages/settings.html")
	}
}

func SettingsCurrent(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		s := loadSettings()
		c.Set("Content-Type", "text/html; charset=utf-8")
		return c.SendString(fmt.Sprintf(
			`<input type="url" name="logo_url" class="form-input" id="settings-logo-url" placeholder="https://ejemplo.com/logo.png" value="%s"/>`,
			template.HTMLEscapeString(s.LogoURL),
		))
	}
}

func SettingsUpdate(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		s := AppSettings{
			LogoURL: strings.TrimSpace(c.FormValue("logo_url")),
		}
		if err := saveSettings(s); err != nil {
			return c.Status(500).SendString(`<div class="toast toast-error">Error guardando configuración</div>`)
		}
		c.Set("Content-Type", "text/html; charset=utf-8")
		return c.SendString(`<div class="toast toast-success">Configuración guardada correctamente</div>`)
	}
}
