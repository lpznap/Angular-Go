import { Component, computed, input, output, signal } from '@angular/core';
import { RouterLink } from '@angular/router';
import { MatButtonModule } from '@angular/material/button';
import { Note, minutesLabel } from '../../core/models';

@Component({
  selector: 'app-calendar',
  imports: [RouterLink, MatButtonModule],
  template: ` <section class="panel month-calendar">
      <div class="panel-heading">
        <button mat-button (click)="move(-1)" aria-label="Previous month">←</button>
        <h2>{{ monthLabel() }}</h2>
        <button mat-button (click)="move(1)" aria-label="Next month">→</button>
      </div>
      <div class="month-grid">
        @for (day of weekdays; track day) {
          <span class="weekday">{{ day }}</span>
        }
        @for (cell of cells(); track cell.date) {
          <div class="month-cell" [class.outside]="!cell.current">
            <span class="day-number">{{ cell.day }}</span>
            @for (n of cell.notes; track n.id) {
              <a [routerLink]="['/notes', n.id]" [attr.aria-label]="n.workDate + ': ' + n.title"
                ><span class="priority-dot" [class.high]="n.priority === 'high'"></span>{{ n.title
                }}<small>{{ time(n.minutes) }}</small></a
              >
            }
          </div>
        }
      </div>
    </section>
    <p class="muted small">
      The calendar shows notes from the current results page. Use Next below when there are more
      notes.
    </p>`,
})
export class Calendar {
  notes = input.required<Note[]>();
  initialDate = input.required<string>();
  rangeChanged = output<{ from: string; to: string }>();
  time = minutesLabel;
  weekdays = ['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun'];
  month = computed(() => {
    const d = new Date(this.initialDate().slice(0, 7) + '-01T12:00:00Z');
    return d;
  });
  monthLabel = computed(() =>
    new Intl.DateTimeFormat('en', { timeZone: 'UTC', month: 'long', year: 'numeric' }).format(
      this.month(),
    ),
  );
  cells = computed(() => {
    const month = this.month(),
      first = new Date(month);
    first.setUTCDate(1 - ((first.getUTCDay() + 6) % 7));
    return Array.from({ length: 42 }, (_, i) => {
      const d = new Date(first);
      d.setUTCDate(d.getUTCDate() + i);
      const date = d.toISOString().slice(0, 10);
      return {
        date,
        day: d.getUTCDate(),
        current: d.getUTCMonth() === month.getUTCMonth(),
        notes: this.notes().filter((n) => n.workDate === date),
      };
    });
  });
  move(direction: number) {
    const first = new Date(this.month());
    first.setUTCMonth(first.getUTCMonth() + direction);
    const last = new Date(first);
    last.setUTCMonth(last.getUTCMonth() + 1);
    last.setUTCDate(0);
    this.rangeChanged.emit({
      from: first.toISOString().slice(0, 10),
      to: last.toISOString().slice(0, 10),
    });
  }
}
