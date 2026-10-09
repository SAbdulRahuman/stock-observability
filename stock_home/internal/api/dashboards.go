package api

import (
	"context"
	"regexp"
	"sort"
	"strings"
)

var grafanaUID = regexp.MustCompile(`^[A-Za-z0-9_-]{1,40}$`)

// DashboardLink is one Grafana dashboard opened through the portal's Grafana path.
type DashboardLink struct {
	UID    string   `json:"uid"`
	Title  string   `json:"title"`
	Folder string   `json:"folder,omitempty"`
	Tags   []string `json:"tags"`
	Href   string   `json:"href"`
}

// Dashboards lists the Grafana dashboards for the Dashboards page.
type Dashboards struct {
	Items []DashboardLink `json:"items"`
}

func (s *Server) dashboards(ctx context.Context) (any, error) {
	list, err := s.graf.Dashboards(ctx)
	if err != nil {
		return nil, err
	}
	base := grafanaPublicPath()
	out := Dashboards{Items: []DashboardLink{}}
	for _, d := range list {
		// Links are built from the uid only; Grafana-supplied URLs are never passed to the browser.
		if !grafanaUID.MatchString(d.UID) {
			continue
		}
		tags := d.Tags
		if tags == nil {
			tags = []string{}
		}
		out.Items = append(out.Items, DashboardLink{UID: d.UID, Title: d.Title, Folder: d.FolderTitle, Tags: tags, Href: base + "d/" + d.UID})
	}
	sort.Slice(out.Items, func(i, j int) bool { return strings.ToLower(out.Items[i].Title) < strings.ToLower(out.Items[j].Title) })
	return out, nil
}

func grafanaPublicPath() string {
	for _, l := range quickLinks {
		if l.ID == "grafana" {
			return strings.TrimRight(l.Href, "/") + "/"
		}
	}
	return "/grafana/"
}
