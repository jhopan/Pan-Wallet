package qris

import "testing"

// Payload statis asli JhopanStore (GoPay merchant). Dipakai sebagai fixture uji.
const staticFixture = "00020101021126610014COM.GO-JEK.WWW011893600914365190135102200000000000000021035190135102520303UMI51440014ID.CO.QRIS.WWW0215UMI102520210303UMI5204549953033605802ID5910J Store156006JAKARTA BARAT61051214062150703A0163043D2B"

func TestCRC16MatchesEmvcoCheckValue(t *testing.T) {
	// Vektor uji baku CRC-16/CCITT-FALSE: crc16("123456789") = 0x29B1.
	if got := crc16("123456789"); got != "29B1" {
		t.Fatalf("crc16 = %q, want 29B1", got)
	}
}

func TestBuildDynamicInjectsAmountAndRecomputesCRC(t *testing.T) {
	dynamic, err := BuildDynamic(staticFixture, 25000)
	if err != nil {
		t.Fatalf("BuildDynamic() error = %v", err)
	}

	if !contains(dynamic, "010212") {
		t.Fatal("payload harus jadi dynamic (010212)")
	}
	if contains(dynamic, "010211") {
		t.Fatal("penanda static 010211 harus hilang")
	}
	// Tag 54: panjang value "25000.00" adalah 8 -> "540825000.00".
	if !contains(dynamic, "540825000.00") {
		t.Fatal("tag 54 nominal 25000.00 tidak disisipkan")
	}
	// CRC harus valid: hitung ulang badan payload.
	body := dynamic[:len(dynamic)-4]
	if got := crc16(body); got != dynamic[len(dynamic)-4:] {
		t.Fatalf("CRC akhir %q tidak cocok hitungan %q", dynamic[len(dynamic)-4:], got)
	}
}

func TestBuildDynamicRejectsForeignPayload(t *testing.T) {
	if _, err := BuildDynamic("bukan-qris", 1000); err == nil {
		t.Fatal("payload asing harus ditolak")
	}
}

func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && indexOf(haystack, needle) >= 0
}

func indexOf(haystack, needle string) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}
