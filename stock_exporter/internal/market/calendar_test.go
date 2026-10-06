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

package market

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func mustIST(t *testing.T) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation("Asia/Kolkata")
	if err != nil {
		t.Fatal(err)
	}
	return loc
}

func TestPhase(t *testing.T) {
	loc := mustIST(t)
	cal := NewCalendar(loc, map[string]string{"2026-10-02": "Gandhi Jayanti"}, false)
	at := func(s string) time.Time {
		tm, err := time.ParseInLocation("2006-01-02 15:04:05", s, loc)
		if err != nil {
			t.Fatal(err)
		}
		return tm
	}
	cases := []struct {
		at   string
		want string
	}{
		{"2026-10-06 08:59:59", PhaseClosed},
		{"2026-10-06 09:00:00", PhasePreOpen},
		{"2026-10-06 09:14:59", PhasePreOpen},
		{"2026-10-06 09:15:00", PhaseOpen},
		{"2026-10-06 15:29:59", PhaseOpen},
		{"2026-10-06 15:30:00", PhasePostClose},
		{"2026-10-06 16:00:00", PhaseClosed},
		{"2026-10-04 11:00:00", PhaseClosed}, // Sunday
		{"2026-10-03 11:00:00", PhaseClosed}, // Saturday
		{"2026-10-02 11:00:00", PhaseClosed}, // holiday
	}
	for _, c := range cases {
		if got := cal.Phase(at(c.at)); got != c.want {
			t.Errorf("Phase(%s) = %s, want %s", c.at, got, c.want)
		}
	}

	// Same instant expressed in UTC must give the same answer.
	if got := cal.Phase(at("2026-10-06 09:15:00").UTC()); got != PhaseOpen {
		t.Errorf("UTC input: got %s, want open", got)
	}
}

func TestNextOpenSkipsWeekendAndHoliday(t *testing.T) {
	loc := mustIST(t)
	cal := NewCalendar(loc, map[string]string{"2026-10-05": "Test Holiday"}, false)
	from := time.Date(2026, 10, 2, 16, 0, 0, 0, loc) // Friday after close
	want := time.Date(2026, 10, 6, 9, 15, 0, 0, loc) // Monday is a holiday
	if got := cal.NextOpen(from); !got.Equal(want) {
		t.Fatalf("NextOpen = %s, want %s", got, want)
	}
	if got := cal.SecondsToOpen(from); got != want.Sub(from).Seconds() {
		t.Fatalf("SecondsToOpen = %v", got)
	}
}

func TestSecondsToClose(t *testing.T) {
	loc := mustIST(t)
	cal := NewCalendar(loc, nil, false)
	if got := cal.SecondsToClose(time.Date(2026, 10, 6, 15, 0, 0, 0, loc)); got != 1800 {
		t.Fatalf("got %v, want 1800", got)
	}
	if got := cal.SecondsToClose(time.Date(2026, 10, 6, 8, 0, 0, 0, loc)); got != 0 {
		t.Fatalf("got %v, want 0 when closed", got)
	}
}

func TestForceOpen(t *testing.T) {
	cal := NewCalendar(mustIST(t), nil, true)
	sunday := time.Date(2026, 10, 4, 23, 0, 0, 0, cal.Location())
	if !cal.IsOpen(sunday) {
		t.Fatal("force-open calendar must report open")
	}
}

func TestLoadHolidays(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "h.yml")
	if err := os.WriteFile(p, []byte("holidays:\n  - date: 2026-01-26\n    name: Republic Day\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	h, err := LoadHolidays(p)
	if err != nil {
		t.Fatal(err)
	}
	if h["2026-01-26"] != "Republic Day" {
		t.Fatalf("unexpected holidays: %v", h)
	}

	if err := os.WriteFile(p, []byte("holidays:\n  - date: 26-01-2026\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadHolidays(p); err == nil {
		t.Fatal("expected error for invalid date")
	}
}
