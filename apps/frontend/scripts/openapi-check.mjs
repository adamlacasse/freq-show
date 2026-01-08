import { execFile } from 'node:child_process';
import { mkdtemp, readFile } from 'node:fs/promises';
import os from 'node:os';
import path from 'node:path';
import { promisify } from 'node:util';

const execFileAsync = promisify(execFile);

const frontendRoot = path.resolve(import.meta.dirname, '..');
const specPath = path.resolve(frontendRoot, '..', '..', 'docs', 'openapi.yaml');
const generatedPath = path.resolve(frontendRoot, 'src', 'app', 'models', 'openapi-types.generated.ts');

const binName = process.platform === 'win32' ? 'openapi-typescript.cmd' : 'openapi-typescript';
const binPath = path.resolve(frontendRoot, 'node_modules', '.bin', binName);

async function main() {
  const tempDir = await mkdtemp(path.join(os.tmpdir(), 'freqshow-openapi-'));
  const tempOut = path.join(tempDir, 'openapi-types.generated.ts');

  try {
    await execFileAsync(binPath, [specPath, '-o', tempOut], {
      cwd: frontendRoot,
      env: process.env,
      maxBuffer: 10 * 1024 * 1024,
    });
  } catch (err) {
    console.error('openapi:check failed to run openapi-typescript.');
    console.error('Try: npm install && npm run -s openapi:gen');
    throw err;
  }

  const [expected, actual] = await Promise.all([
    readFile(tempOut, 'utf8'),
    readFile(generatedPath, 'utf8'),
  ]);

  if (expected !== actual) {
    console.error('OpenAPI generated types are out of date.');
    console.error(`Spec:      ${path.relative(frontendRoot, specPath)}`);
    console.error(`Generated: ${path.relative(frontendRoot, generatedPath)}`);
    console.error('Run: npm run -s openapi:gen');
    process.exitCode = 1;
    return;
  }

  // Keep output quiet for CI usage.
}

main().catch((err) => {
  if (process.exitCode !== 1) {
    process.exitCode = 1;
  }
  // Ensure something useful is printed if the thrown error had no message.
  if (!err?.message) {
    console.error(err);
  }
});
