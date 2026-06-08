const API_BASE_URL = process.env.API_BASE_URL || 'http://127.0.0.1:8080';
const USERNAME = process.env.SMOKE_USERNAME || 'admin';
const PASSWORD = process.env.SMOKE_PASSWORD || 'admin123';
const WEB_SCAN_TARGET = process.env.WEB_SCAN_TARGET || 'http://10.99.99.185/web_01/';
const WEB_SCAN_LOGIN_URL = process.env.WEB_SCAN_LOGIN_URL || WEB_SCAN_TARGET;
const WEB_SCAN_APP_USERNAME = process.env.WEB_SCAN_APP_USERNAME || 'web_01';
const WEB_SCAN_APP_PASSWORD = process.env.WEB_SCAN_APP_PASSWORD || 'Web_01';
const POLL_INTERVAL_MS = Number(process.env.SECURITY_WEB_POLL_INTERVAL_MS || 3000);
const TIMEOUT_MS = Number(process.env.SECURITY_WEB_TIMEOUT_MS || 240000);

async function expectOk(response, label) {
  if (response.ok) {
    return response;
  }
  const body = await response.text();
  throw new Error(`${label} failed with status ${response.status}: ${body}`);
}

async function login() {
  const response = await fetch(`${API_BASE_URL}/api/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ username: USERNAME, password: PASSWORD }),
  });
  await expectOk(response, 'login');
  const payload = await response.json();
  if (!payload?.token) {
    throw new Error('login response missing token');
  }
  return payload.token;
}

async function apiJson(path, token, options = {}) {
  const response = await fetch(`${API_BASE_URL}${path}`, {
    ...options,
    headers: {
      Authorization: `Bearer ${token}`,
      'Content-Type': 'application/json',
      ...(options.headers || {}),
    },
  });
  await expectOk(response, path);
  return response.json();
}

async function generateAuthFlow(token) {
  const payload = {
    preset: 'auto',
    target_url: WEB_SCAN_TARGET,
    login_url: WEB_SCAN_LOGIN_URL,
  };
  const result = await apiJson('/api/security/auth-flow/generate', token, {
    method: 'POST',
    body: JSON.stringify(payload),
  });
  if (!result?.auth_flow) {
    throw new Error('auth flow response missing auth_flow');
  }
  return result.auth_flow;
}

async function createTask(token, authFlow) {
  const payload = {
    name: `security-web-regression-${Date.now()}`,
    target_type: 'url',
    target: WEB_SCAN_TARGET,
    scan_type: 'web',
    web_scan_profile: 'standard',
    discovery_mode: 'browser',
    auth_mode: 'advanced',
    username: WEB_SCAN_APP_USERNAME,
    password: WEB_SCAN_APP_PASSWORD,
    login_url: WEB_SCAN_LOGIN_URL,
    auth_flow: authFlow,
  };
  const task = await apiJson('/api/security/tasks', token, {
    method: 'POST',
    body: JSON.stringify(payload),
  });
  if (!task?.id) {
    throw new Error('create task response missing id');
  }
  return task;
}

async function pollTask(token, taskId) {
  const startedAt = Date.now();
  while (Date.now() - startedAt < TIMEOUT_MS) {
    const task = await apiJson(`/api/security/tasks/${taskId}`, token, {
      headers: { 'Content-Type': 'application/json' },
    });
    const phase = task?.current_run?.phase || task?.latest_run?.phase || '';
    const message = task?.message || '';
    console.log(`poll status=${task.status} phase=${phase} message=${message}`);
    if (['completed', 'failed', 'cancelled'].includes(task.status)) {
      return task;
    }
    await new Promise((resolve) => setTimeout(resolve, POLL_INTERVAL_MS));
  }
  throw new Error(`task ${taskId} did not finish within ${TIMEOUT_MS}ms`);
}

function summarizeTask(task, targets, evidences, occurrences, vulnerabilities) {
  const latestRun = task.latest_run || null;
  return {
    task_id: task.id,
    status: task.status,
    message: task.message || '',
    latest_run_id: latestRun?.id || null,
    latest_run_phase: latestRun?.phase || null,
    target_count: targets.length,
    evidence_count: evidences.length,
    occurrence_count: occurrences.length,
    vulnerability_count: vulnerabilities.length,
    target_kinds: [...new Set(targets.map((item) => item.target_kind).filter(Boolean))].sort(),
    evidence_types: [...new Set(evidences.map((item) => item.evidence_type).filter(Boolean))].sort(),
  };
}

function assertRegression(task, summary) {
  if (task.status !== 'completed') {
    throw new Error(`task ${task.id} ended with status ${task.status}`);
  }
  if (!task.latest_run?.id) {
    throw new Error(`task ${task.id} missing latest_run`);
  }
  if ((summary.target_count || 0) <= 0) {
    throw new Error(`task ${task.id} missing targets`);
  }
  if ((summary.evidence_count || 0) <= 0) {
    throw new Error(`task ${task.id} missing evidences`);
  }
  if ((summary.occurrence_count || 0) <= 0) {
    throw new Error(`task ${task.id} missing occurrences`);
  }
  if ((summary.vulnerability_count || 0) <= 0) {
    throw new Error(`task ${task.id} missing vulnerabilities`);
  }
}

async function run() {
  const token = await login();
  const authFlow = await generateAuthFlow(token);
  const createdTask = await createTask(token, authFlow);
  console.log(`created task ${createdTask.id}`);

  const task = await pollTask(token, createdTask.id);
  const [targets, evidences, occurrences, vulnerabilities] = await Promise.all([
    apiJson(`/api/security/tasks/${createdTask.id}/targets`, token),
    apiJson(`/api/security/tasks/${createdTask.id}/evidences`, token),
    apiJson(`/api/security/tasks/${createdTask.id}/occurrences`, token),
    apiJson(`/api/security/tasks/${createdTask.id}/vulnerabilities`, token),
  ]);

  const summary = summarizeTask(task, targets, evidences, occurrences, vulnerabilities);
  assertRegression(task, summary);

  console.log('SECURITY_WEB_REGRESSION_PASS');
  console.log(JSON.stringify(summary, null, 2));
}

run().catch((error) => {
  console.error('SECURITY_WEB_REGRESSION_FAIL');
  console.error(String(error));
  process.exit(1);
});
