"use client";

import { CheckCircle2 } from "lucide-react";
import { useState } from "react";
import { Button } from "@/components/ui/button";
import { interests, joinWaitlist } from "@/lib/api";

const field =
  "w-full rounded-md border bg-background px-3 py-2 text-sm placeholder:text-muted-foreground focus-visible:outline-2 focus-visible:outline-ring";

export function WaitlistForm() {
  const [state, setState] = useState<"idle" | "sending" | "done">("idle");
  const [error, setError] = useState<{ msg: string; field?: string } | null>(null);

  async function onSubmit(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const f = new FormData(e.currentTarget);
    const s = (k: string) => String(f.get(k) ?? "");
    setState("sending");
    setError(null);
    const res = await joinWaitlist({ email: s("email"), name: s("name"), interest: s("interest"), message: s("message"), website: s("website") });
    if (res.ok) {
      setState("done");
    } else {
      setState("idle");
      setError({ msg: res.error, field: res.field });
    }
  }

  if (state === "done") {
    return (
      <div role="status" className="flex items-start gap-3 rounded-xl border border-ok/30 bg-ok/10 p-5">
        <CheckCircle2 className="mt-0.5 size-5 shrink-0 text-ok" aria-hidden />
        <div>
          <p className="font-semibold">You are on the list.</p>
          <p className="text-sm text-muted-foreground">We will email you when early access opens. No spam, ever.</p>
        </div>
      </div>
    );
  }

  return (
    <form onSubmit={onSubmit} className="grid gap-4">
      <div className="grid gap-1.5">
        <label htmlFor="wl-email" className="text-sm font-medium">
          Email
        </label>
        <input id="wl-email" name="email" type="email" required maxLength={254} autoComplete="email" placeholder="you@example.com" className={field} aria-invalid={error?.field === "email"} />
      </div>
      <div className="grid gap-4 sm:grid-cols-2">
        <div className="grid gap-1.5">
          <label htmlFor="wl-name" className="text-sm font-medium">
            Name <span className="text-muted-foreground">(optional)</span>
          </label>
          <input id="wl-name" name="name" maxLength={100} autoComplete="name" className={field} aria-invalid={error?.field === "name"} />
        </div>
        <div className="grid gap-1.5">
          <label htmlFor="wl-interest" className="text-sm font-medium">
            I am a…
          </label>
          <select id="wl-interest" name="interest" className={field} defaultValue="">
            {interests.map((i) => (
              <option key={i.value} value={i.value}>
                {i.label}
              </option>
            ))}
          </select>
        </div>
      </div>
      <div className="grid gap-1.5">
        <label htmlFor="wl-message" className="text-sm font-medium">
          What would you like to observe? <span className="text-muted-foreground">(optional)</span>
        </label>
        <textarea id="wl-message" name="message" rows={3} maxLength={1000} className={field} aria-invalid={error?.field === "message"} />
      </div>
      {/* Honeypot: hidden from people and assistive tech, filled in by bots. */}
      <div aria-hidden className="absolute -left-[9999px] h-0 w-0 overflow-hidden">
        <label htmlFor="wl-website">Website</label>
        <input id="wl-website" name="website" tabIndex={-1} autoComplete="off" />
      </div>
      {error && (
        <p role="alert" className="text-sm text-critical">
          {error.msg}
        </p>
      )}
      <div className="flex flex-wrap items-center gap-3">
        <Button type="submit" size="lg" disabled={state === "sending"}>
          {state === "sending" ? "Joining…" : "Join the waitlist"}
        </Button>
        <p className="text-xs text-muted-foreground">
          We only use your email to contact you about iSee Ticks. See <a href="/privacy" className="underline">privacy</a>.
        </p>
      </div>
    </form>
  );
}
