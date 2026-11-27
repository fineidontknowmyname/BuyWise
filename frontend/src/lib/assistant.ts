import { resolve } from '$app/paths';

export const assistantQuestions = {
	capacity: 'How much can I set aside each month?',
	goal: 'How is my goal progressing?',
	decision: 'Why did BuyWise make this purchase decision?',
	price: 'What does the recorded price history show?',
	budget: 'How is my monthly budget doing?'
} as const;

export type AssistantTopic = keyof typeof assistantQuestions;
export type AssistantRequest = { topic: AssistantTopic; goalId?: string; productId?: string };
export type AssistantSource =
	| { kind: 'profile' | 'budget' }
	| { kind: 'goal' | 'plan' | 'decision' | 'product' | 'offers'; id: string };

export type AssistantAnswer = {
	title: string;
	facts: { label: string; value: string }[];
	sources: AssistantSource[];
	explanation: string[];
	mode: 'ai_assisted' | 'buywise';
	fallback?: 'not_configured' | 'temporarily_unavailable';
};
export async function askAssistant(
	input:
		AssistantRequest | { message: string; previous?: string; goalId?: string; productId?: string },
	signal?: AbortSignal
): Promise<AssistantAnswer> {
	const response = await fetch(resolve('/api/ai/chat'), {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify(input),
		signal
	});
	const data = await response.json().catch(() => null);
	if (!response.ok || !data)
		throw new Error(data?.error ?? data?.message ?? 'The assistant is temporarily unavailable.');
	return data;
}
