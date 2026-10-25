// Package paypan adalah client untuk PayPan, payment gateway QRIS yang sudah
// berjalan. PanWallet TIDAK membuat QR sendiri: dia meminta invoice ke PayPan,
// menampilkan pay_url, lalu menerima webhook order.paid untuk mengkredit saldo.
//
// Kontrak: POST /api/invoice, GET /api/invoice/{id}, POST /api/invoice/{id}/cancel
// Semua respons berbentuk {"ok":bool,"data":{...}} atau {"ok":false,"error":"..."}.
package paypan

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Invoice adalah bentuk data invoice dari PayPan.
type Invoice struct {
	ID        string `json:"id"`
	Price     int64  `json:"price"`
	Code      int    `json:"code"`
	Total     int64  `json:"total"`
	Status    string `json:"status"`
	ExpiresAt int64  `json:"expires_at"`
	PaidAt    int64  `json:"paid_at"`
	PayURL    string `json:"pay_url"`
	QRURL     string `json:"qr_url"`
	QR        string `json:"qr"`
}

// Client memanggil API PayPan.
type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

// NewClient membuat client dengan base URL dan token scope "order".
func NewClient(baseURL, token string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		token:   token,
		http:    &http.Client{Timeout: 15 * time.Second},
	}
}

// CreateInvoice meminta invoice QRIS baru ke PayPan.
func (client *Client) CreateInvoice(ctx context.Context, price int64, label string) (Invoice, error) {
	payload := map[string]any{"price": price}
	if label != "" {
		payload["label"] = label
	}
	var invoice Invoice
	if err := client.call(ctx, http.MethodPost, "/api/invoice", payload, &invoice); err != nil {
		return Invoice{}, err
	}
	return invoice, nil
}

// GetInvoice membaca status invoice (pending|paid|expired|refunded).
func (client *Client) GetInvoice(ctx context.Context, invoiceID string) (Invoice, error) {
	var invoice Invoice
	if err := client.call(ctx, http.MethodGet, "/api/invoice/"+invoiceID, nil, &invoice); err != nil {
		return Invoice{}, err
	}
	return invoice, nil
}

// CancelInvoice membatalkan invoice yang masih pending.
func (client *Client) CancelInvoice(ctx context.Context, invoiceID string) error {
	return client.call(ctx, http.MethodPost, "/api/invoice/"+invoiceID+"/cancel", nil, nil)
}

// call menjalankan satu request dan membuka pembungkus {"ok","data"}.
func (client *Client) call(ctx context.Context, method, path string, payload any, target any) error {
	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("encode body: %w", err)
		}
		body = bytes.NewReader(encoded)
	}

	request, err := http.NewRequestWithContext(ctx, method, client.baseURL+path, body)
	if err != nil {
		return fmt.Errorf("buat request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+client.token)
	if payload != nil {
		request.Header.Set("Content-Type", "application/json")
	}

	response, err := client.http.Do(request)
	if err != nil {
		return fmt.Errorf("hubungi PayPan: %w", err)
	}
	defer response.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("baca respons PayPan: %w", err)
	}

	var envelope struct {
		OK    bool            `json:"ok"`
		Error string          `json:"error"`
		Data  json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return fmt.Errorf("respons PayPan bukan JSON (HTTP %d)", response.StatusCode)
	}
	if !envelope.OK {
		if envelope.Error != "" {
			return fmt.Errorf("%s", envelope.Error)
		}
		return fmt.Errorf("PayPan menolak permintaan (HTTP %d)", response.StatusCode)
	}
	if target != nil && len(envelope.Data) > 0 {
		if err := json.Unmarshal(envelope.Data, target); err != nil {
			return fmt.Errorf("baca data PayPan: %w", err)
		}
	}
	return nil
}

// Sign menghitung signature HMAC-SHA256 hex dari body mentah.
func Sign(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

// VerifyWebhook memastikan request benar dari PayPan (constant-time compare).
func VerifyWebhook(body []byte, signature, secret string) bool {
	if signature == "" || secret == "" {
		return false
	}
	expected := Sign(body, secret)
	return hmac.Equal([]byte(expected), []byte(strings.ToLower(signature)))
}

// WebhookEvent adalah isi webhook order.paid.
type WebhookEvent struct {
	Event string `json:"event"`
	Order struct {
		ID     string `json:"id"`
		Price  int64  `json:"price"`
		Code   int    `json:"code"`
		Total  int64  `json:"total"`
		PaidAt int64  `json:"paid_at"`
	} `json:"order"`
}

// ParseWebhook membaca body webhook PayPan.
func ParseWebhook(body []byte) (WebhookEvent, error) {
	var event WebhookEvent
	if err := json.Unmarshal(body, &event); err != nil {
		return WebhookEvent{}, fmt.Errorf("parse webhook: %w", err)
	}
	return event, nil
}

// BaseURL mengembalikan base URL PayPan (untuk menyusun pay_url absolut).
func (client *Client) BaseURL() string { return client.baseURL }
