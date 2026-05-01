import { NextResponse } from "next/server";
import type { NextRequest } from "next/server";

const PROXY_TARGET = process.env.API_PROXY_TARGET || "http://127.0.0.1:7330";

export function middleware(request: NextRequest) {
  const targetUrl = new URL(
    request.nextUrl.pathname + request.nextUrl.search,
    PROXY_TARGET,
  );
  return NextResponse.rewrite(targetUrl);
}

export const config = {
  matcher: ["/api/:path*", "/mpk-:postKey", "/p-:qid"],
};
