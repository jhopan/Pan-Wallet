// Package qris mengubah payload QRIS statis milik merchant menjadi payload
// dinamis bernominal. Mekanisme sama dengan PayPan: ubah point-of-initiation
// static (010211) menjadi dynamic (010212), sisipkan tag 54 (nominal) sebelum
// 5802ID, lalu hitung ulang CRC16 CCITT-FALSE di belakang tag 6304.
package qris

import (
	"fmt"
	"strings"
)

// CRC16 menghitung CRC-16/CCITT-FALSE (poly 0x1021, init 0xFFFF) sesuai EMVCo.
func crc16(payload string) string {
	crc := uint16(0xFFFF)
	for index := 0; index < len(payload); index++ {
		crc ^= uint16(payload[index]) << 8
		for bit := 0; bit < 8; bit++ {
			if crc&0x8000 != 0 {
				crc = crc<<1 ^ 0x1021
			} else {
				crc <<= 1
			}
		}
	}
	return fmt.Sprintf("%04X", crc)
}

// BuildDynamic menyuntikkan nominal (rupiah bulat) ke payload statis.
func BuildDynamic(static string, amount int64) (string, error) {
	if amount <= 0 {
		return "", fmt.Errorf("nominal harus lebih dari nol")
	}
	if !strings.Contains(static, "010211") || !strings.Contains(static, "5802ID") {
		return "", fmt.Errorf("payload QRIS statis tidak dikenali (butuh 010211 dan 5802ID)")
	}

	value := fmt.Sprintf("%d.00", amount)
	tag54 := fmt.Sprintf("54%02d%s", len(value), value)

	dynamic := strings.Replace(static, "010211", "010212", 1)
	dynamic = strings.Replace(dynamic, "5802ID", tag54+"5802ID", 1)

	marker := strings.Index(dynamic, "6304")
	if marker < 0 {
		return "", fmt.Errorf("payload QRIS tanpa penanda CRC 6304")
	}
	body := dynamic[:marker+4]
	return body + crc16(body), nil
}

// ValidateStatic memeriksa kelengkapan payload statis sebelum disimpan admin.
func ValidateStatic(static string) error {
	trimmed := strings.TrimSpace(static)
	if !strings.Contains(trimmed, "010211") {
		return fmt.Errorf("payload harus QRIS statis (010211)")
	}
	if !strings.Contains(trimmed, "5802ID") {
		return fmt.Errorf("payload harus merchant Indonesia (5802ID)")
	}
	if !strings.Contains(trimmed, "6304") {
		return fmt.Errorf("payload tanpa CRC 6304")
	}
	return nil
}
