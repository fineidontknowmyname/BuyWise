import { json, type RequestHandler } from '@sveltejs/kit';
import { chatWithAI, type ChatMessage } from '#lib/server/ai/groqClient.js';

export const POST: RequestHandler = async ({ request, fetch }) => {
	try {
		const body = await request.json();
		const messages: ChatMessage[] = body.messages || [];

		if (!Array.isArray(messages) || messages.length === 0) {
			return json({ error: 'Messages array is required.' }, { status: 400 });
		}

		const response = await chatWithAI(messages, fetch);
		return json(response);
	} catch (err: unknown) {
		const message = err instanceof Error ? err.message : 'Unknown communication error.';
		return json(
			{
				text: `Unable to complete AI request: ${message}`,
				error: message
			},
			{ status: 200 }
		);
	}
};
