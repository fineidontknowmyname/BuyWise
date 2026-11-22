export type PurchaseInput = {
	date: string;
	seller: string;
	costPaise: number;
	reference: string;
	paymentAccountId: string;
	categoryId: string;
};

type AccountAmount = { accountId: string; accountName: string; amountPaise: number };

export type PurchaseSummary = PurchaseInput & {
	targetPaise: number;
	contributedPaise: number;
	backedPaise: number;
	shortfallPaise: number;
	surplusPaise: number;
	paymentAccountName: string;
	categoryName: string;
	funding: AccountAmount[];
	surplus: AccountAmount[];
};

export type PurchasePreview = { summary: PurchaseSummary; token: string; key: string };

type Option = { id: string; name: string; purpose?: string };

export type PurchaseLifecycle = {
	purchase: PurchaseSummary | null;
	remainingPaise: number;
	accounts: Option[];
	categories: Option[];
	goals: Option[];
	allocations: {
		recordedAt: string;
		lines: {
			accountName: string;
			goalName?: string;
			purpose: string;
			amountPaise: number;
			sources: AccountAmount[];
		}[];
	}[];
};
