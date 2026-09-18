import { Injectable, inject, signal } from '@angular/core';
import { HttpClient, HttpInterceptorFn, HttpErrorResponse } from '@angular/common/http';
import { Router } from '@angular/router';
import { catchError, of, tap, throwError } from 'rxjs';
import { User } from './models';
@Injectable({ providedIn: 'root' })
export class Api {
  readonly http = inject(HttpClient);
  readonly router = inject(Router);
  readonly user = signal<User | null>(null);
  readonly message = signal('');
  get<T>(path: string, params: Record<string, string | number> = {}) {
    return this.http.get<T>('/api' + path, { params });
  }
  post<T>(path: string, body: unknown) {
    return this.http.post<T>('/api' + path, body);
  }
  put<T>(path: string, body: unknown) {
    return this.http.put<T>('/api' + path, body);
  }
  delete(path: string) {
    return this.http.delete('/api' + path);
  }
  load() {
    return this.get<User>('/me').pipe(
      tap((u) => this.user.set(u)),
      catchError(() => of(null)),
    );
  }
  notify(text: string) {
    this.message.set(text);
    setTimeout(() => {
      if (this.message() === text) this.message.set('');
    }, 6000);
  }
  error(e: unknown) {
    return e instanceof HttpErrorResponse
      ? e.error?.error?.message || `Request failed (${e.status || 'offline'})`
      : 'Something went wrong';
  }
  logout() {
    this.post('/auth/logout', {}).subscribe({
      next: () => {
        this.user.set(null);
        void this.router.navigateByUrl('/auth');
      },
      error: (e) => this.notify(this.error(e)),
    });
  }
}
export const authInterceptor: HttpInterceptorFn = (req, next) => {
  const api = inject(Api);
  const csrf = api.user()?.csrf;
  if (csrf && !['GET', 'HEAD'].includes(req.method))
    req = req.clone({ setHeaders: { 'X-CSRF-Token': csrf } });
  return next(req).pipe(
    catchError((e) => {
      if (e.status === 401 && !req.url.includes('/auth/') && !req.url.endsWith('/me')) {
        api.user.set(null);
        void api.router.navigateByUrl('/auth');
      }
      return throwError(() => e);
    }),
  );
};
