"use client";

import { ExternalLink, Lock, LogIn, RefreshCw, Trash2 } from "lucide-react";
import { useSearchParams } from "next/navigation";
import { useState } from "react";
import { SettingsCard } from "@/components/settings/cards";
import { Badge, type BadgeVariant } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Dialog, DialogClose, DialogContent, DialogFooter } from "@/components/ui/dialog";
import { formatTimeIST } from "@/lib/format";
import { type KiteSetup, kiteLoginErrors, readOnlyMessage, useClearKiteCredentials, useKiteSetup } from "@/lib/kite";
import { CopyField, KiteCredentialsForm } from "./credentials-form";

function state(k: KiteSetup): { label: string; variant: BadgeVariant } {
  if (!k.available) return { label: "Unavailable", variant: "default" };
  if (!k.configured) return { label: "Not set up", variant: "warning" };
  if (!k.sessionValid) return { label: "Login needed", variant: "critical" };
  return { label: "Connected", variant: "ok" };
}

/** Settings card for the Zerodha Kite Connect API key and secret held by stock_exporter. */
export function KiteCard({ icon, notify }: { icon: React.ComponentType<{ className?: string }>; notify: (ok: boolean, text: string) => void }) {
  const params = useSearchParams();
  const loginError = params?.get("kite_error") ?? null;
  const justConnected = params?.get("kite") === "connected";
  const setup = useKiteSetup(justConnected ? 5_000 : 30_000);
  const clear = useClearKiteCredentials();
  const [editing, setEditing] = useState(loginError === "exchange");
  const [disconnecting, setDisconnecting] = useState(false);
  const [testMessage, setTestMessage] = useState<string | null>(null);
  const k = setup.data;
  const testConnection = async () => {
    setTestMessage(null);
    try {
      const result = await setup.refetch();
      if (!result.data?.available) {
        setTestMessage("stock_exporter is unavailable. Check its service and try again.");
      } else if (!result.data.configured) {
        setTestMessage("No Kite credentials are saved yet. Enter the API key and secret, then select Test & Save.");
      } else if (result.data.sessionValid) {
        setTestMessage(`Kite connection is active${result.data.userId ? ` for ${result.data.userId}` : ""}.`);
      } else {
        setTestMessage("Credentials are saved. Sign in to Kite to test them and activate the live feed.");
      }
    } catch {
      setTestMessage("Could not test the connection. Check stock_exporter and try again.");
    }
  };

  return (
    <SettingsCard title="Configure Zerodha Kite Connect" subtitle="Connect the app that provides your live NSE market data" icon={icon}>
      {setup.isError ? (
        <p className="text-muted-foreground">Kite setup is unavailable: {setup.error.message}</p>
      ) : !k ? (
        <p className="text-muted-foreground">Loading…</p>
      ) : (
        <>
          <div className="flex flex-wrap items-center justify-between gap-2">
            <div className="flex items-center gap-2"><span className="text-muted-foreground">Connection status</span><Badge variant={state(k).variant}>{state(k).label}</Badge></div>
            <Button type="button" size="sm" variant="outline" onClick={() => void testConnection()} disabled={setup.isFetching}>
              <RefreshCw className={setup.isFetching ? "animate-spin" : ""} /> Refresh status
            </Button>
          </div>
          {k.configured && <p className="text-xs text-muted-foreground">Saved API key <span className="font-mono">{k.apiKeyHint}</span>{k.sessionValid && k.userId ? ` · Connected as ${k.userId}` : ""}{k.sessionExpiresAt ? ` · valid until ${formatTimeIST(k.sessionExpiresAt)} IST` : ""}</p>}
          {loginError && <p role="alert" className="rounded-md border border-warning/40 bg-warning/10 p-3 text-warning">{kiteLoginErrors[loginError] ?? "Kite login did not complete. Try again."}</p>}
          {justConnected && k.sessionValid && <p role="status" className="text-sm text-ok">Kite connection verified. Live data is streaming.</p>}
          {testMessage && <p role="status" className="text-sm text-info">{testMessage}</p>}
          {readOnlyMessage(k) && (
            <p className="flex items-start gap-2 text-xs text-muted-foreground">
              <Lock className="mt-0.5 size-3.5 shrink-0" aria-hidden />
              {readOnlyMessage(k)}
            </p>
          )}
          {k.editable && (!k.configured || editing) ? (
            <div className="space-y-4 border-t pt-4">
              <p className="text-sm">Enter the API key and secret from your Kite Connect app. Save them, then sign in to Kite to test the credentials and start streaming.</p>
              <KiteCredentialsForm
                submitLabel="Save credentials"
                onCancel={k.configured ? () => setEditing(false) : undefined}
                onSaved={() => {
                  setEditing(false);
                  setTestMessage("Configuration saved. Sign in to Kite to test the API secret and activate live data.");
                  notify(true, "Kite configuration saved. Sign in to test it.");
                }}
              />
              <CopyField label="Kite app redirect URL" value={k.redirectUrl} />
              <p className="text-xs text-muted-foreground">Paste this exact URL into the Redirect URL field in your Kite developer app.</p>
            </div>
          ) : (
            <>
              {k.editable && <div className="flex flex-wrap justify-end gap-2 border-t pt-4">
                <Button type="button" size="sm" variant="outline" onClick={() => setEditing(true)}>Edit configuration</Button>
                {!k.sessionValid && <Button asChild size="sm"><a href={k.loginUrl}><LogIn /> Test with Kite</a></Button>}
                <Button type="button" size="sm" variant="outline" className="text-critical" onClick={() => setDisconnecting(true)}><Trash2 /> Disconnect</Button>
              </div>}
              {k.configured && !k.sessionValid && <p className="text-xs text-muted-foreground">The app details are saved. Select Test with Kite to verify them. Kite requires a new sign-in each trading day.</p>}
              <CopyField label="Kite app redirect URL" value={k.redirectUrl} />
              <Button asChild type="button" size="sm" variant="outline"><a href={k.consoleUrl} target="_blank" rel="noopener noreferrer"><ExternalLink /> Kite developer console</a></Button>
            </>
          )}

          <Dialog open={disconnecting} onOpenChange={setDisconnecting}>
            <DialogContent title="Disconnect Zerodha?" description="This removes the API key, the secret and today's Kite login from this computer. Live market data stops until you set it up again.">
              {clear.isError && <p className="text-sm text-critical">Could not disconnect: {clear.error.message}</p>}
              <DialogFooter>
                <DialogClose asChild>
                  <Button variant="outline">Cancel</Button>
                </DialogClose>
                <Button
                  variant="critical"
                  disabled={clear.isPending}
                  onClick={() =>
                    clear.mutate(undefined, {
                      onSuccess: () => {
                        setDisconnecting(false);
                        setEditing(false);
                        setTestMessage(null);
                        notify(true, "Zerodha disconnected.");
                      },
                    })
                  }
                >
                  {clear.isPending ? "Disconnecting…" : "Disconnect"}
                </Button>
              </DialogFooter>
            </DialogContent>
          </Dialog>
        </>
      )}
    </SettingsCard>
  );
}
