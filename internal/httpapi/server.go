package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jhopan/Pan-Wallet/internal/auth"
	"github.com/jhopan/Pan-Wallet/internal/report"
	"github.com/jhopan/Pan-Wallet/internal/topup"
	"github.com/jhopan/Pan-Wallet/internal/wallet"
)

// ProductAuthenticator memetakan product API key ke nama produk.
type ProductAuthenticator interface {
	Authenticate(context.Context, string) (string, error)
}

// HoldStore menyimpan hold produk.
type HoldStore interface {
	CreateHold(context.Context, wallet.CreateHoldRequest, time.Time) (wallet.Hold, error)
}

// OverviewStore menyediakan ringkasan saldo untuk dashboard.
type OverviewStore interface {
	Overview(context.Context) (report.Overview, error)
}

// TopupStore mengelola invoice top-up QRIS.
type TopupStore interface {
	CreateTopupInvoice(context.Context, string, int64, time.Duration) (topup.Invoice, error)
	TopupInvoice(context.Context, string) (topup.Invoice, error)
	MarkTopupPaid(context.Context, string, string) (topup.Invoice, error)
	RecentTopupInvoices(context.Context, int) ([]topup.Invoice, error)
}

// SettingsStore menyimpan konfigurasi QRIS di database.
type SettingsStore interface {
	Setting(context.Context, string) (string, error)
	SetSetting(context.Context, string, string) error
}

// Deps mengumpulkan dependency server. Field nil berarti fitur terkait belum
// aktif dan endpoint-nya menjawab 503 dengan pesan jelas.
type Deps struct {
	Auth     ProductAuthenticator
	Holds    HoldStore
	Overview OverviewStore
	Topups   TopupStore
	Settings SettingsStore
	Admin    *AdminAuth
}

type Server struct {
	mux      *http.ServeMux
	deps     Deps
	renderer *Renderer
}

func New(deps Deps) *Server {
	server := &Server{mux: http.NewServeMux(), deps: deps, renderer: NewRenderer()}
	server.routes()
	return server
}

func (server *Server) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	server.mux.ServeHTTP(response, request)
}

func (server *Server) routes() {
	mux := server.mux

	// Publik
	mux.HandleFunc("GET /healthz", health)
	mux.HandleFunc("GET /pay/{id}", server.checkout)
	mux.HandleFunc("GET /pay/{id}/qr.png", server.checkoutQR)
	mux.HandleFunc("GET /api/topup/{id}", server.topupStatus)
	mux.HandleFunc("GET /admin/login", server.loginPage)
	mux.HandleFunc("POST /admin/login", server.loginSubmit)
	mux.HandleFunc("GET /admin/logout", server.logout)

	// Panel kendali (butuh sesi admin)
	mux.HandleFunc("GET /", server.dashboard)
	mux.HandleFunc("GET /admin", server.requireAdmin(server.adminHome))
	mux.HandleFunc("GET /admin/topup", server.requireAdmin(server.adminTopup))
	mux.HandleFunc("POST /admin/topup", server.requireAdmin(server.adminTopupCreate))
	mux.HandleFunc("POST /admin/topup/{id}/paid", server.requireAdmin(server.adminTopupPaid))
	mux.HandleFunc("GET /admin/config", server.requireAdmin(server.adminConfig))
	mux.HandleFunc("POST /admin/config", server.requireAdmin(server.adminConfigSave))

	// API produk (butuh product key)
	mux.HandleFunc("POST /v1/holds", server.requireProductAuth(server.hold))
}

func (server *Server) requireProductAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(response http.ResponseWriter, request *http.Request) {
		if server.deps.Auth == nil {
			writeError(response, http.StatusServiceUnavailable, "product API belum siap")
			return
		}
		token, err := bearerToken(request)
		if err != nil {
			writeError(response, http.StatusUnauthorized, "Authorization Bearer wajib diisi")
			return
		}
		product, err := server.deps.Auth.Authenticate(request.Context(), token)
		if err != nil {
			if errors.Is(err, auth.ErrUnauthorized) {
				writeError(response, http.StatusUnauthorized, "product API key tidak valid")
				return
			}
			writeError(response, http.StatusUnauthorized, "product API key tidak valid")
			return
		}
		next.ServeHTTP(response, request.WithContext(withProduct(request.Context(), product)))
	}
}

func bearerToken(request *http.Request) (string, error) {
	parts := strings.Fields(request.Header.Get("Authorization"))
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
		return "", auth.ErrUnauthorized
	}
	return parts[1], nil
}

func (server *Server) hold(response http.ResponseWriter, request *http.Request) {
	if server.deps.Holds == nil {
		writeError(response, http.StatusServiceUnavailable, "wallet store belum siap")
		return
	}
	var body struct {
		WalletUserID string    `json:"wallet_user_id"`
		ReferenceID  string    `json:"reference_id"`
		Amount       int64     `json:"amount"`
		ExpiresAt    time.Time `json:"expires_at"`
	}
	decoder := json.NewDecoder(http.MaxBytesReader(response, request.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&body); err != nil || body.ExpiresAt.IsZero() {
		writeError(response, http.StatusBadRequest, "body hold tidak valid")
		return
	}
	product, _ := productFromContext(request.Context())
	hold, err := server.deps.Holds.CreateHold(request.Context(), wallet.CreateHoldRequest{
		WalletUserID: body.WalletUserID, Product: product, ReferenceID: body.ReferenceID,
		Amount: body.Amount, IdempotencyKey: request.Header.Get("Idempotency-Key"),
	}, body.ExpiresAt)
	if err != nil {
		writeError(response, http.StatusUnprocessableEntity, err.Error())
		return
	}
	writeJSON(response, http.StatusCreated, hold)
}

func health(response http.ResponseWriter, _ *http.Request) {
	writeJSON(response, http.StatusOK, map[string]string{"status": "ok"})
}

func writeJSON(response http.ResponseWriter, status int, payload any) {
	response.Header().Set("Content-Type", "application/json; charset=utf-8")
	response.WriteHeader(status)
	json.NewEncoder(response).Encode(payload)
}

func writeError(response http.ResponseWriter, status int, message string) {
	writeJSON(response, status, map[string]string{"error": message})
}
