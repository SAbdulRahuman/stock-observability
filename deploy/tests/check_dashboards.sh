#!/usr/bin/env bash
# Every Grafana dashboard is valid JSON and every panel and annotation query parses.
set -euo pipefail
cd "$(dirname "$0")/.."

tmp=$(mktemp)
trap 'rm -f "$tmp"' EXIT
python3 - grafana/dashboards/*.json >"$tmp" <<'EOF'
import json, sys
print("groups:")
for n, path in enumerate(sys.argv[1:]):
    d = json.load(open(path))
    exprs = [t["expr"] for p in d["panels"] for t in p.get("targets", [])]
    exprs += [a["expr"] for a in d.get("annotations", {}).get("list", []) if "expr" in a]
    print("  - name: %s\n    rules:" % json.dumps(path))
    for i, e in enumerate(exprs):
        e = e.replace("$symbol", "RELIANCE").replace("$sector", ".*")
        print("      - record: dashboard_%d_query_%d\n        expr: %s" % (n, i, json.dumps(e)))
EOF
bin/promtool check rules "$tmp" >/dev/null
echo "dashboards OK: $(ls grafana/dashboards/*.json | wc -l) files"
