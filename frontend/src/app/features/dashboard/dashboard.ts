import { Component, computed, inject, signal } from '@angular/core';
import { RouterLink } from '@angular/router';
import { MatButtonModule } from '@angular/material/button';
import { Api } from '../../core/api';
import { Note, Page, dateInZone, minutesLabel, weekRange } from '../../core/models';
@Component({
  imports: [RouterLink, MatButtonModule],
  template: `<section class="page-heading">
      <div>
        <p class="eyebrow">{{ dateLabel }}</p>
        <h1>A little more clarity.</h1>
        <p class="muted">Here’s how your day is coming together.</p>
      </div>
      <a mat-stroked-button routerLink="/reports" [queryParams]="{ from: today, to: today }"
        >↗ Export today</a
      >
    </section>
    <div class="stats">
      <article class="stat">
        <span class="stat-icon sage">▤</span>
        <div>
          <p>Notes today</p>
          <strong>{{ todayNotes().length }}</strong
          ><small>A record of your progress</small>
        </div>
      </article>
      <article class="stat">
        <span class="stat-icon peach">✓</span>
        <div>
          <p>Tasks completed</p>
          <strong
            >{{ completedTasks() }}<em> / {{ totalTasks() }}</em></strong
          ><small>One step at a time</small>
        </div>
      </article>
      <article class="stat">
        <span class="stat-icon lavender">◷</span>
        <div>
          <p>Time captured</p>
          <strong>{{ time(todayMinutes()) }}</strong
          ><small>Meaningful work adds up</small>
        </div>
      </article>
    </div>
    <div class="dashboard-grid">
      <section class="panel">
        <div class="panel-heading">
          <h2>
            Today’s notes <span class="count">{{ todayNotes().length }}</span>
          </h2>
          <a routerLink="/notes" [queryParams]="{ from: today, to: today }">View all →</a>
        </div>
        @if (loading()) {
          <p class="empty">Loading your workspace…</p>
        } @else if (error()) {
          <p class="error" role="alert">{{ error() }}</p>
          <button mat-button (click)="load()">Try again</button>
        } @else {
          @for (n of todayNotes(); track n.id) {
            <a class="note-row" [routerLink]="['/notes', n.id]"
              ><span class="note-glyph">▤</span>
              <div>
                <span class="project-label">{{ n.project || 'PERSONAL WORK' }}</span>
                <h3>{{ n.title }}</h3>
                <span class="muted small">{{ n.tasks.length }} tasks · {{ time(n.minutes) }}</span>
              </div>
              <span class="badge" [attr.data-status]="n.status">{{ labels[n.status] }}</span
              ><span class="row-arrow">↗</span></a
            >
          } @empty {
            <div class="empty">
              <span class="empty-symbol">✎</span>
              <h3>A fresh page for today.</h3>
              <p>Your first note is a good place to start.</p>
              <a mat-flat-button routerLink="/notes/new">＋ Write a note</a>
            </div>
          }
        }
        <a class="add-row" routerLink="/notes/new">＋ Capture something new</a>
      </section>
      <section class="week-card">
        <p class="eyebrow">THE BIGGER PICTURE</p>
        <h2>Your week,<br />at a glance.</h2>
        <div class="week-bars">
          @for (d of days(); track d.date) {
            <div>
              <span class="bar-track"
                ><span [style.height.%]="d.height" [class.current]="d.date === today"></span></span
              ><small>{{ d.label }}</small>
            </div>
          }
        </div>
        <div class="week-total">
          <strong>{{ time(weekMinutes()) }}</strong
          ><span>captured this week</span>
        </div>
        <a routerLink="/email" [queryParams]="range">Share your weekly summary →</a>
      </section>
    </div>
    <div class="quiet-banner">
      <span>✳</span>
      <div>
        <h3>Close the day with a clear mind.</h3>
        <p>A quick note now makes tomorrow’s first step a little easier.</p>
      </div>
      <a routerLink="/notes/new">Make a note ↗</a>
    </div>`,
})
export class Dashboard {
  api = inject(Api);
  today = dateInZone(this.api.user()!.timezone);
  range = weekRange(this.today);
  dateLabel = new Intl.DateTimeFormat('en', {
    timeZone: this.api.user()!.timezone,
    weekday: 'long',
    month: 'long',
    day: 'numeric',
  }).format(new Date());
  notes = signal<Note[]>([]);
  loading = signal(true);
  error = signal('');
  time = minutesLabel;
  labels = {
    todo: 'To do',
    progress: 'In progress',
    completed: 'Completed',
    blocked: 'Blocked',
  };
  todayNotes = computed(() => this.notes().filter((n) => n.workDate === this.today));
  todayMinutes = computed(() => this.todayNotes().reduce((s, n) => s + n.minutes, 0));
  weekMinutes = computed(() => this.notes().reduce((s, n) => s + n.minutes, 0));
  completedTasks = computed(
    () =>
      this.todayNotes()
        .flatMap((n) => n.tasks)
        .filter((t) => t.done).length,
  );
  totalTasks = computed(() => this.todayNotes().flatMap((n) => n.tasks).length);
  days = computed(() =>
    Array.from({ length: 7 }, (_, i) => {
      const d = new Date(this.range.from + 'T12:00:00Z');
      d.setUTCDate(d.getUTCDate() + i);
      const date = d.toISOString().slice(0, 10),
        value = this.notes()
          .filter((n) => n.workDate === date)
          .reduce((s, n) => s + n.minutes, 0);
      return {
        date,
        label: ['M', 'T', 'W', 'T', 'F', 'S', 'S'][i],
        height: Math.max(4, Math.min(100, (value / 480) * 100)),
      };
    }),
  );
  constructor() {
    this.load();
  }
  load() {
    this.loading.set(true);
    this.error.set('');
    const all: Note[] = [];
    const next = (page: number) =>
      this.api.get<Page>('/notes', { ...this.range, size: 100, page }).subscribe({
        next: (p) => {
          all.push(...p.items);
          if (p.hasMore) next(page + 1);
          else {
            this.notes.set(all);
            this.loading.set(false);
          }
        },
        error: (e) => {
          this.error.set(this.api.error(e));
          this.loading.set(false);
        },
      });
    next(1);
  }
}
