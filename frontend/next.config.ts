import type { NextConfig } from "next";
import createNextIntlPlugin from "next-intl/plugin";

const nextConfig: NextConfig = {
  output: "standalone",

  async rewrites() {
    const serverPort = process.env.MARKPOST_SERVER__PORT || "7330";
    return [
      {
        source: "/api/:path*",
        destination: `http://localhost:${serverPort}/api/:path*`,
      },
    ];
  },
};

const withNextIntl = createNextIntlPlugin("./src/i18n/request.ts");

export default withNextIntl(nextConfig);
