import { Component, computed, inject, signal, DestroyRef } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { FormBuilder, ReactiveFormsModule } from '@angular/forms';
import { ActivatedRoute, RouterLink } from '@angular/router';
import { MatButtonModule } from '@angular/material/button';
import { MatCheckboxModule } from '@angular/material/checkbox';
import { debounceTime, startWith, switchMap, catchError, of, tap, merge, Subject } from 'rxjs';
import { Api } from '../../core/api';
import { Note, Page, minutesLabel, dateInZone } from '../../core/models';
import { Calendar } from './calendar';
@Component({
  imports: [ReactiveFormsModule, RouterLink, MatButtonModule, MatCheckboxModule, Calendar],
  template: `<section class="page-heading">
      <div>
        <p class="eyebrow">A RECORD OF WHAT MATTERS</p>
        <h1>My notes</h1>
        <p class="muted">Find the details. See the progress. Pick up where you left off.</p>
      </div>
      <div class="segmented">
        <button [class.selected]="view() === 'list'" (click)="view.set('list')">☷ List</button
        ><button [class.selected]="view() === 'calendar'" (click)="view.set('calendar')">
          ▦ Calendar
        </button>
      </div>
    </section>
    <form class="filter-bar" [formGroup]="filters">
      <label class="search-field"
        ><span>⌕</span
        ><input
          formControlName="q"
          aria-label="Search notes"
          placeholder="Search titles, content, projects, tags…" /></label
      ><label>From<input type="date" formControlName="from" /></label
      ><label>To<input type="date" formControlName="to" /></label
      ><label
        >Status<select formControlName="status">
          <option value="">All statuses</option>
          <option value="todo">To do</option>
          <option value="progress">In progress</option>
          <option value="completed">Completed</option>
          <option value="blocked">Blocked</option>
        </select></label
      ><label
        >Priority<select formControlName="priority">
          <option value="">All priorities</option>
          <option value="low">Low</option>
          <option value="medium">Medium</option>
          <option value="high">High</option>
        </select></label
      ><label>Project<input formControlName="project" placeholder="Any project" /></label
      ><label>Tag<input formControlName="tag" placeholder="Any tag" /></label
      ><label
        >Sort<select formControlName="sort">
          <option value="date">Work date</option>
          <option value="updated">Last updated</option>
          <option value="priority">Priority</option>
          <option value="title">Title</option>
        </select></label
      >
    </form>
    @if (selected().size) {
      <div class="selection-bar">
        <strong>{{ selected().size }} selected</strong
        ><a mat-button routerLink="/email" [queryParams]="{ ids: selectedIds() }">Email</a
        ><a mat-button routerLink="/reports" [queryParams]="{ ids: selectedIds() }">Export</a
        ><button mat-button (click)="selected.set(emptySet())">Clear</button>
      </div>
    }
    @if (loading()) {
      <p class="loading" role="status">Finding your notes…</p>
    }
    @if (error()) {
      <p class="error" role="alert">{{ error() }}</p>
    }
    @switch (view()) {
      @case ('list') {
        <section class="panel notes-table">
          <div class="table-header">
            <span>NOTE / PROJECT</span><span>WORK DATE</span><span>STATUS</span><span>TIME</span>
          </div>
          @for (n of notes(); track n.id) {
            <div class="table-row">
              <div class="table-title">
                <mat-checkbox
                  [checked]="selected().has(n.id)"
                  (change)="select(n.id)"
                  [attr.aria-label]="'Select ' + n.title"
                /><a [routerLink]="['/notes', n.id]"
                  ><strong>{{ n.title }}</strong
                  ><small
                    >{{ n.project || 'No project' }}
                    @for (tag of n.tags; track tag) {
                      <span class="tag">{{ tag }}</span>
                    }
                  </small></a
                >
              </div>
              <span class="muted">{{ n.workDate }}</span
              ><span class="badge" [attr.data-status]="n.status">{{ labels[n.status] }}</span
              ><span class="muted">{{ time(n.minutes) }}</span>
            </div>
          } @empty {
            @if (!loading()) {
              <div class="empty">
                <h3>No notes found.</h3>
                <p>Try a different filter or begin a new note.</p>
                <a mat-flat-button routerLink="/notes/new">＋ Add note</a>
              </div>
            }
          }
        </section>
      }
      @case ('calendar') {
        <app-calendar
          [notes]="notes()"
          [initialDate]="calendarDate"
          (rangeChanged)="filters.patchValue($event)"
        />
      }
    }
    <div class="pagination">
      <button mat-button [disabled]="page() === 1 || loading()" (click)="changePage(-1)">
        ← Previous</button
      ><span>Page {{ page() }}</span
      ><button mat-button [disabled]="!hasMore() || loading()" (click)="changePage(1)">
        Next →
      </button>
    </div>`,
})
export class Notes {
  api = inject(Api);
  fb = inject(FormBuilder);
  route = inject(ActivatedRoute);
  destroy = inject(DestroyRef);
  filters = this.fb.nonNullable.group({
    q: '',
    from: this.route.snapshot.queryParamMap.get('from') || '',
    to: this.route.snapshot.queryParamMap.get('to') || '',
    status: '',
    priority: '',
    project: this.route.snapshot.queryParamMap.get('project') || '',
    tag: '',
    sort: 'date',
  });
  notes = signal<Note[]>([]);
  page = signal(1);
  hasMore = signal(false);
  loading = signal(false);
  error = signal('');
  view = signal<'list' | 'calendar'>('list');
  selected = signal(new Set<string>());
  calendarDate =
    this.route.snapshot.queryParamMap.get('from') || dateInZone(this.api.user()!.timezone);
  private pageChanges = new Subject<void>();
  time = minutesLabel;
  labels = { todo: 'To do', progress: 'In progress', completed: 'Completed', blocked: 'Blocked' };
  selectedIds = computed(() => [...this.selected()].join(','));
  constructor() {
    merge(
      this.filters.valueChanges.pipe(
        debounceTime(300),
        tap(() => {
          this.page.set(1);
          this.calendarDate =
            this.filters.controls.from.value || dateInZone(this.api.user()!.timezone);
        }),
      ),
      this.pageChanges,
    )
      .pipe(
        startWith(this.filters.getRawValue()),
        switchMap(() => {
          this.loading.set(true);
          this.error.set('');
          return this.api
            .get<Page>('/notes', { ...this.filters.getRawValue(), page: this.page() })
            .pipe(
              catchError((e) => {
                this.error.set(this.api.error(e));
                return of({ items: [], hasMore: false, page: 1, size: 20 });
              }),
            );
        }),
        takeUntilDestroyed(),
      )
      .subscribe((p) => {
        this.notes.set(p.items);
        this.hasMore.set(p.hasMore);
        this.loading.set(false);
      });
  }
  changePage(delta: number) {
    this.page.update((p) => p + delta);
    this.pageChanges.next();
  }
  select(id: string) {
    this.selected.update((old) => {
      const s = new Set(old);
      s.has(id) ? s.delete(id) : s.add(id);
      return s;
    });
  }
  emptySet() {
    return new Set<string>();
  }
}
