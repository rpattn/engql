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
  if (typeof handler !== 'function') {
    console.error('Handler Error: Expected a function but found:', typeof handler);
    return new Response('Server Configuration Error', { status: 500 });
  }

  const start = Date.now();
  const { req } = event.node;

  try {
    const protocol = req.headers['x-forwarded-proto'] || 'http';
    const host = req.headers.host;
    const url = new URL(req.url, `${protocol}://${host}`);
    
    // Log Inbound
    console.log(`[SSR INBOUND] ${req.method} ${url.pathname}${url.search}`);

    const webRequest = new Request(url, {
      method: req.method,
      headers: req.headers,
      body: ['GET', 'HEAD'].includes(req.method) ? undefined : req,
      duplex: 'half', 
    });

    const response = await handler(webRequest);
    const duration = Date.now() - start;

    // Distinguish between successful SSR and errors
    const statusIcon = response.status >= 400 ? '⚠️' : '✅';
    console.log(`${statusIcon} [SSR RESPONSE] ${response.status} (${duration}ms)`);

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