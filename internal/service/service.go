package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/prakashniraula/portfolio-in-go/internal/config"
	"github.com/prakashniraula/portfolio-in-go/internal/models"
	"github.com/prakashniraula/portfolio-in-go/internal/repo"
	"golang.org/x/crypto/bcrypt"
)

var ErrInvalidCredentials = errors.New("invalid credentials")

type Services struct {
	Store  repo.Store
	Config config.Config
}

func New(store repo.Store, cfg config.Config) *Services {
	return &Services{Store: store, Config: cfg}
}

func (s *Services) Login(ctx context.Context, username, password string) (token string, err error) {
	user, err := s.Store.GetUserByUsername(ctx, username)
	if err != nil {
		return "", err
	}
	if user == nil {
		return "", ErrInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", ErrInvalidCredentials
	}
	raw, err := randomToken(32)
	if err != nil {
		return "", err
	}
	hash := hashToken(raw)
	expires := time.Now().UTC().Add(7 * 24 * time.Hour).Format("2006-01-02 15:04:05")
	if err := s.Store.CreateSession(ctx, user.ID, hash, expires); err != nil {
		return "", err
	}
	return raw, nil
}

func (s *Services) Logout(ctx context.Context, rawToken string) error {
	if rawToken == "" {
		return nil
	}
	return s.Store.DeleteSession(ctx, hashToken(rawToken))
}

func (s *Services) UserFromSession(ctx context.Context, rawToken string) (*models.User, error) {
	if rawToken == "" {
		return nil, nil
	}
	return s.Store.GetSessionUser(ctx, hashToken(rawToken))
}

func (s *Services) ChangePasswordWithUser(ctx context.Context, user *models.User, current, next string) error {
	if user == nil {
		return ErrInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(current)); err != nil {
		return ErrInvalidCredentials
	}
	if len(next) < 8 {
		return errors.New("password must be at least 8 characters")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(next), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return s.Store.UpdatePassword(ctx, user.ID, string(hash))
}

func (s *Services) SubmitContact(ctx context.Context, name, email, subject, message string) error {
	name = strings.TrimSpace(name)
	email = strings.TrimSpace(email)
	subject = strings.TrimSpace(subject)
	message = strings.TrimSpace(message)
	if name == "" || email == "" || subject == "" || message == "" {
		return errors.New("all fields are required")
	}
	if !strings.Contains(email, "@") {
		return errors.New("invalid email")
	}
	id, err := s.Store.CreateContactMessage(ctx, models.ContactMessage{
		Name: name, Email: email, Subject: subject, Message: message,
	})
	if err != nil {
		return err
	}
	if s.Config.TelegramBotToken != "" && s.Config.TelegramChatID != "" {
		text := fmt.Sprintf("New contact message\nName: %s\nEmail: %s\nSubject: %s\nMessage: %s", name, email, subject, message)
		if err := sendTelegram(s.Config.TelegramBotToken, s.Config.TelegramChatID, text); err == nil {
			_ = s.Store.MarkMessageNotified(ctx, id)
		}
	}
	return nil
}

func sendTelegram(token, chatID, text string) error {
	endpoint := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)
	form := url.Values{}
	form.Set("chat_id", chatID)
	form.Set("text", text)
	req, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("telegram status %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

func randomToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func SplitCSV(s string) []string {
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

func JoinCSV(items []string) string {
	return strings.Join(items, ", ")
}

func Slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	prevDash := false
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			prevDash = false
			continue
		}
		if !prevDash {
			b.WriteByte('-')
			prevDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}
