package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakePinger struct{ err error }

func (p fakePinger) Ping(context.Context) error { return p.err }

func TestHealthReportsAllComponents(t *testing.T) {
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)

	New(Deps{Pinger: fakePinger{}}).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}

	var report healthReport
	if err := json.Unmarshal(response.Body.Bytes(), &report); err != nil {
		t.Fatalf("body bukan JSON: %v", err)
	}
	if report.Status != "ok" {
		t.Fatalf("status = %q, want ok", report.Status)
	}

	seen := map[string]bool{}
	for _, component := range report.Components {
		seen[component.Name] = true
		if component.Name == "database" && !component.OK {
			t.Fatal("database harus OK")
		}
	}
	for _, want := range []string{"database", "website"} {
		if !seen[want] {
			t.Fatalf("komponen %q tidak dilaporkan", want)
		}
	}
}

func TestHealthFailsWhenDatabaseDown(t *testing.T) {
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)

	New(Deps{Pinger: fakePinger{err: context.DeadlineExceeded}}).ServeHTTP(response, request)

	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503 saat database mati", response.Code)
	}
}
