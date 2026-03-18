import fs from "node:fs/promises";
import http from "node:http";
import path from "node:path";
import { fileURLToPath } from "node:url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const isProduction = process.env.NODE_ENV === "production";
const port = Number(process.env.PORT || 3000);
const apiProxyTarget = process.env.API_PROXY_TARGET || "http://www.xbetaa.com";

const mimeTypes = new Map([
  [".js", "text/javascript; charset=utf-8"],
  [".css", "text/css; charset=utf-8"],
  [".html", "text/html; charset=utf-8"],
  [".svg", "image/svg+xml"],
  [".json", "application/json; charset=utf-8"],
  [".ico", "image/x-icon"],
  [".png", "image/png"],
  [".jpg", "image/jpeg"],
  [".jpeg", "image/jpeg"],
  [".webp", "image/webp"],
]);

let vite;
let template;
let render;

if (isProduction) {
  template = await fs.readFile(path.resolve(__dirname, "dist/client/index.html"), "utf-8");
  ({ render } = await import("./dist/server/entry-server.js"));
} else {
  const { createServer } = await import("vite");
  vite = await createServer({
    root: __dirname,
    server: { middlewareMode: true },
    appType: "custom",
  });
}

function send(res, statusCode, body, headers = {}) {
  res.writeHead(statusCode, headers);
  res.end(body);
}

async function readRequestBody(req) {
  const chunks = [];
  for await (const chunk of req) {
    chunks.push(chunk);
  }
  return chunks.length ? Buffer.concat(chunks) : undefined;
}

async function proxyAPI(req, res) {
  const url = new URL(req.url, apiProxyTarget);
  const headers = new Headers();

  Object.entries(req.headers).forEach(([key, value]) => {
    if (key === "host" || value === undefined) {
      return;
    }
    if (Array.isArray(value)) {
      value.forEach((item) => headers.append(key, item));
      return;
    }
    headers.set(key, value);
  });

  const body = req.method === "GET" || req.method === "HEAD" ? undefined : await readRequestBody(req);
  const response = await fetch(url, {
    method: req.method,
    headers,
    body,
    redirect: "manual",
  });

  const responseBody = Buffer.from(await response.arrayBuffer());
  const responseHeaders = {};
  response.headers.forEach((value, key) => {
    responseHeaders[key] = value;
  });
  send(res, response.status, responseBody, responseHeaders);
}

async function serveStaticAsset(req, res) {
  const assetPath = path.resolve(__dirname, "dist/client", `.${new URL(req.url, "http://local").pathname}`);
  try {
    const stats = await fs.stat(assetPath);
    if (!stats.isFile()) {
      return false;
    }
    const body = await fs.readFile(assetPath);
    send(res, 200, body, {
      "Content-Type": mimeTypes.get(path.extname(assetPath)) || "application/octet-stream",
      "Cache-Control": assetPath.includes("/assets/") ? "public, max-age=31536000, immutable" : "no-cache",
    });
    return true;
  } catch {
    return false;
  }
}

const server = http.createServer(async (req, res) => {
  try {
    if (!req.url) {
      send(res, 400, "Bad Request");
      return;
    }

    if (req.url.startsWith("/api/")) {
      await proxyAPI(req, res);
      return;
    }

    if (isProduction && (req.url.startsWith("/assets/") || req.url === "/favicon.svg")) {
      const served = await serveStaticAsset(req, res);
      if (served) {
        return;
      }
    }

    const requestOrigin = `http://${req.headers.host || `localhost:${port}`}`;
    const apiBaseUrl = process.env.API_BASE_URL || requestOrigin;
    const url = req.url;

    let htmlTemplate;
    let renderModule;
    if (isProduction) {
      htmlTemplate = template;
      renderModule = { render };
    } else {
      htmlTemplate = await fs.readFile(path.resolve(__dirname, "index.html"), "utf-8");
      htmlTemplate = await vite.transformIndexHtml(url, htmlTemplate);
      renderModule = await vite.ssrLoadModule("/src/entry-server.tsx");
    }

    const { appHtml, dehydratedState, metadata } = await renderModule.render({ url, apiBaseUrl });
    const stateScript = `<script>window.__INITIAL_STATE__=${JSON.stringify(dehydratedState).replace(/</g, "\\u003c")}</script>`;
    const head = [
      `<title>${escapeHtml(metadata.title)}</title>`,
      `<meta name="description" content="${escapeHtml(metadata.description)}" />`,
      `<link rel="canonical" href="${escapeHtml(`${requestOrigin}${metadata.canonicalPath}`)}" />`,
      `<meta property="og:type" content="website" />`,
      `<meta property="og:title" content="${escapeHtml(metadata.title)}" />`,
      `<meta property="og:description" content="${escapeHtml(metadata.description)}" />`,
      `<meta property="og:url" content="${escapeHtml(`${requestOrigin}${metadata.canonicalPath}`)}" />`,
      `<meta name="twitter:card" content="summary_large_image" />`,
      `<meta name="twitter:title" content="${escapeHtml(metadata.title)}" />`,
      `<meta name="twitter:description" content="${escapeHtml(metadata.description)}" />`,
    ].join("");
    const html = htmlTemplate
      .replace("<!--app-head-->", head)
      .replace("<!--app-html-->", appHtml)
      .replace("<!--app-state-->", stateScript);

    send(res, 200, html, { "Content-Type": "text/html; charset=utf-8" });
  } catch (error) {
    if (!isProduction && vite) {
      vite.ssrFixStacktrace(error);
    }
    console.error(error);
    send(res, 500, "Internal Server Error");
  }
});

server.listen(port, () => {
  console.log(`datasrv front SSR listening on http://localhost:${port}`);
});

function escapeHtml(value) {
  return String(value)
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#39;");
}
