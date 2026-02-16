import { HttpInterceptorFn } from '@angular/common/http';

export const apiBaseInterceptor: HttpInterceptorFn = (req, next) => {
  // Skip if URL is absolute (external APIs, blob downloads, etc.)
  if (req.url.startsWith('http://') || req.url.startsWith('https://') || req.url.startsWith('blob:')) {
    return next(req);
  }

  // Skip if it's a WebSocket upgrade request
  if (req.headers.has('Upgrade') && req.headers.get('Upgrade') === 'websocket') {
    return next(req);
  }

  // Request is already prefixed with /api/v1 in environment config
  return next(req);
};
