const { chromium } = require('playwright');

const UI_BASE_URL = process.env.BASE_URL || 'http://127.0.0.1:8890';
const API_BASE_URL = process.env.API_BASE_URL || 'http://127.0.0.1:8080';
const USERNAME = process.env.SMOKE_USERNAME || 'admin';
const PASSWORD = process.env.SMOKE_PASSWORD || '';
const EXPECTED_CONFLICT_MESSAGE = '当前策略与主机已有执行任务进行中，请稍后重试';

async function loginAndGetToken() {
  const response = await fetch(`${API_BASE_URL}/api/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ username: USERNAME, password: PASSWORD }),
  });
  if (!response.ok) {
    throw new Error(`login failed with status ${response.status}`);
  }
  const payload = await response.json();
  const token = payload?.token;
  if (!token) {
    throw new Error('login response missing token');
  }
  return token;
}

async function openScanModal(page) {
  await page.goto(`${UI_BASE_URL}/security/fim/policies`, { waitUntil: 'networkidle' });
  const openBtn = page.getByRole('button', { name: '构建基线 / 扫描' }).first();
  await openBtn.waitFor({ state: 'visible', timeout: 20000 });
  await openBtn.click();

  const modal = page.locator('.ant-modal').filter({ hasText: '执行巡检' }).last();
  await modal.waitFor({ state: 'visible', timeout: 15000 });

  const actionSelect = modal
    .locator('.ant-form-item')
    .filter({ hasText: '执行动作' })
    .locator('.ant-select')
    .first();
  await actionSelect.click();
  await page.getByText('手动扫描', { exact: true }).last().click();

  return modal;
}

async function run() {
  const token = await loginAndGetToken();
  const browser = await chromium.launch({ channel: 'chrome', headless: true });
  const context = await browser.newContext({ viewport: { width: 1440, height: 960 } });
  await context.addInitScript((value) => localStorage.setItem('token', value), token);
  const page = await context.newPage();

  try {
    const modal = await openScanModal(page);

    // Occupy the per-policy/per-host execution lock first.
    const lockRequest = fetch(`${API_BASE_URL}/api/security/fim/policies/2/scan`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${token}`,
      },
      body: JSON.stringify({ server_id: 1, scan_type: 'manual' }),
    });

    await page.waitForTimeout(100);
    const okBtn = modal.locator('.ant-modal-footer .ant-btn-primary');
    await okBtn.click();

    const toast = page
      .locator('.ant-message .ant-message-notice-content')
      .filter({ hasText: EXPECTED_CONFLICT_MESSAGE })
      .first();
    await toast.waitFor({ state: 'visible', timeout: 12000 });

    const lockResponse = await lockRequest;
    if (!lockResponse.ok) {
      throw new Error(`lock request failed with status ${lockResponse.status}`);
    }

    console.log('FIM_ERROR_SMOKE_PASS');
    console.log(`conflict_message=${EXPECTED_CONFLICT_MESSAGE}`);
  } finally {
    await browser.close();
  }
}

run().catch((error) => {
  console.error('FIM_ERROR_SMOKE_FAIL');
  console.error(String(error));
  process.exit(1);
});
