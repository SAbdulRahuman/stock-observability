import type { Metadata } from "next";
import { site } from "@/lib/site";

export const metadata: Metadata = { title: "Privacy" };

export default function PrivacyPage() {
  return (
    <article className="prose mx-auto max-w-3xl px-4 py-16 dark:prose-invert">
      <h1>Privacy</h1>
      <p>This page explains what {site.name} collects on this website and the public demo.</p>
      <h2>Waitlist</h2>
      <p>
        When you join the waitlist we store the email address you enter and, if you provide them, your name, interest and
        message, together with the time of sign-up. We use them only to contact you about {site.name}. We do not sell or share
        them. To be removed, reply to any email from us.
      </p>
      <h2>Logs</h2>
      <p>
        Like any web server, ours keeps short-lived request logs (time, path, status) for security and troubleshooting. IP
        addresses are used in memory for rate limiting and are not stored with your sign-up.
      </p>
      <h2>Cookies and tracking</h2>
      <p>
        The website sets no tracking cookies and uses no third-party analytics. The only browser storage it uses is your
        light/dark theme preference. Grafana in the demo may set a session cookie for anonymous viewing.
      </p>
      <h2>Market data</h2>
      <p>The public demo runs on simulated data. Nothing on this site is investment advice.</p>
    </article>
  );
}
