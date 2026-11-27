import { json, type RequestHandler } from '@sveltejs/kit';
import {
	assistantQuestions,
	type AssistantAnswer,
	type AssistantRequest,
	type AssistantTopic
} from '#lib/assistant.js';
import { AssistantDataError, assistantContext } from '#lib/server/ai/assistantContext.js';
import { applyExplanationOrder, groqAssistant } from '#lib/server/ai/assistantProvider.js';
import { conversationAnswer, type Conversation } from '#lib/server/ai/assistantConversation.js';
import { apiUrl, groqApiKey } from '#lib/server/config.js';

const uuid = /^[0-9a-f]{8}-(?:[0-9a-f]{4}-){3}[0-9a-f]{12}$/i;
const requests = new Map<string, number>();
let rateWindow = 0;

function allowRequest(userId: string) {
	const minute = Math.floor(Date.now() / 60_000);
	if (rateWindow !== minute) {
		rateWindow = minute;
		requests.clear();
	}
	const count = requests.get(userId) ?? 0;
	if (count >= 12 || (count === 0 && requests.size >= 4096)) return false;
	requests.set(userId, count + 1);
	return true;
}

async function parseRequest(request: Request): Promise<AssistantRequest | Conversation | null> {
	const reader = request.body?.getReader();
	if (!reader) return null;
	const decoder = new TextDecoder('utf-8', { fatal: true });
	let text = '';
	let size = 0;
	try {
		while (true) {
			const chunk = await reader.read();
			if (chunk.done) break;
			size += chunk.value.length;
			if (size > 8192) {
				void reader.cancel().catch(() => {});
				return null;
			}
			text += decoder.decode(chunk.value, { stream: true });
		}
		const parsed: unknown = JSON.parse(text + decoder.decode());
		if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) return null;
		const value = parsed as Record<string, unknown>;
		if (typeof value.message === 'string') {
			const message = value.message.trim();
			if (!message || message.length > 1000) return null;
			return {
				message,
				previous: typeof value.previous === 'string' ? value.previous.slice(0, 1000) : undefined,
				goalId:
					typeof value.goalId === 'string' && uuid.test(value.goalId) ? value.goalId : undefined,
				productId:
					typeof value.productId === 'string' && uuid.test(value.productId)
						? value.productId
						: undefined
			};
		}
		if (
			!value ||
			typeof value !== 'object' ||
			Array.isArray(value) ||
			!('topic' in value) ||
			typeof value.topic !== 'string' ||
			!Object.hasOwn(assistantQuestions, value.topic)
		)
			return null;
		const topic = value.topic as AssistantTopic;
		const key =
			topic === 'price' ? 'productId' : topic === 'goal' || topic === 'decision' ? 'goalId' : null;
		if (Object.keys(value).some((field) => field !== 'topic' && field !== key)) return null;
		if (!key) return { topic };
		const id = (value as Record<string, unknown>)[key];
		return typeof id === 'string' && uuid.test(id) ? { topic, [key]: id.toLowerCase() } : null;
	} catch {
		return null;
	}
}

export const POST: RequestHandler = async ({ locals, request, fetch }) => {
	if (!locals.userId) return json({ error: 'Sign in to continue.' }, { status: 401 });
	if (!allowRequest(locals.userId)) {
		return json(
			{ error: 'Too many requests. Try again in a minute.' },
			{ status: 429, headers: { 'Retry-After': '60' } }
		);
	}
	const question = await parseRequest(request);
	if (!question)
		return json(
			{ error: 'Enter a question of up to 1,000 characters or choose a supported topic.' },
			{ status: 400 }
		);
	const started = Date.now();
	const audit = (outcome: string, factCount = 0) =>
		console.info('buywise.assistant', {
			topic: 'message' in question ? 'conversation' : question.topic,
			outcome,
			factCount,
			durationMs: Date.now() - started
		});
	try {
		if ('message' in question) {
			const answer = await conversationAnswer(question, fetch, apiUrl(), request.signal);
			audit(answer.mode, answer.facts.length);
			return json(answer);
		}
		const { points, complete, ...facts } = await assistantContext(question, fetch, apiUrl());
		const answer: AssistantAnswer = { ...facts, explanation: points, mode: 'buywise' };
		if (complete && points.length > 1) {
			if (!groqApiKey()) answer.fallback = 'not_configured';
			else {
				const result = await groqAssistant(
					{ topic: question.topic, points },
					AbortSignal.any([request.signal, AbortSignal.timeout(6000)])
				).catch(() => null);
				const ordered = applyExplanationOrder(result, points);
				if (ordered) {
					answer.explanation = ordered;
					answer.mode = 'ai_assisted';
				} else answer.fallback = 'temporarily_unavailable';
			}
		}
		audit(answer.fallback ?? answer.mode, answer.facts.length);
		return json(answer);
	} catch (err) {
		const status = err instanceof AssistantDataError ? err.status : 503;
		audit(status === 404 ? 'not_found' : 'source_unavailable');
		return json(
			{
				error:
					err instanceof AssistantDataError
						? err.message
						: 'BuyWise could not load the required records.'
			},
			{ status }
		);
	}
};
