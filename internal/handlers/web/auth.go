package web

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"io"
	"log"
	"strings"
	"time"

	"jcp-gestioninmobiliaria/internal/auth"
	"jcp-gestioninmobiliaria/internal/config"
	webtmpl "jcp-gestioninmobiliaria/internal/templates/web"

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

// logVisitor saves visitor login data to visitor_logs collection.
// Silently swallows errors so auth is never affected.
func logVisitor(pb *pocketbase.PocketBase, u googleUserInfo, ip, ua string) {
	collection, err := pb.FindCollectionByNameOrId("visitor_logs")
	if err != nil {
		return // collection not created yet — no-op
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

// RegisterPageHandler serves the email registration page.
func RegisterPageHandler(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		errMsg := c.Query("error")
		switch errMsg {
		case "short":
			errMsg = "La contraseña debe tener al menos 8 caracteres."
		case "save":
			errMsg = "Error al crear la cuenta. Intenta de nuevo."
		case "csrf":
			errMsg = "Sesión inválida. Recarga la página e intenta de nuevo."
		default:
			errMsg = ""
		}
		csrf := randomState()
		c.Cookie(&fiber.Cookie{
			Name:     "reg_csrf",
			Value:    csrf,
			MaxAge:   600,
			HTTPOnly: true,
			SameSite: "Lax",
			Secure:   cfg.IsProd(),
		})
		return renderTempl(c, webtmpl.RegisterPage(errMsg, csrf))
	}
}

// RegisterSubmit handles email+password registration creating a visitor JWT session.
func RegisterSubmit(cfg *config.Config, pb *pocketbase.PocketBase) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if c.FormValue("_csrf") != c.Cookies("reg_csrf") || c.Cookies("reg_csrf") == "" {
			return c.Redirect("/register?error=csrf")
		}
		c.ClearCookie("reg_csrf")

		name := strings.TrimSpace(c.FormValue("name"))
		email := strings.TrimSpace(c.FormValue("email"))
		password := c.FormValue("password")

		// Primero validar campos requeridos
		if email == "" || name == "" {
			return c.Redirect("/register?error=save")
		}
		// Luego validar restricciones
		if len(password) < 8 {
			return c.Redirect("/register?error=short")
		}
		// password validated for length only; not stored — this is a one-shot visitor session

		// Log the new email-registered visitor
		collection, err := pb.FindCollectionByNameOrId("visitor_logs")
		if err == nil {
			record := core.NewRecord(collection)
			record.Set("email", email)
			record.Set("name", name)
			record.Set("picture", "")
			record.Set("ip", c.IP())
			record.Set("user_agent", c.Get("User-Agent"))
			if saveErr := pb.Save(record); saveErr != nil {
				log.Printf("register: visitor_logs save failed for %s: %v", email, saveErr)
			}
		}

		jwtToken, err := auth.GenerateToken(cfg, "", email, "visitor", name)
		if err != nil {
			return c.Redirect("/register?error=save")
		}

		c.Cookie(&fiber.Cookie{
			Name:     "jcp_visitor",
			Value:    jwtToken,
			MaxAge:   int(cfg.JWTExpiration / time.Second),
			HTTPOnly: true,
			SameSite: "Lax",
			Secure:   cfg.IsProd(),
		})

		return c.Redirect("/propiedades")
	}
}
