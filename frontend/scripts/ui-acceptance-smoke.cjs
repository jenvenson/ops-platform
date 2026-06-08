const fs = require('fs');
const path = require('path');
const { chromium } = require('playwright');

const BASE_URL = process.env.BASE_URL || 'http://127.0.0.1:8890';
const USERNAME = process.env.SMOKE_USERNAME || 'admin';
const PASSWORD = process.env.SMOKE_PASSWORD || '';
const HEADLESS = process.env.HEADLESS !== 'false';
const SMOKE_BROWSER = process.env.SMOKE_BROWSER || (process.platform === 'darwin' ? 'chrome' : 'auto');
const SMOKE_CHROME_EXECUTABLE = process.env.SMOKE_CHROME_EXECUTABLE || '';
const SMOKE_ALLOW_BUNDLED_CHROMIUM = process.env.SMOKE_ALLOW_BUNDLED_CHROMIUM === '1';
const ARTIFACT_DIR = process.env.SMOKE_ARTIFACT_DIR || path.join(__dirname, '../artifacts/acceptance-smoke');
const REQUESTED_MODE = process.argv[2] || process.env.SMOKE_MODE || 'full';

const topMenus = ['工作台', '资产中心', '变更发布', '监控中心', '告警中心', '安全中心', '系统管理'];

const submenuGroups = [
  {
    menu: '资产中心',
    items: ['项目管理', '环境管理', '主机管理', '应用流水线管理'],
  },
  {
    menu: '变更发布',
    items: ['迭代部署', '部署记录', '归档打包', '归档历史', '聚合打包', 'Consul配置变更', 'Jenkins任务'],
  },
  {
    menu: '监控中心',
    items: ['监控大屏', '监控概览', 'Grafana仪表盘'],
  },
  {
    menu: '告警中心',
    items: ['告警事件', '告警规则', '联系人管理', '通知渠道', '通知模板'],
  },
  {
    menu: 'Consul配置变更',
    items: ['配置管理', '批量配置下发', '配置操作记录'],
  },
  {
    menu: 'Jenkins任务',
    items: ['视图管理'],
  },
  {
    menu: '安全中心',
    items: ['安全概览', '扫描任务', '安全资产', '漏洞管理', '漏洞工单', '漏洞知识库'],
  },
  {
    menu: '系统管理',
    items: ['用户管理', '角色管理', '菜单管理', '系统设置'],
  },
];

const pageChecks = [
  { path: '/cmdb/projects', marker: { type: 'text', value: '新增项目' } },
  { path: '/cmdb/environments', marker: { type: 'text', value: '新增环境' } },
  { path: '/cmdb/servers', marker: { type: 'text', value: '新增服务器' } },
  { path: '/cmdb/applications', marker: { type: 'text', value: '从 Jenkins 导入' } },
  { path: '/deploy/release', marker: { type: 'text', value: '迭代部署' } },
  { path: '/deploy/history', marker: { type: 'text', value: '部署记录' } },
  { path: '/deploy/archive', marker: { type: 'text', value: '请选择要归档的项目' } },
  { path: '/deploy/archived', marker: { type: 'placeholder', value: '应用名称' } },
  { path: '/deploy/aggregate-package', marker: { type: 'text', value: '安装包聚合打包' } },
  { path: '/deploy/aggregated-history', marker: { type: 'placeholder', value: '项目名称' } },
  { path: '/consul/config', marker: { type: 'text', value: 'Consul配置' } },
  { path: '/consul/batch-all', marker: { type: 'text', value: '批量配置下发' } },
  { path: '/consul/operations', marker: { type: 'text', value: '配置操作记录' } },
  { path: '/jenkins/views', marker: { type: 'text', value: 'Jenkins视图管理功能:' } },
  { path: '/monitor/bigscreen', marker: { type: 'text', value: '监控大屏' } },
  { path: '/monitor/overview', marker: { type: 'contains', value: '服务器资源总览' } },
  { path: '/monitor/dashboards', marker: { type: 'text', value: 'Grafana 仪表盘' } },
  { path: '/platform/events', marker: { type: 'text', value: '平台事件中心' } },
  { path: '/alarm/events', marker: { type: 'text', value: '告警中' } },
  { path: '/alarm/rules', marker: { type: 'text', value: '从 Prometheus 同步规则' } },
  { path: '/alarm/contacts', marker: { type: 'text', value: '添加联系人' } },
  { path: '/alarm/channels', marker: { type: 'text', value: '添加通知渠道' } },
  { path: '/alarm/templates', marker: { type: 'text', value: '新建模板' } },
  { path: '/security/tasks', marker: { type: 'text', value: '扫描任务' } },
  { path: '/security/assets', marker: { type: 'text', value: '安全资产' } },
  { path: '/security/vulnerabilities', marker: { type: 'text', value: '漏洞管理' } },
  { path: '/security/tickets', marker: { type: 'text', value: '漏洞工单' } },
  { path: '/security/vuln-db', marker: { type: 'text', value: '漏洞知识库' } },
  { path: '/admin/users', marker: { type: 'text', value: '用户管理' } },
  { path: '/admin/roles', marker: { type: 'text', value: '角色管理' } },
  { path: '/admin/menus', marker: { type: 'text', value: '菜单管理' } },
  { path: '/admin/settings', marker: { type: 'text', value: '通用设置' } },
];

const assistantChecks = [
  {
    type: 'api',
    query: '查看归档历史',
    pageContext: { pagePath: '/deploy/archived', moduleKey: 'deploy', pageTitle: '归档历史' },
    expectedIntent: 'readonly_query',
    expectedTool: 'query_archive_history',
    expectedPath: '/deploy/archived',
  },
  {
    type: 'api',
    query: '最新告警动作',
    pageContext: { pagePath: '/alarm/events', moduleKey: 'alert', pageTitle: '告警中心' },
    expectedIntent: 'readonly_query',
    expectedTool: 'query_alert_events',
    expectedPath: '/alarm/events',
    expectedSummaryText: '规则：',
  },
  {
    type: 'inline',
    path: '/alarm/events',
    buttonText: '最新告警动作',
    expectedTexts: ['规则名称'],
  },
  {
    type: 'inline',
    path: '/deploy/archived',
    buttonText: '最近有哪些归档失败',
    expectedTexts: ['归档'],
  },
  {
    type: 'inline',
    path: '/deploy/history',
    buttonText: '最近有哪些失败部署',
    expectedTexts: ['部署'],
  },
  { type: 'navigation', query: '打开资产中心', label: '打开资产中心', path: '/cmdb/projects', marker: '新增项目' },
  { type: 'navigation', query: '打开部署记录', label: '打开部署记录', path: '/deploy/history', marker: '部署记录' },
  { type: 'navigation', query: '打开配置管理', label: '打开配置管理', path: '/consul/config', marker: 'Consul配置' },
  { type: 'navigation', query: '打开监控中心', label: '打开监控中心', path: '/monitor/bigscreen', marker: '监控大屏' },
  { type: 'navigation', query: '打开平台事件中心', label: '打开平台事件中心', path: '/platform/events', marker: '平台事件中心' },
  { type: 'navigation', query: '打开告警事件', label: '打开告警事件', path: '/alarm/events', marker: '告警中' },
  { type: 'navigation', query: '打开扫描任务', label: '打开扫描任务', path: '/security/tasks', marker: '扫描任务' },
  { type: 'navigation', query: '打开用户手册', label: '打开用户手册', path: '/user-manual', marker: '用户手册' },
  { type: 'navigation', query: '打开漏洞知识库', label: '打开漏洞知识库', path: '/security/vuln-db', marker: '漏洞知识库' },
  { type: 'navigation', query: '打开角色管理', label: '打开角色管理', path: '/admin/roles', marker: '角色管理' },
  { type: 'navigation', query: '打开我的资料', label: '打开我的资料', path: '/profile', marker: '安全设置' },
  {
    type: 'knowledge',
    query: '如何归档打包',
    expectedActions: ['打开归档打包'],
    citationPath: 'docs/user_manual.md',
  },
  {
    type: 'knowledge',
    query: '如何查看归档历史',
    expectedActions: ['打开归档历史'],
    citationPath: 'docs/user_manual.md',
  },
  {
    type: 'knowledge',
    query: '如何删除会话',
    citationPath: 'docs/user_manual.md',
  },
];

const modeDefinitions = {
  core: ['login', 'dashboard-reload', 'user-menu', 'assistant'],
  navigation: ['login', 'dashboard-reload', 'submenus', 'assistant'],
  pages: ['login', 'dashboard-reload', 'pages'],
  full: ['login', 'dashboard-reload', 'submenus', 'pages', 'user-menu', 'assistant'],
};

function ensureArtifactDir() {
  fs.mkdirSync(ARTIFACT_DIR, { recursive: true });
}

function escapeHtml(value) {
  return String(value)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;');
}

function slugify(text) {
  return text.replace(/[^\w\u4e00-\u9fa5-]+/g, '-').replace(/-+/g, '-').replace(/^-|-$/g, '');
}

function artifactFile(name) {
  return path.join(ARTIFACT_DIR, name);
}

function captureName(label, suffix = 'png') {
  return `${new Date().toISOString().replace(/[:.]/g, '-')}-${slugify(label)}.${suffix}`;
}

function writeReport(results) {
  const screenshotItems = results.steps
    .filter((step) => step.screenshot)
    .map((step) => `
      <article class="shot-card">
        <div class="shot-meta">
          <strong>${escapeHtml(step.label)}</strong>
          <span class="badge ${step.status === 'passed' ? 'ok' : 'fail'}">${escapeHtml(step.status)}</span>
          <span>${step.durationMs} ms</span>
        </div>
        <a href="./${encodeURI(step.screenshot)}" target="_blank" rel="noreferrer">
          <img src="./${encodeURI(step.screenshot)}" alt="${escapeHtml(step.label)}">
        </a>
      </article>
    `)
    .join('');

  const stepRows = results.steps
    .map((step) => `
      <tr>
        <td>${escapeHtml(step.label)}</td>
        <td><span class="badge ${step.status === 'passed' ? 'ok' : 'fail'}">${escapeHtml(step.status)}</span></td>
        <td>${step.durationMs} ms</td>
        <td>${step.screenshot ? `<a href="./${encodeURI(step.screenshot)}" target="_blank" rel="noreferrer">查看截图</a>` : '-'}</td>
      </tr>
    `)
    .join('');

  const html = `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Smoke Report</title>
  <style>
    :root {
      --bg: #f5f7fb;
      --card: #ffffff;
      --text: #172033;
      --muted: #5b6475;
      --border: #d9e0ea;
      --ok: #1f9d55;
      --fail: #d64545;
      --link: #1664d8;
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      padding: 24px;
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
      background: linear-gradient(180deg, #f7f9fc 0%, #eef3f8 100%);
      color: var(--text);
    }
    .wrap {
      max-width: 1180px;
      margin: 0 auto;
      display: grid;
      gap: 20px;
    }
    .hero, .panel {
      background: var(--card);
      border: 1px solid var(--border);
      border-radius: 16px;
      padding: 20px;
      box-shadow: 0 8px 24px rgba(16, 24, 40, 0.06);
    }
    .hero {
      display: grid;
      gap: 12px;
    }
    .title {
      display: flex;
      gap: 12px;
      align-items: center;
      flex-wrap: wrap;
    }
    h1, h2 {
      margin: 0;
      font-size: 24px;
    }
    h2 {
      font-size: 18px;
    }
    .meta, .chips {
      display: flex;
      gap: 10px;
      flex-wrap: wrap;
      color: var(--muted);
      font-size: 14px;
    }
    .badge, .chip {
      display: inline-flex;
      align-items: center;
      border-radius: 999px;
      padding: 4px 10px;
      font-size: 12px;
      font-weight: 600;
      border: 1px solid transparent;
    }
    .badge.ok, .chip.ok {
      color: var(--ok);
      background: #e9f8ef;
      border-color: #ccefd9;
    }
    .badge.fail, .chip.fail {
      color: var(--fail);
      background: #fdecec;
      border-color: #f7cdcd;
    }
    .stats {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
      gap: 12px;
    }
    .stat {
      background: #f8fafc;
      border-radius: 12px;
      padding: 14px;
      border: 1px solid var(--border);
    }
    .stat strong {
      display: block;
      font-size: 24px;
      margin-top: 6px;
    }
    table {
      width: 100%;
      border-collapse: collapse;
      font-size: 14px;
    }
    th, td {
      text-align: left;
      border-bottom: 1px solid var(--border);
      padding: 12px 10px;
      vertical-align: top;
    }
    .shot-grid {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
      gap: 16px;
    }
    .shot-card {
      border: 1px solid var(--border);
      border-radius: 14px;
      overflow: hidden;
      background: #fbfcfe;
    }
    .shot-meta {
      padding: 12px;
      display: grid;
      gap: 6px;
    }
    .shot-card img {
      width: 100%;
      display: block;
      background: #eef2f7;
    }
    a {
      color: var(--link);
      text-decoration: none;
    }
    .error {
      white-space: pre-wrap;
      font-family: ui-monospace, SFMono-Regular, monospace;
      color: var(--fail);
      background: #fff5f5;
      border: 1px solid #f4cccc;
      border-radius: 12px;
      padding: 12px;
      margin-top: 8px;
    }
  </style>
</head>
<body>
  <main class="wrap">
    <section class="hero">
      <div class="title">
        <h1>Frontend Smoke Acceptance</h1>
        <span class="badge ${results.status === 'passed' ? 'ok' : 'fail'}">${escapeHtml(results.status)}</span>
      </div>
      <div class="meta">
        <span>Base URL: <a href="${escapeHtml(results.baseUrl)}" target="_blank" rel="noreferrer">${escapeHtml(results.baseUrl)}</a></span>
        <span>Checked At: ${escapeHtml(results.checkedAt)}</span>
      </div>
      <div class="chips">
        <span class="chip ok">一级菜单 ${results.topMenus.length}</span>
        <span class="chip ok">关键页面 ${results.pageChecks.length}</span>
        <span class="chip ok">助手导航 ${results.assistantChecks.length}</span>
        <span class="chip ok">模式 ${escapeHtml(results.mode)}</span>
        <span class="chip ${results.status === 'passed' ? 'ok' : 'fail'}">步骤 ${results.steps.length}</span>
      </div>
      ${results.error ? `<div class="error">${escapeHtml(results.error)}</div>` : ''}
    </section>

    <section class="panel">
      <h2>统计</h2>
      <div class="stats">
        <div class="stat"><div>通过步骤</div><strong>${results.steps.filter((step) => step.status === 'passed').length}</strong></div>
        <div class="stat"><div>失败步骤</div><strong>${results.steps.filter((step) => step.status === 'failed').length}</strong></div>
        <div class="stat"><div>总耗时</div><strong>${results.steps.reduce((sum, step) => sum + step.durationMs, 0)} ms</strong></div>
        <div class="stat"><div>截图数量</div><strong>${results.steps.filter((step) => step.screenshot).length}</strong></div>
      </div>
    </section>

    <section class="panel">
      <h2>步骤结果</h2>
      <table>
        <thead>
          <tr>
            <th>步骤</th>
            <th>状态</th>
            <th>耗时</th>
            <th>截图</th>
          </tr>
        </thead>
        <tbody>${stepRows}</tbody>
      </table>
    </section>

    <section class="panel">
      <h2>截图</h2>
      <div class="shot-grid">
        ${screenshotItems || '<div>暂无截图</div>'}
      </div>
    </section>
  </main>
</body>
</html>`;

  fs.writeFileSync(artifactFile('report.html'), html, 'utf8');
}

async function expectExactText(locator, text) {
  await locator.getByText(text, { exact: true }).first().waitFor({ state: 'visible', timeout: 15000 });
}

async function expectContentText(page, text) {
  await page.locator('.ant-layout-content').getByText(text, { exact: true }).first().waitFor({
    state: 'visible',
    timeout: 15000,
  });
}

async function expectContentContains(page, text) {
  await page.locator('.ant-layout-content').getByText(text).first().waitFor({
    state: 'visible',
    timeout: 15000,
  });
}

async function expectContentPlaceholder(page, placeholder) {
  await page.locator('.ant-layout-content').getByPlaceholder(placeholder, { exact: true }).first().waitFor({
    state: 'visible',
    timeout: 15000,
  });
}

async function expectMarker(page, marker) {
  if (typeof marker === 'string') {
    await expectContentText(page, marker);
    return;
  }
  if (marker.type === 'contains') {
    await expectContentContains(page, marker.value);
    return;
  }
  if (marker.type === 'placeholder') {
    await expectContentPlaceholder(page, marker.value);
    return;
  }
  await expectContentText(page, marker.value);
}

async function expectPath(page, routePath) {
  await page.waitForURL((url) => url.pathname === routePath, { timeout: 15000 });
}

async function waitForAnyVisible(locators, timeout = 15000) {
  const deadline = Date.now() + timeout;

  while (Date.now() < deadline) {
    for (const locator of locators) {
      if (await locator.first().isVisible().catch(() => false)) {
        return;
      }
    }
    await locators[0].page().waitForTimeout(250);
  }

  throw new Error('dashboard ready signals not found before timeout');
}

async function expectDashboardReady(page) {
  await expectPath(page, '/');
  await page.locator('.ant-layout-content').waitFor({ state: 'visible', timeout: 15000 });
  await waitForAnyVisible([
    page.getByText('平台事件中心', { exact: true }),
    page.getByText('最近部署', { exact: true }),
    page.getByText('更多动态', { exact: true }),
    page.locator('.ant-layout-content .ant-card'),
  ]);
}

async function hasSiderMenuItem(page, text) {
  const item = page.locator('.ant-layout-sider').getByText(text, { exact: true }).first();
  return item.isVisible().catch(() => false);
}

async function login(page) {
  await page.goto(`${BASE_URL}/login`, { waitUntil: 'networkidle' });
  await page.getByPlaceholder('用户名').fill(USERNAME);
  await page.getByPlaceholder('密码').fill(PASSWORD);
  await page.getByRole('button', { name: /登\s*录/ }).click();
  await expectDashboardReady(page);
  await expectExactText(page.locator('.ant-layout-sider'), '工作台');
}

async function openUserMenu(page) {
  const header = page.locator('.ant-layout-header');
  const trigger = header.locator('.ant-dropdown-trigger').first();
  await trigger.waitFor({ state: 'visible', timeout: 10000 });
  await trigger.hover();
  const dropdown = page.locator('.ant-dropdown:visible');
  try {
    await dropdown.waitFor({ state: 'visible', timeout: 3000 });
    return;
  } catch (error) {
    await trigger.click({ force: true });
  }
  await dropdown.waitFor({ state: 'visible', timeout: 10000 });
}

async function clickUserMenuItem(page, label, expectedPath) {
  const dropdown = page.locator('.ant-dropdown:visible');
  await dropdown.waitFor({ state: 'visible', timeout: 10000 });
  const item = dropdown.locator('.ant-dropdown-menu-item').filter({ hasText: label }).first();
  await item.waitFor({ state: 'visible', timeout: 10000 });
  await item.click({ force: true });

  if (!expectedPath) {
    return;
  }

  try {
    await page.waitForURL((url) => url.pathname === expectedPath, { timeout: 5000 });
  } catch (error) {
    await item.evaluate((node) => {
      node.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }));
    });
    await page.waitForURL((url) => url.pathname === expectedPath, { timeout: 15000 });
  }
}

async function openAssistant(page) {
  const container = page.locator('.ai-chatbot-container');
  if (await container.isVisible().catch(() => false)) {
    return;
  }
  const button = page.locator('.ai-chatbot-float-button');
  await button.click();
  await container.waitFor({ state: 'visible', timeout: 10000 });
}

async function waitForAiMessage(page, previousCount) {
  await page.waitForFunction(
    (count) => document.querySelectorAll('.ai-chatbot-message.ai-message').length > count,
    previousCount,
    { timeout: 15000 }
  );
  return page.locator('.ai-chatbot-message.ai-message').last();
}

async function clickAssistantControl(button, afterClick) {
  await button.waitFor({ state: 'visible', timeout: 10000 });
  try {
    await button.click({ force: true });
    await afterClick();
  } catch (error) {
    await button.evaluate((node) => {
      node.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }));
    });
    await afterClick();
  }
}

async function checkAssistantSessions(page) {
  await openAssistant(page);
  const sessionButtons = page.locator('.ai-chatbot-session-btn');
  const panel = page.locator('.ai-chatbot-session-panel');
  await clickAssistantControl(sessionButtons.first(), () => panel.waitFor({ state: 'visible', timeout: 10000 }));
  const sessionItems = panel.locator('.ai-chatbot-session-item');
  const sessionCount = await sessionItems.count();
  if (sessionCount === 0) {
    throw new Error('assistant session list is empty');
  }
  await clickAssistantControl(sessionButtons.nth(1), () => page.locator('.ai-chatbot-welcome').getByText('您好！我是运维小助手。').waitFor({
    state: 'visible',
    timeout: 10000,
  }));
  await page.locator('.ai-chatbot-welcome').getByText('您好！我是运维小助手。').waitFor({
    state: 'visible',
    timeout: 10000,
  });
  await page.locator('.ai-chatbot-welcome').getByText('目前支持基础问答、页面导航和只读查询。').waitFor({
    state: 'visible',
    timeout: 10000,
  });
  await page.locator('.ai-chatbot-welcome').getByText('你可以继续问我部署、监控、告警、安全或系统管理相关问题。').waitFor({
    state: 'visible',
    timeout: 10000,
  });
  await panel.waitFor({ state: 'hidden', timeout: 10000 });
  await clickAssistantControl(sessionButtons.first(), () => panel.waitFor({ state: 'visible', timeout: 10000 }));
  await sessionItems.first().waitFor({ state: 'visible', timeout: 10000 });
}

async function assistantNavigate(page, check) {
  await openAssistant(page);
  const aiMessages = page.locator('.ai-chatbot-message.ai-message');
  const previousAiCount = await aiMessages.count();
  const input = page.locator('.ai-chatbot-input');
  await input.fill(check.query);
  await page.getByRole('button', { name: '发送' }).click();
  const latestMessage = await waitForAiMessage(page, previousAiCount);
  const action = latestMessage.getByRole('button', { name: check.label, exact: true });
  await action.waitFor({ state: 'visible', timeout: 15000 });
  await action.click();
  await expectPath(page, check.path);
  await expectMarker(page, check.marker);
}

async function assistantKnowledgeCheck(page, check) {
  await openAssistant(page);
  const aiMessages = page.locator('.ai-chatbot-message.ai-message');
  const previousAiCount = await aiMessages.count();
  const input = page.locator('.ai-chatbot-input');
  await input.fill(check.query);
  await page.getByRole('button', { name: '发送' }).click();
  const latestMessage = await waitForAiMessage(page, previousAiCount);
  const content = latestMessage.locator('.ai-chatbot-message-content');
  const citationSummary = latestMessage.locator('.ai-chatbot-citation-summary');
  const hasCitationSummary = await citationSummary.count();

  if (hasCitationSummary > 0) {
    const citationDetails = latestMessage.locator('.ai-chatbot-citation-details');
    const isCitationOpen = await citationDetails.evaluate((node) => node.hasAttribute('open')).catch(() => false);
    if (!isCitationOpen) {
      await citationSummary.click();
    }
  }

  const contentText = await content.innerText().catch(() => '');

  for (const text of check.expectedTexts || []) {
    if (contentText.includes(text)) {
      continue;
    }

    const snippetTexts = await latestMessage.locator('.ai-chatbot-citation-snippet').allTextContents().catch(() => []);
    const pathTexts = await latestMessage.locator('.ai-chatbot-citation-path').allTextContents().catch(() => []);
    const titleTexts = await latestMessage.locator('.ai-chatbot-citation-title').allTextContents().catch(() => []);
    const matchedInCitation = [...snippetTexts, ...pathTexts, ...titleTexts].some((value) => value.includes(text));

    if (!matchedInCitation) {
      throw new Error(`assistant knowledge response missing expected text: ${text}`);
    }
  }

  for (const actionLabel of check.expectedActions || []) {
    await latestMessage.getByRole('button', { name: actionLabel, exact: true }).waitFor({
      state: 'visible',
      timeout: 15000,
    });
  }

  if (check.citationPath) {
    await citationSummary.waitFor({ state: 'visible', timeout: 15000 });
    const citationPaths = await latestMessage.locator('.ai-chatbot-citation-path').allTextContents().catch(() => []);
    if (!citationPaths.some((value) => value.trim() === check.citationPath)) {
      throw new Error(`assistant knowledge response missing citation path: ${check.citationPath}`);
    }
  }

  for (const titleText of check.expectedCitationTitles || []) {
    const citationTitles = await latestMessage.locator('.ai-chatbot-citation-title').allTextContents().catch(() => []);
    if (!citationTitles.some((value) => value.includes(titleText))) {
      throw new Error(`assistant knowledge response missing citation title: ${titleText}`);
    }
  }

  if (check.forbidActions) {
    const actionCount = await latestMessage.locator('.ai-chatbot-action-chip').count();
    if (actionCount !== 0) {
      throw new Error(`assistant returned unexpected quick actions for query: ${check.query}`);
    }
  }
}

async function assistantInlinePromptCheck(page, check) {
  await page.goto(`${BASE_URL}${check.path}`, { waitUntil: 'networkidle' });
  await openAssistant(page);
  const aiMessages = page.locator('.ai-chatbot-message.ai-message');
  const previousAiCount = await aiMessages.count();
  await page.getByRole('button', { name: check.buttonText, exact: true }).click();
  const latestMessage = await waitForAiMessage(page, previousAiCount);
  const missingTexts = [];

  for (const text of check.expectedTexts || []) {
    try {
      await latestMessage.getByText(text, { exact: false }).first().waitFor({
        state: 'visible',
        timeout: 5000,
      });
    } catch (error) {
      missingTexts.push(text);
    }
  }

  if (!missingTexts.length) {
    return;
  }

  const resultCardCount = await latestMessage.locator('.ai-chatbot-result-card').count();
  const actionChipCount = await latestMessage.locator('.ai-chatbot-action-chip').count();
  if (resultCardCount > 0 || actionChipCount > 0) {
    return;
  }

  throw new Error(`assistant inline prompt missing expected texts: ${missingTexts.join(', ')}`);
}

async function assistantApiDecisionCheck(page, check) {
  const payload = await page.evaluate(async ({ query, pageContext }) => {
    const token = localStorage.getItem('token');
    if (!token) {
      throw new Error('missing auth token');
    }

    const sessionResp = await fetch('/api/assistant/sessions', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${token}`,
      },
      body: JSON.stringify({
        scene: 'web',
        userAgent: navigator.userAgent,
        ipAddress: '127.0.0.1',
        forceNew: true,
      }),
    });
    const sessionData = await sessionResp.json();
    if (!sessionResp.ok || !sessionData.session?.sessionId) {
      throw new Error(sessionData.error || 'failed to create assistant session');
    }

    const replyResp = await fetch('/api/assistant/messages', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${token}`,
      },
      body: JSON.stringify({
        sessionId: sessionData.session.sessionId,
        message: query,
        pageContext,
      }),
    });
    const replyData = await replyResp.json();
    if (!replyResp.ok) {
      throw new Error(replyData.error || 'assistant api request failed');
    }
    return replyData;
  }, { query: check.query, pageContext: check.pageContext });

  if (!payload.decision) {
    throw new Error('assistant api response missing decision');
  }
  if (payload.decision.intent?.name !== check.expectedIntent) {
    throw new Error(`assistant api returned unexpected intent: ${payload.decision.intent?.name}`);
  }
  const firstStep = payload.decision.executionPlan?.steps?.[0];
  if (!firstStep || firstStep.tool !== check.expectedTool) {
    throw new Error(`assistant api returned unexpected tool: ${firstStep?.tool}`);
  }
  if (!firstStep.readonly) {
    throw new Error('assistant api execution step should be readonly');
  }
  if (payload.decision.context?.pageContext?.pagePath !== check.expectedPath) {
    throw new Error(`assistant api returned unexpected page path: ${payload.decision.context?.pageContext?.pagePath}`);
  }
  if (check.expectedSummaryText && !String(payload.answer || payload.decision.summary || '').includes(check.expectedSummaryText)) {
    throw new Error(`assistant api returned unexpected summary for query: ${check.query}`);
  }
}

async function saveStepScreenshot(page, label) {
  ensureArtifactDir();
  const filename = captureName(label);
  await page.screenshot({ path: artifactFile(filename), fullPage: true });
  return filename;
}

async function runStep(page, label, fn) {
  console.log(label);
  const startedAt = Date.now();
  try {
    await fn();
    return {
      label,
      status: 'passed',
      durationMs: Date.now() - startedAt,
      screenshot: await saveStepScreenshot(page, `${label}-passed`),
    };
  } catch (error) {
    const screenshot = await saveStepScreenshot(page, `${label}-failed`);
    throw Object.assign(error, {
      stepMeta: { label, status: 'failed', durationMs: Date.now() - startedAt, screenshot },
    });
  }
}

async function launchBrowser() {
  const launchAttempts = [];

  if (SMOKE_CHROME_EXECUTABLE) {
    launchAttempts.push({
      label: `custom-chrome:${SMOKE_CHROME_EXECUTABLE}`,
      launch: () => chromium.launch({ executablePath: SMOKE_CHROME_EXECUTABLE, headless: HEADLESS }),
    });
  }

  if (SMOKE_BROWSER === 'chrome' || SMOKE_BROWSER === 'auto') {
    launchAttempts.push({
      label: 'chrome-channel',
      launch: () => chromium.launch({ channel: 'chrome', headless: HEADLESS }),
    });
  }

  const allowBundledChromium = SMOKE_BROWSER === 'chromium' || (SMOKE_BROWSER === 'auto' && process.platform !== 'darwin') || SMOKE_ALLOW_BUNDLED_CHROMIUM;
  if (allowBundledChromium) {
    launchAttempts.push({
      label: 'bundled-chromium',
      launch: () => chromium.launch({ headless: HEADLESS }),
    });
  }

  const failures = [];
  for (const attempt of launchAttempts) {
    try {
      return await attempt.launch();
    } catch (error) {
      failures.push({ label: attempt.label, error });
      console.warn(`[smoke] browser launch failed: ${attempt.label}`);
      console.warn(String(error));
    }
  }

  if (!allowBundledChromium && process.platform === 'darwin') {
    throw new Error(
      [
        `Unable to launch Playwright browser on macOS with SMOKE_BROWSER=${SMOKE_BROWSER}.`,
        'Bundled Chromium fallback is disabled by default on macOS because it crashes with SIGTRAP in this environment.',
        'Try one of:',
        '  1. SMOKE_BROWSER=chrome npm run acceptance:smoke:core',
        '  2. SMOKE_CHROME_EXECUTABLE=\"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome\" npm run acceptance:smoke:core',
        '  3. If you explicitly want bundled Chromium, set SMOKE_ALLOW_BUNDLED_CHROMIUM=1',
        `Failed attempts: ${failures.map(item => item.label).join(', ') || 'none'}`,
      ].join('\n')
    );
  }

  throw new Error(
    `Unable to launch Playwright browser. Failed attempts: ${failures.map(item => item.label).join(', ') || 'none'}`
  );
}

function resolveMode(mode) {
  if (!modeDefinitions[mode]) {
    const supported = Object.keys(modeDefinitions).join(', ');
    throw new Error(`Unsupported smoke mode: ${mode}. Supported modes: ${supported}`);
  }
  return mode;
}

async function run() {
  ensureArtifactDir();
  const mode = resolveMode(REQUESTED_MODE);
  const browser = await launchBrowser();
  const page = await browser.newPage({ viewport: { width: 1440, height: 960 } });

  const stepHandlers = {
    login: async () => {
      await login(page);
      const sider = page.locator('.ant-layout-sider');
      for (const text of topMenus) {
        await expectExactText(sider, text);
      }
    },
    'dashboard-reload': async () => {
      await page.goto(`${BASE_URL}/`, { waitUntil: 'networkidle' });
      await expectDashboardReady(page);
      await page.reload({ waitUntil: 'networkidle' });
      await expectDashboardReady(page);
    },
    submenus: async () => {
      const sider = page.locator('.ant-layout-sider');
      for (const group of submenuGroups) {
        await sider.getByText(group.menu, { exact: true }).click();
        for (const item of group.items) {
          await expectExactText(sider, item);
        }
      }
    },
    pages: async () => {
      const canAccessPlatformEvents = await hasSiderMenuItem(page, '平台事件中心');
      for (const check of pageChecks) {
        if (!canAccessPlatformEvents && check.path === '/platform/events') {
          console.log('SKIP /platform/events: menu not visible for current role');
          continue;
        }
        await page.goto(`${BASE_URL}${check.path}`, { waitUntil: 'networkidle' });
        await expectMarker(page, check.marker);
      }
    },
    'user-menu': async () => {
      await page.goto(`${BASE_URL}/`, { waitUntil: 'networkidle' });
      await openUserMenu(page);
      await clickUserMenuItem(page, '用户手册', '/user-manual');
      await expectContentText(page, '用户手册');

      await page.goto(`${BASE_URL}/`, { waitUntil: 'networkidle' });
      await openUserMenu(page);
      await clickUserMenuItem(page, '我的资料', '/profile');
      await expectContentText(page, '安全设置');
    },
    assistant: async () => {
      await page.goto(`${BASE_URL}/`, { waitUntil: 'networkidle' });
      await checkAssistantSessions(page);
      const canAccessPlatformEvents = await hasSiderMenuItem(page, '平台事件中心');
      for (const check of assistantChecks) {
        if (check.type === 'api') {
          await assistantApiDecisionCheck(page, check);
          continue;
        }
        if (check.type === 'knowledge') {
          await assistantKnowledgeCheck(page, check);
          continue;
        }
        if (check.type === 'inline') {
          await assistantInlinePromptCheck(page, check);
          continue;
        }
        if (!canAccessPlatformEvents && check.path === '/platform/events') {
          console.log('SKIP assistant navigation /platform/events: menu not visible for current role');
          continue;
        }
        await assistantNavigate(page, check);
      }
    },
  };

  const stepMetadata = {
    login: '登录并检查一级菜单',
    'dashboard-reload': '检查工作台首次进入和刷新',
    submenus: '检查关键菜单展开项',
    pages: '检查关键页面标识',
    'user-menu': '检查右上角入口',
    assistant: '检查运维小助手导航',
  };

  const selectedSteps = modeDefinitions[mode];

  const results = {
    baseUrl: BASE_URL,
    checkedAt: new Date().toISOString(),
    mode,
    topMenus,
    submenuGroups: selectedSteps.includes('submenus') ? submenuGroups.map((item) => item.menu) : [],
    pageChecks: selectedSteps.includes('pages') ? pageChecks.map((item) => item.path) : [],
    assistantChecks: selectedSteps.includes('assistant') ? assistantChecks.map((item) => item.query) : [],
    steps: [],
  };

  try {
    for (const [index, stepKey] of selectedSteps.entries()) {
      results.steps.push(
        await runStep(page, `${index + 1}/${selectedSteps.length + 1} ${stepMetadata[stepKey]}`, async () => {
          await stepHandlers[stepKey]();
        })
      );
    }

    results.steps.push(await runStep(page, `${selectedSteps.length + 1}/${selectedSteps.length + 1} 输出验收结果`, async () => {
      fs.writeFileSync(
        artifactFile('last-run.json'),
        JSON.stringify({ ...results, status: 'passed' }, null, 2),
        'utf8'
      );
      writeReport({ ...results, status: 'passed' });
    }));

    console.log('SMOKE PASS');
  } catch (error) {
    if (error?.stepMeta) {
      results.steps.push(error.stepMeta);
    }
    fs.writeFileSync(
      artifactFile('last-run.json'),
      JSON.stringify({ ...results, status: 'failed', error: String(error) }, null, 2),
      'utf8'
    );
    writeReport({ ...results, status: 'failed', error: String(error) });
    console.error('SMOKE FAIL');
    console.error(error);
    process.exitCode = 1;
  } finally {
    await browser.close();
  }
}

run();
