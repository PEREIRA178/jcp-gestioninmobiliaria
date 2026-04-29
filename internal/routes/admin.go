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
	adm.Get("/visitors", admin.VisitorLogs(cfg, pb))

	adm.Get("/funnel", admin.FunnelPage(cfg))
	adm.Get("/funnel/stats", admin.FunnelStats(cfg, pb))

	adm.Get("/settings", admin.SettingsPage(cfg))
	adm.Get("/settings/current", admin.SettingsCurrent(cfg))
	adm.Get("/settings/logo-url", admin.SettingsLogoURL(cfg))
	adm.Post("/settings", admin.SettingsUpdate(cfg))

	// Propiedades
	adm.Get("/propiedades", admin.PropiedadesList(cfg, pb))
	adm.Get("/propiedades/new", admin.PropiedadForm(cfg))
	adm.Post("/propiedades", admin.PropiedadCreate(cfg, pb))
	adm.Get("/propiedades/:id/edit", admin.PropiedadEdit(cfg, pb))
	adm.Put("/propiedades/:id", admin.PropiedadUpdate(cfg, pb))
	adm.Delete("/propiedades/:id", admin.PropiedadDelete(cfg, pb))
	adm.Post("/propiedades/:id/publish", admin.PropiedadToggleStatus(cfg, pb))
}
