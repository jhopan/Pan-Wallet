package paypan

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// PingStatus adalah hasil pemeriksaan koneksi ke PayPan.
type PingStatus struct {
	OK        bool
	Message   string
	LatencyMS int64
	BaseURL   string
	Version   string
}

// Ping benar-benar menghubungi PayPan untuk memastikan gateway hidup,
// bukan sekadar memeriksa "string konfigurasi kosong atau tidak".
//
// Sebelumnya health hanya mengecek apakah base_url dan token terisi.
// Akibatnya status bisa "ok" padahal PayPan sudah mati — pelanggan yang
// mau top-up baru sadar saat gagal bayar.
func (client *Client) Ping(ctx context.Context) PingStatus {
	status := PingStatus{BaseURL: client.baseURL}
	start := time.Now()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, client.baseURL+"/", nil)
	if err != nil {
		status.Message = "alamat PayPan tidak valid"
		return status
	}
	if client.token != "" {
		request.Header.Set("Authorization", "Bearer "+client.token)
	}

	response, err := client.http.Do(request)
	if err != nil {
		status.Message = friendlyError(err)
		return status
	}
	defer response.Body.Close()

	status.LatencyMS = time.Since(start).Milliseconds()

	body := make([]byte, 0, 512)
	buffer := make([]byte, 512)
	for {
		n, readErr := response.Body.Read(buffer)
		body = append(body, buffer[:n]...)
		if readErr != nil {
			break
		}
		if len(body) > 8192 {
			break
		}
	}

	if response.StatusCode == http.StatusUnauthorized || response.StatusCode == http.StatusForbidden {
		status.Message = "token PayPan ditolak"
		return status
	}
	if response.StatusCode >= 300 {
		status.Message = fmt.Sprintf("PayPan membalas HTTP %d", response.StatusCode)
		return status
	}

	// PayPan menjawab di "/" dengan teks "Paypan payment gateway OK".
	// Kalau JSON (instalasi lain), hormati field ok.
	var envelope struct {
		OK      bool   `json:"ok"`
		Version string `json:"version"`
		Error   string `json:"error"`
	}
	if err := json.Unmarshal(body, &envelope); err == nil {
		if !envelope.OK {
			status.Message = "PayPan melaporkan masalah"
			if envelope.Error != "" {
				status.Message = "PayPan: " + envelope.Error
			}
			return status
		}
		status.OK = true
		status.Message = "terhubung"
		status.Version = envelope.Version
		return status
	}

	// Bukan JSON: pastikan isinya memang PayPan, bukan situs lain.
	text := strings.ToLower(string(body))
	if strings.Contains(text, "paypan") && strings.Contains(text, "ok") {
		status.OK = true
		status.Message = "terhubung"
		return status
	}

	status.Message = "alamat ini bukan PayPan"
	return status
}

// friendlyError menerjemahkan error jaringan menjadi kalimat yang berguna.
func friendlyError(err error) string {
	text := err.Error()
	switch {
	case strings.Contains(text, "connection refused"):
		return "PayPan tidak menerima koneksi (mati atau port salah)"
	case strings.Contains(text, "no such host"):
		return "alamat PayPan tidak ditemukan"
	case strings.Contains(text, "timeout"), strings.Contains(text, "deadline exceeded"):
		return "PayPan tidak merespons (timeout)"
	case strings.Contains(text, "certificate"):
		return "sertifikat HTTPS PayPan bermasalah"
	default:
		return "gagal menghubungi PayPan"
	}
}
