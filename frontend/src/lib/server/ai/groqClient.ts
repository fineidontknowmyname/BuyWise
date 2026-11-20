import { aiRateLimiter } from './rateLimiter.js';
import { groqApiKey, groqModel, groqVisionModel } from '#lib/server/config.js';

export interface ChatMessage {
	role: 'system' | 'user' | 'assistant' | 'tool';
	content: string | Array<{ type: string; text?: string; image_url?: { url: string } }> | null;
	name?: string;
	tool_call_id?: string;
	tool_calls?: Array<{
		id: string;
		type: string;
		function: { name: string; arguments: string };
	}>;
}

const GROQ_ENDPOINT = 'https://api.groq.com/openai/v1/chat/completions';
export const DEFAULT_MODEL = groqModel() || 'openai/gpt-oss-120b';
export const VISION_MODEL = groqVisionModel() || 'qwen/qwen3.8-27b';
const FALLBACK_MODEL = 'openai/gpt-oss-20b';

function getApiKey(): string {
	return groqApiKey();
}

export interface GroqChatCompletion {
	choices?: Array<{
		message?: {
			role: 'system' | 'user' | 'assistant' | 'tool';
			content: string | null;
			tool_calls?: Array<{
				id: string;
				type: string;
				function: { name: string; arguments: string };
			}>;
		};
		[key: string]: unknown;
	}>;
	model?: string;
	[key: string]: unknown;
}

export async function callGroq(
	messages: ChatMessage[],
	options: {
		fetchFn?: typeof fetch;

		jsonMode?: boolean;
		maxTokens?: number;
		temperature?: number;
		model?: string;
		signal?: AbortSignal;
		quiet?: boolean;
	} = {}
): Promise<{ success: boolean; data?: GroqChatCompletion; error?: string; retryable?: boolean }> {
	const apiKey = getApiKey();
	if (!apiKey) {
		return {
			success: false,
			error:
				'GROQ_API_KEY is not configured in backend/.env. AI features are in offline fallback mode.'
		};
	}

	// Rate limiter check
	const rateCheck = aiRateLimiter.check();
	if (!rateCheck.allowed) {
		return {
			success: false,
			error: `AI rate limit reached to conserve tokens. Please wait ${rateCheck.retryAfterSeconds}s before retrying.`
		};
	}

	const fetchImpl = options.fetchFn || fetch;
	const model = options.model || DEFAULT_MODEL;

	aiRateLimiter.recordRequest();

	const payload: Record<string, unknown> = {
		model,
		messages,
		max_tokens: options.maxTokens || 2048,
		temperature: options.temperature ?? 0.2
	};

	if (options.jsonMode) {
		payload.response_format = { type: 'json_object' };
	}

	try {
		const res = await fetchImpl(GROQ_ENDPOINT, {
			method: 'POST',
			headers: {
				Authorization: `Bearer ${apiKey}`,
				'Content-Type': 'application/json'
			},
			body: JSON.stringify(payload),
			signal: options.signal ?? AbortSignal.timeout(20000)
		});

		if (res.status === 429) {
			aiRateLimiter.triggerCooldown(60);
			return {
				success: false,
				error: 'Groq free tier rate limit reached. Cooldown activated for 60 seconds.'
			};
		}

		if (!res.ok) {
			const errBody = await res.json().catch(() => ({}));
			const msg = errBody?.error?.message || `Groq error (status ${res.status})`;
			if (!options.quiet)
				console.error(`[GROQ ERROR] Status: ${res.status}, Model: ${model}, Message: ${msg}`);

			// If the specified model is unavailable, attempt retry with fallback model
			if (
				model !== FALLBACK_MODEL &&
				(res.status === 404 || msg.includes('model_not_found') || msg.includes('does not exist'))
			) {
				return callGroq(messages, { ...options, model: FALLBACK_MODEL });
			}

			return { success: false, error: msg, retryable: res.status >= 500 };
		}

		const data = (await res.json()) as GroqChatCompletion;
		return { success: true, data };
	} catch (err: unknown) {
		const isTimeout = err instanceof Error && err.name === 'TimeoutError';
		return {
			success: false,
			retryable: true,
			error: isTimeout ? 'AI request timed out.' : 'Network error connecting to Groq.'
		};
	}
}

export interface ParsedExpenseResult {
	amountPaise: number | null;
	amountFormatted: string;
	categoryId?: string;
	suggestedCategory: string;
	occurredOn: string;
	note: string;
	confidence: string;
}

export async function parseNaturalLanguageExpense(
	input: string,
	categoryNames: string[],
	fetchFn: typeof fetch
): Promise<ParsedExpenseResult> {
	const today = new Date().toISOString().split('T')[0];

	const prompt = `Extract expense fields from this user statement: "${input}".
Available categories: ${JSON.stringify(categoryNames)}.
Today's date in India: ${today}.
Respond strictly in JSON:
{
  "amountRupees": number,
  "suggestedCategory": string (must match one of the available categories if possible, else "Miscellaneous"),
  "occurredOn": "YYYY-MM-DD",
  "note": "short description"
}`;

	const res = await callGroq(
		[
			{
				role: 'system',
				content: 'You are a financial entity extraction assistant. Output strict JSON.'
			},
			{ role: 'user', content: prompt }
		],
		{ fetchFn, jsonMode: true, maxTokens: 1024, temperature: 0.1 }
	);

	if (!res.success || !res.data) {
		// Fallback deterministic extraction
		const match = input.match(/(?:₹|rs\.?|inr)?\s*(\d+(?:\.\d{1,2})?)/i);
		const rupees = match ? parseFloat(match[1]) : 0;
		return {
			amountPaise: rupees > 0 ? Math.round(rupees * 100) : null,
			amountFormatted: rupees > 0 ? `₹${rupees.toFixed(2)}` : '0',
			suggestedCategory: categoryNames[0] || 'General',
			occurredOn: today,
			note: input,
			confidence: 'low (deterministic fallback)'
		};
	}

	try {
		const rawContent = res.data?.choices?.[0]?.message?.content || '{}';
		const parsed = JSON.parse(rawContent);
		const rupees = Number(parsed.amountRupees) || 0;
		const result = {
			amountPaise: rupees > 0 ? Math.round(rupees * 100) : null,
			amountFormatted: rupees > 0 ? `₹${rupees.toFixed(2)}` : '0',
			suggestedCategory: parsed.suggestedCategory || 'Miscellaneous',
			occurredOn: parsed.occurredOn || today,
			note: parsed.note || input,
			confidence: 'high'
		};
		return result;
	} catch {
		return {
			amountPaise: null,
			amountFormatted: '0',
			suggestedCategory: categoryNames[0] || 'General',
			occurredOn: today,
			note: input,
			confidence: 'low'
		};
	}
}
