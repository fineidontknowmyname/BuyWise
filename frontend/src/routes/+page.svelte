<script lang="ts">
	import { resolve } from '$app/paths';
	import { formatRupees } from '#lib/money.js';
	import FinancialChart from '#lib/components/FinancialChart.svelte';
	import ActivityList from '#lib/components/ActivityList.svelte';
	import Modal from '#lib/components/Modal.svelte';
	import ProductListingPreview from '#lib/components/ProductListingPreview.svelte';
	import { fillListingField, listingPrice, type ProductListing } from '#lib/productListing.js';
	import QuickExpenseModal from '#lib/components/QuickExpenseModal.svelte';
	import type { PageProps } from './$types';
	import {
		Wallet,
		Target,
		Plus,
		ArrowRight,
		Sparkles,
		TrendingUp,
		List,
		Clock
	} from '@lucide/svelte';

	let { data, form }: PageProps = $props();

	let showGoalModal = $state(false);
	let showQuickExpenseModal = $state(false);

	const greeting = $derived(
		data.hour < 12 ? 'Good morning' : data.hour < 17 ? 'Good afternoon' : 'Good evening'
	);
	const date = $derived(
		new Intl.DateTimeFormat('en-IN', {
			dateStyle: 'full',
			timeZone: 'Asia/Kolkata'
		}).format(new Date(data.today + 'T00:00:00Z'))
	);
	const capacity = $derived(
		data.foundation?.profile.revision ? data.foundation.capacity.safeSavingsPaise : null
	);
	const metrics = $derived([
		{
			label: 'Recorded income',
			detail: 'This month · entered records',
			icon: Wallet,
			amount: data.weeks ? data.monthlyIncomePaise : null
		},
		{
			label: 'Recorded expenses',
			detail: 'This month · entered records',
			icon: List,
			amount: data.weeks ? data.monthlyExpensePaise : null
		},
		{
			label: 'Monthly saving estimate',
			detail: 'Based on your financial setup',
			icon: TrendingUp,
			amount: capacity
		},
		{
			label: 'Recorded goal savings',
			detail: 'Active goals · backing checked separately',
			icon: Target,
			amount: data.goalsAvailable ? data.totalSavingsPaise : null
		}
	]);
	const verdicts = [
		{ state: 'SAVE_MORE', label: 'Save more' },
		{ state: 'BUY', label: 'Buy' },
		{ state: 'REASSESS', label: 'Reassess' },
		{ state: 'WAIT', label: 'Wait' }
	];
	const verdictCopy = {
		BUY: 'Your goal has enough backed savings and the price meets the buying criteria.',
		WAIT: 'Your funds are ready. Wait for a price that meets your buying criteria.',
		SAVE_MORE: 'The price meets the buying criteria. Build up your goal savings before purchasing.',
		REASSESS: 'Review your savings, price evidence and financial assumptions before buying.'
	};
	const amountLabel = (paise: number) => formatRupees(paise).replace(/\.00$/, '');

	// Budget calculations
	const budgetSpent = $derived(data.budgetReport?.totalSpentPaise ?? 0);
	const budgetLimit = $derived(data.budgetReport?.totalLimitPaise ?? 0);
	const budgetRemaining = $derived(data.budgetReport?.totalRemainingPaise ?? 0);

	const budgetUtilization = $derived(
		budgetLimit > 0 ? Math.min(100, Math.round((budgetSpent / budgetLimit) * 100)) : 0
	);

	// Goal Modal URL extraction bindings
	let newGoalName = $state('');
	let newGoalUrl = $state('');
	let isAnalyzingUrl = $state(false);
	let extractUrlError = $state('');
	let goalListing = $state<ProductListing | null>(null);
	let goalProductName = $state('');
	let goalVariant = $state('');
	let goalTarget = $state('');
	let variantConfirmed = $state(false);
	const currentListing = $derived(goalListing?.url === newGoalUrl ? goalListing : null);
	$effect(() => {
		void newGoalUrl;
		void goalVariant;
		void goalProductName;
		variantConfirmed = false;
	});
	$effect(() => {
		if (form?.error) {
			showGoalModal = true;
			newGoalName = form.name ?? '';
		}
	});

	async function analyzeGoalUrl() {
		if (!newGoalUrl.trim() || isAnalyzingUrl) return;
		const raw = newGoalUrl.trim();
		const previous = goalListing;
		isAnalyzingUrl = true;
		extractUrlError = '';
		try {
			const res = await fetch(resolve('/api/products/extract'), {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ url: newGoalUrl.trim() })
			});
			const d = await res.json();
			if (newGoalUrl.trim() !== raw || !showGoalModal) return;
			if (res.ok) {
				goalListing = d;
				goalProductName = fillListingField(goalProductName, previous?.name, d.name);
				goalVariant = fillListingField(goalVariant, previous?.variant, d.variant);
				newGoalUrl = d.url;
				if (d.name && !newGoalName) newGoalName = `Buy ${d.name}`.slice(0, 200);
				if (String(goalTarget ?? '').trim() === '') goalTarget = listingPrice(d);
			} else {
				extractUrlError = d.error || 'Failed to analyze URL.';
			}
		} catch (err: unknown) {
			extractUrlError = err instanceof Error ? err.message : 'Network error.';
		} finally {
			isAnalyzingUrl = false;
		}
	}
</script>

<svelte:head>
	<title>BuyWise - Overview Dashboard</title>
</svelte:head>

<div class="dashboard-page">
	<div class="overview-heading">
		<div>
			<h1>
				{greeting}{data.workspacePreferences?.displayName
					? `, ${data.workspacePreferences.displayName}`
					: ''}
			</h1>
			<p>{date}</p>
		</div>
		<div class="overview-actions">
			<button
				type="button"
				class="btn btn-secondary"
				onclick={() => (showQuickExpenseModal = true)}
			>
				<Sparkles size={15} /> Quick-add expense
			</button>
			<button type="button" class="btn btn-primary" onclick={() => (showGoalModal = true)}>
				<span class="icon-tile"><Plus size={16} /></span> New goal
			</button>
		</div>
	</div>

	{#if data.loadIncomplete}
		<p class="dashboard-alert" role="alert">Some dashboard figures are temporarily unavailable.</p>
	{/if}

	<section class="next-step" aria-labelledby="next-step-title">
		<div>
			<p class="eyebrow">Your next step</p>
			{#if !data.goalsAvailable}
				<h2 id="next-step-title">Reconnect with your purchase plans</h2>
				<p>Your goals could not be loaded. Open Purchase goals to try again.</p>
			{:else if !data.goals.length}
				<h2 id="next-step-title">What would you like to save for?</h2>
				<p>Start with a purchase goal. Add a target and build a saving plan at your own pace.</p>
			{:else if !data.foundation}
				<h2 id="next-step-title">Your financial setup is temporarily unavailable</h2>
				<p>You can still open your goals. Refresh your setup before relying on an estimate.</p>
			{:else if !data.foundation.profile.revision}
				<h2 id="next-step-title">Give your plan some financial context</h2>
				<p>Add income and planned outflows to estimate your monthly saving capacity.</p>
			{:else}
				<h2 id="next-step-title">
					Keep {data.milestone?.name ?? 'your next purchase'} moving forward
				</h2>
				<p>
					Review your contribution plan. Recorded savings and purchase readiness are checked
					separately.
				</p>
			{/if}
		</div>
		{#if !data.goalsAvailable}
			<a class="btn btn-accent" href={resolve('/goals')}>Open goals <ArrowRight size={16} /></a>
		{:else if !data.goals.length}
			<button type="button" class="btn btn-accent" onclick={() => (showGoalModal = true)}
				>Create a goal <Plus size={16} /></button
			>
		{:else if data.foundation && !data.foundation.profile.revision}
			<a class="btn btn-accent" href={resolve('/finances')}
				>Set up My Money <ArrowRight size={16} /></a
			>
		{:else if data.milestone}
			<a class="btn btn-accent" href={resolve('/goals/[id]/plan', { id: data.milestone.id })}
				>Review saving plan <ArrowRight size={16} /></a
			>
		{/if}
	</section>

	<dl class="summary-strip" aria-label="Financial overview">
		{#each metrics as metric (metric.label)}
			<div>
				<dt><span class="icon-tile"><metric.icon size={14} /></span>{metric.label}</dt>
				<dd class:unavailable={metric.amount === null}>
					{metric.amount === null ? 'Unavailable' : amountLabel(metric.amount)}
				</dd>
				<p class="metric-detail">{metric.detail}</p>
			</div>
		{/each}
	</dl>

	<div class="overview-grid">
		<section class="card">
			<div class="section-heading">
				<h2>Purchase goals</h2>
				<a href={resolve('/goals')}>View all</a>
			</div>
			{#if !data.goalsAvailable}
				<p class="muted">Purchase goals are temporarily unavailable.</p>
			{:else if !data.goals.length}
				<p class="muted">Create a goal to track your savings and see when you’re ready to buy.</p>
			{:else}
				<ul class="goal-list">
					{#each data.goals.slice(0, 4) as goal (goal.id)}
						{@const saved = goal.savedAmountPaise ?? 0}
						{@const target = goal.targetPricePaise ?? 0}
						{@const percent =
							target > 0 ? Math.max(0, Math.min(100, Math.round((saved / target) * 100))) : 0}
						<li>
							<a class="goal" href={resolve('/goals/[id]', { id: goal.id })}>
								<span class="goal-title"><strong>{goal.name}</strong><span>{percent}%</span></span>
								<span class="progress-track" aria-hidden="true"
									><span style:width={percent + '%'}></span></span
								>
								<span class="muted"
									>Recorded {amountLabel(saved)}{target ? ' of ' + amountLabel(target) : ''}</span
								>
							</a>
						</li>
					{/each}
				</ul>
			{/if}
		</section>

		<section class="card decision-card" class:has-decision={!!data.decision}>
			{#if data.milestone && data.decision}
				{@const decision = data.decision}
				<p class="decision-goal">
					<span class="icon-tile"><Sparkles size={15} /></span>{data.milestone.name}
				</p>
				<h2 class="verdict">{decision.state.replaceAll('_', ' ')}</h2>
				<p class="muted">{verdictCopy[decision.state]}</p>
				<dl class="readiness-axes">
					<div>
						<dt>Financial assessment</dt>
						<dd>{decision.finance.state.replaceAll('_', ' ').toLowerCase()}</dd>
					</div>
					<div>
						<dt>Price assessment</dt>
						<dd>{decision.price.state.replaceAll('_', ' ').toLowerCase()}</dd>
					</div>
				</dl>
				{#if data.workspacePreferences?.priceUpdates !== false && decision.checkoutPaise !== null}
					<p class="muted">
						Checkout {amountLabel(decision.checkoutPaise)}{#if decision.selectedOffer?.observedOn}
							· {decision.selectedOffer
								.observedOn}{/if}{#if decision.price.reasonCodes.includes('observation_stale')}
							· stale{/if}
					</p>
				{/if}
				<div class="decision-matrix" aria-label="Purchase readiness matrix">
					{#each verdicts as item (item.state)}
						<span
							class:current={item.state === decision.state}
							aria-current={item.state === decision.state ? 'true' : undefined}>{item.label}</span
						>
					{/each}
				</div>
				<a class="btn btn-accent" href={resolve('/goals/[id]', { id: data.milestone.id })}
					>View decision <ArrowRight size={15} /></a
				>
			{:else}
				<div class="section-heading"><h2>Purchase decision</h2></div>
				<p class="muted">
					{data.milestone
						? 'The current assessment is temporarily unavailable.'
						: 'Your next active goal will appear here.'}
				</p>
				{#if data.milestone}<a href={resolve('/goals/[id]', { id: data.milestone.id })}>Open goal</a
					>{/if}
			{/if}
		</section>
	</div>

	<div class="overview-grid analytics">
		<div class="chart"><FinancialChart weeks={data.weeks} /></div>
		<section class="card budget">
			<div class="section-heading">
				<h2>Budget utilization</h2>
				<a href={resolve('/budget')}>Manage</a>
			</div>
			{#if !data.budgetReport}
				<p class="muted">Budget totals are temporarily unavailable.</p>
			{:else if !data.budgetReport.lines.length}
				<p class="muted">No category budgets are set for this month.</p>
			{:else}
				<p>
					<strong class="budget-value">{amountLabel(budgetSpent)}</strong>
					<span class="muted">of {amountLabel(budgetLimit)} limit</span>
				</p>
				<div
					class="progress-track budget-progress"
					aria-label="Budget utilization: {budgetUtilization}%"
				>
					<span style:width={budgetUtilization + '%'}></span>
				</div>
				<p class="muted">{amountLabel(budgetRemaining)} remaining in budgeted categories</p>
			{/if}
		</section>
	</div>

	<div class="overview-grid">
		<section class="card">
			<div class="section-heading">
				<h2>Recent activity</h2>
				<a href={resolve('/activity')}>View all</a>
			</div>
			{#if data.activity}<ActivityList events={data.activity} />{:else}<p class="muted">
					Activity is temporarily unavailable.
				</p>{/if}
		</section>
		<section class="card position">
			<div class="section-heading">
				<h2>Financial position</h2>
				<a href={resolve('/finances')}>Financial setup</a>
			</div>
			<dl>
				<div>
					<dt>Liquid balance</dt>
					<dd>
						{data.accountsAvailable ? amountLabel(data.totalLiquidBalancePaise) : 'Unavailable'}
					</dd>
				</div>
				<div>
					<dt>Monthly committed outflows</dt>
					<dd>
						{data.foundation?.profile.revision
							? amountLabel(data.foundation.capacity.committedPaise)
							: 'Unavailable'}
					</dd>
				</div>
				<div>
					<dt>Emergency reserve</dt>
					<dd>
						{data.foundation
							? amountLabel(data.foundation.emergencySavingsPaise)
							: 'Unavailable'}{#if data.foundation?.profile.revision}
							/ {amountLabel(data.foundation.profile.emergencyTargetPaise)}{/if}
					</dd>
				</div>
			</dl>
		</section>
	</div>

	{#if data.health.api !== 'up' || data.health.database !== 'up'}
		<p class="connection-note" role="status">
			Some BuyWise services are unavailable. Check the notices above before using your figures.
		</p>
	{/if}
</div>

<Modal isOpen={showGoalModal} title="New Purchase Goal" onclose={() => (showGoalModal = false)}>
	<form method="POST" action="?/createGoal" class="modal-form">
		<div class="form-group">
			<label for="create-goal-name" class="label">Goal Name *</label>
			<input
				id="create-goal-name"
				name="name"
				required
				maxlength="200"
				placeholder="e.g. Ergonomic Office Chair, MacBook Pro"
				class="input"
				bind:value={newGoalName}
			/>
		</div>

		<div class="form-group">
			<label for="create-goal-url" class="label">Product URL (Optional)</label>
			<div class="url-input-row">
				<input
					id="create-goal-url"
					name="productUrl"
					type="url"
					placeholder="https://www.amazon.in/dp/... or store URL"
					class="input"
					bind:value={newGoalUrl}
				/>
				<button
					type="button"
					class="btn btn-secondary btn-sm"
					disabled={!newGoalUrl.trim() || isAnalyzingUrl}
					onclick={analyzeGoalUrl}
				>
					{#if isAnalyzingUrl}
						<Clock size={14} class="animate-spin" />
						<span>Analyzing...</span>
					{:else}
						<Sparkles size={14} />
						<span>Fetch details</span>
					{/if}
				</button>
			</div>
			{#if extractUrlError}
				<span class="text-xs text-danger mt-1">{extractUrlError}</span>
			{/if}
		</div>

		{#if currentListing}<ProductListingPreview listing={currentListing} />{/if}
		<input type="hidden" name="imageUrl" value={currentListing?.imageUrl ?? ''} />
		<input type="hidden" name="brand" value={currentListing?.brand ?? ''} />
		{#if newGoalUrl}
			<label class="form-group"
				>Product name<input
					class="input"
					name="productName"
					maxlength="200"
					bind:value={goalProductName}
				/></label
			>
			<label class="form-group"
				>Exact variant<input
					class="input"
					name="variant"
					maxlength="100"
					bind:value={goalVariant}
					required
				/></label
			>
			<label
				><input
					type="checkbox"
					name="variantConfirmed"
					value="yes"
					bind:checked={variantConfirmed}
					required
				/> The listing matches this product and variant.</label
			>
		{/if}
		<label class="form-group"
			>Target price (INR)<input
				class="input"
				type="number"
				step="0.01"
				min="0"
				name="targetPrice"
				bind:value={goalTarget}
			/></label
		>

		{#if form?.error}
			<p class="form-error" role="alert">{form.error}</p>
		{/if}

		<div class="modal-footer">
			<button type="button" class="btn btn-secondary" onclick={() => (showGoalModal = false)}>
				Cancel
			</button>
			<button type="submit" class="btn btn-primary" disabled={isAnalyzingUrl}> Save Goal </button>
		</div>
	</form>
</Modal>

<!-- Quick Expense Modal -->
<QuickExpenseModal
	isOpen={showQuickExpenseModal}
	accounts={data.accounts}
	categories={data.categories}
	onclose={() => (showQuickExpenseModal = false)}
/>

<style>
	.dashboard-page {
		display: grid;
		gap: 20px;
	}
	.next-step {
		display: flex;
		align-items: center;
		justify-content: space-between;
		flex-wrap: wrap;
		gap: var(--space-6);
		padding: var(--space-6);
		border: 1px solid var(--border);
		border-radius: var(--radius-card);
		background: var(--surface);
		box-shadow: inset 4px 0 var(--brand-dot);
	}
	.next-step > div {
		flex: 1 1 280px;
		min-width: 0;
	}
	.next-step h2 {
		margin-block: var(--space-1) var(--space-2);
		font-size: clamp(1.25rem, 2vw, 1.6rem);
		overflow-wrap: anywhere;
	}
	.next-step p:not(.eyebrow),
	.metric-detail,
	.connection-note {
		color: var(--foreground-secondary);
	}
	.eyebrow {
		color: var(--foreground-muted);
		font-size: var(--text-small);
		font-weight: 600;
	}
	.metric-detail {
		margin-top: var(--space-2);
		font-size: var(--text-caption);
		line-height: 1.5;
	}
	.readiness-axes {
		display: grid;
		grid-template-columns: repeat(2, minmax(0, 1fr));
		gap: var(--space-4);
	}
	.readiness-axes dt {
		font-size: var(--text-caption);
	}
	.readiness-axes dd {
		font-weight: 600;
		text-transform: capitalize;
	}
	.overview-heading,
	.overview-actions,
	.section-heading,
	.decision-goal,
	.summary-strip dt,
	.goal-title {
		display: flex;
		align-items: center;
		gap: 10px;
	}
	.overview-heading {
		justify-content: space-between;
		flex-wrap: wrap;
		gap: 16px;
		margin-bottom: 2px;
	}
	.overview-heading h1 {
		font-size: clamp(1.75rem, 3.2vw, 2.5rem);
		font-weight: 540;
		letter-spacing: -0.045em;
		line-height: 1.12;
		overflow-wrap: anywhere;
	}
	.overview-heading p,
	.muted {
		color: var(--foreground-secondary);
	}
	.overview-heading p {
		margin-top: 4px;
	}
	.overview-actions {
		flex-wrap: wrap;
	}
	.overview-actions .btn {
		min-height: 44px;
	}
	.overview-actions .btn-primary {
		padding: 6px 17px 6px 7px;
	}
	.icon-tile {
		display: grid;
		place-items: center;
		width: 30px;
		height: 30px;
		flex: none;
		border-radius: 9px;
		border: 1px solid #353535;
		background: #0d0d0d;
		color: var(--accent);
	}
	.dashboard-alert {
		padding: 12px 16px;
		border-radius: 12px;
		background: var(--warning-surface);
		color: var(--warning-foreground);
	}
	.summary-strip {
		display: grid;
		grid-template-columns: repeat(4, minmax(0, 1fr));
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: 22px;
		overflow: hidden;
	}
	.summary-strip > div {
		padding: 18px clamp(12px, 1.5vw, 20px);
	}
	.summary-strip > div + div {
		border-inline-start: 1px solid var(--border);
	}
	.summary-strip dt {
		color: var(--foreground-secondary);
		font-size: 0.86rem;
	}
	.summary-strip dd {
		font: 550 clamp(1.15rem, 2.4vw, 2rem) / 1.12 var(--font-mono);
		letter-spacing: -0.04em;
		margin-top: 12px;
		overflow-wrap: anywhere;
	}
	.summary-strip .unavailable {
		font-size: 1rem;
		color: var(--foreground-muted);
	}
	.overview-grid {
		display: grid;
		grid-template-columns: minmax(0, 1.35fr) minmax(0, 1fr);
		align-items: start;
		gap: 16px;
	}
	.overview-grid > * {
		min-width: 0;
	}
	.section-heading {
		justify-content: space-between;
		padding-bottom: 13px;
		border-bottom: 1px solid var(--border);
		margin-bottom: 14px;
	}
	.section-heading h2 {
		font-size: 1.06rem;
		font-weight: 650;
	}
	.section-heading a {
		color: var(--foreground-secondary);
		text-decoration: underline;
		text-underline-offset: 3px;
	}
	.goal-list {
		list-style: none;
	}
	.goal-list li + li {
		border-top: 1px solid var(--border);
	}
	.goal {
		display: grid;
		gap: 9px;
		padding: 13px 0;
		color: var(--foreground);
	}
	.goal-list li:first-child .goal {
		padding-top: 2px;
	}
	.goal-list li:last-child .goal {
		padding-bottom: 0;
	}
	.goal-title {
		justify-content: space-between;
	}
	.goal-title strong {
		overflow-wrap: anywhere;
	}
	.goal:hover strong {
		text-decoration: underline;
		text-underline-offset: 3px;
	}
	.goal-title span {
		font: 600 1rem var(--font-mono);
	}
	.goal .muted {
		font-size: 0.87rem;
	}
	.progress-track {
		display: block;
		height: 8px;
		border-radius: 999px;
		overflow: hidden;
		background: var(--surface-secondary);
	}
	.progress-track > span {
		display: block;
		height: 100%;
		border-radius: inherit;
		background: var(--primary);
		box-shadow: inset -3px 0 var(--accent);
		transition: width 0.45s ease;
	}
	.has-decision {
		display: grid;
		gap: 13px;
		background: var(--primary);
		color: var(--primary-foreground);
		border-color: var(--primary);
	}
	.has-decision .muted,
	.decision-goal {
		color: color-mix(in srgb, var(--primary-foreground) 72%, var(--primary));
	}
	.decision-goal {
		overflow-wrap: anywhere;
	}
	.verdict {
		font: 550 clamp(2rem, 4vw, 3.4rem) / 1.04 var(--font-mono);
		letter-spacing: -0.045em;
	}
	.decision-card .btn {
		justify-self: start;
		min-height: 38px;
	}
	.decision-matrix {
		display: grid;
		grid-template-columns: repeat(2, minmax(0, 1fr));
		border: 1px solid color-mix(in srgb, currentColor 24%, transparent);
		border-radius: 16px;
		overflow: hidden;
	}
	.decision-matrix span {
		display: flex;
		align-items: center;
		gap: 7px;
		min-height: 43px;
		padding: 8px 10px;
		font: 500 0.82rem var(--font-mono);
		text-transform: uppercase;
		color: color-mix(in srgb, currentColor 74%, var(--primary));
	}
	.decision-matrix span:nth-child(odd) {
		border-inline-end: 1px solid color-mix(in srgb, currentColor 22%, transparent);
	}
	.decision-matrix span:nth-child(-n + 2) {
		border-bottom: 1px solid color-mix(in srgb, currentColor 22%, transparent);
	}
	.decision-matrix .current {
		color: var(--primary-foreground);
		background: color-mix(in srgb, currentColor 10%, transparent);
	}
	.decision-matrix .current::before {
		content: '';
		width: 8px;
		height: 8px;
		flex: none;
		border-radius: 50%;
		background: var(--accent);
		box-shadow: 0 0 0 1px #817500;
	}
	.budget-value {
		font: 550 1.65rem var(--font-mono);
	}
	.budget-progress {
		margin-block: 12px;
	}
	.position dl > div {
		display: flex;
		align-items: baseline;
		justify-content: space-between;
		gap: 10px;
		padding: 10px 0;
	}
	.position dl > div + div {
		border-top: 1px solid var(--border);
	}
	.position dt {
		color: var(--foreground-secondary);
	}
	.position dd {
		font-family: var(--font-mono);
		text-align: end;
		overflow-wrap: anywhere;
	}
	.url-input-row {
		display: flex;
		gap: 8px;
	}
	.url-input-row input {
		min-width: 0;
	}

	@media (max-width: 850px) {
		.overview-grid {
			grid-template-columns: minmax(0, 1fr);
		}
	}
	@media (max-width: 700px) {
		.summary-strip {
			grid-template-columns: repeat(2, minmax(0, 1fr));
		}
		.summary-strip > div:nth-child(3) {
			border-inline-start: 0;
		}
		.summary-strip > div:nth-child(n + 3) {
			border-top: 1px solid var(--border);
		}
	}
</style>
