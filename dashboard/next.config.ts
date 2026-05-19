import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  output: "export",
  trailingSlash: true,
  images: { unoptimized: true },
  // The dashboard is embedded into the Go binary at build time — produces /out.
};

export default nextConfig;
