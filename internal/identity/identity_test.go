package identity

import "testing"

func TestNormalizeEmail(t *testing.T) {
	got, err := Normalize(ProviderEmail, "  Budi@Gmail.COM ")
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}
	if got != "budi@gmail.com" {
		t.Fatalf("email = %q, want budi@gmail.com", got)
	}
}

func TestNormalizeRejectsBadEmail(t *testing.T) {
	for _, bad := range []string{"", "bukan-email", "a@b", "a@@b.com"} {
		if _, err := Normalize(ProviderEmail, bad); err == nil {
			t.Fatalf("email %q seharusnya ditolak", bad)
		}
	}
}

func TestNormalizeTelegramID(t *testing.T) {
	got, err := Normalize(ProviderTelegram, " 123456789 ")
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}
	if got != "123456789" {
		t.Fatalf("telegram = %q", got)
	}
	if _, err := Normalize(ProviderTelegram, "abc"); err == nil {
		t.Fatal("telegram id non-angka harus ditolak")
	}
}

func TestNormalizeWhatsappStripsFormatting(t *testing.T) {
	got, err := Normalize(ProviderWhatsapp, "+62 812-3456-789")
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}
	if got != "628123456789" {
		t.Fatalf("whatsapp = %q, want 628123456789", got)
	}
}

func TestNormalizeRejectsUnknownProvider(t *testing.T) {
	if _, err := Normalize("fax", "123"); err == nil {
		t.Fatal("provider asing harus ditolak")
	}
}

func TestGenerateCodeIsSixDigits(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 200; i++ {
		code, err := GenerateCode()
		if err != nil {
			t.Fatalf("GenerateCode: %v", err)
		}
		if len(code) != 6 {
			t.Fatalf("code = %q, panjang %d, want 6", code, len(code))
		}
		for _, ch := range code {
			if ch < '0' || ch > '9' {
				t.Fatalf("code %q mengandung non-angka", code)
			}
		}
		seen[code] = true
	}
	if len(seen) < 150 {
		t.Fatalf("kode terlalu sedikit variasi: %d unik dari 200", len(seen))
	}
}

func TestLinkInstructionMentionsTelegram(t *testing.T) {
	text := LinkInstruction("482913")
	if text == "" {
		t.Fatal("instruksi tidak boleh kosong")
	}
	for _, want := range []string{"482913", "Telegram", "/ikat"} {
		if !contains(text, want) {
			t.Fatalf("instruksi tidak memuat %q: %s", want, text)
		}
	}
}

func contains(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
