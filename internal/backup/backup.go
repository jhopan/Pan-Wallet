package backup

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Backup adalah satu berkas cadangan yang tersedia.
type Backup struct {
	Name      string
	SizeBytes int64
	CreatedAt time.Time
}

// Runner menyalin database ke folder cadangan secara berkala.
type Runner struct {
	sourcePath string
	dir        string
	keep       int
	interval   time.Duration
}

// New membuat runner backup.
// sourcePath = file SQLite aktif, dir = folder cadangan, keep = jumlah berkas disimpan.
func New(sourcePath, dir string, keep int, interval time.Duration) *Runner {
	if keep <= 0 {
		keep = 24
	}
	if interval <= 0 {
		interval = time.Hour
	}
	return &Runner{sourcePath: sourcePath, dir: dir, keep: keep, interval: interval}
}

// Once membuat satu berkas cadangan bertimestamp dan merapikan yang lama.
func (runner *Runner) Once() (Backup, error) {
	if err := os.MkdirAll(runner.dir, 0o755); err != nil {
		return Backup{}, fmt.Errorf("buat folder backup: %w", err)
	}

	name := timestampedName(time.Now())
	target := filepath.Join(runner.dir, name)

	size, err := Snapshot(runner.sourcePath, target)
	if err != nil {
		return Backup{}, err
	}

	backup := Backup{Name: name, SizeBytes: size, CreatedAt: time.Now()}

	if err := runner.prune(); err != nil {
		return backup, err
	}
	return backup, nil
}

// prune menyisakan hanya `keep` berkas terbaru.
func (runner *Runner) prune() error {
	entries, err := runner.List()
	if err != nil {
		return err
	}
	if len(entries) <= runner.keep {
		return nil
	}
	for _, entry := range entries[runner.keep:] {
		_ = os.Remove(filepath.Join(runner.dir, entry.Name))
	}
	return nil
}

// List mengembalikan berkas cadangan, terbaru dulu.
func (runner *Runner) List() ([]Backup, error) {
	items, err := os.ReadDir(runner.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("baca folder backup: %w", err)
	}

	var backups []Backup
	for _, item := range items {
		if item.IsDir() || !strings.HasPrefix(item.Name(), "panwallet-") || !strings.HasSuffix(item.Name(), ".db") {
			continue
		}
		info, err := item.Info()
		if err != nil {
			continue
		}
		backups = append(backups, Backup{Name: item.Name(), SizeBytes: info.Size(), CreatedAt: info.ModTime()})
	}
	sort.Slice(backups, func(a, b int) bool { return backups[a].CreatedAt.After(backups[b].CreatedAt) })
	return backups, nil
}

// Start menjalankan backup berkala sampai ctx dibatalkan.
func (runner *Runner) Start(ctx context.Context) {
	go func() {
		// Satu backup saat start, supaya langsung ada titik pulih.
		_, _ = runner.Once()

		ticker := time.NewTicker(runner.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				_, _ = runner.Once()
			}
		}
	}()
}
