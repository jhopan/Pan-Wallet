package httpapi

import (
	"strings"
	"testing"

	"github.com/jhopan/Pan-Wallet/internal/notify"
)

// Testimoni publik tidak boleh memuat data sensitif apa pun:
// tanpa total saldo, tanpa ID pelanggan. Cukup bukti top-up berhasil.
func TestTopupMessageCompactHidesBalance(t *testing.T) {
	event := notify.TopupEvent{
		PublicID:  "ABC12345",
		Requested: 25000,
		Amount:    25001,
		Balance:   125001,
		PaidAt:    1790000000,
	}
	text := TopupMessageCompact(event)

	if strings.Contains(text, "125.001") {
		t.Fatalf("saldo total tidak boleh muncul:\n%s", text)
	}
	if strings.Contains(text, "Saldo sekarang") {
		t.Fatalf("baris saldo harus hilang:\n%s", text)
	}
	// ID pelanggan adalah identitas pelanggan, jangan dipajang di topic publik.
	if strings.Contains(text, "ABC12345") {
		t.Fatalf("ID pelanggan tidak boleh muncul:\n%s", text)
	}
	if strings.Contains(text, "ID Pelanggan") {
		t.Fatalf("label ID Pelanggan harus hilang:\n%s", text)
	}
	// nominal top-up tetap ada
	if !strings.Contains(text, "25.000") {
		t.Fatalf("nominal top-up harus tetap ada:\n%s", text)
	}
	if !strings.Contains(text, "WIB") {
		t.Fatalf("waktu harus tetap ada:\n%s", text)
	}
	if !strings.Contains(text, "berhasil") {
		t.Fatalf("harus menyatakan berhasil:\n%s", text)
	}
}

// Mode lengkap tetap menyertakan saldo (dipakai notifikasi admin pribadi).
func TestTopupMessageFullStillShowsBalance(t *testing.T) {
	event := notify.TopupEvent{
		PublicID:  "ABC12345",
		Requested: 25000,
		Amount:    25001,
		Balance:   125001,
		PaidAt:    1790000000,
	}
	text := TopupMessage(event)

	if !strings.Contains(text, "125.001") {
		t.Fatalf("mode lengkap harus menyertakan saldo:\n%s", text)
	}
}
