export type MoneyCategory = {
	categoryId: string;
	name: string;
	fixed: boolean;
	archived: boolean;
	monthlyPaise: number;
	allowancePaise: number;
	spentPaise: number;
	remainingPaise: number;
};

export type MoneyPlan = {
	debtStatus: string;
	emergencyTargetPaise: number;
	reviewRequired: boolean;
	plannedGoalsPaise: number;
	unassignedMonthlyPaise: number;
	revision: number;
	configured: boolean;
	period: string;
	currentPeriod: string;
	incomePaise: number;
	fixedPaise: number;
	flexiblePaise: number;
	monthlySavingsPaise: number;
	monthlyShortfallPaise: number;
	protectedPaise: number;
	reservedPaise: number;
	savingsNowPaise: number;
	cashShortfallPaise: number;
	balancePaise: number;
	emergencyPaise: number;
	categories: MoneyCategory[];
};
