package source

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const maxBody = 16 << 20

// Alert is the subset of an Alertmanager v2 alert used by the home page.
type Alert struct {
	Labels       map[string]string `json:"labels"`
	Annotations  map[string]string `json:"annotations"`
	StartsAt     time.Time         `json:"startsAt"`
	EndsAt       time.Time         `json:"endsAt"`
	Fingerprint  string            `json:"fingerprint"`
	GeneratorURL string            `json:"generatorURL"`
	Receivers    []struct {
		Name string `json:"name"`
	} `json:"receivers"`
	Status struct {
		State       string   `json:"state"`
		SilencedBy  []string `json:"silencedBy"`
		InhibitedBy []string `json:"inhibitedBy"`
	} `json:"status"`
}

// AlertGroup is an Alertmanager v2 alert group.
type AlertGroup struct {
	Labels   map[string]string `json:"labels"`
	Receiver struct {
		Name string `json:"name"`
	} `json:"receiver"`
	Alerts []Alert `json:"alerts"`
}

// Silence is the subset of an Alertmanager v2 silence used by stock_home.
type Silence struct {
	ID        string    `json:"id"`
	Matchers  []Matcher `json:"matchers"`
	Comment   string    `json:"comment"`
	CreatedBy string    `json:"createdBy"`
	StartsAt  time.Time `json:"startsAt"`
	EndsAt    time.Time `json:"endsAt"`
	Status    struct {
		State string `json:"state"`
	} `json:"status"`
}

// Matcher is an Alertmanager v2 label matcher.
type Matcher struct {
	Name    string `json:"name"`
	Value   string `json:"value"`
	IsRegex bool   `json:"isRegex"`
	IsEqual bool   `json:"isEqual"`
}

// NewSilence is the body of an Alertmanager v2 silence creation.
type NewSilence struct {
	Matchers  []Matcher `json:"matchers"`
	StartsAt  time.Time `json:"startsAt"`
	EndsAt    time.Time `json:"endsAt"`
	CreatedBy string    `json:"createdBy"`
	Comment   string    `json:"comment"`
}

// Alertmanager reads alerts and silences from the Alertmanager v2 API.
type Alertmanager struct {
	baseURL string
	client  *http.Client
}

// NewAlertmanager returns a client for the Alertmanager at baseURL (including any route prefix).
func NewAlertmanager(baseURL string, client *http.Client) *Alertmanager {
	return &Alertmanager{baseURL: baseURL, client: client}
}

// Alerts returns all alerts, including silenced and inhibited ones.
func (a *Alertmanager) Alerts(ctx context.Context) ([]Alert, error) {
	var out []Alert
	err := getJSON(ctx, a.client, a.baseURL+"/api/v2/alerts?active=true&silenced=true&inhibited=true&unprocessed=true", &out)
	return out, err
}

// Silences returns all silences.
func (a *Alertmanager) Silences(ctx context.Context) ([]Silence, error) {
	var out []Silence
	err := getJSON(ctx, a.client, a.baseURL+"/api/v2/silences", &out)
	return out, err
}

// Groups returns alert groups, including silenced and inhibited alerts.
func (a *Alertmanager) Groups(ctx context.Context) ([]AlertGroup, error) {
	var out []AlertGroup
	err := getJSON(ctx, a.client, a.baseURL+"/api/v2/alerts/groups?active=true&silenced=true&inhibited=true", &out)
	return out, err
}

// AMStatus is the Alertmanager v2 status; Config.Original holds the full configuration and must be redacted before use.
type AMStatus struct {
	Cluster struct {
		Status string            `json:"status"`
		Peers  []json.RawMessage `json:"peers"`
	} `json:"cluster"`
	VersionInfo map[string]string `json:"versionInfo"`
	Uptime      time.Time         `json:"uptime"`
	Config      struct {
		Original string `json:"original"`
	} `json:"config"`
}

// Status returns the Alertmanager status.
func (a *Alertmanager) Status(ctx context.Context) (AMStatus, error) {
	var out AMStatus
	err := getJSON(ctx, a.client, a.baseURL+"/api/v2/status", &out)
	return out, err
}

// ExpireSilence ends a silence now; id must already be validated.
func (a *Alertmanager) ExpireSilence(ctx context.Context, id string) error {
	url := a.baseURL + "/api/v2/silence/" + id
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("DELETE %s: %w", url, err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64<<10))
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("DELETE %s: status %d", url, resp.StatusCode)
	}
	return nil
}

// CreateSilence posts a new silence and returns its ID.
func (a *Alertmanager) CreateSilence(ctx context.Context, s NewSilence) (string, error) {
	body, err := json.Marshal(s)
	if err != nil {
		return "", err
	}
	url := a.baseURL + "/api/v2/silences"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := a.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("POST %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("POST %s: status %d", url, resp.StatusCode)
	}
	var out struct {
		SilenceID string `json:"silenceID"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 64<<10)).Decode(&out); err != nil {
		return "", fmt.Errorf("POST %s: decode: %w", url, err)
	}
	return out.SilenceID, nil
}

func getJSON(ctx context.Context, client *http.Client, url string, v any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("GET %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s: status %d", url, resp.StatusCode)
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxBody)).Decode(v); err != nil {
		return fmt.Errorf("GET %s: decode: %w", url, err)
	}
	return nil
}

// Probe reports whether url answers 200 and how long it took.
func Probe(ctx context.Context, client *http.Client, url string) (ok bool, latency time.Duration) {
	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false, 0
	}
	resp, err := client.Do(req)
	latency = time.Since(start)
	if err != nil {
		return false, latency
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64<<10))
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK, latency
}
