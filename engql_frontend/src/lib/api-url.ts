export const getApiBaseUrl = (path: string) => {
  // path is expected to be like '/api/query' or '/api/ingestion'
  
  if (typeof window !== 'undefined') {
    // BROWSER: Return path as-is (e.g., /api/query)
    // Traefik will strip the /api and forward to backend
    return path;
  }

  // SSR (SERVER): Talk to container directly
  // 1. Start with the internal service address
  const internalBase = 'http://api:8080';
  
  // 2. Strip the '/api' prefix because internal calls bypass Traefik's stripper
  const internalPath = path.startsWith('/api') 
    ? path.replace('/api', '') 
    : path;

  return `${internalBase}${internalPath}`;
};