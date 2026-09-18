import { describe, it, expect } from 'vitest';
import { blankNote, dateInZone, minutesLabel, weekRange } from './models';
describe('work dates and summaries', () => {
  it('uses the configured timezone at UTC day boundaries', () => {
    const now = new Date('2026-09-17T18:30:00Z');
    expect(dateInZone('Asia/Bangkok', now)).toBe('2026-09-18');
    expect(dateInZone('America/New_York', now)).toBe('2026-09-17');
  });
  it('finds inclusive Monday–Sunday ranges over month boundaries', () => {
    expect(weekRange('2026-03-01')).toEqual({ from: '2026-02-23', to: '2026-03-01' });
  });
  it('preserves date-only values for new drafts', () => {
    const n = blankNote('2026-09-18');
    expect(n.workDate).toBe('2026-09-18');
    expect(n.tasks).toEqual([]);
    expect(n.version).toBe(0);
  });
  it('formats durations without decimal rounding', () => {
    expect(minutesLabel(125)).toBe('2h 5m');
  });
});
