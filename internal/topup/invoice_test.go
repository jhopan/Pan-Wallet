package topup

import "testing"

func TestInvoiceStatusTransitionAllowed(t *testing.T) {
	cases := []struct {
		from, to string
		want     bool
	}{
		{"pending", "paid", true},
		{"pending", "expired", true},
		{"pending", "cancelled", true},
		{"paid", "paid", false},
		{"expired", "paid", false},
		{"cancelled", "paid", false},
	}
	for _, c := range cases {
		if got := CanTransition(c.from, c.to); got != c.want {
			t.Errorf("CanTransition(%s,%s) = %v, want %v", c.from, c.to, got, c.want)
		}
	}
}

func TestInvoiceValidateRejectsSmallPrice(t *testing.T) {
	invoice := Invoice{Price: 500, Total: 501}
	if err := invoice.Validate(); err == nil {
		t.Fatal("harga di bawah 1000 harus ditolak")
	}
}

func TestInvoiceValidateAcceptsNormalTopup(t *testing.T) {
	invoice := Invoice{Price: 25000, Code: 3, Total: 25003}
	if err := invoice.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}
