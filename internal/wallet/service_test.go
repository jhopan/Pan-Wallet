package wallet

import "testing"

func TestCreateHoldRequestValidate(t *testing.T) {
	request := CreateHoldRequest{
		WalletUserID:   "11111111-1111-1111-1111-111111111111",
		Product:        "agenpulsa",
		ReferenceID:    "AP-1001",
		Amount:         25_000,
		IdempotencyKey: "hold:agenpulsa:AP-1001",
	}

	if err := request.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestCreateHoldRequestRejectsMissingMoneyIdentity(t *testing.T) {
	request := CreateHoldRequest{Product: "agenpulsa", ReferenceID: "AP-1001", Amount: 1}
	if err := request.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want invalid request")
	}
}

func TestTransitionRequestRequiresProductScope(t *testing.T) {
	request := TransitionRequest{
		HoldID:         "11111111-1111-1111-1111-111111111111",
		IdempotencyKey: "capture:agenpulsa:AP-1001",
	}
	if err := request.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want invalid request")
	}
}
