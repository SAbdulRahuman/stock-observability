package source

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// GrafanaDashboard is the subset of a Grafana search hit used by the Dashboards page.
type GrafanaDashboard struct {
	UID         string   `json:"uid"`
	Title       string   `json:"title"`
	Tags        []string `json:"tags"`
	FolderTitle string   `json:"folderTitle"`
}

// Grafana lists dashboards through the Grafana HTTP API.
type Grafana struct {
	baseURL string
	client  *http.Client
	token   string
}

// NewGrafana returns a client for the Grafana at baseURL (including any sub path); token may be empty for anonymous access.
func NewGrafana(baseURL string, client *http.Client, token string) *Grafana {
	return &Grafana{baseURL: baseURL, client: client, token: token}
}

// Dashboards returns up to 500 dashboards visible to the configured identity.
func (g *Grafana) Dashboards(ctx context.Context) ([]GrafanaDashboard, error) {
	url := g.baseURL + "/api/search?type=dash-db&limit=500"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if g.token != "" {
		req.Header.Set("Authorization", "Bearer "+g.token)
	}
	resp, err := g.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GET %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: status %d", url, resp.StatusCode)
	}
	var out []GrafanaDashboard
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxBody)).Decode(&out); err != nil {
		return nil, fmt.Errorf("GET %s: decode: %w", url, err)
	}
	return out, nil
}
