import { Component, inject, signal } from '@angular/core';
import { FormBuilder, ReactiveFormsModule } from '@angular/forms';
import { ActivatedRoute } from '@angular/router';
import { MatButtonModule } from '@angular/material/button';
import { Api } from '../../core/api';
import { Page, dateInZone, weekRange } from '../../core/models';
import { FileSelection } from './selection';
interface Preview {
  notes: number;
  attachments: number;
  duplicates: string[];
}
@Component({
  imports: [ReactiveFormsModule, MatButtonModule, FileSelection],
  template: `<section class="page-heading">
      <div>
        <p class="eyebrow">TAKE YOUR WORK WITH YOU</p>
        <h1>Reports & backups</h1>
        <p class="muted">Share a polished report, or keep a copy of everything that matters.</p>
      </div>
    </section>
    <div class="two-cols">
      <section class="panel detail-panel">
        <span class="section-icon">↗</span>
        <h2>Export your notes</h2>
        <p class="muted">
          English and Thai supported. PDF reports include tasks, blockers, next steps, and time
          totals.
        </p>
        <form [formGroup]="form">
          @if (ids.length) {
            <p class="notice">{{ ids.length }} notes selected</p>
          } @else {
            <div class="two-cols">
              <label>From<input type="date" formControlName="from" /></label
              ><label>To<input type="date" formControlName="to" /></label>
            </div>
          }
          <label
            >Format<select formControlName="format">
              <option value="pdf">PDF — a formatted work report</option>
              <option value="csv">CSV — spreadsheet-friendly data</option>
              <option value="json">JSON — notes backup</option>
              <option value="zip">ZIP — notes, PDF, and selected files</option>
            </select></label
          >
        </form>
        @if (form.controls.format.value === 'zip') {
          <button mat-stroked-button (click)="loadFiles()">Choose attachments</button>
          @if (noteIds().length) {
            <app-file-selection [noteIds]="noteIds()" (changed)="attachmentIds = $event" />
          }
        }
        <p class="small muted">
          JSON contains notes only. Choose ZIP and select attachments to back up files.
        </p>
        <button mat-flat-button [disabled]="busy()" (click)="export()">
          {{ busy() ? 'Preparing report…' : 'Download export ↓' }}
        </button>
      </section>
      <section class="panel detail-panel">
        <span class="section-icon lavender">↥</span>
        <h2>Restore a backup</h2>
        <p class="muted">
          Bring your notes back into your workspace. Review the contents before importing.
        </p>
        <label class="backup-picker"
          >Choose a JSON or ZIP backup<input
            type="file"
            accept=".json,.zip"
            (change)="choose($event)"
        /></label>
        @if (importBusy()) {
          <p role="status">Validating and restoring…</p>
        }
        @if (preview()) {
          <div class="import-preview">
            <strong>{{ preview()!.notes }} notes · {{ preview()!.attachments }} attachments</strong>
            <p>{{ preview()!.duplicates.length }} duplicate notes found.</p>
            <label
              >When a note already exists<select
                [value]="mode()"
                (change)="mode.set($any($event.target).value)"
              >
                <option value="skip">Skip existing notes</option>
                <option value="replace">Replace existing notes and their attachments</option>
              </select></label
            ><button mat-flat-button [disabled]="importBusy()" (click)="restore()">
              Restore backup
            </button>
          </div>
        }
        @if (result()) {
          <p class="notice" role="status">{{ result() }}</p>
        }
      </section>
    </div>
    @if (error()) {
      <p class="error" role="alert">{{ error() }}</p>
    }
    <div class="quiet-banner">
      <span>◈</span>
      <div>
        <h3>A backup is a little peace of mind.</h3>
        <p>Your files stay private. Every export and restore is scoped to your account.</p>
      </div>
    </div>`,
})
export class Reports {
  api = inject(Api);
  fb = inject(FormBuilder);
  route = inject(ActivatedRoute);
  range = weekRange(dateInZone(this.api.user()!.timezone));
  ids = (this.route.snapshot.queryParamMap.get('ids') || '').split(',').filter(Boolean);
  form = this.fb.nonNullable.group({
    from: this.route.snapshot.queryParamMap.get('from') || this.range.from,
    to: this.route.snapshot.queryParamMap.get('to') || this.range.to,
    format: 'pdf',
  });
  busy = signal(false);
  importBusy = signal(false);
  error = signal('');
  preview = signal<Preview | null>(null);
  mode = signal('skip');
  result = signal('');
  noteIds = signal<string[]>([]);
  attachmentIds: string[] = [];
  file: File | null = null;
  selection() {
    return {
      ids: this.ids,
      from: this.form.controls.from.value,
      to: this.form.controls.to.value,
      attachmentIds: this.attachmentIds,
    };
  }
  loadFiles() {
    this.noteIds.set([]);
    this.attachmentIds = [];
    if (this.ids.length) {
      setTimeout(() => this.noteIds.set([...this.ids]));
      return;
    }
    const ids: string[] = [];
    const next = (page: number) =>
      this.api
        .get<Page>('/notes', {
          from: this.form.controls.from.value,
          to: this.form.controls.to.value,
          size: 100,
          page,
        })
        .subscribe({
          next: (p) => {
            ids.push(...p.items.map((n) => n.id));
            if (p.hasMore && ids.length <= 500) next(page + 1);
            else this.noteIds.set(ids);
          },
          error: (e) => this.error.set(this.api.error(e)),
        });
    next(1);
  }
  export() {
    if (this.busy()) return;
    this.busy.set(true);
    this.error.set('');
    const format = this.form.controls.format.value;
    this.api.http
      .post('/api/reports/' + format, this.selection(), {
        responseType: 'blob',
      })
      .subscribe({
        next: (b) => {
          const url = URL.createObjectURL(b),
            a = document.createElement('a');
          a.href = url;
          a.download = 'daily-work-notes.' + format;
          a.click();
          setTimeout(() => URL.revokeObjectURL(url), 1000);
          this.busy.set(false);
          this.api.notify('Export ready');
        },
        error: async (e) => {
          this.busy.set(false);
          if (e.error instanceof Blob) {
            try {
              this.error.set(JSON.parse(await e.error.text()).error.message);
            } catch {
              this.error.set('Export failed');
            }
          } else this.error.set(this.api.error(e));
        },
      });
  }
  choose(e: Event) {
    this.file = (e.target as HTMLInputElement).files?.[0] || null;
    this.preview.set(null);
    this.result.set('');
    if (!this.file) return;
    const data = new FormData();
    data.append('file', this.file);
    this.importBusy.set(true);
    this.error.set('');
    this.api.post<Preview>('/imports/preview', data).subscribe({
      next: (p) => {
        this.preview.set(p);
        this.importBusy.set(false);
      },
      error: (e) => {
        this.importBusy.set(false);
        this.error.set(this.api.error(e));
      },
    });
  }
  restore() {
    if (!this.file || this.importBusy()) return;
    if (
      this.mode() === 'replace' &&
      !confirm('Replace duplicate notes and their attachments with backup contents?')
    )
      return;
    const data = new FormData();
    data.append('file', this.file);
    data.append('mode', this.mode());
    this.importBusy.set(true);
    this.error.set('');
    this.api
      .post<{
        successful: number;
        skipped: number;
        failed: number;
        attachments: number;
      }>('/imports', data)
      .subscribe({
        next: (r) => {
          this.importBusy.set(false);
          this.result.set(
            `Restored ${r.successful} notes and ${r.attachments} attachments. Skipped ${r.skipped}; failed ${r.failed}.`,
          );
          this.preview.set(null);
        },
        error: (e) => {
          this.importBusy.set(false);
          this.error.set(this.api.error(e));
        },
      });
  }
}
