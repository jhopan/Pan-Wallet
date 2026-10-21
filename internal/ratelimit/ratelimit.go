// Package ratelimit membatasi jumlah permintaan per kunci (IP) dalam jendela
// waktu tertentu. Dipakai endpoint publik supaya ID pelanggan tidak bisa
// ditebak-tebak dengan brute force.
package ratelimit

import (
	"sync"
	"time"
)

// Limiter menyimpan hitungan per kunci di memori.
type Limiter struct {
	mu     sync.Mutex
	hits   map[string][]time.Time
	limit  int
	window time.Duration
}

// New membuat limiter dengan batas dan jendela waktu.
func New(limit int, window time.Duration) *Limiter {
	if limit <= 0 {
		limit = 60
	}
	if window <= 0 {
		window = time.Minute
	}
	return &Limiter{hits: map[string][]time.Time{}, limit: limit, window: window}
}

// Allow mencatat satu permintaan dan melaporkan apakah masih diizinkan.
func (limiter *Limiter) Allow(key string) bool {
	limiter.mu.Lock()
	defer limiter.mu.Unlock()

	now := time.Now()
	kept := limiter.hits[key][:0]
	for _, at := range limiter.hits[key] {
		if now.Sub(at) < limiter.window {
			kept = append(kept, at)
		}
	}

	if len(kept) >= limiter.limit {
		limiter.hits[key] = kept
		return false
	}
	limiter.hits[key] = append(kept, now)
	return true
}

// Remaining menghitung sisa kuota untuk satu kunci.
func (limiter *Limiter) Remaining(key string) int {
	limiter.mu.Lock()
	defer limiter.mu.Unlock()

	now := time.Now()
	count := 0
	for _, at := range limiter.hits[key] {
		if now.Sub(at) < limiter.window {
			count++
		}
	}
	remaining := limiter.limit - count
	if remaining < 0 {
		return 0
	}
	return remaining
}

// Sweep membuang kunci yang sudah tidak punya catatan aktif.
func (limiter *Limiter) Sweep() {
	limiter.mu.Lock()
	defer limiter.mu.Unlock()

	now := time.Now()
	for key, times := range limiter.hits {
		active := false
		for _, at := range times {
			if now.Sub(at) < limiter.window {
				active = true
				break
			}
		}
		if !active {
			delete(limiter.hits, key)
		}
	}
}

// Size melaporkan jumlah kunci yang tersimpan (untuk uji dan pemantauan).
func (limiter *Limiter) Size() int {
	limiter.mu.Lock()
	defer limiter.mu.Unlock()
	return len(limiter.hits)
}
