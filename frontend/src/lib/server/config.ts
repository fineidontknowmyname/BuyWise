import { API_URL } from '$app/env/private';
import fs from 'node:fs';
import path from 'node:path';

export function apiUrl(): string {
	return API_URL;
}

let cachedGroqKey: string | null = null;
let cachedGroqModel: string | null = null;
let cachedGroqVisionModel: string | null = null;

function loadEnvFallback(): { key: string; model: string; visionModel: string } {
	if (cachedGroqKey !== null && cachedGroqModel !== null && cachedGroqVisionModel !== null) {
		return { key: cachedGroqKey, model: cachedGroqModel, visionModel: cachedGroqVisionModel };
	}

	let key = process.env.GROQ_API_KEY || '';
	let model = process.env.GROQ_MODEL || 'openai/gpt-oss-120b';
	let visionModel = process.env.GROQ_VISION_MODEL || 'qwen/qwen3.8-27b';

	if (!key || model === 'openai/gpt-oss-120b' || visionModel === 'qwen/qwen3.8-27b') {
		const candidates = [
			path.resolve(process.cwd(), 'backend/.env'),
			path.resolve(process.cwd(), '../backend/.env'),
			path.resolve(process.cwd(), 'frontend/.env'),
			path.resolve(process.cwd(), '.env'),
			path.resolve(process.cwd(), '../.env')
		];

		for (const cand of candidates) {
			try {
				if (fs.existsSync(cand)) {
					const content = fs.readFileSync(cand, 'utf-8');
					for (const line of content.split(/\r?\n/)) {
						const trimmed = line.trim();
						if (!trimmed || trimmed.startsWith('#')) continue;
						const idx = trimmed.indexOf('=');
						if (idx !== -1) {
							const k = trimmed.slice(0, idx).trim();
							const v = trimmed.slice(idx + 1).trim();
							if (k === 'GROQ_API_KEY' && !key) key = v;
							if (k === 'GROQ_MODEL' && model === 'openai/gpt-oss-120b') model = v;
							if (k === 'GROQ_VISION_MODEL' && visionModel === 'qwen/qwen3.8-27b') visionModel = v;
						}
					}
					if (key && model !== 'openai/gpt-oss-120b' && visionModel !== 'qwen/qwen3.8-27b') break;
				}
			} catch {
				// ignore filesystem read error
			}
		}
	}

	cachedGroqKey = key;
	cachedGroqModel = model;
	cachedGroqVisionModel = visionModel;
	return { key, model, visionModel };
}

export function groqApiKey(): string {
	return loadEnvFallback().key;
}

export function groqModel(): string {
	return loadEnvFallback().model;
}

export function groqVisionModel(): string {
	return loadEnvFallback().visionModel;
}
