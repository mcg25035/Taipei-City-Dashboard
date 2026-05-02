#!/usr/bin/env node
/*
 * chat-proxy-test.js
 *
 * End-to-end test for the generic, runtime-registered reverse proxy.
 *
 *   POST /api/v1/proxy/register  { name, addr, prefix?, upstream?, ping? }
 *   POST /api/v1/proxy/detach    { name }
 *   GET  /api/v1/proxy/list
 *   ANY  <prefix>[/*]            -> http://<addr><upstream>[/*]   (via NoRoute)
 *
 * Defaults derived from name:
 *   prefix   = /api/v1/<name>
 *   upstream = /api/dev/<name>
 *   ping     = /api/dev/ping
 *
 * Statically-registered Gin routes ALWAYS win on collision (NoRoute only
 * fires when no static route matched).
 *
 * The script:
 *   1. Spawns a mock downstream that answers /api/dev/ping with "pong"
 *      and echoes /api/dev/chat[/*] requests as JSON.
 *   2. Registers `chat`, then exercises GET/POST/nested+query through
 *      the proxy.
 *   3. Detaches and asserts the BE returns 404 (NoRoute fallback).
 *   4. Negative: register against bad ping -> 502.
 *   5. Collision check: registers an entry with prefix /api/v1/component
 *      (which IS a real static route) and asserts the static route wins.
 *
 * No third-party deps. Node 18+ for global `fetch`.
 *
 * Usage:
 *   node chat-proxy-test.js [--be http://127.0.0.1:8080]
 */

const http = require("http");

function parseArgs(argv) {
    const out = { be: "http://127.0.0.1:8080" };
    for (let i = 2; i < argv.length; i++) {
        const a = argv[i];
        if (a === "--be") out.be = argv[++i];
    }
    return out;
}

function startMock() {
    return new Promise((resolve, reject) => {
        const seen = []; // { method, url, body }
        const server = http.createServer((req, res) => {
            const chunks = [];
            req.on("data", (c) => chunks.push(c));
            req.on("end", () => {
                const body = Buffer.concat(chunks).toString("utf8");
                if (req.url === "/api/dev/ping" && req.method === "GET") {
                    res.writeHead(200, { "Content-Type": "text/plain" });
                    res.end("pong");
                    return;
                }
                if (req.url.startsWith("/api/dev/chat")) {
                    seen.push({ method: req.method, url: req.url, body });
                    res.writeHead(200, { "Content-Type": "application/json" });
                    res.end(
                        JSON.stringify({
                            ok: true,
                            method: req.method,
                            url: req.url,
                            body,
                        }),
                    );
                    return;
                }
                res.writeHead(404);
                res.end();
            });
        });
        server.on("error", reject);
        server.listen(0, "127.0.0.1", () => {
            const { port } = server.address();
            resolve({ server, port, seen });
        });
    });
}

function startSilentMock() {
    // Never replies "pong" — used for the negative ping test.
    return new Promise((resolve, reject) => {
        const server = http.createServer((req, res) => {
            res.writeHead(404);
            res.end("nope");
        });
        server.on("error", reject);
        server.listen(0, "127.0.0.1", () => {
            const { port } = server.address();
            resolve({ server, port });
        });
    });
}

let passed = 0;
let failed = 0;

function assert(cond, msg) {
    if (cond) {
        passed++;
        console.log("  PASS  " + msg);
    } else {
        failed++;
        console.error("  FAIL  " + msg);
    }
}

async function main() {
    const args = parseArgs(process.argv);
    console.log("BE =", args.be);

    const mock = await startMock();
    const mockAddr = "127.0.0.1:" + mock.port;
    console.log("mock listening on", mockAddr);

    try {
        // 1. Register `chat`
        console.log("\n[1] register name=chat");
        let r = await fetch(args.be + "/api/v1/proxy/register", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ name: "chat", addr: mockAddr }),
        });
        let j = await r.json();
        assert(r.status === 200, "register returns 200 (got " + r.status + ")");
        assert(j.status === "success", "register status=success");
        assert(j.prefix === "/api/v1/chat", "default prefix=/api/v1/chat");
        assert(j.upstream === "/api/dev/chat", "default upstream=/api/dev/chat");

        // 2. List
        console.log("\n[2] list");
        r = await fetch(args.be + "/api/v1/proxy/list");
        j = await r.json();
        assert(
            Array.isArray(j.data) && j.data.some((e) => e.name === "chat"),
            "list contains chat entry",
        );

        // 3. Proxy: GET /api/v1/chat -> /api/dev/chat
        console.log("\n[3] GET /api/v1/chat");
        r = await fetch(args.be + "/api/v1/chat");
        j = await r.json();
        assert(r.status === 200, "GET /chat returns 200 (got " + r.status + ")");
        assert(j.url === "/api/dev/chat", "rewritten path == /api/dev/chat");
        assert(j.method === "GET", "method preserved (GET)");

        // 4. Proxy: POST /api/v1/chat/foo with body
        console.log("\n[4] POST /api/v1/chat/foo");
        r = await fetch(args.be + "/api/v1/chat/foo", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ hello: "world" }),
        });
        j = await r.json();
        assert(r.status === 200, "POST /chat/foo returns 200");
        assert(j.url === "/api/dev/chat/foo", "rewritten path == /api/dev/chat/foo");
        assert(j.method === "POST", "method preserved (POST)");
        assert(
            j.body && JSON.parse(j.body).hello === "world",
            "body forwarded intact",
        );

        // 5. Proxy: nested path + query
        console.log("\n[5] GET /api/v1/chat/foo/bar?x=1");
        r = await fetch(args.be + "/api/v1/chat/foo/bar?x=1");
        j = await r.json();
        assert(r.status === 200, "nested GET returns 200");
        assert(
            j.url === "/api/dev/chat/foo/bar?x=1",
            "nested path + query preserved",
        );

        // 6. Collision: register a proxy whose prefix shadows a real static
        //    route (/api/v1/component is a real GET handler). Static must win.
        console.log("\n[6] collision: register name=component, expect static wins");
        r = await fetch(args.be + "/api/v1/proxy/register", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ name: "component", addr: mockAddr }),
        });
        assert(r.status === 200, "register component proxy returns 200");
        const seenBefore = mock.seen.length;
        r = await fetch(args.be + "/api/v1/component/");
        // GET /api/v1/component/ is a real static handler -> NoRoute never
        // fires -> mock should NOT see this request.
        assert(
            mock.seen.length === seenBefore,
            "static /component route shadowed dynamic proxy (mock unchanged)",
        );
        // Cleanup
        await fetch(args.be + "/api/v1/proxy/detach", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ name: "component" }),
        });

        // 7. Detach `chat`
        console.log("\n[7] detach chat");
        r = await fetch(args.be + "/api/v1/proxy/detach", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ name: "chat" }),
        });
        j = await r.json();
        assert(r.status === 200, "detach returns 200");
        assert(j.status === "success", "detach status=success");

        // 8. Proxy after detach -> 404 (NoRoute default)
        console.log("\n[8] GET /api/v1/chat after detach");
        r = await fetch(args.be + "/api/v1/chat");
        assert(
            r.status === 404,
            "post-detach proxy returns 404 (got " + r.status + ")",
        );

        // 9. Negative: register against bad ping
        console.log("\n[9] register against bad ping");
        const bad = await startSilentMock();
        try {
            r = await fetch(args.be + "/api/v1/proxy/register", {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({
                    name: "broken",
                    addr: "127.0.0.1:" + bad.port,
                }),
            });
            assert(
                r.status === 502,
                "bad-ping register returns 502 (got " + r.status + ")",
            );
        } finally {
            bad.server.close();
        }

        // 10. Mock recorded forwarded requests (3 from steps 3/4/5)
        console.log("\n[10] mock recorded forwarded requests");
        assert(
            mock.seen.length === 3,
            "mock saw 3 forwarded requests (got " + mock.seen.length + ")",
        );
    } finally {
        mock.server.close();
    }

    console.log("\n----");
    console.log("passed:", passed, " failed:", failed);
    process.exit(failed === 0 ? 0 : 1);
}

main().catch((err) => {
    console.error(err);
    process.exit(2);
});
