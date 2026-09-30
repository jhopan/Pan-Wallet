package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jhopan/Pan-Wallet/internal/auth"
	"github.com/jhopan/Pan-Wallet/internal/wallet"
)

type ProductAuthenticator interface {
	Authenticate(context.Context, string) (string, error)
}

type HoldStore interface {
	CreateHold(context.Context, wallet.CreateHoldRequest, time.Time) (wallet.Hold, error)
}

type Server struct {
	mux           *http.ServeMux
	authenticator ProductAuthenticator
	holds         HoldStore
}

func New(authenticator ProductAuthenticator, holds ...HoldStore) *Server {
	mux := http.NewServeMux()
	server := &Server{mux: mux, authenticator: authenticator}
	if len(holds) == 1 {
		server.holds = holds[0]
	}
	mux.HandleFunc("GET /healthz", health)
	mux.HandleFunc("POST /v1/holds", server.requireProductAuth(server.hold))
	return server
}

func (server *Server) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	server.mux.ServeHTTP(response, request)
}

func (server *Server) requireProductAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(response http.ResponseWriter, request *http.Request) {
		if server.authenticator == nil {
			writeError(response, http.StatusServiceUnavailable, "product API belum siap")
			return
		}
		token, err := bearerToken(request)
		if err != nil {
			writeError(response, http.StatusUnauthorized, "Authorization Bearer wajib diisi")
			return
		}
		product, err := server.authenticator.Authenticate(request.Context(), token)
		if errors.Is(err, auth.ErrUnauthorized) {
			writeError(response, http.StatusUnauthorized, "product API key tidak valid")
			return
		}
		if err != nil {
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
	if server.holds == nil {
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
	hold, err := server.holds.CreateHold(request.Context(), wallet.CreateHoldRequest{
		WalletUserID: body.WalletUserID, Product: product, ReferenceID: body.ReferenceID,
		Amount: body.Amount, IdempotencyKey: request.Header.Get("Idempotency-Key"),
	}, body.ExpiresAt)
	if err != nil {
		writeError(response, http.StatusUnprocessableEntity, err.Error())
		return
	}
	response.Header().Set("Content-Type", "application/json; charset=utf-8")
	response.WriteHeader(http.StatusCreated)
	json.NewEncoder(response).Encode(hold)
}

func health(response http.ResponseWriter, _ *http.Request) {
	response.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(response).Encode(map[string]string{"status": "ok"})
}

func writeError(response http.ResponseWriter, status int, message string) {
	response.Header().Set("Content-Type", "application/json; charset=utf-8")
	response.WriteHeader(status)
	json.NewEncoder(response).Encode(map[string]string{"error": message})
}
