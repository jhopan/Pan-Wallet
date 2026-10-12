// Package topup mengatur invoice top-up QRIS: alokasi kode unik 001-999 per
// level harga sehingga tiap nominal total hanya dimiliki satu invoice aktif.
// Desain ini sama dengan PayPan: nominal total menjadi identitas pembayaran,
// karena notifikasi e-wallet hanya membawa jumlah, bukan identitas pengirim.
package topup

// AllocateCode memilih kode terendah yang bebas untuk sebuah harga.
//   - usedInPrice: kode yang sudah dipakai invoice aktif dengan harga sama.
//   - claimedTotals: total (harga+kode) yang sudah dimiliki invoice aktif lain.
//
// Mengembalikan 0 bila semua kode 1-999 habis.
func AllocateCode(price int64, usedInPrice []int, claimedTotals []int64) int {
	used := make(map[int]bool, len(usedInPrice))
	for _, code := range usedInPrice {
		used[code] = true
	}
	claimed := make(map[int64]bool, len(claimedTotals))
	for _, total := range claimedTotals {
		claimed[total] = true
	}

	for code := 1; code <= 999; code++ {
		if used[code] {
			continue
		}
		if claimed[price+int64(code)] {
			continue
		}
		return code
	}
	return 0
}

// Total menghitung nominal akhir yang harus dibayar customer.
func Total(price int64, code int) int64 {
	return price + int64(code)
}
