"use client";

import { ExternalLink } from "lucide-react";
import { useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import { fetchConfig } from "@/lib/api";

/** Button to the public demo; the URL depends on how the VM is exposed, so it comes from the server. */
export function DemoLink({ size = "lg", label = "Open live demo" }: { size?: "default" | "lg"; label?: string }) {
  const [url, setUrl] = useState<string | null>(null);
  useEffect(() => {
    const ctrl = new AbortController();
    fetchConfig(ctrl.signal)
      .then((c) => setUrl(c.demoUrl))
      .catch(() => setUrl(""));
    return () => ctrl.abort();
  }, []);
  if (url === "") {
    return (
      <Button size={size} variant="outline" disabled>
        Demo coming soon
      </Button>
    );
  }
  return (
    <Button size={size} asChild>
      <a href={url ?? "/demo"} target={url ? "_blank" : undefined} rel="noopener">
        {label}
        <ExternalLink aria-hidden />
      </a>
    </Button>
  );
}
