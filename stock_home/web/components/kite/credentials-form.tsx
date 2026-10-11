"use client";

import { Check, Copy, Eye, EyeOff } from "lucide-react";
import { useId, useState } from "react";
import { Button } from "@/components/ui/button";
import { kiteFieldPattern, useSaveKiteCredentials } from "@/lib/kite";
import { cn } from "@/lib/utils";

const inputClass = "h-9 w-full rounded-md border bg-background px-3 font-mono text-sm focus-visible:ring-2 focus-visible:ring-ring focus-visible:outline-none";

/** API key + secret form. The secret is write-only: it is never loaded back from the server. */
export function KiteCredentialsForm({ submitLabel = "Save", onSaved, onCancel }: { submitLabel?: string; onSaved?: () => void; onCancel?: () => void }) {
  const id = useId();
  const save = useSaveKiteCredentials();
  const [apiKey, setApiKey] = useState("");
  const [apiSecret, setApiSecret] = useState("");
  const [show, setShow] = useState(false);
  const [touched, setTouched] = useState(false);
  const keyOk = kiteFieldPattern.test(apiKey.trim());
  const secretOk = kiteFieldPattern.test(apiSecret.trim());

  const submit = (e: React.FormEvent) => {
    e.preventDefault();
    setTouched(true);
    if (!keyOk || !secretOk) return;
    save.mutate(
      { apiKey: apiKey.trim(), apiSecret: apiSecret.trim() },
      {
        onSuccess: () => {
          setApiSecret("");
          onSaved?.();
        },
      },
    );
  };

  return (
    <form onSubmit={submit} className="space-y-3" noValidate>
      <div className="space-y-1">
        <label htmlFor={`${id}-key`} className="text-sm font-medium">
          API key
        </label>
        <input
          id={`${id}-key`}
          value={apiKey}
          onChange={(e) => setApiKey(e.target.value)}
          autoComplete="off"
          spellCheck={false}
          maxLength={128}
          aria-invalid={touched && !keyOk}
          className={inputClass}
          placeholder="e.g. abcd1234efgh5678"
        />
        {touched && !keyOk && <p className="text-xs text-critical">Paste the API key from your Kite Connect app (letters and digits only).</p>}
      </div>
      <div className="space-y-1">
        <label htmlFor={`${id}-secret`} className="text-sm font-medium">
          API secret
        </label>
        <div className="flex gap-2">
          <input
            id={`${id}-secret`}
            type={show ? "text" : "password"}
            value={apiSecret}
            onChange={(e) => setApiSecret(e.target.value)}
            autoComplete="new-password"
            spellCheck={false}
            maxLength={128}
            aria-invalid={touched && !secretOk}
            className={inputClass}
          />
          <Button type="button" variant="outline" size="icon" onClick={() => setShow((s) => !s)} aria-label={show ? "Hide API secret" : "Show API secret"}>
            {show ? <EyeOff /> : <Eye />}
          </Button>
        </div>
        {touched && !secretOk ? (
          <p className="text-xs text-critical">Paste the API secret from your Kite Connect app (letters and digits only).</p>
        ) : (
          <p className="text-xs text-muted-foreground">Stored only on this computer, readable only by your user. It is never shown again.</p>
        )}
      </div>
      {save.isError && (
        <p role="alert" className="text-sm text-critical">
          Could not save: {save.error.message}
        </p>
      )}
      <div className="flex justify-end gap-2">
        {onCancel && (
          <Button type="button" variant="outline" onClick={onCancel}>
            Cancel
          </Button>
        )}
        <Button type="submit" disabled={save.isPending}>
          {save.isPending ? "Saving…" : submitLabel}
        </Button>
      </div>
    </form>
  );
}

export function CopyField({ label, value }: { label: string; value: string }) {
  const id = useId();
  const [copied, setCopied] = useState(false);
  const copy = () => {
    void navigator.clipboard?.writeText(value).then(() => {
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    });
  };
  return (
    <div className="space-y-1">
      <label htmlFor={id} className="text-sm font-medium">
        {label}
      </label>
      <div className="flex gap-2">
        <input id={id} readOnly value={value} onFocus={(e) => e.target.select()} className={cn(inputClass, "bg-muted/40")} />
        <Button type="button" variant="outline" onClick={copy} aria-label={`Copy ${label}`}>
          {copied ? <Check /> : <Copy />}
          {copied ? "Copied" : "Copy"}
        </Button>
      </div>
    </div>
  );
}
