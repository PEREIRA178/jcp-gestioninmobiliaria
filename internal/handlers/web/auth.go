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
	"golang.org/x/crypto/bcrypt"
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
		go logVisitor(pb, gu, c.IP(), c.Get("User-Agent"), "google")

		next := c.Query("next", "/propiedades")
		return c.Redirect(next)
	}
}

// logVisitor saves visitor login data to visitor_logs collection.
func logVisitor(pb *pocketbase.PocketBase, u googleUserInfo, ip, ua, loginType string) {
	collection, err := pb.FindCollectionByNameOrId("visitor_logs")
	if err != nil {
		log.Printf("⚠️  logVisitor: visitor_logs collection not found: %v", err)
		return
	}
	record := core.NewRecord(collection)
	record.Set("email", u.Email)
	record.Set("name", u.Name)
	record.Set("picture", u.Picture)
	record.Set("ip", ip)
	record.Set("user_agent", ua)
	record.Set("login_type", loginType)
	if err := pb.Save(record); err != nil {
		log.Printf("⚠️  logVisitor: save failed for %s: %v", u.Email, err)
	}
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

		if email == "" || name == "" {
			return c.Redirect("/register?error=save")
		}
		if len(password) < 8 {
			return c.Redirect("/register?error=short")
		}

		// Check if email is already registered
		existing, _ := pb.FindRecordsByFilter("visitor_accounts", "email = '"+email+"'", "", 1, 0)
		if len(existing) > 0 {
			return c.Redirect("/login?error=exists")
		}

		// Hash password and save credentials
		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return c.Redirect("/register?error=save")
		}
		if col, err := pb.FindCollectionByNameOrId("visitor_accounts"); err == nil {
			r := core.NewRecord(col)
			r.Set("email", email)
			r.Set("name", name)
			r.Set("password_hash", string(hash))
			if saveErr := pb.Save(r); saveErr != nil {
				log.Printf("register: visitor_accounts save failed for %s: %v", email, saveErr)
			}
		}

		// Log to visitor_logs
		go logVisitor(pb, googleUserInfo{Email: email, Name: name}, c.IP(), c.Get("User-Agent"), "email")

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

// VisitorLoginSubmit handles email+password login for previously registered visitors.
func VisitorLoginSubmit(cfg *config.Config, pb *pocketbase.PocketBase) fiber.Handler {
	return func(c *fiber.Ctx) error {
		email := strings.TrimSpace(c.FormValue("email"))
		password := c.FormValue("password")

		if email == "" || password == "" {
			return c.Redirect("/login?error=invalid")
		}

		records, err := pb.FindRecordsByFilter("visitor_accounts", "email = '"+email+"'", "", 1, 0)
		if err != nil || len(records) == 0 {
			return c.Redirect("/login?error=invalid")
		}
		r := records[0]

		if err := bcrypt.CompareHashAndPassword([]byte(r.GetString("password_hash")), []byte(password)); err != nil {
			return c.Redirect("/login?error=invalid")
		}

		name := r.GetString("name")
		jwtToken, err := auth.GenerateToken(cfg, "", email, "visitor", name)
		if err != nil {
			return c.Redirect("/login?error=invalid")
		}

		c.Cookie(&fiber.Cookie{
			Name:     "jcp_visitor",
			Value:    jwtToken,
			MaxAge:   int(cfg.JWTExpiration / time.Second),
			HTTPOnly: true,
			SameSite: "Lax",
			Secure:   cfg.IsProd(),
		})

		next := c.Query("next", "/propiedades")
		return c.Redirect(next)
	}
}
