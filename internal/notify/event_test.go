package notify

import (
	"testing"
)

// TopupEvent adalah data satu top-up yang sudah lunas.
func TestTopupEventCarriesRealBalance(t *testing.T) {
	event := TopupEvent{
		PublicID:  "ABC12345",
		Amount:    25004,
		Balance:   125004,
		PaidAt:    1790000000,
		InvoiceID: "pp_abc",
	}

	if event.Balance != 125004 {
		t.Fatalf("balance = %d", event.Balance)
	}
	if event.Amount != 25004 {
		t.Fatalf("amount = %d", event.Amount)
	}
}

func TestTopupEventPayloadHasTimestamp(t *testing.T) {
	event := TopupEvent{PublicID: "ABC12345", Amount: 25004, Balance: 125004, PaidAt: 1790000000}
	payload := event.Payload()

	if payload["event"] != "topup.paid" {
		t.Fatalf("event = %v", payload["event"])
	}
	if payload["paid_at"] != int64(1790000000) {
		t.Fatalf("paid_at = %v", payload["paid_at"])
	}
	if payload["balance"] != int64(125004) {
		t.Fatalf("balance = %v", payload["balance"])
	}
}
