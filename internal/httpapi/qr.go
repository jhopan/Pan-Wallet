package httpapi

import (
	"bytes"

	"github.com/skip2/go-qrcode"
)

// encodeQR menghasilkan PNG QR dari payload QRIS, ukuran 512px.
func encodeQR(payload string) ([]byte, error) {
	png, err := qrcode.Encode(payload, qrcode.Medium, 512)
	if err != nil {
		return nil, err
	}
	return bytes.Clone(png), nil
}
