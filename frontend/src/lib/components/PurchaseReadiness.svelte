<script lang="ts">
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	const historyLink = $derived(
		typeof window !== 'undefined' && page.url.hash === '#decision-history'
	);
	import { formatRupees } from '#lib/money.js';
	import type { Decision } from '#lib/server/api.js';

	let {
		decision,
		history,
		historyOnly = false
	}: { decision: Decision; history: Decision[] | null; historyOnly?: boolean } = $props();

	const labels = {
		BUY: 'Buy',
		WAIT: 'Wait',
		SAVE_MORE: 'Save more',
		REASSESS: 'Check incomplete',
		READY: 'Ready',
		NOT_READY: 'Not ready',
		UNKNOWN: 'Unconfirmed',
		ATTRACTIVE: 'Attractive',
		NOT_ATTRACTIVE: 'Not attractive'
	};
	const debtLabels: Record<string, string> = {
		current: 'Current, as declared',
		overdue: 'Overdue, as declared',
		none_declared: 'No debt recorded'
	};
	const money = (paise: number | null) => (paise === null ? 'Unknown' : formatRupees(paise));
	const dateTime = new Intl.DateTimeFormat('en-IN', {
		dateStyle: 'medium',
		timeStyle: 'short',
		timeZone: 'Asia/Kolkata'
	});
	const stamp = (value: string) => dateTime.format(new Date(value));
	const financeStatus = $derived(
		decision.finance.state === 'READY'
			? 'Ready'
			: decision.finance.state === 'UNKNOWN'
				? 'Needs information'
				: 'Needs attention'
	);
	const priceStatus = $derived(
		decision.price.state === 'ATTRACTIVE'
			? 'Meets price checks'
			: decision.price.state === 'UNKNOWN'
				? 'Needs information'
				: 'Wait for a better price'
	);
	const firstReason = (codes: string[]) =>
		decision.reasons.find((reason) => codes.includes(reason.code))?.message;
	const needsGoalFunds = $derived(
		decision.finance.state === 'NOT_READY' &&
			decision.finance.reasonCodes[0] === 'goal_funds_shortfall'
	);
</script>

{#snippet readiness(value: Decision)}
	{@const f = value.evidence.finances}
	{@const p = value.evidence.prices}
	{@const offer = value.selectedOffer}
	<div class="axes">
		<div class="axis">
			<h3>Financial readiness</h3>
			<strong>{labels[value.finance.state]}</strong>
			<ul>
				{#each value.reasons.filter( (reason) => value.finance.reasonCodes.includes(reason.code) ) as reason (reason.code)}
					<li>{reason.message}</li>
				{/each}
			</ul>
			<dl>
				<dt>Backed goal funds</dt>
				<dd>{money(value.backedGoalPaise)}</dd>
				<dt>Emergency reserve</dt>
				<dd>{money(value.emergencyBalancePaise)} / {money(f.emergencyTargetPaise)}</dd>
				<dt>Operating buffer</dt>
				<dd>{f.profileRevision ? money(f.operatingBufferPaise) : 'Not set'}</dd>
				<dt>Liquid cash after purchase and other goals</dt>
				<dd>{money(value.afterPurchasePaise)}</dd>
				<dt>Monthly margin before buffer</dt>
				<dd>{f.profileRevision ? money(f.monthlyMarginPaise) : 'Unknown'}</dd>
				<dt>Debt status</dt>
				<dd>{f.profileRevision ? (debtLabels[f.debtStatus] ?? 'Unconfirmed') : 'Unknown'}</dd>
			</dl>
		</div>
		<div class="axis">
			<h3>Price readiness</h3>
			<strong>{labels[value.price.state]}</strong>
			<ul>
				{#each value.reasons.filter( (reason) => value.price.reasonCodes.includes(reason.code) ) as reason (reason.code)}
					<li>{reason.message}</li>
				{/each}
			</ul>
			<dl>
				<dt>Confirmed checkout</dt>
				<dd>{money(value.checkoutPaise)}</dd>
				{#if p.variant}
					<dt>Tracked variant</dt>
					<dd>{p.variant}</dd>
				{/if}
				{#if offer?.observationId}
					<dt>Retailer and source</dt>
					<dd>
						{[offer.retailer, offer.seller, offer.sourceLabel || offer.source]
							.filter(Boolean)
							.join(' · ')}
					</dd>
					<dt>Observed</dt>
					<dd>{offer.observedAt ? stamp(offer.observedAt) : offer.observedOn}</dd>
					<dt>Item · tax · delivery · discount</dt>
					<dd>
						{[offer.itemPaise, offer.taxPaise, offer.deliveryPaise, offer.discountPaise]
							.map(money)
							.join(' · ')}
					</dd>
					<dt>Comparable history</dt>
					<dd>{offer.historyIds?.length ?? 0} days</dd>
				{/if}
				<dt>Acceptable price</dt>
				<dd>{money(p.goodPaise ?? p.idealPaise ?? (value.goalTargetPaise || null))}</dd>
				<dt>Maximum price</dt>
				<dd>{money(p.maximumPaise ?? (value.goalTargetPaise || null))}</dd>
			</dl>
		</div>
	</div>
	{#each value.reasons.filter((reason) => reason.code === 'goal_inactive') as reason (reason.code)}
		<p>{reason.message}</p>
	{/each}
	{#if value.nextSteps.length}
		<h3>What would change the recommendation</h3>
		<ul>
			{#each value.nextSteps as step (step)}
				<li>{step}</li>
			{/each}
		</ul>
	{/if}
	<details>
		<summary>Evidence assumptions</summary>
		<ul>
			{#each value.assumptions as assumption (assumption)}
				<li>{assumption}</li>
			{/each}
		</ul>
		{#if offer?.observationId}<p>Observation ID {offer.observationId}</p>{/if}
		{#if f.profileRevision}<p>Financial profile revision {f.profileRevision}</p>{/if}
	</details>
{/snippet}

<section aria-label="Purchase readiness">
	{#if !historyOnly}
		<div class="check-row">
			<div>
				<h3>Funds <span>{financeStatus}</span></h3>
				<p>
					{decision.finance.state === 'READY'
						? 'Goal funds cover this purchase and protect your reserves.'
						: firstReason(decision.finance.reasonCodes)}
				</p>
			</div>
			{#if decision.finance.state !== 'READY'}
				<a
					href={needsGoalFunds
						? resolve('/goals/[id]/plan', { id: decision.goalId })
						: resolve('/finances')}
				>
					{needsGoalFunds ? 'Add goal funds' : 'Review finances'}
				</a>
			{/if}
		</div>
		<div class="check-row">
			<div>
				<h3>Price <span>{priceStatus}</span></h3>
				<p>
					{decision.price.state === 'ATTRACTIVE'
						? `${money(decision.checkoutPaise)} confirmed checkout total`
						: firstReason(decision.price.reasonCodes)}
				</p>
			</div>
			{#if decision.evidence.prices.productId}
				<a href={resolve('/products/[id]/offers', { id: decision.evidence.prices.productId })}>
					Check offer
				</a>
			{:else}
				<a href={resolve('/products')}>Find product</a>
			{/if}
		</div>
		{#if decision.reasonCodes.includes('goal_inactive')}
			<p>This goal is inactive. Resume it before checking a purchase.</p>
		{/if}
		<details>
			<summary>How this was checked</summary>
			{@render readiness(decision)}
		</details>
	{/if}
	<details id="decision-history" open={historyLink}>
		<summary>Previous checks</summary>
		{#if history === null}
			<p>Decision history is temporarily unavailable.</p>
		{:else if history.length === 0}
			<p>
				{historyOnly ? 'No saved purchase assessments.' : 'No previous checks yet.'}
			</p>
		{:else}
			{#each history as previous (previous.id)}
				<details class="history-item" id={'decision-' + previous.id}>
					<summary>{labels[previous.state]} · {stamp(previous.decidedAt)}</summary>
					<p>{previous.explanation}</p>
					{@render readiness(previous)}
				</details>
			{/each}
		{/if}
	</details>
</section>

<style>
	section {
		display: grid;
		gap: 14px;
		margin-top: 18px;
		padding-top: 18px;
		border-top: 1px solid var(--border-subtle);
	}
	.axes {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(min(100%, 290px), 1fr));
		gap: 12px;
	}
	.axis {
		background: var(--surface-secondary);
		border-radius: var(--radius-md);
		padding: 14px;
		overflow-wrap: anywhere;
	}
	h3,
	ul {
		margin: 0;
	}
	.axis ul {
		margin-block: 8px;
	}
	dl {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 6px 12px;
		margin: 0;
	}
	dt,
	li {
		color: var(--foreground-secondary);
	}
	dd {
		margin: 0;
	}
	.check-row {
		display: flex;
		flex-wrap: wrap;
		justify-content: space-between;
		align-items: center;
		gap: 12px;
		padding-block: 12px;
		border-bottom: 1px solid var(--border-subtle);
	}
	.check-row > div {
		flex: 1 1 240px;
	}
	.check-row h3 {
		font-size: 0.9375rem;
		font-weight: 600;
	}
	.check-row span {
		margin-left: 10px;
		font-size: 0.8125rem;
		font-weight: 450;
		color: var(--foreground-secondary);
	}
	.check-row p {
		margin: 6px 0 0;
		font-size: 0.875rem;
		color: var(--foreground-secondary);
	}
	.check-row a {
		font-size: 0.875rem;
		font-weight: 550;
	}
	summary {
		cursor: pointer;
	}
	.history-item {
		margin: 10px 0;
		padding: 10px;
		border: 1px solid var(--border-subtle);
		border-radius: var(--radius-md);
	}
</style>
