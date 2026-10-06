import type { NextConfig } from "next";

// Static export: the Go server embeds web/out and serves it together with /api/v1.
const nextConfig: NextConfig = {
  output: "export",
  reactStrictMode: true,
  poweredByHeader: false,
  images: { unoptimized: true },
};

export default nextConfig;
