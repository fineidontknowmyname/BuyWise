export type Provenance = 'stub' | 'fixture' | 'manual' | 'calculated';

export type Goal = {
	id: string;
	name: string;
	targetPricePaise?: number;
	savedAmountPaise?: number;
	targetDate?: string | null;
	priority?: 'high' | 'medium' | 'low';
	productId?: string | null;
	status?: 'active' | 'paused' | 'completed' | 'cancelled';
	createdAt: string;
};

export type Product = {
	id: string;
	name: string;
	brand: string;
	model: string;
	variant?: string;
	category: string;
	description?: string;
	currentPricePaise: number;
	originalPricePaise?: number;
	targetPricePaise: number;
	discountPercent?: number;
	currency?: string;
	availability?: string;
	specifications?: Record<string, string>;
	url: string;
	imageUrl: string;
	retailer: string;
	notes: string;
	lastRetrievedAt?: string | null;
	source?: string;
	verificationStatus?: string;
	warnings?: string[];
	createdAt: string;
	updatedAt: string;
};

export type PriceObservation = {
	id: string;
	productId: string;
	goalId?: string | null;
	pricePaise: number;
	currency?: string;
	priceType?: string;
	observedOn: string;
	retailer?: string;
	source: string;
	provenance: string;
	sourceRef?: string;
	notes?: string;
	importBatchId?: string | null;
	createdAt: string;
};

export type DocumentUpload = {
	id: string;
	goalId: string;
	productId?: string | null;
	filename: string;
	fileType: string;
	fileSize: number;
	purpose: string;
	status: 'pending' | 'previewed' | 'confirmed' | 'failed';
	parsedData: Record<string, unknown>;
	summary: Record<string, unknown>;
	createdAt: string;
	confirmedAt?: string | null;
};

export type ProductStats = {
	productId: string;
	currentPricePaise: number;
	lowestPricePaise: number;
	highestPricePaise: number;
	averagePricePaise: number;
	observationCount: number;
	firstObservedOn: string;
	lastObservedOn: string;
};

export type ImportBatch = {
	id: string;
	productId: string;
	filename: string;
	totalRows: number;
	importedRows: number;
	duplicateRows: number;
	errorRows: number;
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

export type Account = {
	id: string;
	name: string;
	openingBalancePaise: number;
	balancePaise: number;
	createdAt: string;
};

export type Category = {
	id: string;
	name: string;
	kind: 'income' | 'expense';
	committed: boolean;
	archived?: boolean;
};

export type Transaction = {
	id: string;
	accountId: string;
	categoryId: string;
	kind: 'income' | 'expense';
	amountPaise: number;
	occurredOn: string;
	note: string;
	voided: boolean;
	voidedAt: string | null;
	voidReason: string;
	createdAt: string;
};

export type BudgetLine = {
	categoryId: string;
	categoryName: string;
	limitPaise: number;
	spentPaise: number;
	remainingPaise: number;
};

export type BudgetReport = {
	month: string;
	lines: BudgetLine[];
	totalLimitPaise: number;
	totalSpentPaise: number;
	totalRemainingPaise: number;
	unbudgetedDiscretionaryPaise: number;
};

export type ApiError = { code: string; message: string };

export type ApiResult<T> = { ok: true; data: T } | { ok: false; status: number; error: ApiError };

const unreachable: ApiError = {
	code: 'api_unreachable',
	message: 'The BuyWise API is not reachable. Is it running?'
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
