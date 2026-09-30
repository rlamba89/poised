import type { NextConfig } from "next";

// The browser only ever talks to Next.js; /api/* is proxied to the Go API,
// so there is one origin, no CORS, and the auth cookie just works.
const apiUrl = process.env.API_URL ?? "http://localhost:8080";

const nextConfig: NextConfig = {
  // packages/clinical is TypeScript source shared by the browser code.
  transpilePackages: ["@sj/clinical"],
  async rewrites() {
    return [{ source: "/api/:path*", destination: `${apiUrl}/api/:path*` }];
  },
};

export default nextConfig;
