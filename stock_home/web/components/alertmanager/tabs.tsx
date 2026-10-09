"use client";

import { ChevronDown, ChevronRight, ExternalLink, X } from "lucide-react";
import { Fragment, useState } from "react";
import { Empty } from "@/components/dashboard/section-card";
import { LabelList } from "@/components/prometheus/tabs";
import { Badge, type BadgeVariant, severityVariant } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { relativeTime } from "@/lib/format";
import { type AMAlert, type AMGroup, type AMSilence, matcherText, type MatcherInput, type RouteNode, type AMStatus } from "@/lib/am";
import { safeUrl } from "@/lib/openbb";
import { cn } from "@/lib/utils";

const select = "h-9 rounded-lg border bg-card px-2 text-sm";
const search = "h-9 min-w-48 flex-1 rounded-lg border bg-card px-3 text-sm";

export function alertStatus(a: AMAlert): "firing" | "silenced" | "inhibited" | "unprocessed" {
  if (a.silencedBy.length > 0) return "silenced";
  if (a.inhibitedBy.length > 0) return "inhibited";
  return a.state === "active" ? "firing" : "unprocessed";
}
const statusVariant: Record<string, BadgeVariant> = { firing: "critical", silenced: "default", inhibited: "default", unprocessed: "warning", active: "ok", pending: "info", expired: "default" };

export function StatusBadge({ status }: { status: string }) {
  return <Badge variant={statusVariant[status] ?? "default"}>{status}</Badge>;
}

const sevRank: Record<string, number> = { critical: 0, warning: 1, info: 2 };

export function AlertsTab({ groups, now, canSilence, onSilence }: { groups: AMGroup[]; now: Date; canSilence: boolean; onSilence: (m: MatcherInput[]) => void }) {
  const [term, setTerm] = useState("");
  const [severity, setSeverity] = useState("all");
  const [status, setStatus] = useState("all");
  const [receiver, setReceiver] = useState("all");
  const [open, setOpen] = useState<Set<number>>(new Set());
  const [sel, setSel] = useState<{ g: number; fp: string } | null>(null);
  const t = term.trim().toLowerCase();
  const receivers = [...new Set(groups.map((g) => g.receiver))].sort();
  const keep = (a: AMAlert) =>
    (severity === "all" || a.labels.severity === severity) &&
    (status === "all" || alertStatus(a) === status) &&
    (!t || Object.entries(a.labels).some(([k, v]) => `${k}=${v}`.toLowerCase().includes(t)));
  const rows = groups
    .map((g, i) => ({ g, i, alerts: g.alerts.filter(keep) }))
    .filter(({ g, alerts }) => (receiver === "all" || g.receiver === receiver) && (alerts.length > 0 || (!t && severity === "all" && status === "all" && g.alerts.length === 0)));
  const picked = sel ? groups[sel.g]?.alerts.find((a) => a.fingerprint === sel.fp) : undefined;
  const filtered = t || severity !== "all" || status !== "all" || receiver !== "all";

  return (
    <div className="space-y-3">
      <div className="flex flex-wrap gap-2">
        <input aria-label="Search alerts" placeholder="Search alert name or label values…" className={search} value={term} onChange={(e) => setTerm(e.target.value)} />
        <select aria-label="Severity" className={select} value={severity} onChange={(e) => setSeverity(e.target.value)}>
          <option value="all">All severities</option>
          <option value="critical">Critical</option>
          <option value="warning">Warning</option>
          <option value="info">Info</option>
        </select>
        <select aria-label="Status" className={select} value={status} onChange={(e) => setStatus(e.target.value)}>
          <option value="all">All statuses</option>
          <option value="firing">Firing</option>
          <option value="silenced">Silenced</option>
          <option value="inhibited">Inhibited</option>
        </select>
        <select aria-label="Receiver" className={select} value={receiver} onChange={(e) => setReceiver(e.target.value)}>
          <option value="all">All receivers</option>
          {receivers.map((r) => (
            <option key={r} value={r}>
              {r}
            </option>
          ))}
        </select>
        <Button
          variant="outline"
          disabled={!filtered}
          onClick={() => {
            setTerm("");
            setSeverity("all");
            setStatus("all");
            setReceiver("all");
          }}
        >
          Clear
        </Button>
      </div>
      <p className="text-xs text-muted-foreground">Filters apply to the alert groups already loaded from Alertmanager. One group can hold several alert instances.</p>
      <div className={cn("grid gap-4", picked && "xl:grid-cols-[minmax(0,1.6fr)_minmax(0,1fr)]")}>
        <Card className="min-w-0 overflow-x-auto" aria-label="Alert groups">
          {rows.length === 0 ? (
            <Empty>{groups.length === 0 ? "Alertmanager has no alerts" : "No alert groups match the filters"}</Empty>
          ) : (
            <table className="w-full text-sm">
              <thead>
                <tr className="text-xs text-muted-foreground">
                  <th className="w-8 py-2 pl-3" />
                  <th className="py-2 text-left font-normal">Status</th>
                  <th className="py-2 text-left font-normal">Group / Alert</th>
                  <th className="py-2 text-left font-normal">Severity</th>
                  <th className="py-2 text-left font-normal">Active</th>
                  <th className="py-2 pr-3 text-left font-normal">Receiver</th>
                </tr>
              </thead>
              <tbody className="divide-y">
                {rows.map(({ g, i, alerts }) => {
                  const firing = alerts.filter((a) => alertStatus(a) === "firing").length;
                  const sev = alerts.map((a) => a.labels.severity).sort((a, b) => (sevRank[a] ?? 9) - (sevRank[b] ?? 9))[0];
                  const since = alerts.map((a) => a.startsAt).sort()[0];
                  const expanded = open.has(i);
                  return (
                    <Fragment key={i}>
                      <tr className="hover:bg-accent/40">
                        <td className="py-2 pl-3">
                          <button
                            type="button"
                            aria-expanded={expanded}
                            aria-label={`${expanded ? "Collapse" : "Expand"} group ${i + 1}`}
                            onClick={() =>
                              setOpen((s) => {
                                const n = new Set(s);
                                if (n.has(i)) n.delete(i);
                                else n.add(i);
                                return n;
                              })
                            }
                            className="rounded p-0.5 text-muted-foreground hover:text-foreground"
                          >
                            {expanded ? <ChevronDown className="size-4" /> : <ChevronRight className="size-4" />}
                          </button>
                        </td>
                        <td className="py-2 text-xs">
                          {firing} firing / {alerts.length} alert{alerts.length === 1 ? "" : "s"}
                        </td>
                        <td className="py-2">
                          <LabelList labels={g.labels} />
                        </td>
                        <td className="py-2">{sev ? <Badge variant={severityVariant(sev)}>{sev}</Badge> : "—"}</td>
                        <td className="py-2 text-muted-foreground">{since ? relativeTime(since, now) : "—"}</td>
                        <td className="py-2 pr-3">{g.receiver || "—"}</td>
                      </tr>
                      {expanded &&
                        alerts.map((a) => (
                          <tr
                            key={a.fingerprint}
                            className={cn("cursor-pointer bg-muted/30 hover:bg-accent/60", sel?.fp === a.fingerprint && sel.g === i && "bg-primary/15")}
                            onClick={() => setSel({ g: i, fp: a.fingerprint })}
                          >
                            <td />
                            <td className="py-1.5">
                              <StatusBadge status={alertStatus(a)} />
                            </td>
                            <td className="py-1.5">
                              <span className="font-medium">{a.labels.alertname}</span>
                              {a.labels.symbol && <span className="text-muted-foreground"> · {a.labels.symbol}</span>}
                            </td>
                            <td className="py-1.5">{a.labels.severity ?? "—"}</td>
                            <td className="py-1.5 text-muted-foreground">{relativeTime(a.startsAt, now)}</td>
                            <td className="py-1.5 pr-3">
                              <Button size="sm" variant="ghost" onClick={() => setSel({ g: i, fp: a.fingerprint })} aria-label={`Details for ${a.labels.alertname}`}>
                                Details
                              </Button>
                            </td>
                          </tr>
                        ))}
                    </Fragment>
                  );
                })}
              </tbody>
            </table>
          )}
        </Card>
        {picked && sel && (
          <AlertDetail
            alert={picked}
            group={groups[sel.g]}
            canSilence={canSilence}
            onClose={() => setSel(null)}
            onSilence={() => onSilence(Object.entries(picked.labels).map(([name, value]) => ({ name, op: "=", value })))}
          />
        )}
      </div>
    </div>
  );
}

function AlertDetail({ alert, group, canSilence, onClose, onSilence }: { alert: AMAlert; group: AMGroup; canSilence: boolean; onClose: () => void; onSilence: () => void }) {
  const runbook = safeUrl(alert.annotations.runbook_url);
  const prom = safeUrl(alert.generatorUrl);
  const row = (label: string, value: React.ReactNode) => (
    <div>
      <dt className="text-xs text-muted-foreground">{label}</dt>
      <dd className="mt-0.5 break-words">{value}</dd>
    </div>
  );
  return (
    <Card aria-label={`${alert.labels.alertname} details`} className="min-w-0 space-y-3 p-4 text-sm">
      <div className="flex items-center gap-2">
        <h3 className="flex-1 font-semibold">{alert.labels.alertname}</h3>
        <StatusBadge status={alertStatus(alert)} />
        <button type="button" onClick={onClose} aria-label="Close details" className="rounded p-1 text-muted-foreground hover:text-foreground">
          <X className="size-4" />
        </button>
      </div>
      <dl className="space-y-3">
        {row("Active", `${new Date(alert.startsAt).toLocaleString("en-IN")} → ${alert.endsAt ? new Date(alert.endsAt).toLocaleString("en-IN") : "—"}`)}
        {row("Labels", <LabelList labels={alert.labels} />)}
        {row("Annotations", <LabelList labels={alert.annotations} />)}
        {row("Group labels", <LabelList labels={group.labels} />)}
        {row("Fingerprint", <span className="font-mono text-xs">{alert.fingerprint}</span>)}
        {row("Silenced by", alert.silencedBy.length ? <span className="font-mono text-xs">{alert.silencedBy.join(", ")}</span> : "—")}
      </dl>
      <div className="flex flex-wrap gap-2">
        <Button size="sm" onClick={onSilence} disabled={!canSilence} title={canSilence ? undefined : "Silences are disabled on this stock_home (--alertmanager.allow-silences)"}>
          Create silence
        </Button>
        {prom && (
          <Button size="sm" variant="outline" asChild>
            <a href={prom} target="_blank" rel="noopener noreferrer">
              <ExternalLink /> Open in Prometheus
            </a>
          </Button>
        )}
        {runbook && (
          <Button size="sm" variant="outline" asChild>
            <a href={runbook} target="_blank" rel="noopener noreferrer">
              <ExternalLink /> Runbook
            </a>
          </Button>
        )}
      </div>
    </Card>
  );
}

export function SilencesTab({
  silences,
  canSilence,
  onCreate,
  onExpire,
}: {
  silences: AMSilence[];
  canSilence: boolean;
  onCreate: () => void;
  onExpire: (s: AMSilence) => void;
}) {
  const [state, setState] = useState("all");
  const [term, setTerm] = useState("");
  const [sel, setSel] = useState<string | null>(null);
  const t = term.trim().toLowerCase();
  const rows = silences.filter(
    (s) =>
      (state === "all" || s.state === state) &&
      (!t || s.comment.toLowerCase().includes(t) || s.createdBy.toLowerCase().includes(t) || s.matchers.some((m) => matcherText(m).toLowerCase().includes(t))),
  );
  return (
    <div className="space-y-3">
      <div className="flex flex-wrap gap-2">
        <Button onClick={onCreate} disabled={!canSilence} title={canSilence ? undefined : "Silences are disabled on this stock_home (--alertmanager.allow-silences)"}>
          + Create Silence
        </Button>
        <select aria-label="Silence status" className={select} value={state} onChange={(e) => setState(e.target.value)}>
          <option value="all">All statuses</option>
          <option value="active">Active</option>
          <option value="pending">Pending</option>
          <option value="expired">Expired</option>
        </select>
        <input aria-label="Search silences" placeholder="Search matcher, creator or comment…" className={search} value={term} onChange={(e) => setTerm(e.target.value)} />
      </div>
      <Card className="min-w-0 overflow-x-auto" aria-label="Silences">
        {rows.length === 0 ? (
          <Empty>{silences.length === 0 ? "No silences" : "No silences match the filters"}</Empty>
        ) : (
          <table className="w-full text-sm">
            <thead>
              <tr className="text-xs text-muted-foreground">
                <th className="py-2 pl-4 text-left font-normal">Status</th>
                <th className="py-2 text-left font-normal">Matchers</th>
                <th className="py-2 text-left font-normal">Creator</th>
                <th className="py-2 text-left font-normal">Starts / Ends</th>
                <th className="py-2 pr-4 text-right font-normal">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y">
              {rows.map((s) => (
                <Fragment key={s.id}>
                  <tr>
                    <td className="py-2 pl-4">
                      <StatusBadge status={s.state} />
                    </td>
                    <td className="py-2 font-mono text-xs">{s.matchers.map(matcherText).join(", ")}</td>
                    <td className="py-2">{s.createdBy || "—"}</td>
                    <td className="py-2 text-xs text-muted-foreground">
                      {new Date(s.startsAt).toLocaleString("en-IN")} → {new Date(s.endsAt).toLocaleString("en-IN")}
                    </td>
                    <td className="py-2 pr-4 text-right whitespace-nowrap">
                      <Button size="sm" variant="ghost" aria-expanded={sel === s.id} onClick={() => setSel(sel === s.id ? null : s.id)}>
                        Details
                      </Button>
                      {s.state !== "expired" && (
                        <Button size="sm" variant="ghost" className="text-critical" disabled={!canSilence} onClick={() => onExpire(s)} aria-label={`Expire silence ${s.id}`}>
                          Expire
                        </Button>
                      )}
                    </td>
                  </tr>
                  {sel === s.id && (
                    <tr className="bg-muted/30 text-xs">
                      <td colSpan={5} className="px-4 py-2">
                        <div>
                          ID <span className="font-mono">{s.id}</span>
                        </div>
                        <div>Comment: {s.comment || "—"}</div>
                      </td>
                    </tr>
                  )}
                </Fragment>
              ))}
            </tbody>
          </table>
        )}
      </Card>
    </div>
  );
}

function RouteTree({ node, depth = 0 }: { node: RouteNode; depth?: number }) {
  return (
    <li className={cn(depth > 0 && "ml-5 border-l pl-3")}>
      <div className="py-1.5">
        <span className="font-medium">{depth === 0 ? "root" : "route"}</span> → receiver <span className="font-mono">{node.receiver || "(inherited)"}</span>
        {node.continue && <Badge className="ml-2">continue</Badge>}
        <div className="text-xs text-muted-foreground">
          {node.matchers.length > 0 && <span className="font-mono">{node.matchers.join(", ")}</span>}
          {node.groupBy.length > 0 && <span> · group by {node.groupBy.join(", ")}</span>}
          {node.groupWait && <span> · wait {node.groupWait}</span>}
          {node.groupInterval && <span> · interval {node.groupInterval}</span>}
          {node.repeatInterval && <span> · repeat {node.repeatInterval}</span>}
        </div>
      </div>
      {node.routes.length > 0 && (
        <ul>
          {node.routes.map((r, i) => (
            <RouteTree key={i} node={r} depth={depth + 1} />
          ))}
        </ul>
      )}
    </li>
  );
}

export function RoutesTab({ status }: { status: AMStatus }) {
  if (!status.configAvailable || !status.route) {
    return <Empty>Routing configuration is not available from this Alertmanager</Empty>;
  }
  return (
    <div className="grid gap-4 lg:grid-cols-2">
      <Card className="p-4" aria-label="Route tree">
        <h2 className="mb-2 text-sm font-semibold">Route tree (read-only)</h2>
        <ul className="text-sm">
          <RouteTree node={status.route} />
        </ul>
      </Card>
      <Card className="p-4" aria-label="Receivers">
        <h2 className="mb-2 text-sm font-semibold">Receivers (read-only)</h2>
        <ul className="divide-y text-sm">
          {status.receivers.map((r) => (
            <li key={r.name} className="flex flex-wrap items-center gap-2 py-1.5">
              <span className="flex-1 font-medium">{r.name}</span>
              {r.integrations.length === 0 ? (
                <span className="text-xs text-muted-foreground">no integrations</span>
              ) : (
                r.integrations.map((x) => (
                  <Badge key={x.type} variant="outline">
                    {x.type} × {x.count}
                  </Badge>
                ))
              )}
            </li>
          ))}
        </ul>
        <p className="mt-2 text-xs text-muted-foreground">Destinations, URLs and credentials are redacted; only integration types are shown.</p>
      </Card>
    </div>
  );
}

export function StatusTab({ status, now }: { status: AMStatus; now: Date }) {
  const row = (k: string, v: React.ReactNode) => (
    <div key={k} className="flex justify-between gap-3 py-1.5 text-sm">
      <dt className="text-muted-foreground">{k}</dt>
      <dd className="text-right font-mono">{v ?? "—"}</dd>
    </div>
  );
  return (
    <Card className="max-w-xl p-4" aria-label="Alertmanager status">
      <dl className="divide-y">
        {row("Ready", status.ready ? "yes" : "no")}
        {row("Version", status.version.version)}
        {row("Revision", status.version.revision)}
        {row("Up since", status.uptime ? `${new Date(status.uptime).toLocaleString("en-IN")} (${relativeTime(status.uptime, now)})` : "—")}
        {row("Cluster", `${status.cluster || "—"} · ${status.peers} peer${status.peers === 1 ? "" : "s"}`)}
        {row("Configuration", status.configAvailable ? "loaded" : "not available")}
      </dl>
    </Card>
  );
}
