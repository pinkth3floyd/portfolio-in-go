package middleware

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"github.com/prakashniraula/portfolio-in-go/internal/config"
	"github.com/prakashniraula/portfolio-in-go/internal/models"
	"github.com/prakashniraula/portfolio-in-go/internal/service"
)

type ctxKey string

const (
	UserKey  ctxKey = "user"
	CSRFKey  ctxKey = "csrf"
	TokenKey ctxKey = "session_token"
)

const sessionCookie = "session_token"
const csrfCookie = "csrf_token"

func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		next.ServeHTTP(w, r)
	})
}

func Session(svc *service.Services, cfg config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw, _ := r.Cookie(sessionCookie)
			var user *models.User
			token := ""
			if raw != nil && raw.Value != "" {
				token = raw.Value
				u, err := svc.UserFromSession(r.Context(), token)
				if err == nil {
					user = u
				}
			}
			csrf := ensureCSRF(w, r, cfg)
			ctx := context.WithValue(r.Context(), UserKey, user)
			ctx = context.WithValue(ctx, CSRFKey, csrf)
			ctx = context.WithValue(ctx, TokenKey, token)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, _ := r.Context().Value(UserKey).(*models.User)
		if user == nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func RequireCSRF(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}
		cookie, _ := r.Cookie(csrfCookie)
		formToken := r.FormValue("csrf_token")
		if formToken == "" {
			formToken = r.Header.Get("X-CSRF-Token")
		}
		if cookie == nil || formToken == "" || !hmac.Equal([]byte(cookie.Value), []byte(formToken)) {
			http.Error(w, "invalid CSRF token", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func SetSessionCookie(w http.ResponseWriter, token string, cfg config.Config) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   cfg.SecureCookies,
		Expires:  time.Now().Add(7 * 24 * time.Hour),
	})
}

func ClearSessionCookie(w http.ResponseWriter, cfg config.Config) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   cfg.SecureCookies,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})
}

func ensureCSRF(w http.ResponseWriter, r *http.Request, cfg config.Config) string {
	if c, err := r.Cookie(csrfCookie); err == nil && c.Value != "" {
		return c.Value
	}
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	token := hex.EncodeToString(b)
	http.SetCookie(w, &http.Cookie{
		Name:     csrfCookie,
		Value:    token,
		Path:     "/",
		HttpOnly: false,
		SameSite: http.SameSiteLaxMode,
		Secure:   cfg.SecureCookies,
		Expires:  time.Now().Add(24 * time.Hour),
	})
	return token
}

func UserFrom(r *http.Request) *models.User {
	u, _ := r.Context().Value(UserKey).(*models.User)
	return u
}

func CSRFFrom(r *http.Request) string {
	s, _ := r.Context().Value(CSRFKey).(string)
	return s
}

func SessionTokenFrom(r *http.Request) string {
	s, _ := r.Context().Value(TokenKey).(string)
	return s
}

func Sign(secret, payload string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func IsHTMX(r *http.Request) bool {
	return strings.EqualFold(r.Header.Get("HX-Request"), "true")
}
