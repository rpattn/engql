export const getApiBaseUrl = () => {
  // 1. Browser context: use relative path so Traefik handles it
  if (typeof window !== 'undefined') {
    return import.meta.env.VITE_API_URL || '/api';
  }

  // 2. SSR context: Use the internal Docker service name
  // Note: We use 8080 because that's the internal port of your 'api' service
  return 'http://api:8080';
};