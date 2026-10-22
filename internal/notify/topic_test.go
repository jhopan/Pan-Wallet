package notify

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTelegramTopicSendsThreadID(t *testing.T) {
	var gotThread string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		gotThread = r.FormValue("message_thread_id")
		w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	client := NewTelegram("T", "999", server.URL).WithTopic("12")
	if err := client.Send(context.Background(), "Top-up berhasil"); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if gotThread != "12" {
		t.Fatalf("message_thread_id = %q, want 12", gotThread)
	}
}

func TestTelegramWithoutTopicOmitsThread(t *testing.T) {
	var gotThread string = "belum-diset"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		gotThread = r.FormValue("message_thread_id")
		w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	client := NewTelegram("T", "999", server.URL)
	client.Send(context.Background(), "halo")
	if gotThread != "" {
		t.Fatalf("message_thread_id harus kosong, dapat %q", gotThread)
	}
}
