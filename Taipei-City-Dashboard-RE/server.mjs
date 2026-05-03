
import http from "node:http";
import { handleNavigate } from "./src/api/navigate.mjs";
import { handleNavigateAvoid } from "./src/api/navigate-avoid.mjs";

const PORT = Number(process.env.PORT) || 3000;
const HOSTNAME = process.env.HOSTNAME || "0.0.0.0";

const ROUTES = {
  "/api/navigate": handleNavigate,
  "/api/navigate-avoid": handleNavigateAvoid,
};

function sendJson(res, status, body) {
  const payload = JSON.stringify(body);
  res.writeHead(status, {
    "Content-Type": "application/json; charset=utf-8",
    "Content-Length": Buffer.byteLength(payload),
  });
  res.end(payload);
}

const server = http.createServer(async (req, res) => {
  // node:url URL needs an absolute base; the host header is fine here since
  // we only ever read pathname/searchParams.
  const url = new URL(req.url, `http://${req.headers.host || "localhost"}`);

  if (req.method !== "GET") {
    sendJson(res, 405, { error: "Method not allowed" });
    return;
  }

  const handler = ROUTES[url.pathname];
  if (!handler) {
    sendJson(res, 404, { error: "Not found" });
    return;
  }

  try {
    const result = await handler(url.searchParams);
    sendJson(res, result.status, result.body);
  } catch (err) {
    const message =
      err instanceof Error ? err.message : "Unknown server error";
    sendJson(res, 500, { error: message });
  }
});

server.listen(PORT, HOSTNAME, () => {
  console.log(`Server listening on http://${HOSTNAME}:${PORT}`);
});
