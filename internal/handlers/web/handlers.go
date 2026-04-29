package web

import (
	"context"
	"fmt"
	"html/template"
	"io"
	"strings"
	"time"

	"jcp-gestioninmobiliaria/internal/config"
	"jcp-gestioninmobiliaria/internal/services"
	webtmpl "jcp-gestioninmobiliaria/internal/templates/web"

	"github.com/gofiber/fiber/v2"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

type noticiaData struct {
	Title    string
	Date     string
	ImgHTML  template.HTML
	BodyHTML template.HTML
}


// IndexHandler serves the main index.html (with HTMX fragment placeholders)
func IndexHandler(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.SendFile("./web/index.html")
	}
}

// PageHandler serves static sub-pages
func PageHandler(cfg *config.Config, page string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.SendFile(fmt.Sprintf("./web/%s.html", page))
	}
}

// renderTempl renders a Templ component into the Fiber response writer.
func renderTempl(c *fiber.Ctx, component interface{ Render(context.Context, io.Writer) error }) error {
	c.Set("Content-Type", "text/html; charset=utf-8")
	return component.Render(c.Context(), c.Response().BodyWriter())
}

// LandingHandler renders the marketing landing page with 3 preview properties.
func LandingHandler(cfg *config.Config, pb *pocketbase.PocketBase) fiber.Handler {
	return func(c *fiber.Ctx) error {
		recs, _ := pb.FindRecordsByFilter("propiedades", "status='publicado'", "-destacada,-publicado_en", 3, 0)
		var previews []webtmpl.LandingPropPreview
		for _, r := range recs {
			slug := r.GetString("slug")
			if slug == "" {
				slug = r.Id
			}
			previews = append(previews, webtmpl.LandingPropPreview{
				Slug:        slug,
				Titulo:      r.GetString("titulo"),
				CoverImage:  r.GetString("cover_image"),
				Tipo:        r.GetString("tipo"),
				Operacion:   r.GetString("operacion"),
				Dormitorios: r.GetInt("dormitorios"),
				SupUtil:     r.GetFloat("superficie_util"),
				PrecioUF:    r.GetFloat("precio_uf"),
				PrecioCLP:   r.GetFloat("precio_clp"),
				Comuna:      r.GetString("comuna"),
			})
		}
		return renderTempl(c, webtmpl.LandingPage(previews, cfg.ContactPhone))
	}
}

// GuardasHandler renders the saved properties page.
func GuardasHandler(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		name, _ := c.Locals("visitor_name").(string)
		email, _ := c.Locals("visitor_email").(string)
		if name == "" {
			name = "Visitante"
		}
		return renderTempl(c, webtmpl.GuardasPage(name, email))
	}
}

// LoginPageHandler serves the Google OAuth login page.
func LoginPageHandler(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		errMsg := ""
		switch c.Query("error") {
		case "state":
			errMsg = "Error de seguridad. Intentá de nuevo."
		case "exchange":
			errMsg = "No se pudo completar el inicio de sesión. Intentá de nuevo."
		case "invalid":
			errMsg = "Correo o contraseña incorrectos."
		case "exists":
			errMsg = "Ya tienes una cuenta registrada. Ingresa con tu correo y contraseña."
		}
		return renderTempl(c, webtmpl.LoginPage(errMsg))
	}
}

// PropiedadesHandler renders the Templ-based propiedades listing page.
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

// DeviceDisplay serves the kiosk mode display for a horizontal/screen device
func DeviceDisplay(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		code := c.Params("code")
		if code == "" {
			return c.Status(404).SendString("Código de dispositivo inválido")
		}
		return c.SendFile("./internal/templates/devices/display.html")
	}
}

// TotemDisplay serves the vertical totem kiosk for a totem device
func TotemDisplay(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		code := c.Params("code")
		if code == "" {
			return c.Status(404).SendString("Código de dispositivo inválido")
		}
		return c.SendFile("./internal/templates/devices/totem.html")
	}
}

// RSSFeed generates an RSS feed from published events and news
func RSSFeed(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// TODO: Fetch from PocketBase events + news_articles where status = 'publicado'
		now := time.Now().Format(time.RFC1123Z)

		rss := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:atom="http://www.w3.org/2005/Atom">
  <channel>
    <title>Colegio San Lorenzo — Noticias y Eventos</title>
    <link>%s</link>
    <description>Comunicados, eventos y noticias del Colegio San Lorenzo de Copiapó</description>
    <language>es-cl</language>
    <lastBuildDate>%s</lastBuildDate>
    <atom:link href="%s/rss.xml" rel="self" type="application/rss+xml"/>
    <!-- Items from PocketBase -->
    <item>
      <title>Simulacro de Evacuación — 2 de abril</title>
      <link>%s/comunicados.html</link>
      <description>Recordamos a toda la comunidad escolar el simulacro de evacuación obligatorio.</description>
      <pubDate>%s</pubDate>
      <guid>%s/events/1</guid>
    </item>
  </channel>
</rss>`, cfg.BaseURL, now, cfg.BaseURL, cfg.BaseURL, now, cfg.BaseURL)

		c.Set("Content-Type", "application/rss+xml; charset=utf-8")
		return c.SendString(rss)
	}
}

// WhatsAppWebhook handles inbound WhatsApp messages
func WhatsAppWebhook(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.SendString("<Response></Response>")
	}
}

// NoticiaHandler renders a single news article using the full site layout template.
func NoticiaHandler(cfg *config.Config, pb *pocketbase.PocketBase) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id := c.Params("id")
		r, err := pb.FindRecordById("content_blocks", id)
		if err != nil || r.GetString("category") != "NOTICIA" || r.GetString("status") != "publicado" {
			return c.Redirect("/", fiber.StatusFound)
		}

		title := r.GetString("title")
		desc := r.GetString("description")
		body := r.GetString("body")
		imageURL := r.GetString("image_url")

		dateStr := ""
		if dt := r.GetDateTime("date"); !dt.IsZero() {
			dateStr = dt.Time().Format("2 de January de 2006")
		}

		var imgHTML template.HTML
		if imageURL != "" {
			imgHTML = template.HTML(fmt.Sprintf(
				`<div style="width:100%%;aspect-ratio:16/6;border-radius:24px;margin-bottom:48px;overflow:hidden"><img src="%s" style="width:100%%;height:100%%;object-fit:cover" alt="%s"/></div>`,
				template.HTMLEscapeString(imageURL), template.HTMLEscapeString(title)))
		} else {
			imgHTML = `<div style="width:100%;aspect-ratio:16/6;background:var(--md-primary-container);border-radius:24px;margin-bottom:48px;display:flex;align-items:center;justify-content:center"><span style="font-family:'DM Serif Display',Georgia,serif;font-size:80px;color:rgba(155,18,48,0.15)">SL</span></div>`
		}

		src := body
		if src == "" {
			src = desc
		}
		var bodyParts []string
		for _, p := range strings.Split(strings.TrimSpace(src), "\n\n") {
			p = strings.TrimSpace(p)
			if p != "" {
				bodyParts = append(bodyParts, "<p>"+template.HTMLEscapeString(p)+"</p>")
			}
		}
		bodyHTML := template.HTML(strings.Join(bodyParts, "\n"))

		tmpl, err2 := template.ParseFiles("./internal/templates/web/noticia.html")
		if err2 != nil {
			return c.Status(500).SendString("Template error")
		}

		c.Set("Content-Type", "text/html; charset=utf-8")
		return tmpl.ExecuteTemplate(c, "noticia.html", noticiaData{
			Title:    title,
			Date:     dateStr,
			ImgHTML:  imgHTML,
			BodyHTML: bodyHTML,
		})
	}
}

// PropiedadHandler renders a single real-estate listing using the detail template.
// Matches either slug or id.
func PropiedadHandler(cfg *config.Config, pb *pocketbase.PocketBase) fiber.Handler {
	return func(c *fiber.Ctx) error {
		visitorName, _ := c.Locals("visitor_name").(string)
		visitorEmail, _ := c.Locals("visitor_email").(string)
		if visitorName == "" {
			visitorName = "Visitante"
		}
		key := c.Params("key")
		var r *core.Record
		// try by slug first
		recs, err := pb.FindRecordsByFilter("propiedades",
			fmt.Sprintf("slug = '%s' && status = 'publicado'",
				strings.ReplaceAll(key, "'", "")), "", 1, 0)
		if err == nil && len(recs) > 0 {
			r = recs[0]
		} else {
			rec, err2 := pb.FindRecordById("propiedades", key)
			if err2 == nil && rec.GetString("status") == "publicado" {
				r = rec
			}
		}
		if r == nil {
			return c.Redirect("/propiedades.html", fiber.StatusFound)
		}

		titulo := r.GetString("titulo")
		descripcion := r.GetString("descripcion")
		operacion := r.GetString("operacion")
		direccion := r.GetString("direccion")
		comuna := r.GetString("comuna")
		region := r.GetString("region")
		precioUF := r.GetFloat("precio_uf")
		precioCLP := r.GetFloat("precio_clp")
		dorm := r.GetInt("dormitorios")
		banos := r.GetInt("banos")
		estac := r.GetInt("estacionamientos")
		supUtil := r.GetFloat("superficie_util")
		supTotal := r.GetFloat("superficie_total")
		ano := r.GetInt("ano_construccion")
		cover := r.GetString("cover_image")
		gallery := splitAndTrim(r.GetString("gallery"))
		amenities := splitAndTrim(r.GetString("amenidades"))
		destacada := r.GetBool("destacada")
		oportunidad := r.GetBool("oportunidad")
		whatsapp := r.GetString("contacto_whatsapp")

		locLine := strings.TrimSpace(strings.Trim(
			fmt.Sprintf("%s, %s · %s", direccion, comuna, region), ", ·"))
		if locLine == "" {
			locLine = direccion
		}

		// Cover
		var coverHTML template.HTML
		if cover != "" {
			coverHTML = template.HTML(fmt.Sprintf(`<img src="%s" alt="%s"/>`,
				template.HTMLEscapeString(cover), template.HTMLEscapeString(titulo)))
		} else {
			coverHTML = `<span class="gallery-placeholder">JCP</span>`
		}
		// Thumbs: up to 4 cells
		thumbs := ""
		for i := 0; i < 4; i++ {
			if i < len(gallery) {
				thumbs += fmt.Sprintf(`<div class="g-cell"><img src="%s" alt=""/></div>`,
					template.HTMLEscapeString(gallery[i]))
			} else {
				thumbs += `<div class="g-cell"></div>`
			}
		}

		// Chips
		chips := ""
		switch operacion {
		case "VENTA":
			chips += `<span class="op-chip op-venta">En Venta</span>`
		case "ARRIENDO":
			chips += `<span class="op-chip op-arriendo">En Arriendo</span>`
		}
		if destacada {
			chips += `<span class="op-chip op-dest">Destacada</span>`
		}
		if oportunidad {
			chips += `<span class="op-chip op-deal">Oportunidad</span>`
		}

		// Price
		priceMain := ""
		if precioUF > 0 {
			if precioUF == float64(int64(precioUF)) {
				priceMain = fmt.Sprintf("UF %d", int64(precioUF))
			} else {
				priceMain = fmt.Sprintf("UF %.1f", precioUF)
			}
		} else if precioCLP > 0 {
			priceMain = formatCLPString(precioCLP)
		}
		priceSub := ""
		ufVal := services.GetUF()
		if precioUF > 0 && precioCLP > 0 {
			priceSub = "≈ " + formatCLPString(precioCLP)
		} else if precioUF > 0 && ufVal > 0 {
			priceSub = "≈ " + formatCLPString(precioUF*ufVal) + " (UF hoy)"
		} else if precioCLP > 0 && ufVal > 0 {
			eq := precioCLP / ufVal
			if eq == float64(int64(eq)) {
				priceSub = fmt.Sprintf("≈ UF %d (hoy)", int64(eq))
			} else {
				priceSub = fmt.Sprintf("≈ UF %.1f (hoy)", eq)
			}
		}
		if priceSub == "" && operacion == "ARRIENDO" && precioCLP > 0 {
			priceSub = "Mensual"
		}

		// Feature boxes
		feats := ""
		if dorm > 0 {
			feats += featBox(fmt.Sprintf("%d", dorm), "Dormitorios")
		}
		if banos > 0 {
			feats += featBox(fmt.Sprintf("%d", banos), "Baños")
		}
		if supUtil > 0 {
			feats += featBox(fmt.Sprintf("%g m²", supUtil), "Sup. útil")
		}
		if supTotal > 0 && supTotal != supUtil {
			feats += featBox(fmt.Sprintf("%g m²", supTotal), "Sup. total")
		}
		if estac > 0 {
			feats += featBox(fmt.Sprintf("%d", estac), "Estacionamientos")
		}
		if ano > 0 {
			feats += featBox(fmt.Sprintf("%d", ano), "Año")
		}

		// Amenities — just pill items; heading is rendered by Templ
		amenitiesHTML := ""
		if len(amenities) > 0 {
			var ab strings.Builder
			for _, a := range amenities {
				a = strings.ReplaceAll(a, "_", " ")
				ab.WriteString(`<span class="prest-pill">`)
				ab.WriteString(template.HTMLEscapeString(a))
				ab.WriteString(`</span>`)
			}
			amenitiesHTML = ab.String()
		}

		// Description body (preserve paragraph breaks)
		var bodyParts []string
		src := strings.TrimSpace(descripcion)
		if src == "" {
			src = "Sin descripción disponible."
		}
		for _, p := range strings.Split(src, "\n\n") {
			p = strings.TrimSpace(p)
			if p != "" {
				bodyParts = append(bodyParts, "<p>"+template.HTMLEscapeString(p)+"</p>")
			}
		}
		bodyHTML := template.HTML(strings.Join(bodyParts, "\n"))

		whatsappHTML := ""
		if whatsapp != "" {
			msg := "Hola! Me interesa la propiedad: " + titulo
			whatsappHTML = fmt.Sprintf(
				`<a href="https://wa.me/%s?text=%s" target="_blank" rel="noopener" class="pb-btn pb-wa sb-cta sb-wa"><span class="ms ms-sm">chat</span> WhatsApp</a>`,
				onlyDigits(whatsapp), urlEscape(msg))
		}

		data := webtmpl.PropiedadData{
			Titulo:        titulo,
			Direccion:     locLine,
			PriceHTML:     template.HTML(priceMain),
			PriceSub:      priceSub,
			ChipsHTML:     template.HTML(chips),
			CoverHTML:     coverHTML,
			ThumbsHTML:    template.HTML(thumbs),
			FeatsHTML:     template.HTML(feats),
			BodyHTML:      bodyHTML,
			AmenitiesHTML: template.HTML(amenitiesHTML),
			WhatsappHTML:  template.HTML(whatsappHTML),
			WhatsappPhone: onlyDigits(whatsapp),
			Lat:           r.GetFloat("lat"),
			Lng:           r.GetFloat("lng"),
			Comuna:        comuna,
			UserName:      visitorName,
			UserEmail:     visitorEmail,
		}
		return renderTempl(c, webtmpl.PropiedadDetail(data))
	}
}

// ── helpers ──

func splitAndTrim(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

var featIcons = map[string]string{
	"Dormitorios":      "bed",
	"Baños":            "shower",
	"Estacionamientos": "directions_car",
	"Sup. útil":        "square_foot",
	"Sup. total":       "square_foot",
	"Año":              "calendar_today",
}

func featBox(value, label string) string {
	icon := featIcons[label]
	iconHTML := ""
	if icon != "" {
		iconHTML = fmt.Sprintf(`<div class="feat-icon"><span class="ms ms-sm">%s</span></div>`, icon)
	}
	return fmt.Sprintf(`<div class="feat-box">%s<div class="feat-val">%s</div><div class="feat-label">%s</div></div>`,
		iconHTML,
		template.HTMLEscapeString(value),
		template.HTMLEscapeString(label))
}

func formatCLPString(v float64) string {
	if v <= 0 {
		return ""
	}
	n := int64(v)
	s := fmt.Sprintf("%d", n)
	out := make([]byte, 0, len(s)+len(s)/3)
	for i, c := range []byte(s) {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, '.')
		}
		out = append(out, c)
	}
	return "$ " + string(out)
}

func onlyDigits(s string) string {
	b := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] >= '0' && s[i] <= '9' {
			b = append(b, s[i])
		}
	}
	return string(b)
}

func urlEscape(s string) string {
	// minimal: replace spaces
	return strings.ReplaceAll(s, " ", "%20")
}
