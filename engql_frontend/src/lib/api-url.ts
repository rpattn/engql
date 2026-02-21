// src/lib/api-url.ts
export const getApiBaseUrl = (path: string) => {
  const isServer = typeof window === 'undefined';

  if (isServer) {
    /**
     * SSR: Talk to the Go container directly over the Docker network.
     * We MUST strip the /api prefix because the Go server 
     * listens on /query, not /api/query.
     */
    const internalBase = 'http://api:8080';
    const cleanPath = path.startsWith('/api') ? path.replace('/api', '') : path;
    return `${internalBase}${cleanPath}`;
  }

  // BROWSER: Use the relative path. 
  // Our new Proxy in entry.js will catch this if Traefik fails.
  return path;
};