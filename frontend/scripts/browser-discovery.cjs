const { chromium } = require('playwright');
const { URL } = require('node:url');

function rewriteEntryURL(rawURL) {
  const resolved = new URL(rawURL);
  if (['docker-host', 'host.docker.internal'].includes(resolved.hostname)) {
    resolved.hostname = '127.0.0.1';
  }
  return resolved;
}

function restoreTargetURL(rawURL, browserBase, originalBase) {
  const resolved = new URL(rawURL, browserBase);
  if (resolved.protocol === browserBase.protocol && resolved.host === browserBase.host) {
    resolved.hostname = originalBase.hostname;
    resolved.port = originalBase.port;
  }
  return resolved.toString();
}

function classify(url) {
  const lower = String(url || '').toLowerCase();
  if (lower.includes('/api/') || lower.includes('/base/')) return 'api';
  if (lower.includes('/auth/')) return 'auth';
  return 'page';
}

function isStatic(url) {
  const lower = String(url || '').toLowerCase();
  try {
    const parsed = new URL(lower);
    const pathname = parsed.pathname || '/';
    const search = parsed.search || '';
    if (
      pathname.startsWith('/@vite/') ||
      pathname === '/@react-refresh' ||
      pathname.startsWith('/node_modules/') ||
      pathname.startsWith('/src/')
    ) {
      return true;
    }
    const lowValueKeywords = ['logo', 'background', 'favicon', 'avatar', 'watermark', 'qrcode', 'captcha'];
    if (lowValueKeywords.some((keyword) => pathname.includes(keyword) || search.includes(keyword))) {
      return true;
    }
    const hasDownloadHint = ['/download', '/downloadget', '/preview', '/export', '/file-manage/']
      .some((token) => pathname.includes(token));
    if (hasDownloadHint && (search.includes('uuid=') || search.includes('file_id=') || search.includes('fileid='))) {
      return true;
    }
    if (search.includes('uuid=default-')) {
      return true;
    }
    return [
      '.js', '.css', '.png', '.jpg', '.jpeg', '.gif', '.svg', '.ico',
      '.woff', '.woff2', '.ttf', '.eot', '.map', '.mjs', '.ts', '.tsx', '.jsx',
    ].some((suffix) => pathname.endsWith(suffix));
  } catch {
    return [
      '.js', '.css', '.png', '.jpg', '.jpeg', '.gif', '.svg', '.ico',
      '.woff', '.woff2', '.ttf', '.eot', '.map', '.mjs', '.ts', '.tsx', '.jsx',
    ].some((suffix) => lower.endsWith(suffix));
  }
}

function normalizeTarget(raw, base, sameHostOnly) {
  try {
    const resolved = new URL(raw, base);
    resolved.hash = '';
    if (!['http:', 'https:'].includes(resolved.protocol)) return null;
    if (sameHostOnly && resolved.hostname !== base.hostname) return null;
    return resolved.toString();
  } catch {
    return null;
  }
}

async function readStdin() {
  return await new Promise((resolve, reject) => {
    let data = '';
    process.stdin.setEncoding('utf8');
    process.stdin.on('data', (chunk) => { data += chunk; });
    process.stdin.on('end', () => resolve(data));
    process.stdin.on('error', reject);
  });
}

async function launchBrowser() {
  try {
    return await chromium.launch({
      channel: 'chrome',
      headless: true,
      args: ['--no-proxy-server'],
    });
  } catch (error) {
    return await chromium.launch({
      headless: true,
      args: ['--no-proxy-server'],
    });
  }
}

(async () => {
  const raw = await readStdin();
  const req = JSON.parse(raw || '{}');
  const entryURL = String(req.entry_url || '').trim();
  if (!entryURL) {
    throw new Error('entry_url is required');
  }

  const originalBase = new URL(entryURL);
  const browserBase = rewriteEntryURL(entryURL);
  const options = req.options || {};
  const maxURLs = Number(options.maxURLs || options.MaxURLs || 25);
  const sameHostOnly = options.sameHostOnly ?? options.SameHostOnly ?? true;
  const sessionHeaders = Array.isArray(req.session_headers) ? req.session_headers : [];
  const navigationTimeout = Number(options.navigationTimeoutMs || options.NavigationTimeoutMs || 30000);
  const settleTimeout = Number(options.settleTimeoutMs || options.SettleTimeoutMs || 5000);
  const postLoadDelay = Number(options.postLoadDelayMs || options.PostLoadDelayMs || 1200);

  const browser = await launchBrowser();
  const context = await browser.newContext();
  const page = await context.newPage();
  page.setDefaultNavigationTimeout(navigationTimeout);
  page.setDefaultTimeout(navigationTimeout);

  const headers = {};
  for (const item of sessionHeaders) {
    if (!item || !item.name) continue;
    headers[item.name] = item.value || '';
  }
  if (Object.keys(headers).length > 0) {
    await page.setExtraHTTPHeaders(headers);
  }

  const targets = [];
  const seen = new Set();
  const add = (rawURL, source) => {
    if (!rawURL || seen.size >= maxURLs) return;
    const restored = restoreTargetURL(rawURL, browserBase, originalBase);
    const normalized = normalizeTarget(restored, originalBase, sameHostOnly);
    if (!normalized || isStatic(normalized) || seen.has(normalized)) return;
    seen.add(normalized);
    targets.push({
      url: normalized,
      kind: classify(normalized),
      depth: source === 'entry' ? 0 : 1,
      source,
    });
  };

  add(entryURL, 'entry');

  page.on('request', (request) => add(request.url(), 'browser-request'));
  page.on('framenavigated', (frame) => add(frame.url(), 'browser-frame'));

  try {
    // Prefer DOM readiness first; some apps keep long-running requests that never reach stable network idle.
    await page.goto(browserBase.toString(), { waitUntil: 'domcontentloaded', timeout: navigationTimeout });
    try {
      await page.waitForLoadState('networkidle', { timeout: settleTimeout });
    } catch {
      // Best-effort only. We still collect targets from a partially settled page.
    }
    if (postLoadDelay > 0) {
      await page.waitForTimeout(postLoadDelay);
    }
  } catch (error) {
    const currentURL = page.url();
    if (!currentURL || currentURL === 'about:blank') {
      throw error;
    }
  }

  const domTargets = await page.evaluate(() => {
    const values = [];
    document.querySelectorAll('a[href], form[action], script[src]').forEach((node) => {
      const href = node.getAttribute('href') || node.getAttribute('action') || node.getAttribute('src');
      if (href) values.push(href);
    });
    return values;
  });
  for (const item of domTargets) {
    add(item, 'browser-dom');
  }

  console.log(JSON.stringify({ targets }));
  await browser.close();
})().catch((error) => {
  console.error(error && error.stack ? error.stack : String(error));
  process.exit(1);
});
