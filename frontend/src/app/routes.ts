import { inject } from '@angular/core';
import { CanActivateFn, Router, Routes } from '@angular/router';
import { map } from 'rxjs';
import { Api } from './core/api';
const guard:CanActivateFn=()=>{const api=inject(Api),router=inject(Router);return api.user()?true:api.load().pipe(map(u=>u?true:router.parseUrl('/auth')))};
export const routes:Routes=[
 {path:'auth',loadComponent:()=>import('./features/auth/auth').then(m=>m.Auth)},
 {path:'',canActivate:[guard],loadComponent:()=>import('./features/dashboard/dashboard').then(m=>m.Dashboard)},
 {path:'notes',canActivate:[guard],loadComponent:()=>import('./features/notes/notes').then(m=>m.Notes)},
 {path:'notes/:id',canActivate:[guard],loadComponent:()=>import('./features/notes/editor').then(m=>m.Editor)},
 {path:'projects',canActivate:[guard],loadComponent:()=>import('./features/projects/projects').then(m=>m.Projects)},
 {path:'email',canActivate:[guard],loadComponent:()=>import('./features/email/email').then(m=>m.Email)},
 {path:'reports',canActivate:[guard],loadComponent:()=>import('./features/reports/reports').then(m=>m.Reports)},
 {path:'settings',canActivate:[guard],loadComponent:()=>import('./features/settings/settings').then(m=>m.Settings)},
 {path:'**',redirectTo:''}
];
