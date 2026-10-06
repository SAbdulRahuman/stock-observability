// Package holidays reads the exchange holiday list shared with stock_exporter.
package holidays

import (
	"fmt"
	"os"
	"sort"
	"time"

	"go.yaml.in/yaml/v3"
)

// Holiday is one exchange holiday.
type Holiday struct {
	Date time.Time
	Name string
}

type file struct {
	Holidays []struct {
		Date string `yaml:"date"`
		Name string `yaml:"name"`
	} `yaml:"holidays"`
}

// Load parses a holidays YAML file (same format as stock_exporter --market.holidays-file).
func Load(path string, loc *time.Location) ([]Holiday, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var f file
	if err := yaml.Unmarshal(b, &f); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	out := make([]Holiday, 0, len(f.Holidays))
	for _, h := range f.Holidays {
		d, err := time.ParseInLocation(time.DateOnly, h.Date, loc)
		if err != nil {
			return nil, fmt.Errorf("parse %s: holiday %q: %w", path, h.Name, err)
		}
		out = append(out, Holiday{Date: d, Name: h.Name})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Date.Before(out[j].Date) })
	return out, nil
}

// Next returns the first weekday holiday on or after now's date, or nil.
func Next(list []Holiday, now time.Time) *Holiday {
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	for i := range list {
		h := list[i]
		if h.Date.Before(today) || h.Date.Weekday() == time.Saturday || h.Date.Weekday() == time.Sunday {
			continue
		}
		return &h
	}
	return nil
}
