import { spawnSync } from 'node:child_process';
import { backendDir, useShell } from './paths.mjs';

const result = spawnSync('go', process.argv.slice(2), {
	cwd: backendDir,
	stdio: 'inherit',
	shell: useShell
});

if (result.error) {
	console.error(`Could not run go: ${result.error.message}`);
	process.exit(1);
}
process.exit(result.status ?? 1);
