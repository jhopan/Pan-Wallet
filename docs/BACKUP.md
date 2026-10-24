# Backup & Pemulihan Pan-Wallet

Ada dua hal berbeda yang sering tertukar:

| | Isi | Kegunaan |
|---|---|---|
| **Backup** | berkas `.db` utuh | memulihkan sistem persis seperti semula |
| **Ekspor** | berkas `.csv` per tabel | dibaca manusia, dibuka di spreadsheet |

Backup untuk **mengembalikan**, ekspor untuk **melihat**.

---

## Kenapa backup file mentah tidak cukup

SQLite berjalan dalam mode WAL. Data baru bisa masih berada di berkas
`pan-wallet.db-wal` dan belum masuk ke `pan-wallet.db`.

Menyalin `pan-wallet.db` dengan `cp` atau `io.Copy` saat server hidup akan
menghasilkan backup **tidak lengkap**. Ini pernah terjadi: backup berisi
1 wallet dari 17, dan Rp40.000 dari Rp1.465.017.

Pan-Wallet memakai `VACUUM INTO`, yaitu SQLite membuat salinan logisnya
sendiri. Hasilnya selalu utuh dan konsisten, meski server sedang melayani.

---

## 1. Backup lokal (bawaan)

Otomatis tiap jam, simpan 24 terakhir, di folder `backup/`.

```
PAN_WALLET_BACKUP_DIR=backup
```

Tombol manual: panel → **Backup** → *Buat backup sekarang*.

**Kekurangan:** hanya di server yang sama. Kalau disk rusak atau VPS hilang,
database dan backup ikut hilang bersamaan. Ini **bukan** backup sungguhan.

---

## 2. Backup ke Google Drive (disarankan)

Pan-Wallet sudah menyertakan program `backup-drive` yang melakukan
snapshot + unggah + verifikasi + bersih-bersih dalam satu perintah.

```bash
# sekali saja: hubungkan Google Drive ke rclone
rclone config          # pilih n, beri nama "jhopan", pilih Google Drive

# coba dulu tanpa mengunggah
./backup-drive -remote jhopan -dry-run

# jalankan sungguhan
./backup-drive -remote jhopan
```

Keluarannya:

```
[1/4] snapshot data/pan-wallet.db
      backup/pan-wallet-20261025-064820.db (184.0 KB)
[2/4] verifikasi isi
      pelanggan: 1
[3/4] unggah ke jhopan:Pan-Wallet-Backup/
[4/4] hapus backup > 30 hari
selesai.
```

Langkah 2 penting: kalau backup tidak bisa dibuka, proses berhenti dan
**tidak** mengunggah file rusak.

Jadwalkan tiap 6 jam:

```bash
crontab -e
0 */6 * * * cd /opt/pan-wallet && ./backup-drive -remote jhopan >> backup.log 2>&1
```

Windows: Task Scheduler → *Create Basic Task* → jalankan
`backup-drive.exe -remote jhopan` setiap 6 jam.

**Kelebihan:** otomatis, ada versi lama (Drive menyimpan revisi),
tidak hilang kalau VPS mati.

---

## 3. Backup ke GitHub (bisa, tapi hati-hati)

Bisa, **tapi jangan pakai repo publik**. Database berisi `public_id`
pelanggan dan riwayat saldo — itu data pribadi.

```bash
# repo HARUS privat
gh repo create pan-wallet-backup --private

cd backup
git init
git remote add origin git@github.com:USER/pan-wallet-backup.git

# .gitignore: jangan pernah commit .env
echo ".env" > .gitignore

git add *.db
git commit -m "backup $(date +%F)"
git push
```

**Masalahnya:**

| Hal | Kenapa bermasalah |
|---|---|
| Ukuran | Git menyimpan **setiap versi** file. DB 200 KB × 24 backup/hari = repo membengkak cepat |
| Riwayat permanen | Data yang pernah di-commit **sulit dihapus** dari riwayat Git |
| Data pribadi | `public_id` + saldo pelanggan tersimpan di server GitHub |
| Bukan untuk ini | Git dirancang untuk kode, bukan berkas biner berubah terus |

**Kalau tetap mau GitHub:** commit **CSV hasil ekspor**, bukan file `.db`.
CSV jauh lebih kecil, mudah dibaca, dan diff-nya masuk akal.

```bash
cd export/export-20261025-035409
git add *.csv
git commit -m "laporan $(date +%F)"
git push
```

Repo tetap **wajib privat**.

---

## 4. Backup ke Google Sheets (paling mudah dibaca)

Bukan backup teknis, tapi laporan yang bisa dibuka siapa saja di tim.

### Cara A — manual (paling sederhana)

1. Panel → **Ekspor Data** → *Buat ekspor sekarang*
2. Unduh CSV dari folder `export/`
3. Buka Google Sheets → **File → Import → Upload** → pilih CSV

Cocok untuk laporan bulanan.

### Cara B — otomatis lewat Apps Script

Buat di Google Sheets: **Extensions → Apps Script**

```javascript
function tarikDataPanWallet() {
  // Ganti dengan URL Pan-Wallet kamu
  var URL = 'https://wallet.domain-kamu.com/v1/public/65ZMPUNP';

  var response = UrlFetchApp.fetch(URL);
  var data = JSON.parse(response.getContentText());

  var sheet = SpreadsheetApp.getActiveSpreadsheet().getSheetByName('Saldo')
              || SpreadsheetApp.getActiveSpreadsheet().insertSheet('Saldo');

  sheet.appendRow([new Date(), data.public_id, data.balance, data.available]);
}
```

Lalu **Triggers → Add Trigger → tarikDataPanWallet → Time-driven → setiap jam**.

**Batasan:** cara ini hanya membaca **satu** pelanggan per panggilan.
Untuk semua pelanggan, lebih baik ekspor CSV.

### Cara C — kirim CSV ke Drive lewat rclone

Sama seperti opsi 2, tapi targetnya folder Drive yang tersinkron
dengan Google Sheets. Bisa dibuka langsung sebagai spreadsheet.

---

## 5. Pemulihan (restore)

Ini yang paling penting dan paling sering dilupakan.
**Backup yang belum pernah diuji restore bukan backup.**

```bash
# 1. hentikan Pan-Wallet
sudo systemctl stop pan-wallet

# 2. simpan database sekarang (jaga-jaga)
mv data/pan-wallet.db data/pan-wallet.db.rusak

# 3. buang WAL lama — kalau tidak, data lama bisa "kembali"
rm -f data/pan-wallet.db-wal data/pan-wallet.db-shm

# 4. salin backup
cp backup/pan-wallet-20261025-035409.db data/pan-wallet.db

# 5. nyalakan
sudo systemctl start pan-wallet

# 6. verifikasi
curl -s http://127.0.0.1:8080/healthz
```

**Langkah 3 wajib.** Kalau file `-wal` lama tertinggal, SQLite bisa
mencampur data lama dengan backup dan hasilnya kacau.

### Uji restore tanpa mengganggu produksi

```bash
cp backup/pan-wallet-XXXX.db /tmp/uji.db
sqlite3 /tmp/uji.db "select count(*) from wallet_users; select sum(balance) from wallets;"
```

Kalau angkanya masuk akal, backup layak dipercaya.

---

## Ringkasan rekomendasi

| Kebutuhan | Pilihan |
|---|---|
| Pemulihan bencana | **rclone ke Google Drive** (opsi 2) |
| Laporan pembukuan | **Ekspor CSV** → Google Sheets (opsi 4A) |
| Arsip kode & laporan | GitHub **privat**, isi CSV (opsi 3) |
| Backup harian cepat | bawaan `backup/` + rclone |

**Jangan** mengandalkan folder `backup/` saja. Itu satu-satunya salinan,
dan letaknya di mesin yang sama dengan aslinya.
