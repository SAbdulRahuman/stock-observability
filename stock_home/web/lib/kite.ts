import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { z } from "zod";

const kiteSetupSchema = z.object({
  available: z.boolean(),
  configured: z.boolean(),
  source: z.string().optional(),
  editable: z.boolean(),
  readOnlyReason: z.enum(["remote", "managed", "unavailable"]).optional(),
  apiKeyHint: z.string().optional(),
  sessionValid: z.boolean(),
  userId: z.string().optional(),
  sessionExpiresAt: z.string().nullish(),
  redirectUrl: z.string(),
  loginUrl: z.string(),
  consoleUrl: z.string(),
});
export type KiteSetup = z.infer<typeof kiteSetupSchema>;

const KITE_SETUP_KEY = ["kite-setup"] as const;

/** Kite Connect credentials are 6+ letters or digits; Kite itself only checks them at login. */
export const kiteFieldPattern = /^[A-Za-z0-9]{6,128}$/;

/** Messages for the `kite_error` codes stock_exporter appends after the login callback. */
export const kiteLoginErrors: Record<string, string> = {
  exchange: "Kite rejected the login. The API secret is probably wrong: check it in the Kite developer console and save it again.",
  cancelled: "The Kite login was cancelled. Try again when you are ready.",
  state: "The login link expired. Start the login again.",
  not_configured: "Save your API key and secret before logging in.",
};

async function request(method: string, body?: unknown): Promise<KiteSetup> {
  const res = await fetch("/api/v1/kite/setup", {
    method,
    cache: "no-store",
    headers: { Accept: "application/json", ...(body ? { "Content-Type": "application/json" } : {}) },
    body: body ? JSON.stringify(body) : undefined,
  });
  const env = (await res.json().catch(() => ({}))) as { data?: unknown; error?: string };
  if (!res.ok) throw new Error(env.error ?? `HTTP ${res.status}`);
  return kiteSetupSchema.parse(env.data);
}

/** Polls faster while waiting for the browser login to finish. */
export function useKiteSetup(pollMs: number | false = false) {
  return useQuery({ queryKey: KITE_SETUP_KEY, queryFn: () => request("GET"), refetchInterval: pollMs });
}

export function useSaveKiteCredentials() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (c: { apiKey: string; apiSecret: string }) => request("PUT", c),
    onSuccess: (data) => {
      qc.setQueryData(KITE_SETUP_KEY, data);
      void qc.invalidateQueries({ queryKey: ["section"] });
    },
  });
}

export function useClearKiteCredentials() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () => request("DELETE"),
    onSuccess: (data) => {
      qc.setQueryData(KITE_SETUP_KEY, data);
      void qc.invalidateQueries({ queryKey: ["section"] });
    },
  });
}

export function readOnlyMessage(k: KiteSetup): string | null {
  switch (k.readOnlyReason) {
    case "remote":
      return "For security, the API key and secret can only be changed on the computer running iSee Ticks (open http://127.0.0.1 there).";
    case "managed":
      return "The API key and secret are set by the deployment (for example a Kubernetes Secret), so they cannot be changed here.";
    case "unavailable":
      return "stock_home cannot reach stock_exporter (--exporter.url), so the Kite setup is unavailable.";
    default:
      return null;
  }
}
