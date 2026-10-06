package holidays

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadAndNext(t *testing.T) {
	ist, err := time.LoadLocation("Asia/Kolkata")
	if err != nil {
		t.Skip("tzdata not available")
	}
	path := filepath.Join(t.TempDir(), "h.yml")
	data := "holidays:\n" +
		"  - {date: 2026-10-20, name: Dussehra}\n" +
		"  - {date: 2026-10-02, name: Gandhi Jayanti}\n" +
		"  - {date: 2026-11-08, name: Diwali (Sunday)}\n" +
		"  - {date: 2026-11-10, name: Balipratipada}\n"
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	list, err := Load(path, ist)
	if err != nil {
		t.Fatal(err)
	}
	if list[0].Name != "Gandhi Jayanti" {
		t.Fatalf("not sorted: %+v", list)
	}

	cases := []struct {
		now  time.Time
		want string
	}{
		{time.Date(2026, 10, 6, 12, 0, 0, 0, ist), "Dussehra"},
		{time.Date(2026, 10, 20, 9, 0, 0, 0, ist), "Dussehra"},
		{time.Date(2026, 10, 21, 9, 0, 0, 0, ist), "Balipratipada"},
	}
	for _, c := range cases {
		if got := Next(list, c.now); got == nil || got.Name != c.want {
			t.Errorf("Next(%s) = %+v, want %s", c.now, got, c.want)
		}
	}
	if got := Next(list, time.Date(2027, 1, 1, 0, 0, 0, 0, ist)); got != nil {
		t.Errorf("want nil after last holiday, got %+v", got)
	}
}
