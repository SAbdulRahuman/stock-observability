package platform

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// Prefix is the URL prefix of all commands.
const Prefix = "/api/v2/"

const maxBodyBytes = 10 << 20

// OBBject is the response envelope, mirroring openbb_core.app.model.obbject.OBBject.
type OBBject struct {
	Results  any            `json:"results"`
	Provider *string        `json:"provider"`
	Warnings []Warning      `json:"warnings"`
	Chart    any            `json:"chart"`
	Extra    map[string]any `json:"extra"`
}

// CommandInfo is the public metadata of a command served at /api/v2/commands.
type CommandInfo struct {
	Path        string   `json:"path"`
	Method      string   `json:"method"`
	Description string   `json:"description"`
	Model       string   `json:"model,omitempty"`
	Providers   []string `json:"providers"`
	Params      []Param  `json:"params"`
}

// Server serves the registry over HTTP.
type Server struct {
	reg      *Registry
	logger   *slog.Logger
	timeout  time.Duration
	now      func() time.Time
	requests *prometheus.CounterVec
}

// NewServer returns a Server; timeout applies to commands that do not set their own.
func NewServer(reg *Registry, logger *slog.Logger, timeout time.Duration, promReg prometheus.Registerer) *Server {
	s := &Server{
		reg:     reg,
		logger:  logger,
		timeout: timeout,
		now:     time.Now,
		requests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "stock_home_command_requests_total",
			Help: "Calls of /api/v2 commands by HTTP status code.",
		}, []string{"command", "code"}),
	}
	promReg.MustRegister(s.requests)
	return s
}

// Register adds the command routes to mux. Commands registered later are not served.
func (s *Server) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET "+Prefix+"commands", s.listCommands)
	for _, c := range s.reg.Commands() {
		mux.Handle(c.Method+" "+Prefix+c.Path, s.handler(c))
	}
	unknown := func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "unknown command"})
	}
	mux.HandleFunc("GET "+Prefix, unknown)
	mux.HandleFunc("POST "+Prefix, unknown)
}

func (s *Server) listCommands(w http.ResponseWriter, _ *http.Request) {
	cmds := s.reg.Commands()
	out := make([]CommandInfo, 0, len(cmds))
	for _, c := range cmds {
		out = append(out, CommandInfo{
			Path:        c.Path,
			Method:      c.Method,
			Description: c.Description,
			Model:       c.Model,
			Providers:   nonNil(c.Providers),
			Params:      nonNil(c.Params),
		})
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handler(c *Command) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := s.now()
		code, v := s.serve(r, c, start)
		s.requests.WithLabelValues(c.Path, strconv.Itoa(code)).Inc()
		writeJSON(w, code, v)
	}
}

func (s *Server) serve(r *http.Request, c *Command, start time.Time) (int, any) {
	q := r.URL.Query()
	provider, err := pickProvider(c, q.Get("provider"))
	if err != nil {
		return s.fail(c, err)
	}
	params, warns, err := parseParams(c, provider, q)
	if err != nil {
		return s.fail(c, err)
	}
	var body []byte
	if c.Method == http.MethodPost {
		body, err = io.ReadAll(http.MaxBytesReader(nil, r.Body, maxBodyBytes))
		if err != nil {
			var tooBig *http.MaxBytesError
			if errors.As(err, &tooBig) {
				return s.fail(c, &Error{Status: http.StatusRequestEntityTooLarge, Message: "request body too large"})
			}
			return s.fail(c, &Error{Status: http.StatusBadRequest, Message: "unreadable request body", Err: err})
		}
	}
	timeout := c.Timeout
	if timeout <= 0 {
		timeout = s.timeout
	}
	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()
	res, err := c.Handler(ctx, &Request{Command: c, Provider: provider, Params: params, Body: body})
	if err != nil {
		return s.fail(c, err)
	}
	if res == nil {
		res = &Result{}
	}
	out := OBBject{
		Results:  res.Results,
		Warnings: append(warns, res.Warnings...),
		Chart:    res.Chart,
		Extra:    map[string]any{},
	}
	if out.Results == nil {
		out.Results = []any{}
	}
	if out.Warnings == nil {
		out.Warnings = []Warning{}
	}
	if p := res.Provider; p != "" {
		out.Provider = &p
	} else if provider != "" {
		out.Provider = &provider
	}
	for k, v := range res.Extra {
		out.Extra[k] = v
	}
	out.Extra["metadata"] = map[string]any{
		"route":     "/" + c.Path,
		"arguments": map[string]any{"provider": provider, "params": wireParams(params)},
		"duration":  s.now().Sub(start).Nanoseconds(),
		"timestamp": start.UTC().Format(time.RFC3339Nano),
	}
	return http.StatusOK, out
}

func (s *Server) fail(c *Command, err error) (int, any) {
	code, msg := statusOf(err)
	if code >= 500 {
		s.logger.Warn("Command failed", "command", c.Path, "status", code, "err", err)
	}
	return code, map[string]string{"error": msg}
}

func pickProvider(c *Command, given string) (string, error) {
	if len(c.Providers) == 0 {
		if given != "" {
			return "", InvalidParam("provider", "command has no providers")
		}
		return "", nil
	}
	if given == "" {
		return c.Providers[0], nil
	}
	if !slices.Contains(c.Providers, given) {
		return "", InvalidParam("provider", "must be one of %s", strings.Join(c.Providers, ", "))
	}
	return given, nil
}

// wireParams renders dates back to their wire format for the metadata echo.
func wireParams(p Params) map[string]any {
	out := make(map[string]any, len(p))
	for k, v := range p {
		if t, ok := v.(time.Time); ok {
			v = t.Format(DateLayout)
		}
		out[k] = v
	}
	return out
}

func nonNil[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
