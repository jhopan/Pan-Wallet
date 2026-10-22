package notify

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTelegramSendsMessage(t *testing.T) {
	var gotPath, gotText, gotChat string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		r.ParseForm()
		gotText = r.FormValue("text")
		gotChat = r.FormValue("chat_id")
		w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	client := NewTelegram("TOKEN123", "999", server.URL)
	if err := client.Send(context.Background(), "Top-up masuk Rp50.001"); err != nil {
		t.Fatalf("Send: %v", err)
	}

	if !strings.Contains(gotPath, "/botTOKEN123/sendMessage") {
		t.Fatalf("path = %q", gotPath)
	}
	if gotChat != "999" {
		t.Fatalf("chat_id = %q", gotChat)
	}
	if !strings.Contains(gotText, "50.001") {
		t.Fatalf("text = %q", gotText)
	}
}

func TestTelegramSkipsWhenNotConfigured(t *testing.T) {
	client := NewTelegram("", "", "")
	if err := client.Send(context.Background(), "halo"); err != nil {
		t.Fatalf("tanpa token tidak boleh error: %v", err)
	}
}

func TestTelegramMultipleChats(t *testing.T) {
	var chats []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		chats = append(chats, r.FormValue("chat_id"))
		w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	client := NewTelegram("T", "111, 222", server.URL)
	if err := client.Send(context.Background(), "halo"); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if len(chats) != 2 {
		t.Fatalf("chat terkirim = %d, want 2", len(chats))
	}
}

func TestTelegramReportsHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{"ok": false, "description": "chat not found"})
	}))
	defer server.Close()

	client := NewTelegram("T", "999", server.URL)
	if err := client.Send(context.Background(), "x"); err == nil {
		t.Fatal("harus melaporkan error")
	}
}

func TestWebhookFormatContainsEvent(t *testing.T) {
	body := WebhookBody("topup.paid", map[string]any{
		"public_id": "ABC12345",
		"amount":    50001,
	})
	var decoded map[string]any
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("bukan JSON: %v", err)
	}
	if decoded["event"] != "topup.paid" {
		t.Fatalf("event = %v", decoded["event"])
	}
}
