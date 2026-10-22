package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

// Pinger memeriksa apakah database masih bisa dihubungi.
type Pinger interface {
	Ping(context.Context) error
}

type healthComponent struct {
	Name    string `json:"name"`
	OK      bool   `json:"ok"`
	Message string `json:"message"`
}

// healthReport adalah bentuk jawaban /healthz.
type healthReport struct {
	Status     string            `json:"status"`
	Components []healthComponent `json:"components"`
}

// healthDetail memeriksa database dan website, lalu menjawab 200 hanya bila
// semuanya sehat. Dipakai monitoring supaya tahu masalah sebelum pelanggan.
func (server *Server) healthDetail(response http.ResponseWriter, request *http.Request) {
	ctx, cancel := context.WithTimeout(request.Context(), 3*time.Second)
	defer cancel()

	components := []healthComponent{}

	databaseOK := true
	databaseMessage := "ok"
	if server.deps.Pinger == nil {
		databaseOK = false
		databaseMessage = "database belum terhubung"
	} else if err := server.deps.Pinger.Ping(ctx); err != nil {
		databaseOK = false
		databaseMessage = "database tidak merespons"
	}
	components = append(components, healthComponent{Name: "database", OK: databaseOK, Message: databaseMessage})

	websiteOK := server.renderer != nil && len(server.renderer.pages) > 0
	websiteMessage := "ok"
	if !websiteOK {
		websiteMessage = "template tidak termuat"
	}
	components = append(components, healthComponent{Name: "website", OK: websiteOK, Message: websiteMessage})

	paypanOK := false
	paypanMessage := "belum dikonfigurasi"
	if server.deps.Settings != nil {
		base, _ := server.deps.Settings.Setting(ctx, "paypan_base_url")
		token, _ := server.deps.Settings.Setting(ctx, "paypan_token")
		if base != "" && token != "" {
			paypanOK = true
			paypanMessage = "terkonfigurasi"
		}
	}
	components = append(components, healthComponent{Name: "paypan", OK: paypanOK, Message: paypanMessage})

	report := healthReport{Status: "ok", Components: components}
	status := http.StatusOK
	if !databaseOK || !websiteOK {
		report.Status = "degraded"
		status = http.StatusServiceUnavailable
	}

	response.Header().Set("Content-Type", "application/json; charset=utf-8")
	response.WriteHeader(status)
	json.NewEncoder(response).Encode(report)
}
