export const getApiUrl = (path: string) => {
  const isBrowser = typeof window !== 'undefined';
  
  // Ensure path starts with a slash
  const cleanPath = path.startsWith('/') ? path : `/${path}`;

  if (isBrowser) {
    // Browser uses the relative path (e.g., /api/query)
    // Traefik will handle the stripping.
    return cleanPath;
  }

  // SSR: Talk directly to Docker container 'api' on port 8080
  // IMPORTANT: We must remove '/api' from the start of the path 
  // because internal calls bypass the Traefik Stripper.
  const internalPath = cleanPath.replace(/^\/api/, '');
  
  return `http://api:8080${internalPath}`;
};