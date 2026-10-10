import { spawnSync } from 'node:child_process';
import { backendDir, useShell } from './paths.mjs';

const result = spawnSync('gofmt', ['-l', '.'], {
	cwd: backendDir,
	encoding: 'utf8',
	shell: useShell
});

if (result.error || result.status !== 0) {
	console.error(result.stderr || result.error?.message || 'gofmt failed');
	process.exit(1);
}

const unformatted = result.stdout.trim();
if (unformatted) {
	console.error(`These Go files need formatting (run pnpm format):\n${unformatted}`);
	process.exit(1);
}
console.log('Go formatting OK');
