package waitlist

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"
)

// Export writes entries as csv or json.
func Export(w io.Writer, entries []Entry, format string) error {
	switch format {
	case "json":
		if entries == nil {
			entries = []Entry{}
		}
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(entries)
	case "csv":
		cw := csv.NewWriter(w)
		_ = cw.Write([]string{"created_at", "email", "name", "interest", "message"})
		for _, e := range entries {
			_ = cw.Write([]string{e.CreatedAt.UTC().Format(time.RFC3339), cell(e.Email), cell(e.Name), cell(e.Interest), cell(e.Message)})
		}
		cw.Flush()
		return cw.Error()
	default:
		return fmt.Errorf("unknown format %q: want csv or json", format)
	}
}

// cell defuses spreadsheet formulas in user input.
func cell(s string) string {
	if s != "" && strings.ContainsRune("=+-@\t\r", rune(s[0])) {
		return "'" + s
	}
	return s
}
