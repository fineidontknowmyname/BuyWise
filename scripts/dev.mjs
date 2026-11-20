import { spawn, spawnSync } from 'node:child_process';
import { backendDir, frontendDir, useShell } from './paths.mjs';

if (!process.env.DATABASE_URL) {
	console.error('DATABASE_URL is not set. Copy backend/.env.example to backend/.env and fill it in.');
	process.exit(1);
}

const migration = spawnSync('go', ['run', './cmd/migrate'], {
	cwd: backendDir,
	stdio: 'inherit',
	shell: useShell
});
if (migration.status !== 0) {
	console.error('Migrations failed; not starting the app.');
	process.exit(migration.status ?? 1);
}

const webEnv = { ...process.env };
delete webEnv.SERPAPI_API_KEY;

const children = [
	spawn('go', ['run', './cmd/api'], { cwd: backendDir, stdio: 'inherit', shell: useShell, env: process.env }),
	spawn('pnpm', ['dev'], { cwd: frontendDir, stdio: 'inherit', shell: useShell, env: webEnv })
];

let stopping = false;

function stopAll(exitCode) {
	if (stopping) return;
	stopping = true;
	for (const child of children) {
		if (child.exitCode !== null) continue;
		if (process.platform === 'win32') {
			spawnSync('taskkill', ['/pid', String(child.pid), '/T', '/F'], { stdio: 'ignore' });
		} else {
			child.kill('SIGTERM');
		}
	}
	process.exit(exitCode);
}

for (const child of children) {
	child.on('exit', (code) => stopAll(code ?? 1));
}
process.on('SIGINT', () => stopAll(0));
process.on('SIGTERM', () => stopAll(0));
