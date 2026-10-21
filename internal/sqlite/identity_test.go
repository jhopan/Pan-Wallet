package sqlite

import (
	"context"
	"testing"

	"github.com/jhopan/Pan-Wallet/internal/identity"
)

func TestResolveTelegramCreatesWalletOnce(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	first, created, err := store.ResolveIdentity(ctx, identity.ProviderTelegram, "123456789")
	if err != nil {
		t.Fatalf("ResolveIdentity: %v", err)
	}
	if !created {
		t.Fatal("panggilan pertama harus membuat wallet baru")
	}
	if first.PublicID == "" {
		t.Fatal("public_id harus terisi")
	}

	second, created, err := store.ResolveIdentity(ctx, identity.ProviderTelegram, "123456789")
	if err != nil {
		t.Fatalf("ResolveIdentity kedua: %v", err)
	}
	if created {
		t.Fatal("panggilan kedua tidak boleh membuat wallet baru")
	}
	if second.PublicID != first.PublicID {
		t.Fatalf("public_id berubah: %q -> %q", first.PublicID, second.PublicID)
	}
}

func TestResolveDifferentTelegramGetsDifferentWallet(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	a, _, _ := store.ResolveIdentity(ctx, identity.ProviderTelegram, "111111111")
	b, _, _ := store.ResolveIdentity(ctx, identity.ProviderTelegram, "222222222")

	if a.PublicID == b.PublicID {
		t.Fatal("telegram id berbeda harus dapat wallet berbeda")
	}
}

func TestLinkEmailViaTelegramCode(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	// Pelanggan mulai dari Telegram.
	wallet, _, err := store.ResolveIdentity(ctx, identity.ProviderTelegram, "333333333")
	if err != nil {
		t.Fatalf("ResolveIdentity: %v", err)
	}

	// Dari web: minta kode untuk email.
	code, err := store.StartIdentityLink(ctx, identity.ProviderEmail, "Budi@Gmail.com")
	if err != nil {
		t.Fatalf("StartIdentityLink: %v", err)
	}
	if len(code) != 6 {
		t.Fatalf("kode = %q, want 6 angka", code)
	}

	// Pelanggan kirim /ikat <kode> di Telegram.
	linked, err := store.ConfirmIdentityLink(ctx, code, "333333333")
	if err != nil {
		t.Fatalf("ConfirmIdentityLink: %v", err)
	}
	if linked.PublicID != wallet.PublicID {
		t.Fatalf("tertaut ke wallet salah: %q want %q", linked.PublicID, wallet.PublicID)
	}

	// Sekarang email yang sama harus resolve ke wallet yang sama.
	viaEmail, created, err := store.ResolveIdentity(ctx, identity.ProviderEmail, "budi@gmail.com")
	if err != nil {
		t.Fatalf("ResolveIdentity email: %v", err)
	}
	if created {
		t.Fatal("email yang sudah tertaut tidak boleh membuat wallet baru")
	}
	if viaEmail.PublicID != wallet.PublicID {
		t.Fatalf("email resolve ke %q, want %q", viaEmail.PublicID, wallet.PublicID)
	}
}

func TestConfirmCodeCanOnlyBeUsedOnce(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	store.ResolveIdentity(ctx, identity.ProviderTelegram, "444444444")
	code, _ := store.StartIdentityLink(ctx, identity.ProviderEmail, "satu@example.com")

	if _, err := store.ConfirmIdentityLink(ctx, code, "444444444"); err != nil {
		t.Fatalf("konfirmasi pertama: %v", err)
	}
	if _, err := store.ConfirmIdentityLink(ctx, code, "444444444"); err == nil {
		t.Fatal("kode yang sudah dipakai harus ditolak")
	}
}

func TestConfirmRejectsUnknownCode(t *testing.T) {
	store := newTestStore(t)
	store.ResolveIdentity(context.Background(), identity.ProviderTelegram, "555555555")

	if _, err := store.ConfirmIdentityLink(context.Background(), "000000", "555555555"); err == nil {
		t.Fatal("kode asing harus ditolak")
	}
}

func TestIdentityCannotBeStolenByOtherWallet(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	owner, _, _ := store.ResolveIdentity(ctx, identity.ProviderTelegram, "666666666")
	store.ResolveIdentity(ctx, identity.ProviderTelegram, "777777777")

	// Pemilik asli menautkan email.
	code, err := store.StartIdentityLink(ctx, identity.ProviderEmail, "korban@example.com")
	if err != nil {
		t.Fatalf("StartIdentityLink: %v", err)
	}
	if _, err := store.ConfirmIdentityLink(ctx, code, "666666666"); err != nil {
		t.Fatalf("konfirmasi pemilik: %v", err)
	}

	// Penyerang tidak bisa minta kode untuk identitas yang sudah tertaut.
	if _, err := store.StartIdentityLink(ctx, identity.ProviderEmail, "korban@example.com"); err == nil {
		t.Fatal("email yang sudah tertaut harus menolak permintaan kode baru")
	}

	// Pastikan tetap milik pemilik asli.
	check, _, _ := store.ResolveIdentity(ctx, identity.ProviderEmail, "korban@example.com")
	if check.PublicID != owner.PublicID {
		t.Fatalf("email berpindah ke wallet lain: %q want %q", check.PublicID, owner.PublicID)
	}
}
