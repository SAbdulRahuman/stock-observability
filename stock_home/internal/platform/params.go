package platform

import (
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"
)

// DateLayout is the wire format of Date parameters.
const DateLayout = "2006-01-02"

// maxListItems bounds comma-separated parameters.
const maxListItems = 500

// Params holds validated, typed parameter values keyed by name.
type Params map[string]any

// Has reports whether name was given or has a default.
func (p Params) Has(name string) bool { _, ok := p[name]; return ok }

// String returns a string parameter or "".
func (p Params) String(name string) string { s, _ := p[name].(string); return s }

// Int returns an integer parameter or 0.
func (p Params) Int(name string) int64 { v, _ := p[name].(int64); return v }

// Float returns a number parameter or 0.
func (p Params) Float(name string) float64 { v, _ := p[name].(float64); return v }

// Bool returns a boolean parameter or false.
func (p Params) Bool(name string) bool { v, _ := p[name].(bool); return v }

// Date returns a date parameter or the zero time.
func (p Params) Date(name string) time.Time { v, _ := p[name].(time.Time); return v }

// Strings returns a multiple string parameter, or a single string as a one-element slice.
func (p Params) Strings(name string) []string {
	switch v := p[name].(type) {
	case []string:
		return v
	case string:
		return []string{v}
	}
	return nil
}

// parseParams validates q against c for provider and returns the typed values plus warnings for ignored keys.
func parseParams(c *Command, provider string, q url.Values) (Params, []Warning, error) {
	out := Params{}
	var warns []Warning
	known := map[string]bool{}
	for _, name := range reserved {
		known[name] = true
	}
	for _, p := range c.Params {
		known[p.Name] = true
		applies := len(p.Providers) == 0 || slices.Contains(p.Providers, provider)
		raw, given := q[p.Name]
		if given && !applies {
			warns = append(warns, Warning{Category: "OpenBBWarning", Message: "Parameter '" + p.Name + "' is not supported by " + provider + "; ignored."})
			continue
		}
		if !applies {
			continue
		}
		if !given || len(raw) == 0 || raw[len(raw)-1] == "" {
			if p.Required {
				return nil, nil, InvalidParam(p.Name, "required")
			}
			if p.Default != nil {
				out[p.Name] = p.Default
				if s, ok := p.Default.(string); ok && p.Type == Date {
					out[p.Name], _ = time.Parse(DateLayout, s)
				}
			}
			continue
		}
		v, err := parseValue(p, raw[len(raw)-1])
		if err != nil {
			return nil, nil, err
		}
		out[p.Name] = v
	}
	for k := range q {
		if !known[k] {
			warns = append(warns, Warning{Category: "OpenBBWarning", Message: "Unknown parameter '" + k + "' ignored."})
		}
	}
	slices.SortFunc(warns, func(a, b Warning) int { return strings.Compare(a.Message, b.Message) })
	return out, warns, nil
}

func parseValue(p Param, s string) (any, error) {
	if !p.Multiple {
		return parseScalar(p, s)
	}
	parts := strings.Split(s, ",")
	if len(parts) > maxListItems {
		return nil, InvalidParam(p.Name, "at most %d items", maxListItems)
	}
	switch p.Type {
	case Integer:
		out := make([]int64, 0, len(parts))
		for _, part := range parts {
			v, err := parseScalar(p, strings.TrimSpace(part))
			if err != nil {
				return nil, err
			}
			out = append(out, v.(int64))
		}
		return out, nil
	case Number:
		out := make([]float64, 0, len(parts))
		for _, part := range parts {
			v, err := parseScalar(p, strings.TrimSpace(part))
			if err != nil {
				return nil, err
			}
			out = append(out, v.(float64))
		}
		return out, nil
	default:
		out := make([]string, 0, len(parts))
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			v, err := parseScalar(p, part)
			if err != nil {
				return nil, err
			}
			out = append(out, v.(string))
		}
		if len(out) == 0 {
			return nil, InvalidParam(p.Name, "empty list")
		}
		return out, nil
	}
}

func parseScalar(p Param, s string) (any, error) {
	switch p.Type {
	case Integer:
		v, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return nil, InvalidParam(p.Name, "not an integer")
		}
		return v, nil
	case Number:
		v, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return nil, InvalidParam(p.Name, "not a number")
		}
		return v, nil
	case Boolean:
		v, err := strconv.ParseBool(s)
		if err != nil {
			return nil, InvalidParam(p.Name, "not a boolean")
		}
		return v, nil
	case Date:
		v, err := time.Parse(DateLayout, s)
		if err != nil {
			return nil, InvalidParam(p.Name, "not a date (YYYY-MM-DD)")
		}
		return v, nil
	default:
		if len(p.Choices) > 0 && !slices.Contains(p.Choices, s) {
			return nil, InvalidParam(p.Name, "must be one of %s", strings.Join(p.Choices, ", "))
		}
		return s, nil
	}
}
