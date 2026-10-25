package httpapi

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/jhopan/PanWallet/internal/export"
)

// adminExport menampilkan halaman ekspor CSV.
func (server *Server) adminExport(response http.ResponseWriter, request *http.Request) {
	data := map[string]any{
		"Title":  "Ekspor Data",
		"Tab":    "export",
		"Tables": export.Tables,
	}
	if server.deps.Database != nil {
		if text, err := export.Ringkasan(server.deps.Database); err == nil {
			data["Ringkasan"] = text
		}
	}
	server.renderer.Render("export", response, data)
}

// adminExportRun menjalankan ekspor semua tabel.
func (server *Server) adminExportRun(response http.ResponseWriter, request *http.Request) {
	if server.deps.Database == nil {
		redirectWith(response, request, "/admin/export", "err", "database belum siap")
		return
	}
	result, err := export.ExportAll(server.deps.Database, "export")
	if err != nil {
		redirectWith(response, request, "/admin/export", "err", err.Error())
		return
	}
	total := 0
	for _, count := range result {
		total += count
	}
	redirectWith(response, request, "/admin/export", "ok",
		"Ekspor selesai: "+itoa(total)+" baris di folder export/")
}

// adminExportDownload mengirim satu berkas CSV.
func (server *Server) adminExportDownload(response http.ResponseWriter, request *http.Request) {
	name := filepath.Base(request.URL.Query().Get("file"))
	if name == "" || filepath.Ext(name) != ".csv" {
		http.Error(response, "berkas tidak valid", http.StatusBadRequest)
		return
	}
	path := filepath.Join("export", name)
	if _, err := os.Stat(path); err != nil {
		http.Error(response, "berkas tidak ditemukan", http.StatusNotFound)
		return
	}
	response.Header().Set("Content-Type", "text/csv; charset=utf-8")
	response.Header().Set("Content-Disposition", `attachment; filename="`+name+`"`)
	http.ServeFile(response, request, path)
}

// itoa sederhana untuk menyusun pesan.
func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	var digits []byte
	for value > 0 {
		digits = append([]byte{byte('0' + value%10)}, digits...)
		value /= 10
	}
	return string(digits)
}
