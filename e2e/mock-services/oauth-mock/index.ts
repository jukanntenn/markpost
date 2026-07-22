import { OAuth2Server } from "oauth2-mock-server";

const PORT = parseInt(process.env.PORT || "3001", 10);
const MOCK_USER_ID = parseInt(process.env.MOCK_USER_ID || "12345", 10);
const MOCK_USERNAME = process.env.MOCK_USERNAME || "testuser";
const MOCK_EMAIL = process.env.MOCK_EMAIL || "test@example.com";

const server = new OAuth2Server(undefined, undefined, {
  endpoints: {
    authorize: "/login/oauth/authorize",
    token: "/login/oauth/access_token",
  },
});

await server.issuer.keys.generate("RS256");

// Custom route: GitHub-style user info
server.service.addRoute("GET", "/user", (_req, res) => {
  res.setHeader("Content-Type", "application/json");
  res.end(
    JSON.stringify({
      id: MOCK_USER_ID,
      login: MOCK_USERNAME,
      avatar_url: "https://example.com/avatar.png",
      email: MOCK_EMAIL,
    })
  );
});

// Custom route: GitHub-style user emails
server.service.addRoute("GET", "/user/emails", (_req, res) => {
  res.setHeader("Content-Type", "application/json");
  res.end(
    JSON.stringify([
      { email: MOCK_EMAIL, primary: true, verified: true },
      { email: "secondary@example.com", primary: false, verified: true },
    ])
  );
});

// Health check
server.service.addRoute("GET", "/health", (_req, res) => {
  res.setHeader("Content-Type", "application/json");
  res.end(JSON.stringify({ status: "ok" }));
});

// Test helper: no-op request log clear (for test compatibility)
server.service.addRoute("POST", "/requests/clear", (_req, res) => {
  res.setHeader("Content-Type", "application/json");
  res.end(JSON.stringify({ success: true }));
});

await server.start(PORT, "0.0.0.0");
console.log(`OAuth2 mock server running on port ${PORT}`);
console.log(`Issuer URL: ${server.issuer.url}`);
