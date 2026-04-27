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
