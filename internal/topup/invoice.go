package topup

import (
	"fmt"
	"strings"
	"time"
)

// MinPrice adalah nominal top-up terendah. Di bawah ini kode 001-999 terlalu
// besar porsinya terhadap total dan rawan bentrok nominal.
const MinPrice int64 = 1000

// Invoice mewakili satu permintaan top-up QRIS.
type Invoice struct {
	ID           string
	WalletUserID string
	Price        int64
	Code         int
	Total        int64
	Status       string
	Source       string
	ExpiresAt    time.Time
}

func (invoice Invoice) Validate() error {
	if invoice.Price < MinPrice {
		return fmt.Errorf("nominal top-up minimal Rp%d", MinPrice)
	}
	if invoice.Code < 1 || invoice.Code > 999 {
		return fmt.Errorf("kode invoice harus 1-999")
	}
	if invoice.Total != invoice.Price+int64(invoice.Code) {
		return fmt.Errorf("total harus sama dengan harga + kode")
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

// NormalizeSource membersihkan nama sumber pembayaran dari notifikasi.
func NormalizeSource(source string) string {
	return strings.TrimSpace(source)
}
