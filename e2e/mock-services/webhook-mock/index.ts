import * as http from "http";

interface WebhookRequest {
  timestamp: string;
  method: string;
  path: string;
  headers: Record<string, string | string[] | undefined>;
  body: unknown;
}

const receivedWebhooks: WebhookRequest[] = [];

const server = http.createServer((req, res) => {
  res.setHeader("Access-Control-Allow-Origin", "*");
  res.setHeader("Access-Control-Allow-Methods", "GET, POST, OPTIONS");
  res.setHeader("Access-Control-Allow-Headers", "Content-Type");

  if (req.method === "OPTIONS") {
    res.writeHead(200);
    res.end();
    return;
  }

  const url = new URL(req.url || "/", `http://${req.headers.host}`);

  if (req.method === "POST" && url.pathname.startsWith("/webhook")) {
    let body = "";
    req.on("data", (chunk) => { body += chunk.toString(); });
    req.on("end", () => {
      let parsedBody: unknown;
      try { parsedBody = JSON.parse(body); } catch { parsedBody = body; }

      receivedWebhooks.push({
        timestamp: new Date().toISOString(),
        method: req.method || "POST",
        path: url.pathname,
        headers: req.headers as Record<string, string | string[] | undefined>,
        body: parsedBody,
      });

      res.writeHead(200, { "Content-Type": "application/json" });
      res.end(JSON.stringify({ code: 0, msg: "success", data: {} }));
    });
    return;
  }

  if (req.method === "GET" && url.pathname === "/webhooks") {
    res.writeHead(200, { "Content-Type": "application/json" });
    res.end(JSON.stringify(receivedWebhooks));
    return;
  }

  if (req.method === "POST" && url.pathname === "/webhooks/clear") {
    receivedWebhooks.length = 0;
    res.writeHead(200, { "Content-Type": "application/json" });
    res.end(JSON.stringify({ success: true }));
    return;
  }

  if (req.method === "GET" && url.pathname === "/health") {
    res.writeHead(200, { "Content-Type": "application/json" });
    res.end(JSON.stringify({ status: "ok" }));
    return;
  }

  res.writeHead(404, { "Content-Type": "application/json" });
  res.end(JSON.stringify({ error: "Not found" }));
});

const port = parseInt(process.env.PORT || "3002", 10);
server.listen(port, "0.0.0.0", () => {
  console.log(`Webhook mock server running on port ${port}`);
});
