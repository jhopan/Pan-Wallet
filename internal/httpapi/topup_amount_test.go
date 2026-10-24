package httpapi

import (
	"strings"
	"testing"

	"github.com/jhopan/Pan-Wallet/internal/notify"
)

// Testimoni harus menyebut nominal yang diminta pelanggan (25.000),
// bukan total unik yang sudah ditambah kode PayPan.
func TestTopupMessageReportsRequestedAmount(t *testing.T) {
	event := notify.TopupEvent{
		PublicID:  "ABC12345",
		Requested: 25000,
		Amount:    25004,
		Balance:   125004,
		PaidAt:    1790000000,
	}
	text := TopupMessage(event)

	if !strings.Contains(text, "Rp 25.000") {
		t.Fatalf("harus menyebut nominal permintaan Rp 25.000:\n%s", text)
	}
	// baris "Nominal:" harus angka permintaan, bukan total unik
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(line, "Nominal:") && strings.Contains(line, "25.004") {
			t.Fatalf("baris Nominal tidak boleh pakai total unik: %s", line)
		}
	}
	if !strings.Contains(text, "Rp 125.004") {
		t.Fatalf("saldo harus tetap saldo asli:\n%s", text)
	}
}

// Saldo yang dikredit wajib persis total unik, tidak boleh kurang 1 rupiah.
func TestTopupMessageShowsExactCreditedAmount(t *testing.T) {
	event := notify.TopupEvent{
		PublicID:  "ABC12345",
		Requested: 25000,
		Amount:    25004,
		Balance:   25004,
		PaidAt:    1790000000,
	}
	text := TopupMessage(event)

	// saldo masuk harus menyebut angka persis 25.004
	if !strings.Contains(text, "25.004") {
		t.Fatalf("saldo masuk harus persis total unik:\n%s", text)
	}
}

// Tanpa Requested (data lama), pesan tetap masuk akal.
func TestTopupMessageFallsBackWhenRequestedEmpty(t *testing.T) {
	event := notify.TopupEvent{PublicID: "ABC12345", Amount: 25004, Balance: 25004, PaidAt: 1790000000}
	text := TopupMessage(event)
	if !strings.Contains(text, "25.004") {
		t.Fatalf("fallback harus pakai Amount:\n%s", text)
	}
}
