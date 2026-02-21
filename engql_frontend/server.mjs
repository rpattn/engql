import { createServer } from 'node:http';
import { join, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';
import serveStatic from 'serve-static';
import { createApp, eventHandler, fromNodeMiddleware } from 'h3';
import { toNodeListener } from 'h3/node';

// Import the build
import * as serverModule from './dist/server/server.js';

/** * TARGET THE FETCH PROPERTY
 * Based on your cat output, the export is { default: { fetch: Function } }
 */
const handler = serverModule.default?.fetch;

const __dirname = dirname(fileURLToPath(import.meta.url));
const app = createApp();

// Serve Static Assets
app.use(fromNodeMiddleware(serveStatic(join(__dirname, 'dist', 'client'))));

// Handle SSR Requests
app.use(eventHandler(async (event) => {
  if (typeof handler !== 'function') {
    console.error('Handler Error: Expected a function but found:', typeof handler);
    return new Response('Server Configuration Error', { status: 500 });
  }

  const start = Date.now(); // Start timer
  const { req } = event.node;

  try {
    const protocol = req.headers['x-forwarded-proto'] || 'http';
    const url = new URL(req.url, `${protocol}://${req.headers.host}`);
    
    // Log the incoming request to the SSR server
    console.log(`[SSR] ${req.method} ${url.pathname}${url.search}`);

    const webRequest = new Request(url, {
      method: req.method,
      headers: req.headers,
      body: ['GET', 'HEAD'].includes(req.method) ? undefined : req,
      duplex: 'half', 
    });

    // Call the TanStack Start handler
    const response = await handler(webRequest);

    // Log the result of the SSR execution
    const duration = Date.now() - start;
    console.log(`[SSR] Response: ${response.status} (${duration}ms)`);

    return response;
  } catch (error) {
    console.error('SSR Critical Error:', error);
    return new Response('Internal Server Error', { status: 500 });
  }
}));

const port = process.env.PORT || 3010;
createServer(toNodeListener(app)).listen(port, () => {
  console.log(`🚀 Server started at http://localhost:${port}`);
});
