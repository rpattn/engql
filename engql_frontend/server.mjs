import { createServer } from 'node:http';
import { join, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';
import serveStatic from 'serve-static';
import { createApp, eventHandler, fromNodeMiddleware } from 'h3';
import { toNodeListener } from 'h3/node';

// Import the build
import * as serverModule from './dist/server/server.js';

// Configuration
const INTERNAL_SECRET = process.env.INTERNAL_SECRET || 'change-me-in-prod';
const API_URL = process.env.INTERNAL_API_URL || 'http://api:8080';

// 1. GLOBAL OUTBOUND LOGGER & AUTH INJECTOR
// This handles fetches made INTERNALLY by the server during SSR
const originalFetch = globalThis.fetch;
globalThis.fetch = async (url, config = {}) => {
  const start = Date.now();
  const method = config?.method || 'GET';
  
  // Create a Headers object from existing config headers
  const headers = new Headers(config.headers || {});

  // AUTOMATICALLY inject the secret if we are hitting our internal API
  if (url.toString().includes(API_URL) || url.toString().includes('api:8080')) {
    headers.set('X-Internal-Secret', INTERNAL_SECRET);
  }

  const newConfig = { ...config, headers };

  console.log(`    ➡️  [OUTBOUND] ${method} ${url}`);
  try {
    const response = await originalFetch(url, newConfig);
    const duration = Date.now() - start;
    console.log(`    ⬅️  [OUTBOUND] ${response.status} ${url} (${duration}ms)`);
    return response;
  } catch (err) {
    console.error(`    ❌ [OUTBOUND ERROR] ${method} ${url}:`, err.message);
    throw err;
  }
};

const handler = serverModule.default?.fetch;
const __dirname = dirname(fileURLToPath(import.meta.url));
const app = createApp();

// Serve Static Assets
app.use(fromNodeMiddleware(serveStatic(join(__dirname, 'dist', 'client'))));

// 2. UNIFIED INBOUND HANDLER (Proxy + SSR)
app.use(eventHandler(async (event) => {
  const { req } = event.node;
  const protocol = req.headers['x-forwarded-proto'] || 'http';
  const host = req.headers.host;
  const url = new URL(req.url, `${protocol}://${host}`);

  // --- A. PROXY LOGIC (Browser -> Node -> Go Backend) ---
  if (url.pathname.startsWith('/api')) {
    const targetPath = url.pathname.replace('/api', ''); 
    const targetUrl = `${API_URL}${targetPath}${url.search}`;
    
    console.log(`[PROXY] Forwarding to Backend: ${url.pathname} -> ${targetUrl}`);

    // Clone incoming headers and add the Secret Key
    const proxyHeaders = new Headers(req.headers);
    proxyHeaders.set('X-Internal-Secret', INTERNAL_SECRET);
    
    // Safety: The 'host' header from the browser might confuse the Go backend
    // especially if it's expecting 'api:8080' or localhost
    proxyHeaders.delete('host'); 

    try {
      // Note: We use the patched globalThis.fetch here
      return await fetch(targetUrl, {
        method: req.method,
        headers: proxyHeaders,
        // Duplex 'half' is required for streaming request bodies in Node fetch
        body: ['GET', 'HEAD', 'OPTIONS'].includes(req.method) ? undefined : req,
        duplex: 'half',
      });
    } catch (err) {
      console.error(`[PROXY ERROR]:`, err.message);
      return new Response('API Bridge Error', { status: 502 });
    }
  }

  // --- B. SSR LOGIC (For actual page rendering) ---
  if (typeof handler !== 'function') {
    return new Response('Server Handler Not Found', { status: 500 });
  }

  console.log(`[SSR INBOUND] ${req.method} ${url.pathname}`);
  const start = Date.now();

  try {
    const webRequest = new Request(url, {
      method: req.method,
      headers: req.headers,
      body: ['GET', 'HEAD'].includes(req.method) ? undefined : req,
      duplex: 'half',
    });

    const response = await handler(webRequest);
    console.log(`✅ [SSR RESPONSE] ${response.status} (${Date.now() - start}ms)`);
    return response;
  } catch (error) {
    console.error('🔥 [SSR CRITICAL ERROR]:', error);
    return new Response('Internal Server Error', { status: 500 });
  }
}));

const port = process.env.PORT || 3010;
createServer(toNodeListener(app)).listen(port, () => {
  console.log(`\n🚀 SSR Server ready at http://localhost:${port}\n`);
});