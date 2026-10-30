package paypan

import (
	"context"
	"net/http"
)

// RefundInvoice menandai invoice yang sudah lunas sebagai refunded.
//
// CATATAN PENTING: ini pembukuan administratif di PayPan — status invoice
// berubah menjadi "refunded". PayPan TIDAK mengembalikan dana ke pelanggan.
// Pengembalian dana tetap urusan manual admin.
//
// Endpoint PayPan: POST /api/invoice/{id}/refund (butuh scope "order").
func (client *Client) RefundInvoice(ctx context.Context, invoiceID string) error {
	return client.call(ctx, http.MethodPost, "/api/invoice/"+invoiceID+"/refund", nil, nil)
}
