export type Provenance = 'stub' | 'fixture' | 'manual' | 'calculated';

export type Goal = {
	id: string;
	name: string;
	createdAt: string;
};

export type DecisionInput = {
	key: string;
	label: string;
	value: string;
	provenance: Provenance;
};

export type Decision = {
	goalId: string;
	goalName: string;
	state: 'BUY' | 'WAIT' | 'SAVE_MORE' | 'REASSESS';
	reasonCodes: string[];
	inputs: DecisionInput[];
	rulesVersion: string;
	decidedAt: string;
	allInputsReal: boolean;
};

export type ApiError = { code: string; message: string };

export type ApiResult<T> = { ok: true; data: T } | { ok: false; status: number; error: ApiError };

const unreachable: ApiError = {
	code: 'api_unreachable',
	message: 'The Self-EMI API is not reachable. Is it running?'
};

export async function apiRequest<T>(
	fetchFn: typeof fetch,
	baseUrl: string,
	path: string,
	init?: RequestInit
): Promise<ApiResult<T>> {
	let response: Response;
	try {
		response = await fetchFn(`${baseUrl}${path}`, {
			...init,
			headers: { 'Content-Type': 'application/json', ...init?.headers },
			signal: AbortSignal.timeout(5000)
		});
	} catch {
		return { ok: false, status: 503, error: unreachable };
	}

	const body = await response.json().catch(() => undefined);
	if (response.ok) {
		return { ok: true, data: body as T };
	}
	const error = (body as { error?: ApiError } | undefined)?.error;
	return {
		ok: false,
		status: response.status,
		error: error ?? {
			code: 'unexpected_response',
			message: 'The API returned an unexpected response.'
		}
	};
}
