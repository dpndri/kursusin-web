# KursusIn Web Integrated

Versi web terintegrasi dari aplikasi console KursusIn.

## Fitur

- CRUD Bidang Minat
- CRUD Kursus
- CRUD Peserta
- Peserta terintegrasi dengan Kursus dan Bidang Minat melalui dropdown
- Sequential Search berdasarkan nama dan/atau ID bidang
- Binary Search berdasarkan nama lengkap
- Selection Sort berdasarkan ID peserta
- Insertion Sort berdasarkan nama peserta
- Statistik total peserta, peserta aktif, dan jumlah peserta per bidang
- Penyimpanan permanen ke `data/kursusin.json`

## Cara menjalankan

```bash
go mod tidy
go run main.go
```

Buka browser:

```text
http://localhost:8080
```

## Alur input yang benar

1. Tambahkan Bidang Minat terlebih dahulu.
2. Tambahkan Kursus dan pilih Bidang Minat.
3. Tambahkan Peserta dan pilih Kursus + Bidang Minat.
