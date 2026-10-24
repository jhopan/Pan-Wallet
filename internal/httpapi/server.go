package httpapi

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/jhopan/Pan-Wallet/internal/notify"
	"net/http"
	"strings"
	"time"

	"github.com/jhopan/Pan-Wallet/internal/auth"
	"github.com/jhopan/Pan-Wallet/internal/paypan"
	"github.com/jhopan/Pan-Wallet/internal/ratelimit"
	"github.com/jhopan/Pan-Wallet/internal/report"
	"github.com/jhopan/Pan-Wallet/internal/sqlite"
	"github.com/jhopan/Pan-Wallet/internal/topup"
	"github.com/jhopan/Pan-Wallet/internal/wallet"
)

// ProductAuthenticator memetakan product API key ke nama produk.
type ProductAuthenticator interface {
	Authenticate(context.Context, string) (string, error)
}

// HoldStore menyimpan hold produk dan transisinya.
type HoldStore interface {
	CreateHold(context.Context, wallet.CreateHoldRequest, time.Time) (wallet.Hold, error)
	CaptureHold(context.Context, wallet.TransitionRequest) (wallet.Hold, error)
	ReleaseHold(context.Context, wallet.TransitionRequest) (wallet.Hold, error)
	WalletBalance(context.Context, string) (report.OverviewWallet, error)
}

// OverviewStore menyediakan ringkasan saldo untuk dashboard.
type OverviewStore interface {
	Overview(context.Context) (report.Overview, error)
}

// TopupStore mengelola invoice top-up lewat PayPan.
type TopupStore interface {
	CreateTopupInvoice(context.Context, *paypan.Client, string, int64) (topup.Invoice, error)
	TopupInvoice(context.Context, string) (topup.Invoice, error)
	MarkTopupPaidByPaypan(context.Context, string, string) (topup.Invoice, error)
	MarkTopupPaidByLocalID(context.Context, string, string) (topup.Invoice, error)
	RecentTopupInvoices(context.Context, int) ([]topup.Invoice, error)
	ExpireTopupInvoice(context.Context, string) error
}

// SettingsStore menyimpan konfigurasi QRIS di database.
type SettingsStore interface {
	Setting(context.Context, string) (string, error)
	SetSetting(context.Context, string, string) error
}

// LedgerStore membaca riwayat mutasi saldo.
type LedgerStore interface {
	LedgerEntries(context.Context, string, int) ([]sqlite.LedgerEntry, error)
}

// KeyStore mengelola API key produk.
type KeyStore interface {
	ListProductKeys(context.Context) ([]sqlite.ProductKey, error)
	CreateProductKey(context.Context, string) (string, error)
	RevokeProductKey(context.Context, string) error
}

// WalletStore mengelola daftar wallet user.
type WalletStore interface {
	ListWallets(context.Context, int) ([]sqlite.WalletRow, error)
	CreateWalletUser(context.Context, int64) (string, error)
	SetWalletStatus(context.Context, string, string) error
}

// Deps mengumpulkan dependency server. Field nil berarti fitur terkait belum
// aktif dan endpoint-nya menjawab 503 dengan pesan jelas.
type Deps struct {
	Auth       ProductAuthenticator
	Holds      HoldStore
	Overview   OverviewStore
	Topups     TopupStore
	Settings   SettingsStore
	Admin      *AdminAuth
	Ledger     LedgerStore
	Keys       KeyStore
	Wallets    WalletStore
	PublicIDs  PublicIDStore
	Identities IdentityStore
	Reconcile  ReconcileStore
	Pinger     Pinger
	Audit      AuditStore
	Revenue    RevenueStore
	Backup     BackupRunner
	Notify     NotifyStore
	// TrustedProxies adalah daftar IP/CIDR reverse proxy (cloudflared, nginx)
	// yang boleh mengirim header X-Forwarded-For / CF-Connecting-IP.
	TrustedProxies []string
	// PublicBaseURL dipakai menyusun URL absolut (webhook, pay_url).
	PublicBaseURL string
	// Database dipakai halaman ekspor CSV.
	Database *sql.DB
}

// NotifyStore mengirim notifikasi keluar (Telegram / webhook).
type NotifyStore interface {
	SendTopupPaid(ctx context.Context, event notify.TopupEvent) error
	SendTest(ctx context.Context) error
}

type Server struct {
	mux      *http.ServeMux
	deps     Deps
	renderer *Renderer
	limiter  *ratelimit.Limiter
	// loginLimiter membatasi percobaan login admin per alamat IP.
	// Tanpa ini, password bisa ditebak ribuan kali per detik.
	loginLimiter *ratelimit.Limiter
	// trustedProxies menentukan siapa yang boleh mengirim header IP.
	trustedProxies *trustedProxies
}

// publicLimit: 30 permintaan per menit per IP untuk endpoint publik.
// Cukup longgar untuk bot normal, cukup ketat untuk mencegah brute force ID.
const publicLimitPerMinute = 30

func New(deps Deps) *Server {
	server := &Server{
		mux:      http.NewServeMux(),
		deps:     deps,
		renderer: NewRenderer(),
		limiter:  ratelimit.New(publicLimitPerMinute, time.Minute),
		// 8 percobaan per menit: cukup untuk salah ketik, tidak cukup untuk menebak.
		loginLimiter: ratelimit.New(8, time.Minute),

		trustedProxies: newTrustedProxies(deps.TrustedProxies),
	}
	server.routes()
	return server
}

// limitPublic membatasi endpoint publik per IP. Balas 429 saat habis.
func (server *Server) limitPublic(next http.HandlerFunc) http.HandlerFunc {
	return func(response http.ResponseWriter, request *http.Request) {
		key := server.clientIP(request)
		if !server.limiter.Allow(key) {
			response.Header().Set("Retry-After", "60")
			writeError(response, http.StatusTooManyRequests, "terlalu banyak permintaan, coba lagi sebentar")
			return
		}
		next.ServeHTTP(response, request)
	}
}

// limitLogin membatasi percobaan login admin per alamat IP.
// Sengaja lebih ketat daripada endpoint publik: 8 kali per menit.
func (server *Server) limitLogin(next http.HandlerFunc) http.HandlerFunc {
	return func(response http.ResponseWriter, request *http.Request) {
		key := server.clientIP(request)
		if !server.loginLimiter.Allow(key) {
			response.Header().Set("Retry-After", "60")
			writeError(response, http.StatusTooManyRequests,
				"terlalu banyak percobaan login, tunggu satu menit")
			return
		}
		next.ServeHTTP(response, request)
	}
}

// clientIP mengambil IP pemanggil.
// Header X-Forwarded-For / CF-Connecting-IP HANYA dipercaya kalau
// koneksi datang dari reverse proxy yang terdaftar. Tanpa ini,
// penyerang bisa memalsukan IP dan melewati rate limit sepenuhnya.
func (server *Server) clientIP(request *http.Request) string {
	remote := hostOnly(request.RemoteAddr)

	if server.trustedProxies.contains(remote) {
		if forwarded := strings.TrimSpace(request.Header.Get("CF-Connecting-IP")); forwarded != "" {
			return forwarded
		}
		if forwarded := request.Header.Get("X-Forwarded-For"); forwarded != "" {
			if comma := strings.Index(forwarded, ","); comma > 0 {
				return strings.TrimSpace(forwarded[:comma])
			}
			return strings.TrimSpace(forwarded)
		}
	}
	return remote
}

// hostOnly membuang port dari RemoteAddr.
func hostOnly(address string) string {
	if colon := strings.LastIndex(address, ":"); colon > 0 {
		return address[:colon]
	}
	return address
}

// SetNotifier memasang pengirim notifikasi setelah server dibuat.
// Dipisah karena notifier membaca setting lewat server itu sendiri.
func (server *Server) SetNotifier(notifier NotifyStore) {
	server.deps.Notify = notifier
}

func (server *Server) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	// Header pengaman + penjagaan origin (CSRF) untuk semua permintaan.
	sameOriginOnly(securityHeaders(server.mux)).ServeHTTP(response, request)
}

func (server *Server) routes() {
	mux := server.mux

	// Publik
	mux.HandleFunc("GET /healthz", server.healthDetail)
	mux.HandleFunc("GET /pay/{id}", server.checkout)
	mux.HandleFunc("GET /api/topup/{id}", server.topupStatus)
	mux.HandleFunc("POST /v1/identity/resolve", server.limitPublic(server.identityResolve))
	mux.HandleFunc("POST /v1/identity/link/start", server.limitPublic(server.identityLinkStart))
	mux.HandleFunc("POST /v1/identity/link/confirm", server.limitPublic(server.identityLinkConfirm))
	mux.HandleFunc("GET /v1/public/{id}/identities", server.limitPublic(server.identityList))
	mux.HandleFunc("GET /v1/public/{id}", server.limitPublic(server.publicWallet))
	mux.HandleFunc("POST /v1/public/{id}/topup", server.limitPublic(server.publicTopup))
	mux.HandleFunc("POST /webhook/paypan", server.paypanWebhook)
	mux.HandleFunc("GET /masuk", server.limitPublic(server.portalLogin))
	mux.HandleFunc("POST /masuk", server.limitPublic(server.portalStart))
	mux.HandleFunc("POST /masuk/cek", server.limitPublic(server.portalCheck))
	mux.HandleFunc("GET /admin/login", server.loginPage)
	mux.HandleFunc("POST /admin/login", server.limitLogin(server.loginSubmit))
	mux.HandleFunc("GET /admin/logout", server.logout)

	// Panel kendali (butuh sesi admin)
	mux.HandleFunc("GET /", server.root)
	mux.HandleFunc("GET /admin", server.requireAdmin(server.adminHome))
	mux.HandleFunc("GET /admin/topup", server.requireAdmin(server.adminTopup))
	mux.HandleFunc("POST /admin/topup", server.requireAdmin(server.adminTopupCreate))
	mux.HandleFunc("POST /admin/topup/{id}/paid", server.requireAdmin(server.adminTopupPaid))
	mux.HandleFunc("POST /admin/topup/{id}/sync", server.requireAdmin(server.paypanReconcile))
	mux.HandleFunc("GET /admin/wallets", server.requireAdmin(server.adminWallets))
	mux.HandleFunc("POST /admin/wallets", server.requireAdmin(server.adminWalletsCreate))
	mux.HandleFunc("POST /admin/wallets/{id}/status", server.requireAdmin(server.adminWalletsStatus))
	mux.HandleFunc("GET /admin/ledger", server.requireAdmin(server.adminLedger))
	mux.HandleFunc("GET /admin/audit", server.requireAdmin(server.adminAudit))
	mux.HandleFunc("GET /admin/laporan", server.requireAdmin(server.adminReport))
	mux.HandleFunc("POST /admin/wallets/adjust", server.requireAdmin(server.adminAdjust))
	mux.HandleFunc("GET /admin/export", server.requireAdmin(server.adminExport))
	mux.HandleFunc("POST /admin/export", server.requireAdmin(server.adminExportRun))
	mux.HandleFunc("GET /admin/export/download", server.requireAdmin(server.adminExportDownload))
	mux.HandleFunc("GET /admin/backup", server.requireAdmin(server.adminBackup))
	mux.HandleFunc("POST /admin/backup", server.requireAdmin(server.adminBackupRun))
	mux.HandleFunc("GET /admin/keys", server.requireAdmin(server.adminKeys))
	mux.HandleFunc("POST /admin/keys", server.requireAdmin(server.adminKeysCreate))
	mux.HandleFunc("POST /admin/keys/{product}/revoke", server.requireAdmin(server.adminKeysRevoke))
	mux.HandleFunc("GET /admin/config", server.requireAdmin(server.adminConfig))
	mux.HandleFunc("POST /admin/config", server.requireAdmin(server.adminConfigSave))
	mux.HandleFunc("POST /admin/notify", server.requireAdmin(server.adminNotifySave))
	mux.HandleFunc("POST /admin/notify/test", server.requireAdmin(server.adminNotifyTest))

	// API produk (butuh product key)
	mux.HandleFunc("POST /v1/holds", server.requireProductAuth(server.hold))
	mux.HandleFunc("POST /v1/holds/{id}/capture", server.requireProductAuth(server.capture))
	mux.HandleFunc("POST /v1/holds/{id}/release", server.requireProductAuth(server.release))
	mux.HandleFunc("GET /v1/wallets/{id}", server.requireProductAuth(server.walletBalance))
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
		if errors.Is(err, wallet.ErrWalletBlocked) {
			writeError(response, http.StatusForbidden, "wallet diblokir")
			return
		}
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

// decodeJSON membaca body JSON dengan batas 1 MiB dan menolak field asing.
func decodeJSON(response http.ResponseWriter, request *http.Request, target any) error {
	decoder := json.NewDecoder(http.MaxBytesReader(response, request.Body, 1<<20))
	decoder.DisallowUnknownFields()
	return decoder.Decode(target)
}
