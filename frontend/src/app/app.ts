import { Component, inject, signal } from '@angular/core';
import { RouterLink, RouterLinkActive, RouterOutlet } from '@angular/router';
import { MatButtonModule } from '@angular/material/button';
import { Api } from './core/api';
@Component({selector:'app-root',imports:[RouterOutlet,RouterLink,RouterLinkActive,MatButtonModule],template:`
@if(api.user()) {
 <aside [class.open]="menu()"><a class="brand" routerLink="/" (click)="menu.set(false)"><span class="brand-mark">d<span>·</span></span><span>daily<span class="brand-sub">WORK NOTES</span></span></a><div class="workspace-label">YOUR WORKSPACE</div>
 <nav aria-label="Main navigation">@for(link of links;track link.path){<a [routerLink]="link.path" routerLinkActive="active" [routerLinkActiveOptions]="{exact:true}" (click)="menu.set(false)"><span class="nav-symbol">{{link.icon}}</span>{{link.name}}</a>}</nav>
 <div class="sidebar-bottom"><div class="small-note"><span class="leaf">✳</span><strong>A little progress, every day.</strong><p>Make room for the work that matters.</p></div><button class="account" (click)="api.logout()"><span class="avatar">{{api.user()!.email.slice(0,1).toUpperCase()}}</span><span>{{api.user()!.email}}<small>Sign out ↗</small></span></button></div></aside>
 <div class="shell"><header><button class="menu-button" mat-button (click)="menu.set(!menu())" aria-label="Toggle navigation">☰</button><div class="breadcrumb">My workspace <span>/</span> Daily Work Notes</div><div class="header-actions"><span class="private-label">◉ Private workspace</span><button mat-button (click)="toggleTheme()" aria-label="Toggle color theme">{{dark()?'☀':'☾'}}</button><a mat-flat-button routerLink="/notes/new">＋ Add note</a></div></header><main><router-outlet/></main><footer>Thoughtfully captured. Ready for tomorrow.<span>DAILY WORK NOTES</span></footer></div>
} @else {<router-outlet/>}
@if(api.message()){<div class="toast" role="status">{{api.message()}}<button aria-label="Dismiss notification" (click)="api.message.set('')">×</button></div>}
`})
export class App {api=inject(Api);menu=signal(false);dark=signal(localStorage.getItem('theme')==='dark');links=[{path:'/',name:'Overview',icon:'◫'},{path:'/notes',name:'My notes',icon:'▤'},{path:'/projects',name:'Projects',icon:'▦'},{path:'/email',name:'Email summaries',icon:'✉'},{path:'/reports',name:'Reports & backups',icon:'↗'},{path:'/settings',name:'Settings',icon:'⚙'}];constructor(){document.documentElement.classList.toggle('dark',this.dark())}toggleTheme(){this.dark.update(v=>!v);localStorage.setItem('theme',this.dark()?'dark':'light');document.documentElement.classList.toggle('dark',this.dark())}}
