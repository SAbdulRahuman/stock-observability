package waitlist

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/seenimo/stock_website/internal/ratelimit"
)

func newHandler(t *testing.T) (*Handler, *int) {
	t.Helper()
	st, err := Open(filepath.Join(t.TempDir(), "site.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	added := 0
	return &Handler{
		Store:  st,
		Limit:  ratelimit.New(0, 100),
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		OnAdd:  func() { added++ },
		Now:    func() time.Time { return time.Date(2026, 10, 10, 9, 15, 0, 0, time.UTC) },
	}, &added
}

func post(h http.Handler, body string, hdr map[string]string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodPost, "http://isee.test/api/site/waitlist", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	for k, v := range hdr {
		r.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	return rec
}

func TestSignup(t *testing.T) {
	h, added := newHandler(t)
	body := `{"email":" Asha@Example.com ","name":"Asha","interest":"trader","message":"Hi"}`
	if rec := post(h, body, map[string]string{"Origin": "http://isee.test"}); rec.Code != http.StatusAccepted {
		t.Fatalf("code %d: %s", rec.Code, rec.Body)
	}
	// A repeat with different case gets the same answer and is not stored again.
	if rec := post(h, `{"email":"asha@example.COM"}`, nil); rec.Code != http.StatusAccepted {
		t.Fatalf("repeat code %d", rec.Code)
	}
	if *added != 1 {
		t.Fatalf("added %d, want 1", *added)
	}
	list, err := h.Store.List(context.Background())
	if err != nil || len(list) != 1 || list[0].Email != "Asha@Example.com" || list[0].Interest != "trader" {
		t.Fatalf("list %+v, %v", list, err)
	}
}

func TestRejects(t *testing.T) {
	h, added := newHandler(t)
	for _, tc := range []struct {
		name, body string
		hdr        map[string]string
		code       int
	}{
		{"bad email", `{"email":"nope"}`, nil, 400},
		{"no tld", `{"email":"a@localhost"}`, nil, 400},
		{"display name", `{"email":"Bob <b@example.com>"}`, nil, 400},
		{"long name", `{"email":"a@example.com","name":"` + strings.Repeat("x", 101) + `"}`, nil, 400},
		{"control in name", `{"email":"a@example.com","name":"a\u0007"}`, nil, 400},
		{"interest", `{"email":"a@example.com","interest":"admin"}`, nil, 400},
		{"long message", `{"email":"a@example.com","message":"` + strings.Repeat("x", 1001) + `"}`, nil, 400},
		{"unknown field", `{"email":"a@example.com","admin":true}`, nil, 400},
		{"cross origin", `{"email":"a@example.com"}`, map[string]string{"Origin": "https://evil.example"}, 403},
		{"form post", `email=a@example.com`, map[string]string{"Content-Type": "application/x-www-form-urlencoded"}, 415},
		{"too large", `{"email":"a@example.com","message":"` + strings.Repeat("x", maxBody) + `"}`, nil, 400},
	} {
		if rec := post(h, tc.body, tc.hdr); rec.Code != tc.code {
			t.Errorf("%s: code %d, want %d: %s", tc.name, rec.Code, tc.code, rec.Body)
		}
	}
	if *added != 0 {
		t.Fatalf("added %d", *added)
	}
}

func TestHoneypot(t *testing.T) {
	h, added := newHandler(t)
	if rec := post(h, `{"email":"bot@example.com","website":"http://spam"}`, nil); rec.Code != http.StatusAccepted {
		t.Fatalf("code %d", rec.Code)
	}
	if *added != 0 {
		t.Fatal("honeypot sign-up stored")
	}
}

func TestRateLimit(t *testing.T) {
	h, _ := newHandler(t)
	h.Limit = ratelimit.New(0, 2)
	codes := []int{}
	for range 3 {
		codes = append(codes, post(h, `{"email":"a@example.com"}`, nil).Code)
	}
	if codes[2] != http.StatusTooManyRequests {
		t.Fatalf("codes %v", codes)
	}
}

func TestExport(t *testing.T) {
	at := time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC)
	var buf bytes.Buffer
	if err := Export(&buf, []Entry{{Email: "a@example.com", Name: "=cmd()", CreatedAt: at}}, "csv"); err != nil {
		t.Fatal(err)
	}
	want := "created_at,email,name,interest,message\n2026-10-10T00:00:00Z,a@example.com,'=cmd(),,\n"
	if buf.String() != want {
		t.Fatalf("got %q", buf.String())
	}
	buf.Reset()
	if err := Export(&buf, nil, "json"); err != nil || strings.TrimSpace(buf.String()) != "[]" {
		t.Fatalf("json %q %v", buf.String(), err)
	}
	if err := Export(&buf, nil, "xml"); err == nil {
		t.Fatal("want error for unknown format")
	}
}
