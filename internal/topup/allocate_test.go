package topup

import "testing"

func TestAllocatePicksLowestFreeCode(t *testing.T) {
	// Belum ada yang terpakai: kode pertama untuk harga 10000 harus 001.
	if code := AllocateCode(10000, nil, nil); code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
}

func TestAllocateSkipsCodeUsedInSamePrice(t *testing.T) {
	// Harga 10000 sudah pakai kode 1 dan 2 -> berikutnya 3.
	used := []int{1, 2}
	if code := AllocateCode(10000, used, nil); code != 3 {
		t.Fatalf("code = %d, want 3", code)
	}
}

func TestAllocateSkipsTotalCollisionAcrossLevels(t *testing.T) {
	// Harga 1100, kode 1 -> total 1101. Tapi total 1101 sudah diklaim
	// order harga lain -> harus geser ke kode 2 (total 1102).
	claimed := []int64{1101}
	if code := AllocateCode(1100, nil, claimed); code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
}

func TestAllocateReturnsZeroWhenFull(t *testing.T) {
	claimed := make([]int64, 0, 999)
	for amount := int64(1); amount <= 999; amount++ {
		claimed = append(claimed, 1000+amount)
	}
	if code := AllocateCode(1000, nil, claimed); code != 0 {
		t.Fatalf("code = %d, want 0 saat penuh", code)
	}
}

func TestTotalIsPricePlusCode(t *testing.T) {
	if total := Total(25000, 7); total != 25007 {
		t.Fatalf("total = %d, want 25007", total)
	}
}
