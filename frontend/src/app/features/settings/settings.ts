import { Component, inject } from "@angular/core";
import { FormBuilder, ReactiveFormsModule, Validators } from "@angular/forms";
import { MatButtonModule } from "@angular/material/button";
import { Api } from "../../core/api";
import { User } from "../../core/models";
@Component({
  imports: [ReactiveFormsModule, MatButtonModule],
  template: `<section class="page-heading">
      <div>
        <p class="eyebrow">MAKE THIS SPACE YOURS</p>
        <h1>Settings</h1>
        <p class="muted">Small preferences for a workspace that feels right.</p>
      </div>
    </section>
    <form
      class="panel detail-panel settings-panel"
      [formGroup]="form"
      (ngSubmit)="save()"
    >
      <h2>Time & place</h2>
      <p class="muted">
        Your work dates stay exactly as entered. This timezone controls “today”
        and the timestamps you see.
      </p>
      <label
        >Display timezone<input
          formControlName="timezone"
          list="timezones"
          placeholder="Asia/Bangkok" /><datalist id="timezones">
          @for (z of zones; track z) {
            <option [value]="z"></option>
          }</datalist></label
      ><button mat-flat-button [disabled]="form.invalid">
        Save preferences
      </button>
      <hr />
      <h3>Your account</h3>
      <p>{{ api.user()!.email }}</p>
      <p class="muted small">
        Drafts are saved in this browser, scoped to your account. Use Reports &
        backups for a portable copy of saved notes.
      </p>
      <button type="button" mat-stroked-button (click)="clearDrafts()">
        Clear my local drafts
      </button>
    </form>`,
})
export class Settings {
  api = inject(Api);
  fb = inject(FormBuilder);
  form = this.fb.nonNullable.group({
    timezone: [this.api.user()!.timezone, Validators.required],
  });
  zones = Intl.supportedValuesOf("timeZone");
  save() {
    this.api.put<User>("/settings", this.form.getRawValue()).subscribe({
      next: (u) => {
        this.api.user.set(u);
        this.api.notify("Preferences saved");
      },
      error: (e) => this.api.notify(this.api.error(e)),
    });
  }
  clearDrafts() {
    if (confirm("Permanently clear your unsaved drafts from this browser?")) {
      const prefix = "daily-draft:" + this.api.user()!.id + ":";
      for (const k of Object.keys(localStorage))
        if (k.startsWith(prefix)) localStorage.removeItem(k);
      this.api.notify("Local drafts cleared");
    }
  }
}
