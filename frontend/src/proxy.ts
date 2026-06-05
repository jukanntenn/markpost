import { NextResponse } from "next/server";
import type { NextRequest } from "next/server";

const BACKEND_URL = process.env.BACKEND_URL || "http://127.0.0.1:7330";

export function proxy(request: NextRequest) {
  const targetUrl = new URL(
    request.nextUrl.pathname + request.nextUrl.search,
    BACKEND_URL,
  );
  return NextResponse.rewrite(targetUrl);
}

export const config = {
  matcher: ["/api/:path*", "/mpk-:postKey", "/p-:qid"],
};
