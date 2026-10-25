package topup

import (
	"fmt"
	"strings"
	"time"
)

// MinPrice adalah nominal top-up terendah. Ditetapkan Rp25.000 supaya
// biaya QRIS dan kode unik tetap wajar dibanding nominalnya.
const MinPrice int64 = 25_000

// MaxPrice adalah batas atas PayPan (kode unik hanya 3 digit).
const MaxPrice int64 = 9_000_000

// Invoice mewakili satu permintaan top-up yang sudah dibuat di PayPan.
type Invoice struct {
	ID            string
	WalletUserID  string
	PaypanInvoice string
	Price         int64
	Code          int
	Total         int64
	Status        string
	Source        string
	ExpiresAt     time.Time
	PayURL        string

	// BalanceAfter adalah saldo wallet setelah top-up ini dikredit.
	// Diisi oleh store supaya notifikasi melaporkan saldo yang benar,
	// bukan menebak dari nominal invoice.
	BalanceAfter int64
	// FreshlyPaid true hanya pada pelunasan pertama. Webhook ganda dari
	// PayPan tidak boleh mengirim notifikasi atau testimoni dua kali.
	FreshlyPaid bool
}

func (invoice Invoice) Validate() error {
	if invoice.Price < MinPrice {
		return fmt.Errorf("nominal top-up minimal Rp25.000")
	}
	if invoice.Price > MaxPrice {
		return fmt.Errorf("nominal top-up maksimal Rp%d", MaxPrice)
	}
	if invoice.Total < invoice.Price {
		return fmt.Errorf("total tidak boleh lebih kecil dari nominal")
	}
	return nil
}

// CanTransition menjaga status invoice hanya bergerak dari pending ke satu
// keadaan akhir. Paid bersifat final; tidak ada kredit ganda.
func CanTransition(from, to string) bool {
	if from != "pending" {
		return false
	}
	switch to {
	case "paid", "expired", "cancelled":
		return true
	}
	return false
}

// NormalizeSource membersihkan nama sumber pembayaran.
func NormalizeSource(source string) string {
	return strings.TrimSpace(source)
}

// PaypanToLocalStatus memetakan status PayPan ke status lokal PanWallet.
func PaypanToLocalStatus(status string) string {
	switch status {
	case "paid":
		return "paid"
	case "expired":
		return "expired"
	case "refunded":
		return "cancelled"
	default:
		return "pending"
	}
}
