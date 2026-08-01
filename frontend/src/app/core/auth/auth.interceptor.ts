import { inject } from '@angular/core';
import { HttpInterceptorFn } from '@angular/common/http';
import { from, switchMap } from 'rxjs';
import { APP_CONFIG } from '../config/app-config';
import { AuthService } from './auth.service';

export const authInterceptor: HttpInterceptorFn = (request, next) => {
  const config = inject(APP_CONFIG);
  if (!request.url.startsWith(config.apiBaseUrl)) {
    return next(request);
  }

  const auth = inject(AuthService);
  return from(auth.accessToken()).pipe(
    switchMap((token) => {
      const headers: Record<string, string> = {
        'X-Correlation-ID': crypto.randomUUID(),
      };
      if (token) {
        headers['Authorization'] = `Bearer ${token}`;
      }
      return next(request.clone({ setHeaders: headers }));
    }),
  );
};
