package httpapi

import (
	"context"
	"log"
	"net/http"
	"runtime"
	"time"
)

// statusPage adalah halaman pantau publik /status.
// Isinya sengaja TIDAK memuat saldo, public_id, atau data pelanggan —
// hanya kesehatan sistem, supaya aman dibuka tanpa login.
type statusView struct {
	// Field yang diwajibkan layout admin.
	Tab        string
	User       string
	Flash      string
	Error      string
	Title      string
	Overall    string
	OverallOK  bool
	CheckedAt  string
	Components []statusComponent
	Facts      []statusFact
	Uptime     string
	GoVersion  string
}

type statusComponent struct {
	Name     string
	OK       bool
	Message  string
	Detail   string
	Latency  string
	Critical bool
}

type statusFact struct {
	Label string
	Value string
}

// statusPage merender halaman status publik.
func (server *Server) statusPage(response http.ResponseWriter, request *http.Request) {
	ctx, cancel := context.WithTimeout(request.Context(), 5*time.Second)
	defer cancel()

	view := statusView{
		Title:     "Status PanWallet",
		CheckedAt: time.Now().Format("02 Jan 2006, 15:04:05 WIB"),
		GoVersion: runtime.Version(),
		Uptime:    time.Since(server.startedAt).Round(time.Second).String(),
	}

	allOK := true

	// 1. Database
	database := statusComponent{Name: "Database", Critical: true}
	if server.deps.Pinger == nil {
		database.Message = "belum terhubung"
	} else if err := server.deps.Pinger.Ping(ctx); err != nil {
		database.Message = "tidak merespons"
		database.Detail = err.Error()
	} else {
		database.OK = true
		database.Message = "normal"
	}
	if !database.OK {
		allOK = false
	}
	view.Components = append(view.Components, database)

	// 2. Tampilan / template
	website := statusComponent{Name: "Tampilan web", Critical: true}
	if server.renderer != nil && len(server.renderer.pages) > 0 {
		website.OK = true
		website.Message = "normal"
		website.Detail = itoa(len(server.renderer.pages)) + " halaman termuat"
	} else {
		website.Message = "template gagal dimuat"
	}
	if !website.OK {
		allOK = false
	}
	view.Components = append(view.Components, website)

	// 3. Gateway pembayaran (opsional: mati tidak mematikan sistem)
	paypan := statusComponent{Name: "Gateway pembayaran", Critical: false}
	if server.deps.Settings == nil {
		paypan.Message = "belum diatur"
	} else {
		base, _ := server.deps.Settings.Setting(ctx, "paypan_base_url")
		token, _ := server.deps.Settings.Setting(ctx, "paypan_token")
		if base == "" || token == "" {
			paypan.Message = "belum diatur"
			paypan.Detail = "isi di panel Konfigurasi untuk menerima top-up"
		} else if server.paypanPinger != nil {
			result := server.paypanPinger.Ping(ctx)
			paypan.OK = result.OK
			paypan.Message = result.Message
			paypan.Latency = itoa(int(result.LatencyMS)) + " ms"
			if result.Version != "" {
				paypan.Detail = "versi " + result.Version
			}
		} else {
			paypan.OK = true
			paypan.Message = "terkonfigurasi"
		}
	}
	view.Components = append(view.Components, paypan)

	// 4. Notifikasi (opsional)
	notify := statusComponent{Name: "Notifikasi", Critical: false}
	if server.deps.Settings != nil {
		token, _ := server.deps.Settings.Setting(ctx, "notify_tg_token")
		chat, _ := server.deps.Settings.Setting(ctx, "notify_tg_chat")
		hook, _ := server.deps.Settings.Setting(ctx, "notify_webhook_url")
		switch {
		case token != "" && chat != "":
			notify.OK = true
			notify.Message = "Telegram aktif"
		case hook != "":
			notify.OK = true
			notify.Message = "webhook aktif"
		default:
			notify.Message = "belum diatur"
		}
	}
	view.Components = append(view.Components, notify)

	// 5. Backup
	backup := statusComponent{Name: "Backup otomatis", Critical: false}
	if server.deps.Backup != nil {
		if items, err := server.deps.Backup.List(); err == nil {
			if len(items) == 0 {
				backup.Message = "belum ada berkas"
			} else {
				backup.OK = true
				backup.Message = "normal"
				backup.Detail = itoa(len(items)) + " berkas, terbaru " + items[0].CreatedAt.Format("02 Jan 15:04")
			}
		} else {
			backup.Message = "gagal membaca folder"
		}
	} else {
		backup.Message = "tidak aktif"
	}
	view.Components = append(view.Components, backup)

	// Ringkasan
	if allOK {
		view.Overall = "Semua sistem normal"
		view.OverallOK = true
	} else {
		view.Overall = "Ada gangguan"
		view.OverallOK = false
	}

	view.Facts = []statusFact{
		{Label: "Versi Go", Value: view.GoVersion},
		{Label: "Waktu server", Value: view.CheckedAt},
		{Label: "Lama berjalan", Value: view.Uptime},
	}

	if err := server.renderer.Render("status", response, view); err != nil {
		// Tanpa ini, template yang gagal hanya menghasilkan halaman terpotong
		// dan tidak ada jejak apa pun di log.
		log.Printf("render status gagal: %v", err)
	}
}

// statusJSON memberi bentuk mesin untuk monitoring (uptime robot, dll).
func (server *Server) statusJSON(response http.ResponseWriter, request *http.Request) {
	server.healthDetail(response, request)
}
