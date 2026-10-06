// Copyright 2026 The Stock Exporter Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package market implements the NSE/BSE equity trading calendar.
package market

import (
	"fmt"
	"os"
	"time"

	"go.yaml.in/yaml/v3"
)

// Session phases used in the phase label.
const (
	PhaseClosed    = "closed"
	PhasePreOpen   = "pre_open"
	PhaseOpen      = "open"
	PhasePostClose = "post_close"
)

// Phases lists all phases in display order.
var Phases = []string{PhaseClosed, PhasePreOpen, PhaseOpen, PhasePostClose}

const dateLayout = "2006-01-02"

// Minutes since midnight in exchange local time.
const (
	preOpenStart   = 9 * 60
	openStart      = 9*60 + 15
	openEnd        = 15*60 + 30
	postCloseEnd   = 16 * 60
	maxHolidayScan = 30
)

// Calendar answers market session questions for a timezone and holiday list.
type Calendar struct {
	loc       *time.Location
	holidays  map[string]string
	forceOpen bool
}

// NewCalendar creates a calendar. forceOpen makes every instant report as open (testing/simulation).
func NewCalendar(loc *time.Location, holidays map[string]string, forceOpen bool) *Calendar {
	if holidays == nil {
		holidays = map[string]string{}
	}
	return &Calendar{loc: loc, holidays: holidays, forceOpen: forceOpen}
}

// Location returns the exchange timezone.
func (c *Calendar) Location() *time.Location { return c.loc }

type holidayFile struct {
	Holidays []struct {
		Date string `yaml:"date"`
		Name string `yaml:"name"`
	} `yaml:"holidays"`
}

// LoadHolidays reads a YAML holiday file keyed by YYYY-MM-DD.
func LoadHolidays(path string) (map[string]string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var f holidayFile
	if err := yaml.Unmarshal(b, &f); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	out := make(map[string]string, len(f.Holidays))
	for _, h := range f.Holidays {
		if _, err := time.Parse(dateLayout, h.Date); err != nil {
			return nil, fmt.Errorf("invalid holiday date %q in %s: %w", h.Date, path, err)
		}
		out[h.Date] = h.Name
	}
	return out, nil
}

// IsTradingDay reports whether t falls on a weekday that is not a holiday.
func (c *Calendar) IsTradingDay(t time.Time) bool {
	if c.forceOpen {
		return true
	}
	t = t.In(c.loc)
	if wd := t.Weekday(); wd == time.Saturday || wd == time.Sunday {
		return false
	}
	_, holiday := c.holidays[t.Format(dateLayout)]
	return !holiday
}

// Phase returns the session phase at t.
func (c *Calendar) Phase(t time.Time) string {
	if c.forceOpen {
		return PhaseOpen
	}
	if !c.IsTradingDay(t) {
		return PhaseClosed
	}
	t = t.In(c.loc)
	m := t.Hour()*60 + t.Minute()
	switch {
	case m >= preOpenStart && m < openStart:
		return PhasePreOpen
	case m >= openStart && m < openEnd:
		return PhaseOpen
	case m >= openEnd && m < postCloseEnd:
		return PhasePostClose
	default:
		return PhaseClosed
	}
}

// IsOpen reports whether continuous trading is in progress at t.
func (c *Calendar) IsOpen(t time.Time) bool { return c.Phase(t) == PhaseOpen }

func (c *Calendar) at(day time.Time, minutes int) time.Time {
	y, mo, d := day.In(c.loc).Date()
	return time.Date(y, mo, d, minutes/60, minutes%60, 0, 0, c.loc)
}

// SecondsToClose returns the seconds until the normal session ends, or 0 when not open.
func (c *Calendar) SecondsToClose(t time.Time) float64 {
	if c.forceOpen || !c.IsOpen(t) {
		return 0
	}
	return c.at(t, openEnd).Sub(t).Seconds()
}

// NextOpen returns the next normal session start strictly after t, or zero if none is found.
func (c *Calendar) NextOpen(t time.Time) time.Time {
	for i := 0; i <= maxHolidayScan; i++ {
		day := t.In(c.loc).AddDate(0, 0, i)
		open := c.at(day, openStart)
		if open.After(t) && c.IsTradingDay(open) {
			return open
		}
	}
	return time.Time{}
}

// SecondsToOpen returns the seconds until the next session opens, or 0 when open.
func (c *Calendar) SecondsToOpen(t time.Time) float64 {
	if c.forceOpen || c.IsOpen(t) {
		return 0
	}
	n := c.NextOpen(t)
	if n.IsZero() {
		return 0
	}
	return n.Sub(t).Seconds()
}

// Holiday returns the holiday name for t's date, if any.
func (c *Calendar) Holiday(t time.Time) (string, bool) {
	n, ok := c.holidays[t.In(c.loc).Format(dateLayout)]
	return n, ok
}
