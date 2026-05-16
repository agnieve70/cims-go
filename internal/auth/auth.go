package auth

import (
	"context"
	"errors"
	"net/http"

	"cims-go/internal/models"

	"github.com/gorilla/securecookie"
	"golang.org/x/crypto/bcrypt"
)

type contextKey string

const userContextKey contextKey = "user"

type UserStore interface {
	GetUserByUsername(ctx context.Context, username string) (models.User, error)
	GetUserByID(ctx context.Context, id int64) (models.User, error)
}

type Manager struct {
	store  UserStore
	codec  *securecookie.SecureCookie
	secure bool
}

func NewManager(store UserStore, hashKey, blockKey string) *Manager {
	return &Manager{
		store: store,
		codec: securecookie.New([]byte(hashKey), []byte(blockKey)),
	}
}

func (m *Manager) Login(ctx context.Context, w http.ResponseWriter, username, password string) (models.User, error) {
	user, err := m.store.GetUserByUsername(ctx, username)
	if err != nil {
		return models.User{}, errors.New("invalid username or password")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return models.User{}, errors.New("invalid username or password")
	}
	encoded, err := m.codec.Encode("session", map[string]int64{"user_id": user.ID})
	if err != nil {
		return models.User{}, err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "cims_session",
		Value:    encoded,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   m.secure,
	})
	return user, nil
}

func (m *Manager) Logout(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "cims_session",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   m.secure,
	})
}

func (m *Manager) LoadUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("cims_session")
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}
		value := map[string]int64{}
		if err := m.codec.Decode("session", cookie.Value, &value); err != nil {
			next.ServeHTTP(w, r)
			return
		}
		userID := value["user_id"]
		if userID == 0 {
			next.ServeHTTP(w, r)
			return
		}
		user, err := m.store.GetUserByID(r.Context(), userID)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}
		next.ServeHTTP(w, r.WithContext(WithUser(r.Context(), user)))
	})
}

func (m *Manager) RequireLogin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := CurrentUser(r.Context()); !ok {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func RequireWrite(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := CurrentUser(r.Context())
		if !ok || !user.CanWrite() {
			http.Error(w, "write access required", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func WithUser(ctx context.Context, user models.User) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

func CurrentUser(ctx context.Context) (models.User, bool) {
	user, ok := ctx.Value(userContextKey).(models.User)
	return user, ok
}

func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hash), err
}
