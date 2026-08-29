import { APIRequestContext } from "@playwright/test";
import { test, expect } from "../lib/fixtures";
import {
  waitForBackend,
  apiLogin,
  deleteAllPosts,
  getPostKey,
  createPost,
} from "../lib/helpers";

test.beforeEach(async ({ request }) => {
  await waitForBackend(request);
  const auth = await apiLogin(request);
  await deleteAllPosts(request, auth.token);
});

// The caching contract of specs/backend/caching.md, exercised through the
// production Caddy path: browser TTL 300 s, shared-cache TTL 3600 s, ETag =
// xxhash64 of the exact response bytes, Last-Modified = Post.CreatedAt. The
// raw and html variants share every header value except the ETag — their
// bodies differ, so their validators must differ too, or the CDN would
// answer one variant's revalidation with the other's ETag.
//
// On the wire the ETag is the xxhash core plus an optional encoding suffix:
// Caddy appends "-gzip"/"-zstd" when it compresses the representation
// (specs/backend/compression.md), so small uncompressed bodies carry the
// bare form. Identity lives in the core; the suffix is environmental.

function etagCore(etag: string | undefined): string {
  expect(etag).toMatch(/^"[0-9a-f]{16}(-[a-z0-9]+)?"$/);
  return (etag as string).replace(/-[a-z0-9]+"$/, '"');
}

async function seedPost(request: APIRequestContext) {
  const auth = await apiLogin(request);
  const postKey = await getPostKey(request, auth.token);
  return createPost(request, auth.token, postKey, "Caching E2E", "cached body");
}

test("html and raw responses carry the spec'd cache policy", async ({
  request,
}) => {
  const { id } = await seedPost(request);

  const html = await request.get(`/${id}`);
  expect(html.status()).toBe(200);
  expect((html.headers()["content-type"] || "").toLowerCase()).toContain("text/html");
  expect(html.headers()["cache-control"]).toBe("public, max-age=300, s-maxage=3600");
  expect(html.headers()["cache-tag"]).toBe(`post-${id}`);
  expect((html.headers()["vary"] || "").toLowerCase()).toContain("accept-encoding");

  const raw = await request.get(`/${id}?format=raw`);
  expect(raw.status()).toBe(200);
  expect((raw.headers()["content-type"] || "").toLowerCase()).toContain("text/markdown");
  expect(await raw.text()).toContain("# Caching E2E");
  expect(raw.headers()["cache-control"]).toBe("public, max-age=300, s-maxage=3600");
  expect(raw.headers()["cache-tag"]).toBe(`post-${id}`);
  expect((raw.headers()["vary"] || "").toLowerCase()).toContain("accept-encoding");
});

test("each form's ETag hashes its own bytes and is stable across fetches", async ({
  request,
}) => {
  const { id } = await seedPost(request);

  const htmlCore = etagCore((await request.get(`/${id}`)).headers()["etag"]);
  const rawCore = etagCore((await request.get(`/${id}?format=raw`)).headers()["etag"]);

  // The two forms must never share a validator.
  expect(htmlCore).not.toBe(rawCore);

  // The origin render cache serves byte-identical output per variant, so
  // repeat fetches return the same validator core.
  expect(etagCore((await request.get(`/${id}`)).headers()["etag"])).toBe(htmlCore);
  expect(etagCore((await request.get(`/${id}?format=raw`)).headers()["etag"])).toBe(rawCore);
});

test("If-None-Match revalidates each form separately", async ({ request }) => {
  const { id } = await seedPost(request);

  // Revalidate with the validator exactly as received — what a browser or
  // CDN stored, suffix included.
  const htmlTag = (await request.get(`/${id}`)).headers()["etag"];
  const rawTag = (await request.get(`/${id}?format=raw`)).headers()["etag"];

  const html304 = await request.get(`/${id}`, {
    headers: { "if-none-match": htmlTag as string },
  });
  expect(html304.status()).toBe(304);
  expect(etagCore(html304.headers()["etag"])).toBe(etagCore(htmlTag));

  const raw304 = await request.get(`/${id}?format=raw`, {
    headers: { "if-none-match": rawTag as string },
  });
  expect(raw304.status()).toBe(304);
  expect(etagCore(raw304.headers()["etag"])).toBe(etagCore(rawTag));

  // Cross-form confusion switch: presenting the raw validator to the html
  // form must fall through to a full 200 — a shared validator would wrongly
  // answer 304 here.
  const mixed = await request.get(`/${id}`, {
    headers: { "if-none-match": rawTag as string },
  });
  expect(mixed.status()).toBe(200);
});

test("Last-Modified is the post creation time on both forms", async ({
  request,
}) => {
  const { id } = await seedPost(request);

  const htmlLm = (await request.get(`/${id}`)).headers()["last-modified"];
  const rawLm = (await request.get(`/${id}?format=raw`)).headers()["last-modified"];

  // RFC 9110 HTTP-date (Go's http.TimeFormat, always GMT).
  const httpDate = /^[A-Z][a-z]{2}, \d{2} [A-Z][a-z]{2} \d{4} \d{2}:\d{2}:\d{2} GMT$/;
  expect(htmlLm).toMatch(httpDate);
  expect(rawLm).toMatch(httpDate);
  // Posts are write-once: both variants stamp the same Post.CreatedAt.
  expect(htmlLm).toBe(rawLm);
  // The post was created moments ago — a Last-Modified pointing anywhere
  // else means it stopped reflecting creation time.
  expect(Math.abs(Date.parse(htmlLm as string) - Date.now())).toBeLessThan(5 * 60 * 1000);
});
