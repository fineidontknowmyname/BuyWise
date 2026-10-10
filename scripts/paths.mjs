import path from 'node:path';
import { fileURLToPath } from 'node:url';

export const rootDir = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
export const backendDir = path.join(rootDir, 'backend');
export const frontendDir = path.join(rootDir, 'frontend');
export const useShell = process.platform === 'win32';
