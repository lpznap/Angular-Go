import { Component, input, output, inject, signal, OnInit } from '@angular/core';
import { MatCheckboxModule } from '@angular/material/checkbox';
import { Api } from '../../core/api';
import { Attachment } from '../../core/models';
@Component({
  selector: 'app-file-selection',
  imports: [MatCheckboxModule],
  template: `<div class="file-selection">
    <h3>Include attachments</h3>
    @for (f of files(); track f.id) {
      <mat-checkbox [checked]="selected().includes(f.id)" (change)="toggle(f.id)"
        >{{ f.name }} <small>({{ (f.size / 1024).toFixed(1) }} KB)</small></mat-checkbox
      >
    } @empty {
      <p class="muted small">No attachments in the selected notes.</p>
    }
  </div>`,
})
export class FileSelection implements OnInit {
  api = inject(Api);
  noteIds = input.required<string[]>();
  changed = output<string[]>();
  files = signal<Attachment[]>([]);
  selected = signal<string[]>([]);
  ngOnInit() {
    for (const id of this.noteIds())
      this.api.get<Attachment[]>('/notes/' + id + '/attachments').subscribe({
        next: (items) => this.files.update((v) => [...v, ...items]),
        error: (e) => this.api.notify(this.api.error(e)),
      });
  }
  toggle(id: string) {
    this.selected.update((v) => (v.includes(id) ? v.filter((x) => x !== id) : [...v, id]));
    this.changed.emit(this.selected());
  }
}
