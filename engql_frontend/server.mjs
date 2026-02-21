import { createServer } from 'node:http';
import { join, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';
import serveStatic from 'serve-static';
import { createApp, eventHandler, fromNodeMiddleware } from 'h3';
import { toNodeListener } from 'h3/node';

// Import the build
import * as serverModule from './dist/server/server.js';

/** * 1. GLOBAL OUTBOUND LOGGER
 * This intercepts every 'fetch' call your SSR server makes.
 * This will show us EXACTLY what URL getApiBaseUrl is producing.
 */
const originalFetch = globalThis.fetch;
globalThis.fetch = async (url, config) => {
  const start = Date.now();
  const method = config?.method || 'GET';
  
  console.log(`   ➡️  [OUTBOUND] ${method} ${url}`);
  
  try {
    const response = await originalFetch(url, config);
    const duration = Date.now() - start;
    console.log(`   ⬅️  [OUTBOUND] ${response.status} ${url} (${duration}ms)`);
    return response;
  } catch (err) {
    console.error(`   ❌ [OUTBOUND ERROR] ${method} ${url}:`, err.message);
    throw err;
  }
};

const handler = serverModule.default?.fetch;
const __dirname = dirname(fileURLToPath(import.meta.url));
const app = createApp();

// Serve Static Assets
app.use(fromNodeMiddleware(serveStatic(join(__dirname, 'dist', 'client'))));

// 2. ENHANCED INBOUND SSR HANDLER
app.use(eventHandler(async (event) => {
  const { req, res } = event.node;
  const protocol = req.headers['x-forwarded-proto'] || 'http';
  const url = new URL(req.url, `${protocol}://${req.headers.host}`);

  // --- NEW: PROXY LOGIC ---
  // If the request is for /api, proxy it to the Go backend container
  if (url.pathname.startsWith('/api')) {
    const targetPath = url.pathname.replace('/api', ''); // remove /api
    const targetUrl = `http://api:8080${targetPath}${url.search}`;
    
    console.log(`[PROXY] Forwarding ${url.pathname} -> ${targetUrl}`);

    try {
      const proxyResponse = await fetch(targetUrl, {
        method: req.method,
        headers: req.headers,
        body: ['GET', 'HEAD'].includes(req.method) ? undefined : req,
        duplex: 'half',
      });

      // Forward the status and headers back to the browser
      return proxyResponse;
    } catch (err) {
      console.error(`[PROXY ERROR]:`, err.message);
      return new Response('API Bridge Error', { status: 502 });
    }
  }
  // --- END PROXY LOGIC ---

  // Handle SSR Requests (Your existing code)
  console.log(`[SSR INBOUND] ${req.method} ${url.pathname}${url.search}`);
  // ... rest of your handler code
}));

const port = process.env.PORT || 3010;
createServer(toNodeListener(app)).listen(port, () => {
  console.log(`\n🚀 SSR Server ready at http://localhost:${port}\n`);
});