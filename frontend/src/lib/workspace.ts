export type ActivityEvent = {
	key: string;
	kind: string;
	at: string;
	title: string;
	detail: string;
	paise: number | null;
	resource: string;
	resourceId: string;
};
export type ActivityPage = { events: ActivityEvent[]; next: string };
export type Preferences = { displayName: string; priceUpdates: boolean; reducedMotion: boolean };
export type Foundation = {
	profile: { revision: number; emergencyTargetPaise: number; moneyPlan?: { incomePaise: number } };
	capacity: { safeSavingsPaise: number; committedPaise: number };
	emergencySavingsPaise: number;
	generalSavingsPaise: number;
};
