import { Component, inject, signal } from '@angular/core';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { ActivatedRoute } from '@angular/router';
import { MatButtonModule } from '@angular/material/button';
import { Api } from '../../core/api';
import { Note, Page, dateInZone, weekRange } from '../../core/models';
import { FileSelection } from '../reports/selection';
interface Attempt {
  id: string;
  subject: string;
  outcome: string;
  detail: string;
  createdAt: string;
  recipients: { to: string[]; cc: string[] };
}
@Component({
  imports: [ReactiveFormsModule, MatButtonModule, FileSelection],
  template: `<section class="page-heading">
      <div>
        <p class="eyebrow">KEEP EVERYONE IN THE LOOP</p>
        <h1>Email summaries</h1>
        <p class="muted">Turn your progress into a thoughtful update.</p>
      </div>
    </section>
    <div class="two-cols email-layout">
      <form class="panel detail-panel" [formGroup]="form" (ngSubmit)="send()">
        <h2>Compose an update</h2>
        @if (ids.length) {
          <p class="badge">{{ ids.length }} selected notes</p>
        } @else {
          <div class="segmented">
            <button type="button" (click)="todayRange()">Today</button
            ><button type="button" (click)="weeklyRange()">This week</button>
          </div>
          <div class="two-cols">
            <label>From date<input type="date" formControlName="from" /></label
            ><label>To date<input type="date" formControlName="endDate" /></label>
          </div>
        }
        <label
          >To<input
            aria-label="To"
            formControlName="to"
            placeholder="colleague@example.com"
            type="text"
          /><small class="muted">Separate addresses with commas</small></label
        ><label>CC (optional)<input formControlName="cc" placeholder="manager@example.com" /></label
        ><label>Subject<input formControlName="subject" /></label
        ><label class="autosave"
          ><input type="checkbox" formControlName="pdf" /> Include PDF report</label
        ><button mat-stroked-button type="button" (click)="preview()" [disabled]="busy()">
          Refresh preview & attachments
        </button>
        @if (noteIds().length) {
          <app-file-selection [noteIds]="noteIds()" (changed)="attachmentIds = $event" />
        }
        @if (error()) {
          <p class="error" role="alert">{{ error() }}</p>
        }
        @if (outcome()) {
          <p class="notice" role="status">{{ outcome() }}</p>
        }
        <button mat-flat-button [disabled]="busy() || form.invalid || sent()">
          {{ busy() ? 'Sending…' : 'Send summary →' }}
        </button>
        @if (sent()) {
          <button mat-stroked-button type="button" (click)="retry()">
            Start an intentional new send
          </button>
        }
      </form>
      <section class="panel detail-panel">
        <p class="eyebrow">EMAIL PREVIEW</p>
        @if (html()) {
          <iframe
            class="email-preview"
            title="Email summary preview"
            sandbox=""
            [srcdoc]="html()"
          ></iframe>
        } @else {
          <div class="empty">
            <span class="empty-symbol">✉</span>
            <h3>A clear update, ready to share.</h3>
            <p>Refresh the preview to review the notes and choose attachments.</p>
          </div>
        }
      </section>
    </div>
    <section class="panel history">
      <div class="panel-heading">
        <h2>Send history</h2>
        <span class="muted small">SMTP acceptance is not confirmed delivery</span>
      </div>
      @for (h of history(); track h.id) {
        <div class="history-row">
          <div>
            <strong>{{ h.subject }}</strong
            ><small>{{ h.recipients.to.join(', ') }} · {{ h.createdAt }}</small>
            <p class="muted small">{{ h.detail }}</p>
          </div>
          <span class="badge">{{ h.outcome }}</span>
        </div>
      } @empty {
        <p class="empty">Your sent updates will appear here.</p>
      }
    </section>`,
})
export class Email {
  api = inject(Api);
  fb = inject(FormBuilder);
  route = inject(ActivatedRoute);
  today = dateInZone(this.api.user()!.timezone);
  ids = (this.route.snapshot.queryParamMap.get('ids') || '').split(',').filter(Boolean);
  form = this.fb.nonNullable.group({
    from: this.route.snapshot.queryParamMap.get('from') || this.today,
    endDate: this.route.snapshot.queryParamMap.get('to') || this.today,
    to: ['', Validators.required],
    cc: '',
    subject: ['Work summary', Validators.required],
    pdf: false,
  });
  busy = signal(false);
  sent = signal(false);
  error = signal('');
  outcome = signal('');
  html = signal('');
  history = signal<Attempt[]>([]);
  noteIds = signal<string[]>([]);
  attachmentIds: string[] = [];
  key = crypto.randomUUID();
  constructor() {
    this.loadHistory();
  }
  selection() {
    return {
      ids: this.ids,
      from: this.form.controls.from.value,
      to: this.form.controls.endDate.value,
      attachmentIds: this.attachmentIds,
    };
  }
  todayRange() {
    this.form.controls.from.setValue(this.today);
    this.form.controls.endDate.setValue(this.today);
  }
  weeklyRange() {
    const r = weekRange(this.today);
    this.form.controls.from.setValue(r.from);
    this.form.controls.endDate.setValue(r.to);
  }
  preview() {
    this.noteIds.set([]);
    this.attachmentIds = [];
    this.api.post<{ html: string }>('/email/preview', this.selection()).subscribe({
      next: (v) => this.html.set(v.html),
      error: (e) => this.error.set(this.api.error(e)),
    });
    if (this.ids.length) {
      setTimeout(() => this.noteIds.set([...this.ids]));
      return;
    }
    const all: Note[] = [];
    const load = (page: number) =>
      this.api
        .get<Page>('/notes', {
          from: this.form.controls.from.value,
          to: this.form.controls.endDate.value,
          size: 100,
          page,
        })
        .subscribe({
          next: (p) => {
            all.push(...p.items);
            if (p.hasMore && all.length <= 500) load(page + 1);
            else this.noteIds.set(all.map((n) => n.id));
          },
          error: (e) => this.error.set(this.api.error(e)),
        });
    load(1);
  }
  send() {
    if (this.busy() || this.sent() || this.form.invalid) return;
    const v = this.form.getRawValue();
    if (!v.to.trim()) {
      this.error.set('Enter at least one recipient');
      return;
    }
    this.busy.set(true);
    this.error.set('');
    this.api
      .post<{ outcome: string; detail?: string; duplicate: boolean }>('/email/send', {
        ...this.selection(),
        dateTo: this.form.controls.endDate.value,
        to: v.to
          .split(',')
          .map((s) => s.trim())
          .filter(Boolean),
        cc: v.cc
          .split(',')
          .map((s) => s.trim())
          .filter(Boolean),
        subject: v.subject,
        pdf: v.pdf,
        key: this.key,
      })
      .subscribe({
        next: (r) => {
          this.busy.set(false);
          this.sent.set(true);
          this.outcome.set(r.detail || 'Recorded outcome: ' + r.outcome);
          this.loadHistory();
        },
        error: (e) => {
          this.busy.set(false);
          this.error.set(this.api.error(e));
        },
      });
  }
  retry() {
    if (
      confirm('Start a new send? Check history and your mail provider first to avoid duplicates.')
    ) {
      this.key = crypto.randomUUID();
      this.sent.set(false);
      this.outcome.set('');
    }
  }
  loadHistory() {
    this.api.get<Attempt[]>('/email/history').subscribe({
      next: (h) => this.history.set(h),
      error: (e) => this.error.set(this.api.error(e)),
    });
  }
}
