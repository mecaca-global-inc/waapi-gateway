package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mecaca/waapi-gateway/internal/store"
)

func TestEnvelopeJSON(t *testing.T) {
	ev := Envelope{
		Event:   EventMessage,
		Session: "default",
		Time:    1700000000,
		Payload: map[string]any{"id": "abc"},
	}
	b, err := json.Marshal(ev)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `{"event":"message","session":"default","timestamp":1700000000,"payload":{"id":"abc"}}` {
		t.Fatalf("unexpected json: %s", b)
	}
}

func TestSendWithHMAC(t *testing.T) {
	var hits atomic.Int32
	var gotSig string
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		gotSig = r.Header.Get("X-Webhook-Signature")
		gotBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	d := &Dispatcher{client: &http.Client{Timeout: time.Second}}
	body := []byte(`{"event":"message"}`)
	d.send(store.Webhook{URL: srv.URL, Secret: "supersecret", Enabled: true}, body)

	if hits.Load() != 1 {
		t.Fatalf("expected 1 hit, got %d", hits.Load())
	}
	mac := hmac.New(sha256.New, []byte("supersecret"))
	mac.Write(body)
	want := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	if gotSig != want {
		t.Fatalf("signature mismatch:\n got %s\nwant %s", gotSig, want)
	}
	if string(gotBody) != string(body) {
		t.Fatalf("body mismatch: %s", gotBody)
	}
}

func TestSendRetriesOn500(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	d := &Dispatcher{client: &http.Client{Timeout: time.Second}}
	start := time.Now()
	d.send(store.Webhook{URL: srv.URL, Enabled: true}, []byte(`{}`))
	elapsed := time.Since(start)

	if hits.Load() != 3 {
		t.Fatalf("expected 3 retries, got %d", hits.Load())
	}
	if elapsed < 6*time.Second {
		t.Fatalf("retry backoff too short: %v", elapsed)
	}
}

func TestContainsFilter(t *testing.T) {
	if !contains([]string{"a", "b", "c"}, "b") {
		t.Fatal("should contain b")
	}
	if contains([]string{"a", "b"}, "z") {
		t.Fatal("should not contain z")
	}
}
