// src/lib/api-url.ts
export const getApiBaseUrl = (path: string) => {
  // Check if we are in a browser environment
  const isBrowser = typeof window !== 'undefined';

  if (isBrowser) {
    // Browser: Use relative path so Traefik handles the routing/stripping
    return path;
  }

  // SSR (SERVER): We MUST use the absolute internal address
  // Note: 'api' is the service name in your docker-compose
  const internalBase = 'http://api:8080';
  
  // Strip '/api' because the internal Go server listens on /query (not /api/query)
  const internalPath = path.startsWith('/api') 
    ? path.replace('/api', '') 
    : path;

  const finalUrl = `${internalBase}${internalPath}`;

  // This log will appear in your Coolify 'web' service logs
  console.log(`[SSR Fetch] Path: ${path} -> Target: ${finalUrl}`);

  return finalUrl;
};