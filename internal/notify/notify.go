// Package notify mengirim notifikasi keluar saat ada kejadian penting.
//
// Dua jalur, keduanya memakai curl-friendly HTTP:
//   - Telegram: push langsung ke admin
//   - Webhook: POST JSON ke URL mana pun (bisa dipakai bot lain)
package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Telegram mengirim pesan ke satu atau beberapa chat.
type Telegram struct {
	token   string
	chats   []string
	baseURL string
	http    *http.Client
}

// NewTelegram membuat pengirim Telegram. chats boleh dipisah koma.
func NewTelegram(token, chats, baseURL string) *Telegram {
	if baseURL == "" {
		baseURL = "https://api.telegram.org"
	}
	var list []string
	for _, chat := range strings.Split(chats, ",") {
		if trimmed := strings.TrimSpace(chat); trimmed != "" {
			list = append(list, trimmed)
		}
	}
	return &Telegram{
		token:   strings.TrimSpace(token),
		chats:   list,
		baseURL: strings.TrimRight(baseURL, "/"),
		http:    &http.Client{Timeout: 10 * time.Second},
	}
}

// Enabled melaporkan apakah notifikasi Telegram siap dipakai.
func (telegram *Telegram) Enabled() bool {
	return telegram.token != "" && len(telegram.chats) > 0
}

// Send mengirim pesan ke semua chat. Tanpa konfigurasi = tidak melakukan apa pun.
func (telegram *Telegram) Send(ctx context.Context, text string) error {
	if !telegram.Enabled() {
		return nil
	}

	var firstErr error
	for _, chat := range telegram.chats {
		form := url.Values{}
		form.Set("chat_id", chat)
		form.Set("text", text)
		form.Set("disable_web_page_preview", "true")

		endpoint := fmt.Sprintf("%s/bot%s/sendMessage", telegram.baseURL, telegram.token)
		request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		response, err := telegram.http.Do(request)
		if err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("kirim telegram: %w", err)
			}
			continue
		}
		body := make([]byte, 0, 512)
		buffer := make([]byte, 512)
		for {
			n, readErr := response.Body.Read(buffer)
			body = append(body, buffer[:n]...)
			if readErr != nil {
				break
			}
		}
		response.Body.Close()

		if response.StatusCode >= 300 {
			var failure struct {
				Description string `json:"description"`
			}
			json.Unmarshal(body, &failure)
			if firstErr == nil {
				firstErr = fmt.Errorf("telegram menolak: %s", failure.Description)
			}
		}
	}
	return firstErr
}

// WebhookBody menyusun payload JSON untuk penerima webhook.
func WebhookBody(event string, data map[string]any) []byte {
	payload := map[string]any{"event": event}
	for key, value := range data {
		payload[key] = value
	}
	encoded, _ := json.Marshal(payload)
	return encoded
}

// PostWebhook mengirim JSON ke URL penerima (bot lain, dashboard, dll).
func PostWebhook(ctx context.Context, targetURL string, body []byte) error {
	if strings.TrimSpace(targetURL) == "" {
		return nil
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("buat request webhook: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("kirim webhook: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode >= 300 {
		return fmt.Errorf("webhook menolak (HTTP %d)", response.StatusCode)
	}
	return nil
}
