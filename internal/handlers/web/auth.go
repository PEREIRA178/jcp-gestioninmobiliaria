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
