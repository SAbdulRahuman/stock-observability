package waitlist

import (
	"encoding/json"
	"errors"
	"log/slog"
	"mime"
	"net/http"
	"net/mail"
	"net/url"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/seenimo/stock_website/internal/ratelimit"
)

// Interests are the accepted values of the interest field.
var Interests = []string{"", "investor", "trader", "developer", "other"}

const maxBody = 8 << 10

// Handler serves POST /api/site/waitlist.
type Handler struct {
	Store  *Store
	Limit  *ratelimit.Limiter
	Logger *slog.Logger
	// OnAdd is called after a new email is stored.
	OnAdd func()
	Now   func() time.Time
}

type request struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Interest string `json:"interest"`
	Message  string `json:"message"`
	// Website is a honeypot: hidden from people, filled in by bots.
	Website string `json:"website"`
}

type fieldError struct{ field, msg string }

func (e fieldError) Error() string { return e.msg }

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !sameOrigin(r) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "Cross-origin request rejected."})
		return
	}
	if ct, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type")); ct != "application/json" {
		writeJSON(w, http.StatusUnsupportedMediaType, map[string]string{"error": "Content-Type must be application/json."})
		return
	}
	if h.Limit != nil && !h.Limit.Allow(ratelimit.ClientIP(r)) {
		w.Header().Set("Retry-After", "60")
		writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "Too many requests. Please try again in a minute."})
		return
	}
	var req request
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBody))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body."})
		return
	}
	if req.Website != "" {
		h.Logger.Info("Waitlist honeypot triggered")
		writeJSON(w, http.StatusAccepted, map[string]bool{"ok": true})
		return
	}
	e, err := req.entry()
	if err != nil {
		var fe fieldError
		errors.As(err, &fe)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": fe.msg, "field": fe.field})
		return
	}
	e.CreatedAt = h.now()
	added, err := h.Store.Add(r.Context(), e)
	if err != nil {
		h.Logger.Error("Waitlist insert failed", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Could not save your sign-up. Please try again later."})
		return
	}
	if added && h.OnAdd != nil {
		h.OnAdd()
	}
	// Same answer for repeats, so the endpoint does not reveal who signed up.
	writeJSON(w, http.StatusAccepted, map[string]bool{"ok": true})
}

func (h *Handler) now() time.Time {
	if h.Now != nil {
		return h.Now()
	}
	return time.Now()
}

func (req request) entry() (Entry, error) {
	e := Entry{
		Email:    strings.TrimSpace(req.Email),
		Name:     strings.TrimSpace(req.Name),
		Interest: strings.TrimSpace(req.Interest),
		Message:  strings.TrimSpace(req.Message),
	}
	if !validEmail(e.Email) {
		return e, fieldError{"email", "Please enter a valid email address."}
	}
	if utf8.RuneCountInString(e.Name) > 100 || strings.ContainsFunc(e.Name, unicode.IsControl) {
		return e, fieldError{"name", "Name must be at most 100 characters on one line."}
	}
	if !contains(Interests, e.Interest) {
		return e, fieldError{"interest", "Unknown interest."}
	}
	if utf8.RuneCountInString(e.Message) > 1000 || strings.ContainsFunc(e.Message, func(r rune) bool { return unicode.IsControl(r) && r != '\n' && r != '\r' && r != '\t' }) {
		return e, fieldError{"message", "Message must be at most 1000 characters."}
	}
	return e, nil
}

func validEmail(s string) bool {
	if s == "" || len(s) > 254 {
		return false
	}
	a, err := mail.ParseAddress(s)
	if err != nil || a.Name != "" || a.Address != s {
		return false
	}
	_, domain, _ := strings.Cut(s, "@")
	return strings.Contains(domain, ".") && !strings.HasPrefix(domain, ".") && !strings.HasSuffix(domain, ".")
}

// sameOrigin rejects browser requests from other sites; non-browser clients send no Origin.
func sameOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	u, err := url.Parse(origin)
	return err == nil && strings.EqualFold(u.Host, r.Host)
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
