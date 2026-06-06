package main

import (
	"context"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

var db *pgxpool.Pool

func connectDB() error {
	var err error

	databaseURL := os.Getenv("DATABASE_URL")

	db, err = pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		return err
	}

	return db.Ping(context.Background())
}

func migrateDB() error {
	query := `
	CREATE TABLE IF NOT EXISTS bidang (
		id_bidang INTEGER PRIMARY KEY,
		nama_bidang TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS kursus (
		id_kursus INTEGER PRIMARY KEY,
		nama_kursus TEXT NOT NULL,
		id_bidang INTEGER NOT NULL REFERENCES bidang(id_bidang) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS peserta (
		id_pendaftaran INTEGER PRIMARY KEY,
		nama_lengkap TEXT NOT NULL,
		tanggal_daftar TEXT NOT NULL,
		id_kursus INTEGER NOT NULL REFERENCES kursus(id_kursus) ON DELETE CASCADE,
		id_bidang INTEGER NOT NULL REFERENCES bidang(id_bidang) ON DELETE CASCADE,
		status_aktif BOOLEAN NOT NULL
	);
	`

	_, err := db.Exec(context.Background(), query)
	return err
}

func getAllBidang() ([]BidangMinat, error) {
	rows, err := db.Query(
		context.Background(),
		"SELECT id_bidang, nama_bidang FROM bidang ORDER BY id_bidang",
	)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var bidang []BidangMinat

	for rows.Next() {
		var b BidangMinat

		err := rows.Scan(
			&b.IDBidang,
			&b.NamaBidang,
		)

		if err != nil {
			return nil, err
		}

		bidang = append(bidang, b)
	}

	return bidang, nil
}

func insertBidang(b BidangMinat) error {
	_, err := db.Exec(
		context.Background(),
		`
		INSERT INTO bidang (id_bidang, nama_bidang)
		VALUES ($1, $2)
		`,
		b.IDBidang,
		b.NamaBidang,
	)

	return err
}

func deleteBidang(id int) error {
	_, err := db.Exec(
		context.Background(),
		"DELETE FROM bidang WHERE id_bidang = $1",
		id,
	)

	return err
}

func getAllKursus() ([]Kursus, error) {
	rows, err := db.Query(
		context.Background(),
		`
		SELECT id_kursus, nama_kursus, id_bidang
		FROM kursus
		ORDER BY id_kursus
		`,
	)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var kursus []Kursus

	for rows.Next() {
		var k Kursus

		err := rows.Scan(
			&k.IDKursus,
			&k.NamaKursus,
			&k.IDBidang,
		)

		if err != nil {
			return nil, err
		}

		kursus = append(kursus, k)
	}

	return kursus, nil
}

func insertKursus(k Kursus) error {
	_, err := db.Exec(
		context.Background(),
		`
		INSERT INTO kursus (id_kursus, nama_kursus, id_bidang)
		VALUES ($1, $2, $3)
		`,
		k.IDKursus,
		k.NamaKursus,
		k.IDBidang,
	)

	return err
}

func deleteKursus(id int) error {
	_, err := db.Exec(
		context.Background(),
		"DELETE FROM kursus WHERE id_kursus = $1",
		id,
	)

	return err
}

func getAllPeserta() ([]Peserta, error) {
	rows, err := db.Query(
		context.Background(),
		`
		SELECT
			id_pendaftaran,
			nama_lengkap,
			tanggal_daftar,
			id_kursus,
			id_bidang,
			status_aktif
		FROM peserta
		ORDER BY id_pendaftaran
		`,
	)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var peserta []Peserta

	for rows.Next() {
		var p Peserta

		err := rows.Scan(
			&p.IDPendaftaran,
			&p.NamaLengkap,
			&p.TanggalDaftar,
			&p.IDKursus,
			&p.IDBidang,
			&p.StatusAktif,
		)

		if err != nil {
			return nil, err
		}

		peserta = append(peserta, p)
	}

	return peserta, nil
}

func insertPeserta(p Peserta) error {
	_, err := db.Exec(
		context.Background(),
		`
		INSERT INTO peserta (
			id_pendaftaran,
			nama_lengkap,
			tanggal_daftar,
			id_kursus,
			id_bidang,
			status_aktif
		)
		VALUES ($1, $2, $3, $4, $5, $6)
		`,
		p.IDPendaftaran,
		p.NamaLengkap,
		p.TanggalDaftar,
		p.IDKursus,
		p.IDBidang,
		p.StatusAktif,
	)

	return err
}

func getPesertaByID(id int) (Peserta, error) {
	var p Peserta

	err := db.QueryRow(
		context.Background(),
		`
		SELECT
			id_pendaftaran,
			nama_lengkap,
			tanggal_daftar,
			id_kursus,
			id_bidang,
			status_aktif
		FROM peserta
		WHERE id_pendaftaran = $1
		`,
		id,
	).Scan(
		&p.IDPendaftaran,
		&p.NamaLengkap,
		&p.TanggalDaftar,
		&p.IDKursus,
		&p.IDBidang,
		&p.StatusAktif,
	)

	return p, err
}

func updatePesertaDB(p Peserta) error {
	_, err := db.Exec(
		context.Background(),
		`
		UPDATE peserta
		SET
			nama_lengkap = $1,
			tanggal_daftar = $2,
			id_kursus = $3,
			id_bidang = $4,
			status_aktif = $5
		WHERE id_pendaftaran = $6
		`,
		p.NamaLengkap,
		p.TanggalDaftar,
		p.IDKursus,
		p.IDBidang,
		p.StatusAktif,
		p.IDPendaftaran,
	)

	return err
}

func deletePeserta(id int) error {
	_, err := db.Exec(
		context.Background(),
		"DELETE FROM peserta WHERE id_pendaftaran = $1",
		id,
	)

	return err
}

func getDashboardStats() (int, int, int, int, error) {

	var totalPeserta int
	var totalAktif int
	var totalKursus int
	var totalBidang int

	err := db.QueryRow(
		context.Background(),
		"SELECT COUNT(*) FROM peserta",
	).Scan(&totalPeserta)

	if err != nil {
		return 0, 0, 0, 0, err
	}

	err = db.QueryRow(
		context.Background(),
		"SELECT COUNT(*) FROM peserta WHERE status_aktif = true",
	).Scan(&totalAktif)

	if err != nil {
		return 0, 0, 0, 0, err
	}

	err = db.QueryRow(
		context.Background(),
		"SELECT COUNT(*) FROM kursus",
	).Scan(&totalKursus)

	if err != nil {
		return 0, 0, 0, 0, err
	}

	err = db.QueryRow(
		context.Background(),
		"SELECT COUNT(*) FROM bidang",
	).Scan(&totalBidang)

	if err != nil {
		return 0, 0, 0, 0, err
	}

	return totalPeserta, totalAktif, totalKursus, totalBidang, nil
}

func getJumlahPesertaPerBidang() (map[int]int, error) {
	rows, err := db.Query(
		context.Background(),
		`
		SELECT id_bidang, COUNT(*)
		FROM peserta
		GROUP BY id_bidang
		`,
	)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	jumlahPerBidang := make(map[int]int)

	for rows.Next() {
		var idBidang int
		var jumlah int

		err := rows.Scan(&idBidang, &jumlah)
		if err != nil {
			return nil, err
		}

		jumlahPerBidang[idBidang] = jumlah
	}

	return jumlahPerBidang, nil
}