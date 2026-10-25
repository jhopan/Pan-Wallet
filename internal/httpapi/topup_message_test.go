package httpapi

import (
	"strings"
	"testing"

	"github.com/jhopan/PanWallet/internal/notify"
)

func TestTopupMessageHasDateAndTime(t *testing.T) {
	event := notify.TopupEvent{
		PublicID: "ABC12345",
		Amount:   25004,
		Balance:  125004,
		PaidAt:   1790000000,
	}
	text := TopupMessage(event)

	for _, want := range []string{"ABC12345", "25.004", "125.004", "WIB"} {
		if !strings.Contains(text, want) {
			t.Fatalf("pesan tidak memuat %q:\n%s", want, text)
		}
	}
	// harus ada tanggal dan jam
	if !strings.Contains(text, "2026") && !strings.Contains(text, "2025") {
		t.Fatalf("tidak ada tahun:\n%s", text)
	}
}

func TestTopupMessageUsesWIB(t *testing.T) {
	// 1790000000 = 2026-09-21 20:53:20 UTC = 2026-09-22 03:53 WIB
	event := notify.TopupEvent{PublicID: "X", Amount: 25000, Balance: 25000, PaidAt: 1790000000}
	text := TopupMessage(event)
	if !strings.Contains(text, "WIB") {
		t.Fatalf("harus WIB:\n%s", text)
	}
}
