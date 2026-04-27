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
		titulo, slug, descripcion string
		operacion, tipo           string
		direccion, comuna, region string
		precioUF, precioCLP       float64
		dorm, banos, estac        int
		supUtil, supTotal         float64
		ano                       int
		estado, amenidades        string
		destacada, oportunidad    bool
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
			supTotal: 5000, estado: "usada",
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
