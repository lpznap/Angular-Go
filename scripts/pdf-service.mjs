// Optional native development renderer. Compose uses Gotenberg instead.
// Run: node scripts/pdf-service.mjs (after frontend npm ci + playwright install chromium).
import http from 'node:http';
import { chromium } from '../frontend/node_modules/playwright/index.mjs';

const browser = await chromium.launch({ headless: true });
const server = http.createServer(async (req, res) => {
  if (req.method === 'GET' && req.url === '/health') {
    res.writeHead(200, { 'Content-Type': 'application/json' });
    return res.end('{"status":"ready"}');
  }
  if (req.method !== 'POST' || req.url !== '/forms/chromium/convert/html') {
    res.writeHead(404); return res.end();
  }
  let page;
  try {
    const chunks = []; let size = 0;
    for await (const chunk of req) {
      size += chunk.length;
      if (size > 20 * 1024 * 1024) throw new Error('Report exceeds 20 MB');
      chunks.push(chunk);
    }
    const body = new Request('http://localhost/render', {
      method: 'POST', headers: { 'Content-Type': req.headers['content-type'] },
      body: Buffer.concat(chunks),
    });
    const form = await body.formData();
    const file = form.get('files');
    if (!(file instanceof File) || file.name !== 'index.html') throw new Error('Expected index.html');
    page = await browser.newPage({ javaScriptEnabled: false });
    await page.route('**/*', route => route.abort()); // No remote resource access.
    await page.setContent(await file.text(), { waitUntil: 'load', timeout: 30000 });
    // Embedded font loading is independent of remote resources.
    await page.evaluate(() => document.fonts.ready);
    const pdf = await page.pdf({ format: 'A4', printBackground: true, preferCSSPageSize: true });
    res.writeHead(200, { 'Content-Type': 'application/pdf' }); res.end(pdf);
  } catch {
    res.writeHead(500, { 'Content-Type': 'application/json' });
    res.end('{"error":"PDF rendering failed"}');
  } finally {
    await page?.close();
  }
});
server.listen(3000, '127.0.0.1', () => console.log('Native Chromium PDF service: http://127.0.0.1:3000'));
for (const signal of ['SIGINT', 'SIGTERM']) process.on(signal, async () => {
  server.close(); await browser.close(); process.exit(0);
});
