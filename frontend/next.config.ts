import type { NextConfig } from "next";
import createNextIntlPlugin from "next-intl/plugin";

const nextConfig: NextConfig = {
  output: "standalone",

  async rewrites() {
    const target = process.env.API_PROXY_TARGET || "http://127.0.0.1:7330";
    return [
      {
        source: "/api/:path*",
        destination: `${target}/api/:path*`,
      },
      {
        source: "/mpk-:postKey",
        destination: `${target}/mpk-:postKey`,
      },
      {
        source: "/p-:qid",
        destination: `${target}/p-:qid`,
      },
    ];
  },
};

const withNextIntl = createNextIntlPlugin("./src/i18n/request.ts");

export default withNextIntl(nextConfig);
