// Package identity mengatur identitas pelanggan lintas channel.
//
// Aturan utama: TELEGRAM adalah identitas primer. Telegram ID diberikan oleh
// server Telegram, tidak bisa dipalsukan pelanggan, jadi aman dijadikan kunci.
// Channel lain (WhatsApp, email, web) DITAUTKAN ke wallet yang sama lewat kode
// yang dikonfirmasi di Telegram — bukan lewat kirim email.
package identity

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"regexp"
	"strings"
)

// Provider adalah channel tempat pelanggan datang.
const (
	ProviderTelegram = "telegram"
	ProviderWhatsapp = "whatsapp"
	ProviderEmail    = "email"
	ProviderWeb      = "web"
)

var (
	emailPattern    = regexp.MustCompile(`^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}$`)
	telegramPattern = regexp.MustCompile(`^[0-9]{5,20}$`)
	digitsPattern   = regexp.MustCompile(`[^0-9]`)
)

// Normalize membersihkan dan memvalidasi external_id sesuai provider.
func Normalize(provider, externalID string) (string, error) {
	value := strings.TrimSpace(externalID)

	switch provider {
	case ProviderTelegram:
		if !telegramPattern.MatchString(value) {
			return "", fmt.Errorf("telegram id harus berupa angka")
		}
		return value, nil

	case ProviderWhatsapp:
		// Buang spasi, tanda hubung, dan plus; sisakan angka.
		cleaned := digitsPattern.ReplaceAllString(value, "")
		// Normalisasi awalan Indonesia ke 62.
		if strings.HasPrefix(cleaned, "0") {
			cleaned = "62" + cleaned[1:]
		}
		if len(cleaned) < 10 || len(cleaned) > 15 {
			return "", fmt.Errorf("nomor whatsapp tidak valid")
		}
		return cleaned, nil

	case ProviderEmail:
		lower := strings.ToLower(value)
		if !emailPattern.MatchString(lower) {
			return "", fmt.Errorf("email tidak valid")
		}
		return lower, nil

	case ProviderWeb:
		if value == "" {
			return "", fmt.Errorf("id web wajib diisi")
		}
		return lower(value), nil
	}

	return "", fmt.Errorf("provider %q tidak dikenal", provider)
}

func lower(value string) string { return strings.ToLower(value) }

// GenerateCode membuat kode 6 angka untuk mengikat channel baru.
func GenerateCode() (string, error) {
	max := big.NewInt(1000000)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", fmt.Errorf("buat kode: %w", err)
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

// LinkInstruction adalah teks yang dikirim bot ke pelanggan.
func LinkInstruction(code string) string {
	return fmt.Sprintf(
		"Buka bot Telegram PanWallet, lalu kirim:\n\n/ikat %s\n\nKode berlaku 10 menit.",
		code,
	)
}

// IsPrimary menandai provider yang boleh dijadikan identitas utama.
func IsPrimary(provider string) bool {
	return provider == ProviderTelegram
}
