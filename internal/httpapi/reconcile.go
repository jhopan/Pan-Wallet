package httpapi

import (
	"context"
	"log"
	"time"

	"github.com/jhopan/PanWallet/internal/topup"
)

// ReconcileStore menyediakan invoice pending untuk dicek ulang ke PayPan.
type ReconcileStore interface {
	PendingTopupInvoices(context.Context, time.Time) ([]topup.Invoice, error)
	MarkTopupPaidByPaypan(context.Context, string, string) (topup.Invoice, error)
	ExpireTopupInvoice(context.Context, string) error
}

// reconcileInterval adalah jeda antar putaran pemeriksaan.
const reconcileInterval = 2 * time.Minute

// reconcileAge: hanya invoice yang lebih tua dari ini yang dicek, supaya
// tidak menabrak webhook yang mungkin masih dalam perjalanan.
const reconcileAge = 3 * time.Minute

// StartReconciler menjalankan pemeriksaan berkala di latar belakang.
// Ini cadangan kalau webhook PayPan tidak sampai (jaringan, firewall, restart).
func (server *Server) StartReconciler(ctx context.Context) {
	if server.deps.Reconcile == nil {
		return
	}
	go func() {
		ticker := time.NewTicker(reconcileInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				server.reconcileOnce(ctx)
			}
		}
	}()
}

// reconcileOnce menanyakan status setiap invoice pending ke PayPan.
func (server *Server) reconcileOnce(ctx context.Context) {
	if server.deps.Reconcile == nil || server.deps.Settings == nil {
		return
	}
	client, err := server.paypanClientFrom(ctx)
	if err != nil {
		// PayPan belum dikonfigurasi: lewati tanpa berisik.
		return
	}

	invoices, err := server.deps.Reconcile.PendingTopupInvoices(ctx, time.Now().Add(-reconcileAge))
	if err != nil {
		log.Printf("rekonsiliasi: baca invoice: %v", err)
		return
	}

	for _, invoice := range invoices {
		if invoice.PaypanInvoice == "" {
			continue
		}
		remote, err := client.GetInvoice(ctx, invoice.PaypanInvoice)
		if err != nil {
			continue
		}

		switch remote.Status {
		case "paid":
			if _, err := server.deps.Reconcile.MarkTopupPaidByPaypan(ctx, remote.ID, "paypan-reconcile"); err != nil {
				log.Printf("rekonsiliasi: kredit %s: %v", remote.ID, err)
			} else {
				log.Printf("rekonsiliasi: invoice %s lunas, saldo dikredit", remote.ID)
			}
		case "expired", "refunded":
			if err := server.deps.Reconcile.ExpireTopupInvoice(ctx, invoice.ID); err != nil {
				log.Printf("rekonsiliasi: tandai %s: %v", invoice.ID, err)
			}
		}
	}
}
