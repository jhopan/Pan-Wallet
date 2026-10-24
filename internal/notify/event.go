package notify

import "time"

// TopupEvent adalah satu top-up yang baru lunas.
// Dipakai untuk notifikasi admin, webhook bot lain, dan pesan testimoni.
type TopupEvent struct {
	PublicID string
	// Requested adalah nominal yang diminta pelanggan (mis. 25.000).
	// Ini yang ditampilkan di testimoni.
	Requested int64
	// Amount adalah total unik yang benar-benar masuk ke saldo
	// (nominal + kode unik PayPan, mis. 25.004).
	Amount    int64
	Balance   int64
	PaidAt    int64
	InvoiceID string
}

// Payload menyusun JSON untuk penerima webhook.
func (event TopupEvent) Payload() map[string]any {
	return map[string]any{
		"event":      "topup.paid",
		"public_id":  event.PublicID,
		"requested":  event.Requested,
		"amount":     event.Amount,
		"balance":    event.Balance,
		"paid_at":    event.PaidAt,
		"invoice_id": event.InvoiceID,
	}
}

// PaidTime mengembalikan waktu lunas dalam zona Asia/Jakarta.
func (event TopupEvent) PaidTime() time.Time {
	location, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		location = time.FixedZone("WIB", 7*3600)
	}
	return time.Unix(event.PaidAt, 0).In(location)
}
