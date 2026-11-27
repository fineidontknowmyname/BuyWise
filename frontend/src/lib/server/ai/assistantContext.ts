import type { AssistantAnswer, AssistantRequest } from '#lib/assistant.js';
import { currentMonth } from '#lib/dates.js';
import { formatRupees } from '#lib/money.js';
import {
	apiRequest,
	type BudgetReport,
	type Decision,
	type Goal,
	type Product,
	type ProductStats
} from '#lib/server/api.js';

type Context = Pick<AssistantAnswer, 'title' | 'facts' | 'sources'> & {
	points: string[];
	complete: boolean;
};

export class AssistantDataError extends Error {
	constructor(readonly status: 404 | 503) {
		super(
			status === 404 ? 'That record was not found.' : 'BuyWise could not load the required records.'
		);
	}
}

type Capacity = {
	profile: { revision: number };
	capacity: {
		fixedIncomePaise: number;
		countedVariablePaise: number;
		committedPaise: number;
		discretionaryPaise: number;
		bufferPaise: number;
		safeSavingsPaise: number;
	};
};
type Plan = {
	fundedPaise: number;
	profileReady: boolean;
	backingAtRisk: boolean;
	projection: {
		remainingPaise: number;
		monthlyAllocatedPaise: number;
		estimatedDate: string | null;
	};
};
type Offer = { id: string; retailer: string; seller: string };
type History = {
	offer: Offer;
	freshness: 'missing' | 'stale' | 'recent';
	stats: {
		count: number;
		minimumPaise: number | null;
		maximumPaise: number | null;
		averagePaise: number | null;
	};
	observations: { observedOn: string; itemPaise: number; landedPaise: number | null }[];
};

const money = (value: number | undefined | null) =>
	value == null ? 'Unavailable' : formatRupees(value);
const fact = (label: string, value: string) => ({ label, value });

export async function assistantContext(
	request: AssistantRequest,
	fetchFn: typeof fetch,
	base: string
): Promise<Context> {
	async function read<T>(path: string): Promise<T> {
		const result = await apiRequest<T>(fetchFn, base, path);
		if (!result.ok) throw new AssistantDataError(result.status === 404 ? 404 : 503);
		return result.data;
	}
	switch (request.topic) {
		case 'capacity': {
			const { profile, capacity } = await read<Capacity>('/financial-profile');
			const ready = profile.revision > 0;
			return {
				title: 'Monthly saving capacity',
				sources: [{ kind: 'profile' }],
				complete: ready,
				facts: ready
					? [
							fact('Safe monthly savings', money(capacity.safeSavingsPaise)),
							fact('Fixed monthly income', money(capacity.fixedIncomePaise)),
							fact('Counted variable income', money(capacity.countedVariablePaise)),
							fact('Committed outflows', money(capacity.committedPaise)),
							fact('Planned discretionary spending', money(capacity.discretionaryPaise)),
							fact('Operating buffer', money(capacity.bufferPaise))
						]
					: [],
				points: ready
					? [
							capacity.safeSavingsPaise > 0
								? 'This is the amount the saved profile leaves for monthly saving after planned outflows and the operating buffer.'
								: 'The saved profile leaves no room for monthly saving after planned outflows and the operating buffer.',
							'BuyWise counts only the portion of variable income configured in the financial profile.',
							'This is a monthly estimate based on the saved profile. Account balances are tracked separately.'
						]
					: [
							'Save a financial profile to see an estimate based on your income and planned outflows.'
						]
			};
		}
		case 'budget': {
			const budget = await read<BudgetReport>(`/budgets?month=${currentMonth()}`);
			const configured = budget.lines.length > 0;
			return {
				title: 'Monthly budget',
				sources: [{ kind: 'budget' }],
				complete: configured,
				facts: [
					fact('Month', budget.month),
					...(configured
						? [
								fact('Budgeted amount', money(budget.totalLimitPaise)),
								fact('Spent in budgeted categories', money(budget.totalSpentPaise)),
								fact('Budget remaining', money(budget.totalRemainingPaise))
							]
						: []),
					fact('Unbudgeted discretionary spending', money(budget.unbudgetedDiscretionaryPaise))
				],
				points: configured
					? [
							budget.totalRemainingPaise < 0
								? 'Spending exceeds the combined limits for budgeted categories.'
								: 'The remaining amount compares posted spending with the combined limits for budgeted categories.',
							'Some categories can exceed their individual limits even when the combined budget has room.',
							'Unbudgeted discretionary spending is reported separately.'
						]
					: ['There are no category limits for this month.']
			};
		}
		case 'goal': {
			const id = request.goalId!;
			const [goal, plan] = await Promise.all([
				read<Goal>(`/goals/${id}`),
				read<Plan>(`/goals/${id}/plan`)
			]);
			const active = goal.status === 'active';
			return {
				title: goal.name,
				sources: [
					{ kind: 'goal', id },
					{ kind: 'plan', id }
				],
				complete: true,
				facts: [
					fact('Goal status', goal.status ?? 'Unavailable'),
					fact('Priority', goal.priority ?? 'Unset'),
					fact('Target date', goal.targetDate ?? 'Unset'),
					fact('Target', goal.targetPricePaise ? money(goal.targetPricePaise) : 'Unset'),
					fact('Total earmarked contributions', money(plan.fundedPaise)),
					...(active
						? [
								fact('Remaining toward target', money(plan.projection.remainingPaise)),
								fact(
									'Planned monthly allocation',
									plan.profileReady
										? money(plan.projection.monthlyAllocatedPaise)
										: 'Profile required'
								),
								fact('Estimated funding date', plan.projection.estimatedDate ?? 'Unavailable')
							]
						: [])
				],
				points: active
					? [
							plan.backingAtRisk
								? 'Some recorded earmarks may no longer be backed by available funds. Review the savings plan.'
								: 'BuyWise measures progress using contributions earmarked for this goal.',
							plan.profileReady
								? 'The funding date is an estimate based on the saved plan and monthly capacity.'
								: 'A saved financial profile is required to estimate a funding date.',
							'Purchase readiness also depends on financial protections and the available checkout price.'
						]
					: ['The goal is inactive. Its saved contribution history remains available.']
			};
		}
		case 'decision': {
			const id = request.goalId!;
			const [goal, decisions] = await Promise.all([
				read<Goal>(`/goals/${id}`),
				read<Decision[]>(`/goals/${id}/decision/history`)
			]);
			const latest = decisions[0];
			if (!latest)
				return {
					title: goal.name,
					facts: [],
					sources: [{ kind: 'decision', id }],
					complete: false,
					points: [
						'There is no saved assessment for this goal. Open the goal to review and save an assessment.'
					]
				};
			const labels = { BUY: 'Buy', WAIT: 'Wait', SAVE_MORE: 'Save more', REASSESS: 'Reassess' };
			return {
				title: latest.goalName,
				sources: [{ kind: 'decision', id }],
				complete: true,
				facts: [
					fact('Saved decision', labels[latest.state]),
					fact('Recorded at', latest.decidedAt),
					fact('Rules', latest.rulesVersion),
					fact('Saved assessment summary', latest.explanation),
					fact('Backed goal funds at assessment', money(latest.backedGoalPaise)),
					fact('Price used at assessment', money(latest.checkoutPaise)),
					...latest.reasons.map((reason) => fact('BuyWise decision evidence', reason.message))
				],
				points: [
					'Review the saved evidence to see which checks passed and which inputs need attention.',
					'This explanation uses the saved assessment. Current balances and prices may have changed.',
					'BuyWise evaluates financial readiness and price readiness separately. Review both on the goal page.'
				]
			};
		}
		case 'price': {
			const id = request.productId!;
			const { product, offers } = await read<{ product: Product; offers: Offer[] }>(
				`/products/${id}/offers`
			);
			if (offers.length === 0) {
				const stats = await read<ProductStats>(`/products/${id}/stats`);
				const hasPrices = stats.observationCount > 0;
				return {
					title: product.name,
					sources: [
						{ kind: 'product', id },
						{ kind: 'offers', id }
					],
					complete: hasPrices,
					facts: [
						fact('Product-level observations', String(stats.observationCount)),
						...(hasPrices
							? [
									fact('First observation date', stats.firstObservedOn || 'Unavailable'),
									fact('Last observation date', stats.lastObservedOn || 'Unavailable')
								]
							: [])
					],
					points: hasPrices
						? [
								'These older product-level observations do not verify a current retailer checkout price.',
								'Add a confirmed retailer offer for listing-specific price comparisons.'
							]
						: ['There are no recorded price observations for this product.']
				};
			}
			const histories = await Promise.all(
				offers
					.slice(0, 3)
					.map((offer) =>
						read<History>(`/products/${id}/offers/${encodeURIComponent(offer.id)}/history`)
					)
			);
			const facts = histories.flatMap(({ offer, freshness, stats, observations }) => {
				const label = offer.seller ? `${offer.retailer} (${offer.seller})` : offer.retailer;
				const latest = observations[0];
				return [
					fact(`${label} · observations`, String(stats.count)),
					...(latest
						? [
								fact(
									`${label} · last observed`,
									`${latest.observedOn}${freshness === 'stale' ? ' (stale)' : ''}`
								),
								fact(`${label} · last observed item price`, money(latest.itemPaise)),
								fact(`${label} · last observed checkout total`, money(latest.landedPaise)),
								fact(
									`${label} · historical item-price range`,
									`${money(stats.minimumPaise)} to ${money(stats.maximumPaise)}`
								)
							]
						: [])
				];
			});
			const observed = histories.some((entry) => entry.stats.count > 0);
			return {
				title: product.name,
				sources: [{ kind: 'offers', id }],
				complete: observed,
				facts: [
					...(offers.length > histories.length
						? [fact('Retailer offers shown', `${histories.length} of ${offers.length}`)]
						: []),
					...facts
				],
				points: observed
					? [
							'Historical ranges use observed item prices for each offer. Checkout totals require known taxes, delivery and discounts.',
							'Recorded offers may change. Check their latest date and availability before using them in a decision.',
							'Price history describes past observations. It does not predict future prices.'
						]
					: [
							'No price observations are recorded for the offers shown. Open the offer history to check or add observations.'
						]
			};
		}
	}
}
