package httpapi

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	"sync"
	"time"
)

// AdminAuth menyimpan kredensial admin dan sesi login di memori.
// Password disimpan sebagai hash SHA-256; sesi punya masa berlaku.
type AdminAuth struct {
	username     string
	passwordHash [32]byte
	ttl          time.Duration

	mu       sync.Mutex
	sessions map[string]time.Time
}

const adminCookie = "panwallet_session"

// NewAdminAuth membuat auth admin dari username dan password awal.
func NewAdminAuth(username, password string) *AdminAuth {
	auth := &AdminAuth{
		username: username,
		ttl:      12 * time.Hour,
		sessions: map[string]time.Time{},
	}
	auth.passwordHash = sha256.Sum256([]byte(password))
	return auth
}

// CheckPassword membandingkan password secara constant-time.
func (auth *AdminAuth) CheckPassword(password string) bool {
	candidate := sha256.Sum256([]byte(password))
	return subtle.ConstantTimeCompare(candidate[:], auth.passwordHash[:]) == 1
}

func (auth *AdminAuth) Username() string { return auth.username }

// StartSession membuat token sesi baru dan mengembalikannya.
func (auth *AdminAuth) StartSession() string {
	token := randomToken()
	auth.mu.Lock()
	auth.sessions[token] = time.Now().Add(auth.ttl)
	auth.mu.Unlock()
	return token
}

// Valid memeriksa apakah token sesi masih berlaku.
func (auth *AdminAuth) Valid(token string) bool {
	if token == "" {
		return false
	}
	auth.mu.Lock()
	defer auth.mu.Unlock()
	expiry, ok := auth.sessions[token]
	if !ok {
		return false
	}
	if time.Now().After(expiry) {
		delete(auth.sessions, token)
		return false
	}
	return true
}

// EndSession menghapus token sesi.
func (auth *AdminAuth) EndSession(token string) {
	auth.mu.Lock()
	delete(auth.sessions, token)
	auth.mu.Unlock()
}

// Sweep membuang sesi kedaluwarsa. Panggil berkala dari goroutine.
func (auth *AdminAuth) Sweep() {
	auth.mu.Lock()
	defer auth.mu.Unlock()
	now := time.Now()
	for token, expiry := range auth.sessions {
		if now.After(expiry) {
			delete(auth.sessions, token)
		}
	}
}

func randomToken() string {
	buffer := make([]byte, 32)
	if _, err := rand.Read(buffer); err != nil {
		// crypto/rand gagal berarti masalah sistem; biarkan panic terlihat.
		panic(err)
	}
	return hex.EncodeToString(buffer)
}

// requireAdmin membungkus handler yang hanya boleh diakses sesi admin valid.
func (server *Server) requireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(response http.ResponseWriter, request *http.Request) {
		if server.deps.Admin == nil {
			http.Redirect(response, request, "/admin/login", http.StatusFound)
			return
		}
		cookie, err := request.Cookie(adminCookie)
		if err != nil || !server.deps.Admin.Valid(cookie.Value) {
			http.Redirect(response, request, "/admin/login", http.StatusFound)
			return
		}
		next.ServeHTTP(response, request)
	}
}

func (server *Server) loginPage(response http.ResponseWriter, request *http.Request) {
	response.Header().Set("Content-Type", "text/html; charset=utf-8")
	server.renderer.Render("login", response, map[string]any{})
}

func (server *Server) loginSubmit(response http.ResponseWriter, request *http.Request) {
	if server.deps.Admin == nil {
		http.Error(response, "admin belum dikonfigurasi", http.StatusServiceUnavailable)
		return
	}
	if err := request.ParseForm(); err != nil {
		http.Error(response, "form tidak valid", http.StatusBadRequest)
		return
	}
	username := request.FormValue("username")
	password := request.FormValue("password")
	if username != server.deps.Admin.Username() || !server.deps.Admin.CheckPassword(password) {
		response.Header().Set("Content-Type", "text/html; charset=utf-8")
		response.WriteHeader(http.StatusUnauthorized)
		server.renderer.Render("login", response, map[string]any{"Error": "Username atau password salah."})
		return
	}
	token := server.deps.Admin.StartSession()
	http.SetCookie(response, &http.Cookie{
		Name:     adminCookie,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int((12 * time.Hour).Seconds()),
	})
	http.Redirect(response, request, "/admin", http.StatusFound)
}

func (server *Server) logout(response http.ResponseWriter, request *http.Request) {
	if server.deps.Admin != nil {
		if cookie, err := request.Cookie(adminCookie); err == nil {
			server.deps.Admin.EndSession(cookie.Value)
		}
	}
	http.SetCookie(response, &http.Cookie{Name: adminCookie, Value: "", Path: "/", MaxAge: -1})
	http.Redirect(response, request, "/admin/login", http.StatusFound)
}
