package middleware

import (
	"time"

	"jcp-gestioninmobiliaria/internal/config"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
)

// LoginRateLimiter allows 5 login attempts per IP per 15 minutes.
func LoginRateLimiter() fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        5,
		Expiration: 15 * time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			if c.Get("HX-Request") == "true" {
				return c.Status(fiber.StatusTooManyRequests).SendString(
					`<div class="toast toast-error">Demasiados intentos. Intentá de nuevo en 15 minutos.</div>`,
				)
			}
			return c.Status(fiber.StatusTooManyRequests).
				SendString("Demasiados intentos de login. Intentá de nuevo en 15 minutos.")
		},
	})
}

// FragmentsRateLimiter allows 60 fragment requests per IP per minute.
func FragmentsRateLimiter() fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        60,
		Expiration: 1 * time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).
				SendString("Límite de solicitudes alcanzado. Intentá en un momento.")
		},
	})
}

// GlobalRateLimiter allows 200 requests per IP per minute across all routes.
func GlobalRateLimiter() fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        200,
		Expiration: 1 * time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).
				SendString("Límite de solicitudes alcanzado.")
		},
	})
}

// SecurityHeaders adds standard security headers to every response.
// HSTS is only set in production (requires HTTPS to be effective).
func SecurityHeaders(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Set("X-Frame-Options", "DENY")
		c.Set("X-Content-Type-Options", "nosniff")
		c.Set("X-XSS-Protection", "1; mode=block")
		c.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		c.Set("Content-Security-Policy",
			"default-src 'self'; "+
				"script-src 'self' unpkg.com cdn.jsdelivr.net 'unsafe-inline'; "+
				"style-src 'self' 'unsafe-inline'; "+
				"img-src 'self' data: blob: *; "+
				"connect-src 'self'; "+
				"font-src 'self' data:",
		)
		if cfg.IsProd() {
			c.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		return c.Next()
	}
}
