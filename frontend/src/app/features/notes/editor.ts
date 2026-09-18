import { Component, DestroyRef, inject, signal } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { FormArray, FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { ActivatedRoute, RouterLink } from '@angular/router';
import { MatButtonModule } from '@angular/material/button';
import { MatCheckboxModule } from '@angular/material/checkbox';
import { debounceTime, Subject } from 'rxjs';
import { Api } from '../../core/api';
import { Note, Project, blankNote, dateInZone } from '../../core/models';
import { Attachments } from '../attachments/attachments';
import { RichText } from '../../core/rich-text';
@Component({
  imports: [
    ReactiveFormsModule,
    RouterLink,
    MatButtonModule,
    MatCheckboxModule,
    Attachments,
    RichText,
  ],
  template: `<a class="back-link" routerLink="/notes">← Back to notes</a>
    <section class="page-heading">
      <div>
        <p class="eyebrow">MAKE TODAY COUNT</p>
        <h1>{{ id() ? 'Your work note' : 'A fresh page.' }}</h1>
      </div>
      <div class="editor-actions">
        <span class="save-state" role="status">{{ saveState() }}</span
        ><button
          mat-flat-button
          [disabled]="saving() || form.invalid || conflict()"
          (click)="save()"
        >
          Save note
        </button>
      </div>
    </section>
    @if (error()) {
      <div class="error" role="alert">
        {{ error() }}
        @if (conflict()) {
          <button mat-button (click)="reload()">Reload server version</button
          ><button mat-button (click)="downloadDraft()">Download my draft</button>
        }
      </div>
    }
    @if (draftAvailable()) {
      <div class="selection-bar">
        A local draft is available.<button mat-button (click)="restoreDraft()">Restore draft</button
        ><button mat-button (click)="discardDraft()">Discard draft</button>
      </div>
    }
    @if (loading()) {
      <p class="loading">Loading note…</p>
    } @else {
      <form [formGroup]="form" class="editor-grid">
        <section class="panel editor-main">
          <input
            class="title-input"
            formControlName="title"
            placeholder="Give your day a headline…"
            aria-label="Note title"
          />
          <div class="editor-meta">
            <label>Work date<input type="date" formControlName="workDate" /></label
            ><label
              >Project or customer<input
                formControlName="project"
                list="projects"
                placeholder="Choose or type a project" /><datalist id="projects">
                @for (p of projects(); track p.id) {
                  <option [value]="p.name"></option>
                }</datalist
            ></label>
          </div>
          <p class="section-label">THE DETAILS</p>
          <app-rich-text formControlName="description" />
          <div class="panel-heading task-heading">
            <h3>Tasks</h3>
            <button mat-button type="button" (click)="addTask()">＋ Add task</button>
          </div>
          <div formArrayName="tasks">
            @for (task of tasks.controls; track task; let i = $index) {
              <div class="task-input" [formGroupName]="i">
                <mat-checkbox formControlName="done" aria-label="Task completed" /><input
                  formControlName="text"
                  aria-label="Task description"
                  placeholder="One thing to move forward"
                /><button
                  mat-button
                  type="button"
                  (click)="tasks.removeAt(i)"
                  aria-label="Remove task"
                >
                  ×
                </button>
              </div>
            } @empty {
              <p class="muted small">Break your work into a few manageable steps.</p>
            }
          </div>
          <div class="two-cols">
            <label
              >Blockers<textarea
                formControlName="blockers"
                rows="3"
                placeholder="What’s getting in the way?"
              ></textarea></label
            ><label
              >Next steps<textarea
                formControlName="nextSteps"
                rows="3"
                placeholder="Where will you pick up tomorrow?"
              ></textarea>
            </label>
          </div>
        </section>
        <div class="editor-side">
          <section class="panel detail-panel">
            <h3>Note details</h3>
            <label
              >Status<select formControlName="status">
                <option value="todo">To do</option>
                <option value="progress">In progress</option>
                <option value="completed">Completed</option>
                <option value="blocked">Blocked</option>
              </select></label
            ><label
              >Priority<select formControlName="priority">
                <option value="low">Low</option>
                <option value="medium">Medium</option>
                <option value="high">High</option>
              </select></label
            >
            <div class="two-cols">
              <label>Hours<input type="number" min="0" max="24" formControlName="hours" /></label
              ><label
                >Minutes<input type="number" min="0" max="59" formControlName="minutes"
              /></label>
            </div>
            <label
              >Tags<input formControlName="tags" placeholder="design, planning, review" /><small
                class="muted"
                >Separate tags with commas</small
              ></label
            ><label class="autosave"
              ><input type="checkbox" [checked]="autosave()" (change)="autosave.set(!autosave())" />
              Autosave</label
            >
            @if (id()) {
              <p class="small muted">Version {{ version() }}<br />Updated {{ updated() }}</p>
            }
          </section>
          @if (id()) {
            <section class="panel detail-panel">
              <h3>Share & organize</h3>
              <a mat-button routerLink="/email" [queryParams]="{ ids: id() }">✉ Email this note</a
              ><a mat-button routerLink="/reports" [queryParams]="{ ids: id() }">↗ Export note</a
              ><button mat-button type="button" (click)="duplicate()">▤ Duplicate note</button
              ><button mat-button class="danger" type="button" (click)="remove()">
                Delete note
              </button>
            </section>
          }
        </div>
      </form>
      @if (id()) {
        <app-attachments [noteId]="id()" />
      } @else {
        <p class="muted small">Save your note to add attachments.</p>
      }
    }`,
})
export class Editor {
  api = inject(Api);
  route = inject(ActivatedRoute);
  fb = inject(FormBuilder);
  destroy = inject(DestroyRef);
  id = signal('');
  version = signal(0);
  updated = signal('');
  loading = signal(false);
  saving = signal(false);
  error = signal('');
  conflict = signal(false);
  saveState = signal('Not saved yet');
  autosave = signal(true);

  draftAvailable = signal(false);
  projects = signal<Project[]>([]);
  private changes = new Subject<void>();
  private revision = 0;
  private draftKey = '';
  private createdAt = '';
  form = this.fb.nonNullable.group({
    title: ['', Validators.required],
    workDate: [dateInZone(this.api.user()!.timezone), Validators.required],
    project: '',
    description: '',
    status: this.fb.nonNullable.control<Note['status']>('todo'),
    priority: this.fb.nonNullable.control<Note['priority']>('medium'),
    tags: '',
    hours: [0, [Validators.min(0), Validators.max(24)]],
    minutes: [0, [Validators.min(0), Validators.max(59)]],
    blockers: '',
    nextSteps: '',
    tasks: this.fb.array<ReturnType<Editor['taskGroup']>>([]),
  });
  get tasks() {
    return this.form.controls.tasks;
  }
  taskGroup(text = '', done = false) {
    return this.fb.nonNullable.group({ text, done });
  }
  constructor() {
    this.route.paramMap.pipe(takeUntilDestroyed()).subscribe((p) => {
      const nextId = p.get('id') === 'new' ? '' : p.get('id') || '';
      if (nextId === this.id() && this.version() > 0) return;
      this.id.set(nextId);
      this.error.set('');
      this.conflict.set(false);
      this.saveState.set('Not saved yet');
      this.draftKey = 'daily-draft:' + this.api.user()!.id + ':' + (this.id() || 'new');
      this.draftAvailable.set(!!localStorage.getItem(this.draftKey));
      if (this.id()) this.reload();
      else this.apply(blankNote(dateInZone(this.api.user()!.timezone)));
    });
    this.api.get<Project[]>('/projects').subscribe((p) => this.projects.set(p));
    this.form.valueChanges.pipe(takeUntilDestroyed()).subscribe(() => {
      this.revision++;
      this.saveState.set('Unsaved changes · draft on this device');
      this.persistDraft();
      this.changes.next();
    });
    this.changes.pipe(debounceTime(1000), takeUntilDestroyed()).subscribe(() => {
      if (this.autosave() && !this.conflict()) this.save();
    });
  }
  apply(n: Note) {
    this.tasks.clear({ emitEvent: false });
    for (const t of n.tasks) this.tasks.push(this.taskGroup(t.text, t.done), { emitEvent: false });
    this.form.patchValue(
      { ...n, tags: n.tags.join(', '), hours: Math.floor(n.minutes / 60), minutes: n.minutes % 60 },
      { emitEvent: false },
    );
    this.version.set(n.version);
    this.createdAt = n.createdAt;
    this.updated.set(
      n.updatedAt
        ? new Intl.DateTimeFormat('en', {
            timeZone: this.api.user()!.timezone,
            dateStyle: 'medium',
            timeStyle: 'short',
          }).format(new Date(n.updatedAt))
        : '',
    );
    this.form.markAsPristine();
  }
  note(): Note {
    const v = this.form.getRawValue();
    return {
      ...v,
      id: this.id(),
      minutes: Number(v.hours) * 60 + Number(v.minutes),
      tags: [
        ...new Set(
          v.tags
            .split(',')
            .map((t) => t.trim())
            .filter(Boolean),
        ),
      ],
      version: this.version(),
      createdAt: this.createdAt,
      updatedAt: '',
    };
  }
  persistDraft() {
    try {
      localStorage.setItem(this.draftKey, JSON.stringify(this.note()));
    } catch {
      this.saveState.set('Draft storage unavailable — save manually');
    }
  }
  save() {
    if (this.form.invalid || this.saving() || this.conflict()) return;
    const revision = this.revision;
    this.saving.set(true);
    this.saveState.set('Saving…');
    this.error.set('');
    const request = this.id()
      ? this.api.put<Note>('/notes/' + this.id(), this.note())
      : this.api.post<Note>('/notes', this.note());
    request.pipe(takeUntilDestroyed(this.destroy)).subscribe({
      next: (n) => {
        const wasNew = !this.id();
        this.id.set(n.id);
        this.version.set(n.version);
        this.saving.set(false);
        if (revision === this.revision) {
          localStorage.removeItem(this.draftKey);
          this.apply(n);
          this.saveState.set('All changes saved');
        } else {
          this.persistDraft();
          this.saveState.set('Unsaved changes');
          this.changes.next();
        }
        if (wasNew) {
          localStorage.removeItem(this.draftKey);
          this.draftKey = 'daily-draft:' + this.api.user()!.id + ':' + n.id;
          if (revision !== this.revision) this.persistDraft();
          void this.api.router.navigate(['/notes', n.id], { replaceUrl: true });
        }
      },
      error: (e) => {
        this.saving.set(false);
        this.conflict.set(e.status === 409);
        this.error.set(this.api.error(e));
        this.saveState.set('Save failed · draft preserved');
        this.persistDraft();
      },
    });
  }
  reload() {
    this.loading.set(true);
    this.api
      .get<Note>('/notes/' + this.id())
      .pipe(takeUntilDestroyed(this.destroy))
      .subscribe({
        next: (n) => {
          if (n.id !== this.id()) return;
          this.apply(n);
          this.conflict.set(false);
          this.error.set('');
          this.loading.set(false);
          this.saveState.set('Server version loaded');
          this.draftAvailable.set(!!localStorage.getItem(this.draftKey));
        },
        error: (e) => {
          this.error.set(this.api.error(e));
          this.loading.set(false);
        },
      });
  }
  restoreDraft() {
    try {
      const n = JSON.parse(localStorage.getItem(this.draftKey) || 'null') as Note;
      if (n) {
        if (n.version !== this.version() && this.id()) {
          if (
            !confirm(
              'This draft is based on an older version. Load it over the current server version for review? Saving will replace that version.',
            )
          )
            return;
          n.version = this.version();
        }
        this.apply(n);
        this.saveState.set('Local draft restored — review and save');
      }
    } catch {
      this.error.set('Could not read the saved draft');
    }
    this.draftAvailable.set(false);
  }
  discardDraft() {
    if (confirm('Discard the local draft?')) {
      localStorage.removeItem(this.draftKey);
      this.draftAvailable.set(false);
    }
  }
  addTask() {
    this.tasks.push(this.taskGroup());
  }
  duplicate() {
    this.api.post<Note>('/notes/' + this.id() + '/duplicate', {}).subscribe({
      next: (n) => {
        void this.api.router.navigate(['/notes', n.id]);
        this.api.notify('Note duplicated. Attachments remain on the original.');
      },
      error: (e) => this.api.notify(this.api.error(e)),
    });
  }
  remove() {
    if (!confirm('Delete this note and all its attachments?')) return;
    this.api.delete('/notes/' + this.id() + '?version=' + this.version()).subscribe({
      next: () => {
        localStorage.removeItem(this.draftKey);
        void this.api.router.navigateByUrl('/notes');
      },
      error: (e) => this.api.notify(this.api.error(e)),
    });
  }
  downloadDraft() {
    const url = URL.createObjectURL(
      new Blob([JSON.stringify(this.note(), null, 2)], { type: 'application/json' }),
    );
    const a = document.createElement('a');
    a.href = url;
    a.download = 'unsaved-draft.json';
    a.click();
    URL.revokeObjectURL(url);
  }
}
