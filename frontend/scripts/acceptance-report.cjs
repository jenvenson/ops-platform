const fs = require('fs');
const path = require('path');

const artifactDir = path.join(__dirname, '../artifacts/acceptance-smoke');
const jsonFile = path.join(artifactDir, 'last-run.json');
const htmlFile = path.join(artifactDir, 'report.html');

if (!fs.existsSync(jsonFile) || !fs.existsSync(htmlFile)) {
  console.error('Smoke report not found. Run `npm run acceptance:smoke` first.');
  process.exit(1);
}

const result = JSON.parse(fs.readFileSync(jsonFile, 'utf8'));
const passedSteps = (result.steps || []).filter((step) => step.status === 'passed').length;
const failedSteps = (result.steps || []).filter((step) => step.status === 'failed').length;
const totalDuration = (result.steps || []).reduce((sum, step) => sum + (step.durationMs || 0), 0);
const mode = result.mode || 'full';

console.log(`Status: ${result.status}`);
console.log(`Mode: ${mode}`);
console.log(`Checked At: ${result.checkedAt}`);
console.log(`Base URL: ${result.baseUrl}`);
console.log(`Passed Steps: ${passedSteps}`);
console.log(`Failed Steps: ${failedSteps}`);
console.log(`Duration: ${totalDuration} ms`);
if ((result.steps || []).length) {
  console.log('Steps:');
  for (const step of result.steps) {
    const screenshot = step.screenshot ? ` | screenshot: ${step.screenshot}` : '';
    console.log(`- [${step.status}] ${step.label} | ${step.durationMs} ms${screenshot}`);
  }
}
console.log(`Report: ${htmlFile}`);
console.log(`JSON: ${jsonFile}`);
