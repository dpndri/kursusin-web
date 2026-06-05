package main

import (
	"encoding/json"
	"errors"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

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

type AppData struct {
	Peserta []Peserta     `json:"peserta"`
	Kursus  []Kursus      `json:"kursus"`
	Bidang  []BidangMinat `json:"bidang"`
}

var (
	storePath = filepath.Join("data", "kursusin.json")
	appData   = AppData{}
	mu        sync.Mutex
)

func main() {
	mustLoadData()

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

func mustLoadData() {
	mu.Lock()
	defer mu.Unlock()

	_ = os.MkdirAll(filepath.Dir(storePath), 0755)

	bytes, err := os.ReadFile(storePath)
	if errors.Is(err, os.ErrNotExist) {
		appData = AppData{}
		mustSaveDataLocked()
		return
	}
	if err != nil {
		panic(err)
	}
	if len(strings.TrimSpace(string(bytes))) == 0 {
		appData = AppData{}
		return
	}
	if err := json.Unmarshal(bytes, &appData); err != nil {
		panic(err)
	}
}

func mustSaveDataLocked() {
	bytes, err := json.MarshalIndent(appData, "", "  ")
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(storePath, bytes, 0644); err != nil {
		panic(err)
	}
}

func saveData() {
	mu.Lock()
	defer mu.Unlock()
	mustSaveDataLocked()
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
	totalAktif := 0
	for _, p := range appData.Peserta {
		if p.StatusAktif {
			totalAktif++
		}
	}
	c.HTML(http.StatusOK, "index.html", gin.H{
		"title":        "Dashboard KursusIn",
		"totalPeserta": len(appData.Peserta),
		"totalAktif":   totalAktif,
		"totalKursus":  len(appData.Kursus),
		"totalBidang":  len(appData.Bidang),
	})
}

func tampilPeserta(c *gin.Context) {
	c.HTML(http.StatusOK, "peserta.html", gin.H{
		"title":   "Data Peserta",
		"peserta": appData.Peserta,
	})
}

func formTambahPeserta(c *gin.Context) {
	c.HTML(http.StatusOK, "form_peserta.html", gin.H{
		"title":   "Tambah Peserta",
		"mode":    "tambah",
		"peserta": Peserta{StatusAktif: true},
		"kursus":  appData.Kursus,
		"bidang":  appData.Bidang,
	})
}

func tambahPeserta(c *gin.Context) {
	idPendaftaran := atoi(c.PostForm("id_pendaftaran"))
	if idPendaftaran == 0 || indexPesertaByID(idPendaftaran) != -1 {
		redirectWithError(c, "/peserta", "ID pendaftaran tidak valid atau sudah dipakai")
		return
	}

	p := Peserta{
		IDPendaftaran: idPendaftaran,
		NamaLengkap:   strings.TrimSpace(c.PostForm("nama_lengkap")),
		TanggalDaftar: c.PostForm("tanggal_daftar"),
		IDKursus:      atoi(c.PostForm("id_kursus")),
		IDBidang:      atoi(c.PostForm("id_bidang")),
		StatusAktif:   c.PostForm("status_aktif") == "true",
	}
	appData.Peserta = append(appData.Peserta, p)
	saveData()
	c.Redirect(http.StatusSeeOther, "/peserta")
}

func formEditPeserta(c *gin.Context) {
	id := atoi(c.Param("id"))
	idx := indexPesertaByID(id)
	if idx == -1 {
		c.String(http.StatusNotFound, "Peserta tidak ditemukan")
		return
	}
	c.HTML(http.StatusOK, "form_peserta.html", gin.H{
		"title":   "Edit Peserta",
		"mode":    "edit",
		"peserta": appData.Peserta[idx],
		"kursus":  appData.Kursus,
		"bidang":  appData.Bidang,
	})
}

func updatePeserta(c *gin.Context) {
	id := atoi(c.Param("id"))
	idx := indexPesertaByID(id)
	if idx == -1 {
		c.String(http.StatusNotFound, "Peserta tidak ditemukan")
		return
	}
	appData.Peserta[idx].NamaLengkap = strings.TrimSpace(c.PostForm("nama_lengkap"))
	appData.Peserta[idx].TanggalDaftar = c.PostForm("tanggal_daftar")
	appData.Peserta[idx].IDKursus = atoi(c.PostForm("id_kursus"))
	appData.Peserta[idx].IDBidang = atoi(c.PostForm("id_bidang"))
	appData.Peserta[idx].StatusAktif = c.PostForm("status_aktif") == "true"
	saveData()
	c.Redirect(http.StatusSeeOther, "/peserta")
}

func hapusPeserta(c *gin.Context) {
	id := atoi(c.Param("id"))
	idx := indexPesertaByID(id)
	if idx == -1 {
		c.String(http.StatusNotFound, "Peserta tidak ditemukan")
		return
	}
	appData.Peserta = append(appData.Peserta[:idx], appData.Peserta[idx+1:]...)
	saveData()
	c.Redirect(http.StatusSeeOther, "/peserta")
}

func cariPesertaSequential(c *gin.Context) {
	keyword := strings.ToLower(strings.TrimSpace(c.Query("keyword")))
	bidang := atoi(c.Query("id_bidang"))
	hasil := []Peserta{}

	for _, p := range appData.Peserta {
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
		"bidang":  appData.Bidang,
	})
}

func cariPesertaBinary(c *gin.Context) {
	keyword := strings.ToLower(strings.TrimSpace(c.Query("keyword")))
	sorted := copyPeserta(appData.Peserta)
	insertionSortPesertaByNama(sorted)

	kiri, kanan := 0, len(sorted)-1
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
	selectionSortPesertaByID(appData.Peserta)
	saveData()
	c.Redirect(http.StatusSeeOther, "/peserta")
}

func sortPesertaByNama(c *gin.Context) {
	insertionSortPesertaByNama(appData.Peserta)
	saveData()
	c.Redirect(http.StatusSeeOther, "/peserta")
}

func tampilKursus(c *gin.Context) {
	c.HTML(http.StatusOK, "kursus.html", gin.H{"title": "Data Kursus", "kursus": appData.Kursus})
}

func formTambahKursus(c *gin.Context) {
	c.HTML(http.StatusOK, "form_kursus.html", gin.H{"title": "Tambah Kursus", "bidang": appData.Bidang})
}

func tambahKursus(c *gin.Context) {
	id := atoi(c.PostForm("id_kursus"))
	if id == 0 || indexKursusByID(id) != -1 {
		redirectWithError(c, "/kursus", "ID kursus tidak valid atau sudah dipakai")
		return
	}
	appData.Kursus = append(appData.Kursus, Kursus{
		IDKursus:   id,
		NamaKursus: strings.TrimSpace(c.PostForm("nama_kursus")),
		IDBidang:   atoi(c.PostForm("id_bidang")),
	})
	saveData()
	c.Redirect(http.StatusSeeOther, "/kursus")
}

func hapusKursus(c *gin.Context) {
	id := atoi(c.Param("id"))
	idx := indexKursusByID(id)
	if idx != -1 {
		appData.Kursus = append(appData.Kursus[:idx], appData.Kursus[idx+1:]...)
		saveData()
	}
	c.Redirect(http.StatusSeeOther, "/kursus")
}

func tampilBidang(c *gin.Context) {
	c.HTML(http.StatusOK, "bidang.html", gin.H{"title": "Data Bidang Minat", "bidang": appData.Bidang})
}

func formTambahBidang(c *gin.Context) {
	c.HTML(http.StatusOK, "form_bidang.html", gin.H{"title": "Tambah Bidang Minat"})
}

func tambahBidang(c *gin.Context) {
	id := atoi(c.PostForm("id_bidang"))
	if id == 0 || indexBidangByID(id) != -1 {
		redirectWithError(c, "/bidang", "ID bidang tidak valid atau sudah dipakai")
		return
	}
	appData.Bidang = append(appData.Bidang, BidangMinat{
		IDBidang:   id,
		NamaBidang: strings.TrimSpace(c.PostForm("nama_bidang")),
	})
	saveData()
	c.Redirect(http.StatusSeeOther, "/bidang")
}

func hapusBidang(c *gin.Context) {
	id := atoi(c.Param("id"))
	idx := indexBidangByID(id)
	if idx != -1 {
		appData.Bidang = append(appData.Bidang[:idx], appData.Bidang[idx+1:]...)
		saveData()
	}
	c.Redirect(http.StatusSeeOther, "/bidang")
}

func statistik(c *gin.Context) {
	totalAktif := 0
	jumlahPerBidang := map[int]int{}
	for _, p := range appData.Peserta {
		if p.StatusAktif {
			totalAktif++
		}
		jumlahPerBidang[p.IDBidang]++
	}
	c.HTML(http.StatusOK, "statistik.html", gin.H{
		"title":           "Statistik Peserta",
		"totalPeserta":    len(appData.Peserta),
		"totalAktif":      totalAktif,
		"jumlahPerBidang": jumlahPerBidang,
		"bidang":          appData.Bidang,
	})
}

func indexPesertaByID(id int) int {
	for i, p := range appData.Peserta {
		if p.IDPendaftaran == id {
			return i
		}
	}
	return -1
}

func indexKursusByID(id int) int {
	for i, k := range appData.Kursus {
		if k.IDKursus == id {
			return i
		}
	}
	return -1
}

func indexBidangByID(id int) int {
	for i, b := range appData.Bidang {
		if b.IDBidang == id {
			return i
		}
	}
	return -1
}

func namaBidangByID(id int) string {
	for _, b := range appData.Bidang {
		if b.IDBidang == id {
			return b.NamaBidang
		}
	}
	return "-"
}

func namaKursusByID(id int) string {
	for _, k := range appData.Kursus {
		if k.IDKursus == id {
			return k.NamaKursus
		}
	}
	return "-"
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
