// Package platform implements the OpenBB-style command registry and the /api/v2 router.
package platform

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"
)

// ParamType is the wire type of a command parameter.
type ParamType string

// Supported parameter types.
const (
	String  ParamType = "string"
	Integer ParamType = "integer"
	Number  ParamType = "number"
	Boolean ParamType = "boolean"
	Date    ParamType = "date"
)

// Param describes one query parameter of a command.
type Param struct {
	Name        string    `json:"name"`
	Type        ParamType `json:"type"`
	Description string    `json:"description"`
	Required    bool      `json:"required"`
	Default     any       `json:"default,omitempty"`
	Choices     []string  `json:"choices,omitempty"`
	// Multiple accepts a comma-separated list; the value is then []string, []int64 or []float64.
	Multiple bool `json:"multiple,omitempty"`
	// Providers limits a provider-specific parameter; empty means it applies to all providers.
	Providers []string `json:"providers,omitempty"`
}

// Warning is a non-fatal message returned alongside results.
type Warning struct {
	Category string `json:"category"`
	Message  string `json:"message"`
}

// Request is the validated input of a command call.
type Request struct {
	Command  *Command
	Provider string
	Params   Params
	// Body is the raw request body of POST commands.
	Body []byte
}

// Result is what a handler returns; the server wraps it in the OBBject envelope.
type Result struct {
	Results  any
	Provider string
	Warnings []Warning
	Chart    any
	Extra    map[string]any
}

// Handler executes a command.
type Handler func(ctx context.Context, req *Request) (*Result, error)

// Command is one OpenBB-equivalent endpoint.
type Command struct {
	// Path is the OpenBB command path with slashes, e.g. "equity/price/historical".
	Path        string
	Description string
	// Model is the OpenBB standard model name, e.g. "EquityHistorical".
	Model string
	// Method is GET (default) or POST.
	Method string
	Params []Param
	// Providers lists the providers able to serve the command; the first is the default.
	Providers []string
	Handler   Handler
	// Timeout overrides the server default for slow upstreams.
	Timeout time.Duration
}

// Registry holds all commands.
type Registry struct {
	mu       sync.RWMutex
	commands map[string]*Command
}

// NewRegistry returns an empty registry.
func NewRegistry() *Registry { return &Registry{commands: map[string]*Command{}} }

var pathRE = regexp.MustCompile(`^[a-z0-9_]+(/[a-z0-9_]+)*$`)

// reserved query parameters handled by the server itself.
var reserved = []string{"provider"}

// Register adds c. It panics on invalid or duplicate definitions since those are programming errors.
func (r *Registry) Register(c Command) {
	if err := c.validate(); err != nil {
		panic(fmt.Sprintf("platform: command %q: %v", c.Path, err))
	}
	if c.Method == "" {
		c.Method = "GET"
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.commands[c.Path]; ok {
		panic(fmt.Sprintf("platform: duplicate command %q", c.Path))
	}
	r.commands[c.Path] = &c
}

// Get returns the command at path.
func (r *Registry) Get(path string) (*Command, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.commands[path]
	return c, ok
}

// Commands returns all commands sorted by path.
func (r *Registry) Commands() []*Command {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*Command, 0, len(r.commands))
	for _, c := range r.commands {
		out = append(out, c)
	}
	slices.SortFunc(out, func(a, b *Command) int { return strings.Compare(a.Path, b.Path) })
	return out
}

func (c *Command) validate() error {
	if !pathRE.MatchString(c.Path) {
		return fmt.Errorf("invalid path")
	}
	if c.Handler == nil {
		return fmt.Errorf("nil handler")
	}
	if c.Method != "" && c.Method != "GET" && c.Method != "POST" {
		return fmt.Errorf("unsupported method %q", c.Method)
	}
	seen := map[string]bool{}
	for _, p := range c.Params {
		if p.Name == "" || seen[p.Name] || slices.Contains(reserved, p.Name) {
			return fmt.Errorf("invalid or duplicate param %q", p.Name)
		}
		seen[p.Name] = true
		switch p.Type {
		case String, Integer, Number, Boolean, Date:
		default:
			return fmt.Errorf("param %q: unknown type %q", p.Name, p.Type)
		}
		if p.Multiple && (p.Type == Boolean || p.Type == Date) {
			return fmt.Errorf("param %q: multiple not supported for %s", p.Name, p.Type)
		}
		if p.Default != nil && !defaultMatches(p) {
			return fmt.Errorf("param %q: default %T does not match type %s", p.Name, p.Default, p.Type)
		}
		for _, pr := range p.Providers {
			if !slices.Contains(c.Providers, pr) {
				return fmt.Errorf("param %q: unknown provider %q", p.Name, pr)
			}
		}
	}
	return nil
}

func defaultMatches(p Param) bool {
	switch v := p.Default.(type) {
	case string:
		if p.Type == Date {
			_, err := time.Parse(DateLayout, v)
			return err == nil
		}
		return p.Type == String && (len(p.Choices) == 0 || slices.Contains(p.Choices, v))
	case []string:
		return p.Type == String && p.Multiple
	case int64:
		return p.Type == Integer && !p.Multiple
	case float64:
		return p.Type == Number && !p.Multiple
	case bool:
		return p.Type == Boolean
	}
	return false
}
