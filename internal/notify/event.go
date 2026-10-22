package notify

import "time"

// TopupEvent adalah satu top-up yang baru lunas.
// Dipakai untuk notifikasi admin, webhook bot lain, dan pesan testimoni.
type TopupEvent struct {
	PublicID  string
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
