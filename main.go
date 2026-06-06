package main

import (
	"context"
	"html/template"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type Peserta struct {
	IDPendaftaran int    `json:"id_pendaftaran"`
	NamaLengkap   string `json:"nama_lengkap"`
	TanggalDaftar string `json:"tanggal_daftar"`
	IDKursus      int    `json:"id_kursus"`
	IDBidang      int    `json:"id_bidang"`
	StatusAktif   bool   `json:"status_aktif"`
}

type Kursus struct {
	IDKursus   int    `json:"id_kursus"`
	NamaKursus string `json:"nama_kursus"`
	IDBidang   int    `json:"id_bidang"`
}

type BidangMinat struct {
	IDBidang   int    `json:"id_bidang"`
	NamaBidang string `json:"nama_bidang"`
}

func main() {
	err := connectDB()
	if err != nil {
		panic(err)
	}
	defer db.Close()

	err = migrateDB()
	if err != nil {
		panic(err)
	}

	r := gin.Default()
	r.Static("/static", "./static")
	r.SetFuncMap(template.FuncMap{
		"namaBidang": namaBidangByID,
		"namaKursus": namaKursusByID,
	})
	r.LoadHTMLGlob("templates/*")

	r.GET("/login", formLogin)
	r.POST("/login", prosesLogin)
	r.GET("/logout", logout)

	protected := r.Group("/")
	protected.Use(authMiddleware())

	protected.GET("/", dashboard)

	protected.GET("/peserta", tampilPeserta)
	protected.GET("/peserta/tambah", formTambahPeserta)
	protected.POST("/peserta/tambah", tambahPeserta)
	protected.GET("/peserta/edit/:id", formEditPeserta)
	protected.POST("/peserta/edit/:id", updatePeserta)
	protected.POST("/peserta/hapus/:id", hapusPeserta)
	protected.GET("/peserta/cari", cariPesertaSequential)
	protected.GET("/peserta/binary-search", cariPesertaBinary)
	protected.GET("/peserta/sort/id", sortPesertaByID)
	protected.GET("/peserta/sort/nama", sortPesertaByNama)

	protected.GET("/kursus", tampilKursus)
	protected.GET("/kursus/tambah", formTambahKursus)
	protected.POST("/kursus/tambah", tambahKursus)
	protected.POST("/kursus/hapus/:id", hapusKursus)

	protected.GET("/bidang", tampilBidang)
	protected.GET("/bidang/tambah", formTambahBidang)
	protected.POST("/bidang/tambah", tambahBidang)
	protected.POST("/bidang/hapus/:id", hapusBidang)

	protected.GET("/statistik", statistik)

	port := os.Getenv("PORT")

	if port == "" {
		port = "8080"
	}

	r.Run(":" + port)
}

func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		cookie, err := c.Cookie("is_logged_in")

		if err != nil || cookie != "true" {
			c.Redirect(http.StatusSeeOther, "/login")
			c.Abort()
			return
		}

		c.Next()
	}
}

func formLogin(c *gin.Context) {
	c.HTML(http.StatusOK, "login.html", gin.H{
		"title": "Login Admin",
	})
}

func prosesLogin(c *gin.Context) {
	username := c.PostForm("username")
	password := c.PostForm("password")

	if username == "admin" && password == "admin123" {
		c.SetCookie("is_logged_in", "true", 3600, "/", "", false, true)
		c.Redirect(http.StatusSeeOther, "/")
		return
	}

	c.HTML(http.StatusUnauthorized, "login.html", gin.H{
		"title": "Login Admin",
		"error": "Username atau password salah",
	})
}

func logout(c *gin.Context) {
	c.SetCookie("is_logged_in", "", -1, "/", "", false, true)
	c.Redirect(http.StatusSeeOther, "/login")
}

func dashboard(c *gin.Context) {

	totalPeserta,
		totalAktif,
		totalKursus,
		totalBidang,
		err := getDashboardStats()

	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	c.HTML(http.StatusOK, "index.html", gin.H{
		"title":        "Dashboard KursusIn",
		"totalPeserta": totalPeserta,
		"totalAktif":   totalAktif,
		"totalKursus":  totalKursus,
		"totalBidang":  totalBidang,
	})
}

func tampilPeserta(c *gin.Context) {

	peserta, err := getAllPeserta()

	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	c.HTML(http.StatusOK, "peserta.html", gin.H{
		"title":   "Data Peserta",
		"peserta": peserta,
	})
}

func formTambahPeserta(c *gin.Context) {

	kursus, err := getAllKursus()
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	bidang, err := getAllBidang()
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	c.HTML(http.StatusOK, "form_peserta.html", gin.H{
		"title":   "Tambah Peserta",
		"mode":    "tambah",
		"peserta": Peserta{StatusAktif: true},
		"kursus":  kursus,
		"bidang":  bidang,
	})
}

func tambahPeserta(c *gin.Context) {
	idPendaftaran := atoi(c.PostForm("id_pendaftaran"))
	namaLengkap := strings.TrimSpace(c.PostForm("nama_lengkap"))
	tanggalDaftar := c.PostForm("tanggal_daftar")
	idKursus := atoi(c.PostForm("id_kursus"))
	idBidang := atoi(c.PostForm("id_bidang"))
	statusAktif := c.PostForm("status_aktif") == "true"

	if idPendaftaran == 0 || namaLengkap == "" || tanggalDaftar == "" || idKursus == 0 || idBidang == 0 {
		redirectWithError(c, "/peserta", "Data peserta tidak valid")
		return
	}

	err := insertPeserta(Peserta{
		IDPendaftaran: idPendaftaran,
		NamaLengkap:   namaLengkap,
		TanggalDaftar: tanggalDaftar,
		IDKursus:      idKursus,
		IDBidang:      idBidang,
		StatusAktif:   statusAktif,
	})

	if err != nil {
		redirectWithError(c, "/peserta", "ID peserta sudah dipakai atau relasi kursus/bidang tidak valid")
		return
	}

	c.Redirect(http.StatusSeeOther, "/peserta")
}

func formEditPeserta(c *gin.Context) {
	id := atoi(c.Param("id"))

	peserta, err := getPesertaByID(id)
	if err != nil {
		c.String(http.StatusNotFound, "Peserta tidak ditemukan")
		return
	}

	kursus, err := getAllKursus()
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	bidang, err := getAllBidang()
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	c.HTML(http.StatusOK, "form_peserta.html", gin.H{
		"title":   "Edit Peserta",
		"mode":    "edit",
		"peserta": peserta,
		"kursus":  kursus,
		"bidang":  bidang,
	})
}

func updatePeserta(c *gin.Context) {
	id := atoi(c.Param("id"))

	namaLengkap := strings.TrimSpace(c.PostForm("nama_lengkap"))
	tanggalDaftar := c.PostForm("tanggal_daftar")
	idKursus := atoi(c.PostForm("id_kursus"))
	idBidang := atoi(c.PostForm("id_bidang"))
	statusAktif := c.PostForm("status_aktif") == "true"

	if id == 0 || namaLengkap == "" || tanggalDaftar == "" || idKursus == 0 || idBidang == 0 {
		redirectWithError(c, "/peserta", "Data peserta tidak valid")
		return
	}

	err := updatePesertaDB(Peserta{
		IDPendaftaran: id,
		NamaLengkap:   namaLengkap,
		TanggalDaftar: tanggalDaftar,
		IDKursus:      idKursus,
		IDBidang:      idBidang,
		StatusAktif:   statusAktif,
	})

	if err != nil {
		redirectWithError(c, "/peserta", "Gagal memperbarui data peserta")
		return
	}

	c.Redirect(http.StatusSeeOther, "/peserta")
}

func hapusPeserta(c *gin.Context) {
	id := atoi(c.Param("id"))

	err := deletePeserta(id)

	if err != nil {
		redirectWithError(c, "/peserta", "Gagal menghapus peserta")
		return
	}

	c.Redirect(http.StatusSeeOther, "/peserta")
}

func cariPesertaSequential(c *gin.Context) {
	keyword := strings.ToLower(strings.TrimSpace(c.Query("keyword")))
	bidang := atoi(c.Query("id_bidang"))

	peserta, err := getAllPeserta()
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	bidangList, err := getAllBidang()
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	hasil := []Peserta{}

	for _, p := range peserta {
		cocokNama := keyword == "" || strings.Contains(strings.ToLower(p.NamaLengkap), keyword)
		cocokBidang := bidang == 0 || p.IDBidang == bidang

		if cocokNama && cocokBidang {
			hasil = append(hasil, p)
		}
	}

	c.HTML(http.StatusOK, "peserta.html", gin.H{
		"title":   "Hasil Sequential Search",
		"peserta": hasil,
		"keyword": keyword,
		"bidang":  bidangList,
	})
}

func cariPesertaBinary(c *gin.Context) {
	keyword := strings.ToLower(strings.TrimSpace(c.Query("keyword")))

	peserta, err := getAllPeserta()
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	sorted := copyPeserta(peserta)
	insertionSortPesertaByNama(sorted)

	kiri := 0
	kanan := len(sorted) - 1
	hasil := []Peserta{}

	for kiri <= kanan {
		tengah := (kiri + kanan) / 2
		namaTengah := strings.ToLower(sorted[tengah].NamaLengkap)

		if namaTengah == keyword {
			hasil = append(hasil, sorted[tengah])
			break
		}

		if namaTengah < keyword {
			kiri = tengah + 1
		} else {
			kanan = tengah - 1
		}
	}

	c.HTML(http.StatusOK, "peserta.html", gin.H{
		"title":   "Hasil Binary Search",
		"peserta": hasil,
		"keyword": keyword,
	})
}

func sortPesertaByID(c *gin.Context) {
	peserta, err := getAllPeserta()
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	selectionSortPesertaByID(peserta)

	c.HTML(http.StatusOK, "peserta.html", gin.H{
		"title":   "Peserta Diurutkan Berdasarkan ID",
		"peserta": peserta,
	})
}

func sortPesertaByNama(c *gin.Context) {
	peserta, err := getAllPeserta()
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	insertionSortPesertaByNama(peserta)

	c.HTML(http.StatusOK, "peserta.html", gin.H{
		"title":   "Peserta Diurutkan Berdasarkan Nama",
		"peserta": peserta,
	})
}

func tampilKursus(c *gin.Context) {

	kursus, err := getAllKursus()

	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	c.HTML(http.StatusOK, "kursus.html", gin.H{
		"title":  "Data Kursus",
		"kursus": kursus,
	})
}

func formTambahKursus(c *gin.Context) {

	bidang, err := getAllBidang()

	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	c.HTML(http.StatusOK, "form_kursus.html", gin.H{
		"title":  "Tambah Kursus",
		"bidang": bidang,
	})
}

func tambahKursus(c *gin.Context) {
	idKursus := atoi(c.PostForm("id_kursus"))
	namaKursus := strings.TrimSpace(c.PostForm("nama_kursus"))
	idBidang := atoi(c.PostForm("id_bidang"))

	if idKursus == 0 || namaKursus == "" || idBidang == 0 {
		redirectWithError(c, "/kursus", "Data kursus tidak valid")
		return
	}

	err := insertKursus(Kursus{
		IDKursus:   idKursus,
		NamaKursus: namaKursus,
		IDBidang:   idBidang,
	})

	if err != nil {
		redirectWithError(c, "/kursus", "ID kursus sudah dipakai atau bidang tidak valid")
		return
	}

	c.Redirect(http.StatusSeeOther, "/kursus")
}

func hapusKursus(c *gin.Context) {
	id := atoi(c.Param("id"))

	err := deleteKursus(id)

	if err != nil {
		redirectWithError(c, "/kursus", "Gagal menghapus kursus")
		return
	}

	c.Redirect(http.StatusSeeOther, "/kursus")
}

func tampilBidang(c *gin.Context) {

	bidang, err := getAllBidang()

	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	c.HTML(http.StatusOK, "bidang.html", gin.H{
		"title":  "Data Bidang Minat",
		"bidang": bidang,
	})
}

func formTambahBidang(c *gin.Context) {
	c.HTML(http.StatusOK, "form_bidang.html", gin.H{"title": "Tambah Bidang Minat"})
}

func tambahBidang(c *gin.Context) {
	idBidang := atoi(c.PostForm("id_bidang"))
	namaBidang := strings.TrimSpace(c.PostForm("nama_bidang"))

	if idBidang == 0 || namaBidang == "" {
		redirectWithError(c, "/bidang", "ID bidang tidak valid atau nama kosong")
		return
	}

	err := insertBidang(BidangMinat{
		IDBidang:   idBidang,
		NamaBidang: namaBidang,
	})

	if err != nil {
		redirectWithError(c, "/bidang", "ID bidang sudah dipakai atau data tidak valid")
		return
	}

	c.Redirect(http.StatusSeeOther, "/bidang")
}

func hapusBidang(c *gin.Context) {
	id := atoi(c.Param("id"))

	err := deleteBidang(id)

	if err != nil {
		redirectWithError(c, "/bidang", "Gagal menghapus bidang")
		return
	}

	c.Redirect(http.StatusSeeOther, "/bidang")
}

func statistik(c *gin.Context) {
	totalPeserta, totalAktif, _, _, err := getDashboardStats()

	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	bidang, err := getAllBidang()

	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	jumlahPerBidang, err := getJumlahPesertaPerBidang()

	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	c.HTML(http.StatusOK, "statistik.html", gin.H{
		"title":           "Statistik Peserta",
		"totalPeserta":    totalPeserta,
		"totalAktif":      totalAktif,
		"bidang":          bidang,
		"jumlahPerBidang": jumlahPerBidang,
	})
}

func namaBidangByID(id int) string {
	var nama string

	err := db.QueryRow(
		context.Background(),
		"SELECT nama_bidang FROM bidang WHERE id_bidang = $1",
		id,
	).Scan(&nama)

	if err != nil {
		return "-"
	}

	return nama
}

func namaKursusByID(id int) string {
	var nama string

	err := db.QueryRow(
		context.Background(),
		"SELECT nama_kursus FROM kursus WHERE id_kursus = $1",
		id,
	).Scan(&nama)

	if err != nil {
		return "-"
	}

	return nama
}

func selectionSortPesertaByID(data []Peserta) {
	for i := 0; i < len(data)-1; i++ {
		min := i
		for j := i + 1; j < len(data); j++ {
			if data[j].IDPendaftaran < data[min].IDPendaftaran {
				min = j
			}
		}
		data[i], data[min] = data[min], data[i]
	}
}

func insertionSortPesertaByNama(data []Peserta) {
	for i := 1; i < len(data); i++ {
		key := data[i]
		j := i - 1
		for j >= 0 && strings.ToLower(data[j].NamaLengkap) > strings.ToLower(key.NamaLengkap) {
			data[j+1] = data[j]
			j--
		}
		data[j+1] = key
	}
}

func copyPeserta(src []Peserta) []Peserta {
	dst := make([]Peserta, len(src))
	copy(dst, src)
	return dst
}

func atoi(s string) int {
	n, _ := strconv.Atoi(strings.TrimSpace(s))
	return n
}

func redirectWithError(c *gin.Context, path string, message string) {
	c.Redirect(http.StatusSeeOther, path+"?error="+message)
}
