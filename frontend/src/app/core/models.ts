export interface Task {
  text: string;
  done: boolean;
}
export interface Note {
  id: string;
  workDate: string;
  title: string;
  project: string;
  description: string;
  tasks: Task[];
  status: 'todo' | 'progress' | 'completed' | 'blocked';
  priority: 'low' | 'medium' | 'high';
  tags: string[];
  minutes: number;
  blockers: string;
  nextSteps: string;
  version: number;
  createdAt: string;
  updatedAt: string;
}
export interface Attachment {
  id: string;
  noteId: string;
  name: string;
  mime: string;
  size: number;
}
export interface User {
  id: string;
  email: string;
  timezone: string;
  csrf: string;
  maxFileBytes: number;
}
export interface Page {
  items: Note[];
  page: number;
  size: number;
  hasMore: boolean;
}
export interface Project {
  id: string;
  name: string;
  color: string;
}
export function dateInZone(zone: string, now = new Date()): string {
  return new Intl.DateTimeFormat('en-CA', {
    timeZone: zone,
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
  }).format(now);
}
export function minutesLabel(value: number): string {
  return `${Math.floor(value / 60)}h ${value % 60}m`;
}
export function blankNote(date: string): Note {
  return {
    id: '',
    workDate: date,
    title: '',
    project: '',
    description: '',
    tasks: [],
    status: 'todo',
    priority: 'medium',
    tags: [],
    minutes: 0,
    blockers: '',
    nextSteps: '',
    version: 0,
    createdAt: '',
    updatedAt: '',
  };
}
export function weekRange(day: string): { from: string; to: string } {
  const d = new Date(day + 'T12:00:00Z');
  d.setUTCDate(d.getUTCDate() - ((d.getUTCDay() + 6) % 7));
  const from = d.toISOString().slice(0, 10);
  d.setUTCDate(d.getUTCDate() + 6);
  return { from, to: d.toISOString().slice(0, 10) };
}
