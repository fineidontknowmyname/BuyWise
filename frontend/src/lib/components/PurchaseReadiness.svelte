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
		REASSESS: 'Reassess',
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
				<dd>{money(f.operatingBufferPaise)}</dd>
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
		{@render readiness(decision)}
		<nav aria-label="Update decision inputs">
			<a href={resolve('/finances')}>Review financial profile</a>
			{#if decision.evidence.prices.productId}
				<a href={resolve('/products/[id]/offers', { id: decision.evidence.prices.productId })}>
					Review retailer offers
				</a>
			{/if}
		</nav>
	{/if}
	<details id="decision-history" open={historyLink}>
		<summary>Saved decision history</summary>
		{#if history === null}
			<p>Decision history is temporarily unavailable.</p>
		{:else if history.length === 0}
			<p>
				{historyOnly
					? 'No saved purchase assessments.'
					: 'Choose Reassess and save to record your first assessment.'}
			</p>
		{:else}
			{#each history as previous (previous.id)}
				<details class="history-item" id={'decision-' + previous.id}>
					<summary
						>{labels[previous.state]} · {stamp(previous.decidedAt)} · {previous.rulesVersion}</summary
					>
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
	nav {
		display: flex;
		flex-wrap: wrap;
		gap: 16px;
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
