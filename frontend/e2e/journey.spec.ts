import { test, expect } from '@playwright/test';
import fs from 'node:fs/promises';
test('account, note autosave, attachment, Mailpit email, export and restore', async ({
  page,
  request,
}, testInfo) => {
  const email = `journey-${Date.now()}@example.com`,
    password = 'test-password-12345';
  await page.goto('/auth');
  await page.getByRole('button', { name: 'New here? Create an account' }).click();
  await page.getByLabel('Email address').fill(email);
  await page.getByLabel('Password', { exact: true }).fill(password);
  await page.getByRole('button', { name: 'Create account', exact: true }).click();
  await expect(page.getByRole('heading', { name: 'A little more clarity.' })).toBeVisible();
  await page.getByRole('button', { name: /Sign out/ }).click();
  await page.getByLabel('Email address').fill(email);
  await page.getByLabel('Password', { exact: true }).fill(password);
  await page.getByRole('button', { name: 'Sign in', exact: true }).click();
  await expect(page.getByRole('heading', { name: 'A little more clarity.' })).toBeVisible();
  await page.getByRole('link', { name: '＋ Add note', exact: true }).first().click();
  await page.getByLabel('Note title', { exact: true }).fill('Release review งานวันนี้');
  await page.getByLabel('Project or customer').fill('Daily Work Notes');
  await page.getByLabel('THE DETAILS').fill('Completed review. ภาษาไทย ทดสอบการแสดงผล');
  await page.getByLabel('THE DETAILS').press('ControlOrMeta+A');
  await page.getByRole('button', { name: 'Bold', exact: true }).click();
  await expect(page.getByLabel('THE DETAILS').locator('strong')).toContainText('ภาษาไทย');
  await page.getByLabel('Hours', { exact: true }).fill('1');
  await page.getByLabel('Minutes', { exact: true }).fill('30');
  await expect(page.getByText('All changes saved', { exact: true })).toBeVisible();
  await page.getByRole('button', { name: '＋ Add task' }).click();
  await page.getByLabel('Task description').fill('Verify the release');
  await page.getByLabel('Task completed').check();
  await expect(page.getByText('All changes saved', { exact: true })).toBeVisible();
  await page.getByLabel('Upload attachment').setInputFiles({
    name: 'work.txt',
    mimeType: 'text/plain',
    buffer: Buffer.from('Persistent attachment ภาษาไทย'),
  });
  await expect(page.getByText('work.txt', { exact: true })).toBeVisible();
  const noteURL = page.url();
  await page.reload();
  await expect(page.getByLabel('Note title', { exact: true })).toHaveValue(
    'Release review งานวันนี้',
  );
  await expect(page.getByText('work.txt', { exact: true })).toBeVisible();
  await page.getByRole('link', { name: '✉ Email this note' }).click();
  await page.getByLabel('To', { exact: true }).fill('team@example.com');
  await page.getByLabel('Subject', { exact: true }).fill(`Review ${email}`);
  await page.getByRole('button', { name: 'Refresh preview & attachments' }).click();
  await page.getByRole('checkbox', { name: /work.txt/ }).check();
  if (process.env['SKIP_PDF'] !== '1') await page.getByLabel('Include PDF report').check();
  await page.getByRole('button', { name: 'Send summary →' }).click();
  await expect(
    page
      .getByText('SMTP accepted the message. Delivery is not confirmed.', { exact: true })
      .first(),
  ).toBeVisible();
  const mailpit = process.env['MAILPIT_URL'] || 'http://localhost:8025';
  await expect
    .poll(async () => {
      const response = await request.get(mailpit + '/api/v1/messages');
      const data = await response.json();
      return data.messages.some((m: { Subject: string }) => m.Subject === `Review ${email}`);
    })
    .toBe(true);
  await page.goto(noteURL);
  await page.getByRole('link', { name: '↗ Export note' }).click();
  await page.getByRole('combobox', { name: 'Format', exact: true }).selectOption('json');
  const downloadEvent = page.waitForEvent('download');
  await page.getByRole('button', { name: 'Download export ↓' }).click();
  const download = await downloadEvent;
  const backup = await fs.readFile((await download.path())!);
  const parsed = JSON.parse(backup.toString());
  expect(parsed.notes[0].title).toContain('งานวันนี้');
  await page
    .getByLabel('Choose a JSON or ZIP backup')
    .setInputFiles({ name: 'backup.json', mimeType: 'application/json', buffer: backup });
  await expect(page.getByText('1 duplicate notes found.')).toBeVisible();
  await page.getByRole('button', { name: 'Restore backup', exact: true }).click();
  await expect(page.getByRole('status').filter({ hasText: 'Skipped 1' })).toBeVisible();
  if (process.env['SKIP_PDF'] !== '1') {
    await page.getByRole('combobox', { name: 'Format', exact: true }).selectOption('pdf');
    const pdfEvent = page.waitForEvent('download');
    await page.getByRole('button', { name: 'Download export ↓' }).click();
    const pdf = await pdfEvent;
    await pdf.saveAs(testInfo.outputPath('thai-report.pdf'));
    expect((await fs.readFile((await pdf.path())!)).subarray(0, 5).toString()).toBe('%PDF-');
    await page.getByRole('combobox', { name: 'Format', exact: true }).selectOption('zip');
    await page.getByRole('button', { name: 'Choose attachments', exact: true }).click();
    await page.getByRole('checkbox', { name: /work.txt/ }).check();
    const zipEvent = page.waitForEvent('download');
    await page.getByRole('button', { name: 'Download export ↓' }).click();
    const zip = await zipEvent;
    const zipBuffer = await fs.readFile((await zip.path())!);
    await page
      .getByLabel('Choose a JSON or ZIP backup')
      .setInputFiles({ name: 'backup.zip', mimeType: 'application/zip', buffer: zipBuffer });
    await expect(page.getByText('1 notes · 1 attachments')).toBeVisible();
    await page
      .getByRole('combobox', { name: 'When a note already exists' })
      .selectOption('replace');
    page.once('dialog', (d) => d.accept());
    await page.getByRole('button', { name: 'Restore backup', exact: true }).click();
    await expect(
      page.getByRole('status').filter({ hasText: 'Restored 1 notes and 1 attachments' }),
    ).toBeVisible();
  }
  await page.goto(noteURL);
  await expect(page.getByText('work.txt', { exact: true })).toBeVisible();
  await page.setViewportSize({ width: 390, height: 844 });
  await expect(page.getByRole('button', { name: 'Save note', exact: true })).toBeVisible();
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(
    true,
  );
  await page.screenshot({ path: testInfo.outputPath('mobile-editor.png'), fullPage: true });
  await page.setViewportSize({ width: 1440, height: 1000 });
  await page.goto('/');
  await page.screenshot({ path: testInfo.outputPath('dashboard.png'), fullPage: true });
});
test('stale edits are rejected and drafts survive', async ({ page, context }) => {
  const email = `conflict-${Date.now()}@example.com`;
  await page.goto('/auth');
  await page.getByRole('button', { name: 'New here? Create an account' }).click();
  await page.getByLabel('Email address').fill(email);
  await page.getByLabel('Password', { exact: true }).fill('test-password-12345');
  await page.getByRole('button', { name: 'Create account', exact: true }).click();
  await page.getByRole('link', { name: '＋ Add note', exact: true }).first().click();
  await page.getByLabel('Note title', { exact: true }).fill('Original');
  await expect(page.getByText('All changes saved', { exact: true })).toBeVisible();
  const url = page.url();
  const other = await context.newPage();
  await other.goto(url);
  await expect(other.getByLabel('Note title', { exact: true })).toHaveValue('Original');
  await page.getByLabel('Note title', { exact: true }).fill('New server version');
  await expect(page.getByText('All changes saved', { exact: true })).toBeVisible();
  await other.getByLabel('Note title', { exact: true }).fill('My stale draft');
  await expect(other.getByRole('alert')).toContainText('changed elsewhere');
  await expect(other.getByText('Save failed · draft preserved')).toBeVisible();
  await other.reload();
  await expect(other.getByText('A local draft is available.')).toBeVisible();
  await other.close();
});
