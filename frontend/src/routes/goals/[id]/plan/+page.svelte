<script lang="ts">
	import '#lib/styles/workbench.css';
	import { enhance } from '$app/forms';
	import { resolve } from '$app/paths';
	import { formatRupees } from '#lib/money.js';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();
	const states: Record<string, string> = {
		on_track: 'On track',
		funded: 'Target earmarked',
		replan: 'Replanning needed',
		profile_required: 'Review your monthly money plan',
		target_required: 'Target price required',
		contribution_required: 'Approve a contribution',
		paused: 'Paused',
		cancelled: 'Cancelled',
		completed: 'Completed'
	};
	const canEdit = $derived(
		data.plan.goal.status === 'active' || data.plan.goal.status === 'paused'
	);
	const accounts = $derived(
		data.plan.profileReady ? data.plan.accounts.filter((account) => account.availablePaise > 0) : []
	);

	const displayDate = (date: string | null | undefined) =>
		date
			? new Intl.DateTimeFormat('en-IN', { dateStyle: 'medium', timeZone: 'Asia/Kolkata' }).format(
					new Date(date + 'T12:00:00Z')
				)
			: 'Set a contribution';
</script>

<svelte:head><title>{data.plan.goal.name} · Saving plan | BuyWise</title></svelte:head>
<div class="workbench">
	<header class="wb-heading">
		<div>
			<a class="wb-muted" href={resolve('/goals/[id]', { id: data.id })}>← Back to goal</a>
			<h1>{data.plan.goal.name}</h1>
			<p class="wb-muted">Your saved plan, contributions and purchase timeline.</p>
		</div>
		<span class="wb-badge accent"
			>{states[data.plan.projection.state] ?? data.plan.projection.state}</span
		>
	</header>
	{#if form?.error}<p class="wb-notice" role="alert">{form.error}</p>{/if}{#if form?.success}<p
			class="wb-notice"
			role="status"
		>
			{form.success}
		</p>{/if}
	{#if data.plan.backingAtRisk}<p class="wb-notice" role="alert">
			Some reserved savings are no longer backed by account balances. Review your accounts.
		</p>{/if}
	{#if data.setupError}<p class="wb-notice" role="alert">
			Your goal was created. {data.setupError} Complete the remaining setup below.
		</p>{/if}
	{#if data.plan.projection.overdue}<p class="wb-notice" role="status">
			A planned saving is overdue. Record the contribution or choose a new first saving date to
			update your timeline.
		</p>{/if}
	{#if !data.plan.profileReady}<p class="wb-notice">
			<a href={resolve('/accounts')}>Set your monthly income and spending</a> to calculate savings for
			your goals.
		</p>{/if}
	<section class="wb-card goal-progress">
		<div class="wb-metrics">
			<div class="wb-metric">
				<span>Purchase target</span><strong
					>{formatRupees(data.plan.goal.targetPricePaise ?? 0)}</strong
				>
			</div>
			<div class="wb-metric">
				<span>Reserved savings</span><strong>{formatRupees(data.plan.fundedPaise)}</strong>
			</div>
			<div class="wb-metric">
				<span>Still to save</span><strong
					>{formatRupees(data.plan.projection.remainingPaise)}</strong
				>
			</div>
		</div>
		{#if data.plan.goal.targetPricePaise}<progress
				aria-label="Reserved savings progress"
				value={Math.min(data.plan.fundedPaise, data.plan.goal.targetPricePaise)}
				max={data.plan.goal.targetPricePaise}
			></progress>{/if}
		<div class="wb-row">
			<span class="wb-muted">Estimated completion</span><strong
				>{displayDate(data.plan.projection.estimatedDate)}</strong
			>
		</div>
	</section>
	<div class="wb-columns">
		<div class="wb-stack">
			{#if canEdit}{#key data.id}
					<section class="wb-card">
						<div class="wb-card-head">
							<h2>Your contribution plan</h2>
							<span class="wb-badge">Saved to this goal</span>
						</div>
						<form
							method="POST"
							action="?/terms"
							use:enhance={() =>
								async ({ update }) =>
									update({ reset: false })}
						>
							<input type="hidden" name="revision" value={data.plan.terms.revision} />
							<div class="wb-fields">
								<label
									>Contribution (₹)<input
										name="approved"
										inputmode="decimal"
										value={data.plan.terms.approvedPaise / 100}
										required
									/></label
								><label
									>Frequency<select name="frequency" value={data.plan.terms.frequency}
										><option value="weekly">Weekly</option><option value="biweekly"
											>Every two weeks</option
										><option value="monthly">Monthly</option></select
									></label
								><label
									>First saving date<input
										name="firstSavingOn"
										type="date"
										min={data.today}
										value={data.firstSavingOn}
										required
									/></label
								>
							</div>
							<p class="wb-muted plan-hint">
								{formatRupees(data.plan.availableMonthlyPaise)} per month is left after your other goals.
								Saving a plan does not move money.
							</p>
							<div class="wb-actions">
								<button class="btn btn-secondary" name="intent" value="preview">Preview</button
								><button class="btn btn-primary" name="intent" value="approve">Save plan</button>
							</div>
						</form>
						{#if form?.preview}<div class="wb-notice" role="status">
								<strong>{displayDate(form.preview.estimatedDate)}</strong>
								<p class="wb-muted">
									Preview · {states[form.preview.state] ?? form.preview.state}
								</p>
							</div>{/if}
					</section>
				{/key}{/if}
			<section class="wb-card">
				<div class="wb-card-head">
					<h2>Add savings to this goal</h2>
					<span class="wb-badge">From your accounts</span>
				</div>
				{#if data.plan.goal.status === 'active' && data.plan.projection.remainingPaise > 0}
					{#if accounts.length}<form method="POST" action="?/fund" use:enhance>
							<input type="hidden" name="fundingKey" value={form?.fundingKey ?? data.fundingKey} />
							<label class="plan-hint"
								>Saving type<select name="kind"
									><option value="contribution">Planned contribution</option><option value="topup"
										>Existing savings or extra top-up</option
									></select
								></label
							>
							<div class="wb-fields">
								<label
									>Source account<select name="accountId" required
										><option value="">Choose an account</option
										>{#each accounts as account (account.id)}<option value={account.id}
												>{account.name} · {formatRupees(account.availablePaise)} available</option
											>{/each}</select
									></label
								><label>Amount (₹)<input name="amount" inputmode="decimal" required /></label>
							</div>
							<div class="wb-actions plan-hint">
								<button class="btn btn-primary">Reserve funds</button><span class="wb-muted"
									>This sets aside recorded cash for this goal.</span
								>
							</div>
						</form>{:else}<p class="wb-empty">
							No unallocated funds. <a href={resolve('/accounts')}>Review accounts →</a>
						</p>{/if}
				{:else}<p class="wb-muted">
						{data.plan.projection.remainingPaise === 0
							? 'Your target is fully reserved.'
							: 'Resume this goal to reserve funds.'}
					</p>{/if}
				{#if data.plan.goal.legacySavedPaise}<p class="wb-muted">
						Previously reported: {formatRupees(data.plan.goal.legacySavedPaise)}. Reserve account
						funds to include it here.
					</p>{/if}
				<details class="inline-help">
					<summary
						>Contribution history <span class="wb-badge">{data.plan.contributions.length}</span
						></summary
					>
					<ul class="wb-list">
						{#each data.plan.contributions as contribution (contribution.id)}<li class="wb-row">
								<div>
									<strong>{formatRupees(contribution.amountPaise)}</strong>
									<p class="wb-muted">
										{data.plan.accounts.find((a) => a.id === contribution.accountId)?.name ??
											'Account'}
									</p>
								</div>
								<span class="wb-muted">{displayDate(contribution.createdAt.slice(0, 10))}</span>
							</li>{:else}<li class="wb-muted">No savings reserved yet.</li>{/each}
					</ul>
				</details>
			</section>
			{#if canEdit}<details class="wb-fold">
					<summary>Edit goal details</summary>
					<div class="wb-fold-body">
						<form method="POST" action="?/edit" use:enhance>
							<div class="wb-fields">
								<label
									>Name<input
										name="name"
										maxlength="200"
										value={data.plan.goal.name}
										required
									/></label
								><label
									>Target (₹)<input
										name="target"
										inputmode="decimal"
										value={(data.plan.goal.targetPricePaise ?? 0) / 100}
										required
									/></label
								><label
									>Desired purchase date<input
										name="targetDate"
										type="date"
										value={data.plan.goal.targetDate ?? ''}
									/></label
								><label
									>Priority<select name="priority" value={data.plan.goal.priority}
										><option value="high">High</option><option value="medium">Medium</option><option
											value="low">Low</option
										></select
									></label
								>
							</div>
							<button class="btn btn-primary plan-hint">Save goal</button>
						</form>
					</div>
				</details>{/if}
		</div>
		<aside class="wb-stack wb-sticky">
			<section class="wb-card">
				<p class="wb-eyebrow">Your monthly savings</p>
				<p class="wb-amount">
					{data.plan.profileReady ? formatRupees(data.plan.capacityPaise) : 'Not calculated'}
				</p>
				<p class="wb-muted">Income minus fixed and flexible spending</p>
				<dl class="wb-lines">
					<div>
						<dt>Other active goals</dt>
						<dd>{formatRupees(data.plan.otherGoalsPaise)}</dd>
					</div>
					<div>
						<dt>Left for this goal each month</dt>
						<dd>
							{data.plan.profileReady
								? formatRupees(data.plan.availableMonthlyPaise)
								: 'Set your monthly plan'}
						</dd>
					</div>
					<div>
						<dt>Free savings available now</dt>
						<dd>
							{data.plan.profileReady
								? formatRupees(accounts.reduce((total, a) => total + a.availablePaise, 0))
								: 'Review My Money first'}
						</dd>
					</div>
					<div>
						<dt>Desired date</dt>
						<dd>
							{data.plan.goal.targetDate ? displayDate(data.plan.goal.targetDate) : 'Flexible'}
						</dd>
					</div>
				</dl>
				{#if data.plan.projection.overCapacity}<p class="wb-notice" role="alert">
						Your monthly savings cannot cover this contribution alongside your other goals. <a
							href={resolve('/accounts')}>Review your monthly plan</a
						> or reduce the contribution.
					</p>{/if}<a href={resolve('/simulator')}>Explore a different purchase →</a>
			</section>
			{#if data.plan.history.length}<details class="wb-fold">
					<summary>Saved plan history</summary>
					<div class="wb-fold-body">
						<ul class="wb-list">
							{#each data.plan.history as revision (revision.revision)}<li class="wb-row">
									<span class="wb-muted">Version {revision.revision} · {revision.frequency}</span
									><strong>{formatRupees(revision.approvedPaise)}</strong>
								</li>{/each}
						</ul>
					</div>
				</details>{/if}
			{#if canEdit}<details class="wb-fold">
					<summary>Manage goal</summary>
					<div class="wb-fold-body">
						<form method="POST" action="?/lifecycle" use:enhance>
							<button
								class="btn btn-secondary"
								name="status"
								value={data.plan.goal.status === 'paused' ? 'active' : 'paused'}
								>{data.plan.goal.status === 'paused' ? 'Resume goal' : 'Pause goal'}</button
							>
						</form>
						<form method="POST" action="?/lifecycle" use:enhance>
							<label class="wb-check"
								><input type="checkbox" name="confirm" value="yes" required />Release this goal's
								reserved funds</label
							><button class="btn btn-danger plan-hint" name="status" value="cancelled"
								>Cancel goal</button
							>
						</form>
					</div>
				</details>{/if}
		</aside>
	</div>
</div>

<style>
	.plan-hint {
		margin-top: 16px;
		margin-bottom: 16px;
	}
	.inline-help > summary {
		cursor: pointer;
		color: var(--foreground-secondary);
		font-size: 12px;
	}
</style>
