package topup

import "testing"

func TestMinPriceIs25Thousand(t *testing.T) {
	if MinPrice != 25_000 {
		t.Fatalf("MinPrice = %d, want 25000", MinPrice)
	}
}

func TestValidateRejectsBelow25Thousand(t *testing.T) {
	invoice := Invoice{Price: 10_000, Total: 10_001}
	if err := invoice.Validate(); err == nil {
		t.Fatal("Rp10.000 harus ditolak, minimal Rp25.000")
	}
}

func TestValidateAccepts25Thousand(t *testing.T) {
	invoice := Invoice{Price: 25_000, Total: 25_001}
	if err := invoice.Validate(); err != nil {
		t.Fatalf("Rp25.000 harus diterima: %v", err)
	}
}

func TestValidateAcceptsAbove(t *testing.T) {
	for _, price := range []int64{50_000, 100_000, 500_000} {
		invoice := Invoice{Price: price, Total: price + 1}
		if err := invoice.Validate(); err != nil {
			t.Fatalf("Rp%d harus diterima: %v", price, err)
		}
	}
}
