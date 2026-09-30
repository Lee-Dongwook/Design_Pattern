import type { NextConfig } from "next";

const controlPlaneURL = process.env.CONTROL_PLANE_URL ?? "http://localhost:8080";

const nextConfig: NextConfig = {
  async rewrites() {
    return [{ source: "/api/control-plane/:path*", destination: `${controlPlaneURL}/:path*` }];
  },
};

export default nextConfig;
