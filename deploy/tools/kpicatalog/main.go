// Command kpicatalog validates deploy/prometheus/kpi-catalog.yaml against the
// Prometheus rule files and renders the stock_kpi_info textfile.
package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"go.yaml.in/yaml/v3"
)

const exporterSource = "exporter"

type Entry struct {
	Name           string   `yaml:"name"`
	Kind           string   `yaml:"kind"`
	Family         string   `yaml:"family"`
	Level          string   `yaml:"level"`
	Capability     string   `yaml:"capability"`
	Release        string   `yaml:"release"`
	Formula        string   `yaml:"formula"`
	Inputs         []string `yaml:"inputs"`
	RuleFile       string   `yaml:"rule_file"`
	Frequency      string   `yaml:"frequency"`
	Scope          string   `yaml:"scope"`
	Unit           string   `yaml:"unit"`
	Range          *Range   `yaml:"range"`
	Warmup         string   `yaml:"warmup"`
	WarmupSessions int      `yaml:"warmup_sessions"`
	Quality        string   `yaml:"quality"`
	Method         string   `yaml:"method"`
	Limitation     string   `yaml:"limitation"`
	Alertable      bool     `yaml:"alertable"`
	Coverage       bool     `yaml:"coverage"`
	Dashboard      bool     `yaml:"dashboard_default"`
}

type Range struct {
	Min *float64 `yaml:"min"`
	Max *float64 `yaml:"max"`
}

type Catalog struct {
	KPIs []Entry `yaml:"kpis"`
}

type ruleFile struct {
	Groups []struct {
		Name  string `yaml:"name"`
		Rules []struct {
			Record string            `yaml:"record"`
			Alert  string            `yaml:"alert"`
			Expr   string            `yaml:"expr"`
			Labels map[string]string `yaml:"labels"`
		} `yaml:"rules"`
	} `yaml:"groups"`
}

var enums = map[string][]string{
	"kind":    {"kpi", "snapshot", "signal", "quality", "helper"},
	"family":  {"technical", "microstructure", "market", "risk", "derivatives", "predictive"},
	"level":   {"market_health", "sector_intelligence", "stock_intelligence", "risk_intelligence", "derivatives_intelligence", "predictive_intelligence"},
	"release": {"R1", "R2", "R3", "R4", "R5", "R6"},
	"quality": {"exact", "approximation", "proxy", "heuristic", "model"},
	"scope":   {"core", "broad", "all_eq", "market", "sector"},
}

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	fs := flag.NewFlagSet(os.Args[1], flag.ExitOnError)
	catalogPath := fs.String("catalog", "prometheus/kpi-catalog.yaml", "KPI catalogue")
	rulesGlob := fs.String("rules", "prometheus/rules/*.rules.yml", "rule files glob")
	contracted := fs.String("contracted", "nse_*.rules.yml", "rule files whose every record needs a contract")
	textfile := fs.String("textfile", "", "generated textfile that must match the catalogue (check only)")
	_ = fs.Parse(os.Args[2:])

	cat, err := loadCatalog(*catalogPath)
	if err != nil {
		fatal(err)
	}
	switch os.Args[1] {
	case "check":
		errs := check(cat, *rulesGlob, *contracted)
		if *textfile != "" {
			if err := checkTextfile(cat, *textfile); err != nil {
				errs = append(errs, err)
			}
		}
		if len(errs) > 0 {
			for _, e := range errs {
				fmt.Fprintln(os.Stderr, "kpicatalog:", e)
			}
			os.Exit(1)
		}
		fmt.Printf("kpicatalog: %d contracts OK\n", len(cat.KPIs))
	case "textfile":
		os.Stdout.Write(renderTextfile(cat))
	default:
		usage()
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: kpicatalog check|textfile [-catalog file] [-rules glob] [-contracted glob] [-textfile file]")
	os.Exit(2)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "kpicatalog:", err)
	os.Exit(1)
}

func loadCatalog(path string) (*Catalog, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c Catalog
	dec := yaml.NewDecoder(bytes.NewReader(b))
	dec.KnownFields(true)
	if err := dec.Decode(&c); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return &c, nil
}

// records maps rule file base name -> recorded series -> value of the rule's kpi label.
type records map[string]map[string][]string

func loadRules(glob string) (records, error) {
	paths, err := filepath.Glob(glob)
	if err != nil {
		return nil, err
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("no rule files match %q", glob)
	}
	out := records{}
	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err != nil {
			return nil, err
		}
		var rf ruleFile
		if err := yaml.Unmarshal(b, &rf); err != nil {
			return nil, fmt.Errorf("%s: %w", p, err)
		}
		m := map[string][]string{}
		for _, g := range rf.Groups {
			for _, r := range g.Rules {
				if r.Record != "" {
					m[r.Record] = append(m[r.Record], r.Labels["kpi"])
				}
			}
		}
		out[filepath.Base(p)] = m
	}
	return out, nil
}

func check(cat *Catalog, rulesGlob, contracted string) []error {
	recs, err := loadRules(rulesGlob)
	if err != nil {
		return []error{err}
	}
	var errs []error
	seen := map[string]bool{}
	for i, e := range cat.KPIs {
		where := fmt.Sprintf("kpis[%d] %s", i, e.Name)
		if e.Name == "" {
			errs = append(errs, fmt.Errorf("%s: name is required", where))
			continue
		}
		if seen[e.Name] {
			errs = append(errs, fmt.Errorf("%s: duplicate contract", where))
		}
		seen[e.Name] = true
		errs = append(errs, validateEntry(where, e)...)

		if e.RuleFile == exporterSource {
			if strings.Contains(e.Name, ":") {
				errs = append(errs, fmt.Errorf("%s: exporter metrics have no colon in their name", where))
			}
			continue
		}
		m, ok := recs[e.RuleFile]
		if !ok {
			errs = append(errs, fmt.Errorf("%s: rule_file %q not found", where, e.RuleFile))
			continue
		}
		if _, ok := m[e.Name]; !ok {
			errs = append(errs, fmt.Errorf("%s: no rule in %s records it", where, e.RuleFile))
		}
	}

	for file, m := range recs {
		if ok, _ := filepath.Match(contracted, file); !ok {
			continue
		}
		for _, name := range sortedKeys(m) {
			if !seen[name] {
				errs = append(errs, fmt.Errorf("%s: %s has no contract in the catalogue", file, name))
			}
		}
	}
	return append(errs, checkCoverage(cat, recs)...)
}

func validateEntry(where string, e Entry) []error {
	var errs []error
	req := func(field, v string) {
		if v == "" {
			errs = append(errs, fmt.Errorf("%s: %s is required", where, field))
		}
	}
	enum := func(field, v string) {
		if v != "" && !slices.Contains(enums[field], v) {
			errs = append(errs, fmt.Errorf("%s: %s %q not in %v", where, field, v, enums[field]))
		}
	}
	req("kind", e.Kind)
	req("release", e.Release)
	req("rule_file", e.RuleFile)
	req("formula", e.Formula)
	req("scope", e.Scope)
	enum("kind", e.Kind)
	enum("family", e.Family)
	enum("level", e.Level)
	enum("release", e.Release)
	enum("quality", e.Quality)
	enum("scope", e.Scope)
	if e.Kind == "kpi" {
		req("family", e.Family)
		req("level", e.Level)
		req("capability", e.Capability)
		req("quality", e.Quality)
		req("method", e.Method)
		req("unit", e.Unit)
		req("frequency", e.Frequency)
	}
	if e.Coverage && e.Kind != "kpi" {
		errs = append(errs, fmt.Errorf("%s: coverage only applies to kind kpi", where))
	}
	if e.Range != nil && e.Range.Min != nil && e.Range.Max != nil && *e.Range.Min >= *e.Range.Max {
		errs = append(errs, fmt.Errorf("%s: range min must be below max", where))
	}
	return errs
}

// checkCoverage requires a kpi:coverage:ratio rule (kpi label) for every coverage KPI,
// once such rules exist, and no rule for any other series.
func checkCoverage(cat *Catalog, recs records) []error {
	covered := map[string]int{}
	for _, m := range recs {
		for _, kpi := range m["kpi:coverage:ratio"] {
			covered[kpi]++
		}
	}
	if len(covered) == 0 {
		return nil
	}
	var errs []error
	want := map[string]bool{}
	for _, e := range cat.KPIs {
		if !e.Coverage {
			continue
		}
		want[e.Name] = true
		if covered[e.Name] == 0 {
			errs = append(errs, fmt.Errorf("kpi:coverage:ratio has no rule for %s", e.Name))
		}
	}
	keys := make([]string, 0, len(covered))
	for kpi := range covered {
		keys = append(keys, kpi)
	}
	slices.Sort(keys)
	for _, kpi := range keys {
		n := covered[kpi]
		switch {
		case kpi == "":
			errs = append(errs, fmt.Errorf("kpi:coverage:ratio rule without a kpi label"))
		case !want[kpi]:
			errs = append(errs, fmt.Errorf("kpi:coverage:ratio covers %s, which is not a coverage KPI", kpi))
		case n > 1:
			errs = append(errs, fmt.Errorf("kpi:coverage:ratio has %d rules for %s", n, kpi))
		}
	}
	return errs
}

func renderTextfile(cat *Catalog) []byte {
	var b bytes.Buffer
	b.WriteString("# Generated by deploy/tools/kpicatalog from deploy/prometheus/kpi-catalog.yaml. DO NOT EDIT.\n")
	b.WriteString("# HELP stock_kpi_info KPI contract metadata (value is always 1).\n")
	b.WriteString("# TYPE stock_kpi_info gauge\n")
	for _, e := range cat.KPIs {
		if e.Kind != "kpi" {
			continue
		}
		fmt.Fprintf(&b, "stock_kpi_info{kpi=%s,family=%s,level=%s,capability=%s,release=%s,quality=%s,method=%s,scope=%s,unit=%s} 1\n",
			q(e.Name), q(e.Family), q(e.Level), q(e.Capability), q(e.Release), q(e.Quality), q(e.Method), q(e.Scope), q(e.Unit))
	}
	return b.Bytes()
}

func checkTextfile(cat *Catalog, path string) error {
	got, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("%s missing; run `make kpi-textfile`", path)
	} else if err != nil {
		return err
	}
	if !bytes.Equal(got, renderTextfile(cat)) {
		return fmt.Errorf("%s is out of date; run `make kpi-textfile`", path)
	}
	return nil
}

func q(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`)
	return `"` + r.Replace(s) + `"`
}

func sortedKeys(m map[string][]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}
