const http = require('http');
const { spawn } = require('child_process');
const path = require('path');

const port = Number(process.env.BROWSER_DISCOVERY_PORT || 31730);
const scriptPath = process.env.BROWSER_DISCOVERY_SCRIPT || path.join(__dirname, 'browser-discovery.cjs');

function readBody(req) {
  return new Promise((resolve, reject) => {
    let data = '';
    req.setEncoding('utf8');
    req.on('data', (chunk) => { data += chunk; });
    req.on('end', () => resolve(data));
    req.on('error', reject);
  });
}

function runHelper(input) {
  return new Promise((resolve, reject) => {
    const child = spawn('node', [scriptPath], {
      stdio: ['pipe', 'pipe', 'pipe'],
      env: process.env,
    });

    let stdout = '';
    let stderr = '';
    child.stdout.on('data', (chunk) => { stdout += chunk.toString(); });
    child.stderr.on('data', (chunk) => { stderr += chunk.toString(); });
    child.on('error', reject);
    child.on('close', (code) => {
      if (code !== 0) {
        reject(new Error(stderr.trim() || `helper exited with code ${code}`));
        return;
      }
      resolve(stdout);
    });
    child.stdin.end(input);
  });
}

const server = http.createServer(async (req, res) => {
  if (req.method !== 'POST' || req.url !== '/discover') {
    res.writeHead(404, { 'Content-Type': 'text/plain; charset=utf-8' });
    res.end('not found');
    return;
  }

  try {
    const body = await readBody(req);
    const output = await runHelper(body);
    res.writeHead(200, { 'Content-Type': 'application/json; charset=utf-8' });
    res.end(output);
  } catch (error) {
    res.writeHead(500, { 'Content-Type': 'application/json; charset=utf-8' });
    res.end(JSON.stringify({ error: String(error && error.message ? error.message : error) }));
  }
});

server.listen(port, '0.0.0.0', () => {
  console.log(`browser discovery helper listening on ${port}`);
});
