package routes

import (
	"jcp-gestioninmobiliaria/internal/config"
	"jcp-gestioninmobiliaria/internal/handlers/fragments"
	"jcp-gestioninmobiliaria/internal/middleware"

	"github.com/gofiber/fiber/v2"
	"github.com/pocketbase/pocketbase"
)

// RegisterFragments monta los endpoints HTMX de propiedades y noticias.
func RegisterFragments(app *fiber.App, cfg *config.Config, pb *pocketbase.PocketBase) {
	frag := app.Group("/fragments", middleware.FragmentsRateLimiter())
	frag.Get("/hero", fragments.HeroCarousel(cfg, pb))
	frag.Get("/propiedades-destacadas", fragments.PropiedadesDestacadas(cfg, pb))
	frag.Get("/propiedades-page", fragments.PropiedadesPage(cfg, pb))
	frag.Get("/noticias", fragments.Noticias(cfg, pb))
	frag.Get("/noticias-page", fragments.NoticiasPage(cfg, pb))
	frag.Get("/guardadas", fragments.GuardasFragment(cfg, pb))
}
