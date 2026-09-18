import { bootstrapApplication } from '@angular/platform-browser';
import { provideZonelessChangeDetection } from '@angular/core';
import { provideHttpClient, withInterceptors, withXhr } from '@angular/common/http';
import { provideRouter } from '@angular/router';
import { App } from './app/app';
import { routes } from './app/routes';
import { authInterceptor } from './app/core/api';
bootstrapApplication(App, {providers:[provideZonelessChangeDetection(),provideHttpClient(withXhr(),withInterceptors([authInterceptor])),provideRouter(routes)]}).catch(console.error);
