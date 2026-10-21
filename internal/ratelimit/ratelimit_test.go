package ratelimit

import (
	"testing"
	"time"
)

func TestAllowsUpToLimit(t *testing.T) {
	limiter := New(3, time.Minute)

	for i := 0; i < 3; i++ {
		if !limiter.Allow("1.2.3.4") {
			t.Fatalf("permintaan ke-%d harus diizinkan", i+1)
		}
	}
	if limiter.Allow("1.2.3.4") {
		t.Fatal("permintaan ke-4 harus ditolak")
	}
}

func TestDifferentKeysIndependent(t *testing.T) {
	limiter := New(2, time.Minute)

	limiter.Allow("a")
	limiter.Allow("a")
	if limiter.Allow("a") {
		t.Fatal("a harus kena limit")
	}
	if !limiter.Allow("b") {
		t.Fatal("b tidak boleh terpengaruh limit a")
	}
}

func TestWindowExpiry(t *testing.T) {
	limiter := New(1, 50*time.Millisecond)

	if !limiter.Allow("x") {
		t.Fatal("permintaan pertama harus lolos")
	}
	if limiter.Allow("x") {
		t.Fatal("permintaan kedua harus ditolak")
	}
	time.Sleep(80 * time.Millisecond)
	if !limiter.Allow("x") {
		t.Fatal("setelah jendela lewat harus lolos lagi")
	}
}

func TestRemainingCountsDown(t *testing.T) {
	limiter := New(3, time.Minute)

	if got := limiter.Remaining("k"); got != 3 {
		t.Fatalf("remaining awal = %d, want 3", got)
	}
	limiter.Allow("k")
	if got := limiter.Remaining("k"); got != 2 {
		t.Fatalf("remaining setelah 1 = %d, want 2", got)
	}
}

func TestSweepRemovesStaleEntries(t *testing.T) {
	limiter := New(5, 30*time.Millisecond)
	limiter.Allow("lama")
	time.Sleep(60 * time.Millisecond)

	limiter.Sweep()
	if got := limiter.Size(); got != 0 {
		t.Fatalf("setelah sweep ukuran = %d, want 0", got)
	}
}
