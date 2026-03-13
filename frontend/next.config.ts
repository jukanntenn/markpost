import type { NextConfig } from "next";
import createNextIntlPlugin from "next-intl/plugin";

const nextConfig: NextConfig = {
  output: "standalone",

  async rewrites() {
    return [
      {
        source: "/api/:path*",
        destination: "http://localhost:8080/api/:path*",
      },
    ];
  },

  env: {
    NEXT_PUBLIC_PRIMARY_DOMAIN: process.env.PRIMARY_DOMAIN,
    NEXT_PUBLIC_USE_HTTPS: process.env.USE_HTTPS,
  },
};

const withNextIntl = createNextIntlPlugin("./src/i18n/request.ts");

export default withNextIntl(nextConfig);
