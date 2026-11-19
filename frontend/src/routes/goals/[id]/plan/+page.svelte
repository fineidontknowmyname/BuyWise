<script lang="ts">
	import { enhance } from '$app/forms';
	import { resolve } from '$app/paths';
	import { formatRupees } from '#lib/money.js';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();
	const states: Record<string, string> = {
		on_track: 'On track',
		funded: 'Target earmarked',
		replan: 'Replanning needed',
		profile_required: 'Financial profile required',
		target_required: 'Target price required',
		contribution_required: 'Approve a contribution',
		paused: 'Paused',
		cancelled: 'Cancelled',
		completed: 'Completed'
	};
	const canEdit = $derived(
		data.plan.goal.status === 'active' || data.plan.goal.status === 'paused'
	);
	const suggested = $derived(data.plan.projection.recommendedPaise);
	const accounts = $derived(data.plan.accounts.filter((account) => account.availablePaise > 0));
</script>

<svelte:head><title>{data.plan.goal.name} · Reverse EMI | BuyWise</title></svelte:head>
<a href={resolve('/goals/[id]', { id: data.id })}>Back to goal</a>
<h1>{data.plan.goal.name}</h1>
<p>Reverse EMI plan · {states[data.plan.projection.state] ?? data.plan.projection.state}</p>
{#if form?.error}<p role="alert">{form.error}</p>{/if}
{#if form?.success}<p role="status">{form.success}</p>{/if}

<section>
	<h2>Progress and timeline</h2>
	{#if data.plan.goal.targetPricePaise}
		<progress
			aria-label="Earmarked goal progress"
			value={Math.min(data.plan.fundedPaise, data.plan.goal.targetPricePaise)}
			max={data.plan.goal.targetPricePaise}
		></progress>
	{/if}
	<p>
		{formatRupees(data.plan.fundedPaise)} recorded in contributions · {formatRupees(
			data.plan.projection.remainingPaise
		)} remaining
	</p>
	<p>
		Estimated completion: {data.plan.projection.estimatedDate ??
			'Unavailable'}{#if data.plan.goal.targetDate}
			· Target date: {data.plan.goal.targetDate}{/if}
	</p>
	{#if data.plan.backingAtRisk}<p role="alert">
			Some existing goal allocations are no longer fully backed by eligible account balances. Review
			the source accounts before relying on the progress estimate.
		</p>{/if}
	{#if data.plan.goal.legacySavedPaise}<p>
			Previous manual entry: {formatRupees(data.plan.goal.legacySavedPaise)}. Earmark funds from an
			account to include them in the plan.
		</p>{/if}
	{#if data.plan.projection.state === 'replan'}<p>
			Review the target date, contribution, and priority before approving an updated plan.
		</p>{/if}
	{#if !data.plan.profileReady}<p>
			<a href={resolve('/finances')}>Set up your financial profile</a> to calculate available capacity.
		</p>{/if}
</section>

{#if canEdit}
	<section>
		<h2>Goal details</h2>
		<form method="POST" action="?/edit" use:enhance>
			<label>Name<input name="name" maxlength="200" value={data.plan.goal.name} required /></label>
			<label
				>Target price (₹)<input
					name="target"
					inputmode="decimal"
					value={(data.plan.goal.targetPricePaise ?? 0) / 100}
					required
				/></label
			>
			<label
				>Target date<input
					type="date"
					name="targetDate"
					value={data.plan.goal.targetDate ?? ''}
				/></label
			>
			<label
				>Priority<select name="priority" value={data.plan.goal.priority}>
					<option value="high">High</option><option value="medium">Medium</option><option
						value="low">Low</option
					>
				</select></label
			>
			<button class="btn btn-primary">Save goal</button>
		</form>
	</section>

	<section>
		<h2>Contribution plan</h2>
		<p>
			Safe monthly capacity: {formatRupees(data.plan.capacityPaise)} · Assigned to this goal: {formatRupees(
				data.plan.projection.monthlyAllocatedPaise
			)} per month.
		</p>
		<p>
			Suggested amount for the saved {data.plan.terms.frequency} frequency: {formatRupees(
				suggested
			)} per contribution.
		</p>
		<p>
			Active goals draw from shared capacity in priority and target-date order. A simple plan shows
			your chosen pace. An intelligent plan uses your available capacity.
		</p>
		<form
			method="POST"
			action="?/terms"
			use:enhance={() =>
				async ({ update }) => {
					await update({ reset: false });
				}}
		>
			<input type="hidden" name="revision" value={data.plan.terms.revision} />
			<label
				>Mode<select name="mode" value={data.plan.terms.mode}>
					<option value="intelligent">Intelligent</option><option value="simple">Simple</option>
				</select></label
			>
			<label
				>Frequency<select name="frequency" value={data.plan.terms.frequency}>
					<option value="weekly">Weekly</option><option value="biweekly">Every two weeks</option
					><option value="monthly">Monthly</option>
				</select></label
			>
			<label
				>Amount per contribution (₹)<input
					name="approved"
					inputmode="decimal"
					value={data.plan.terms.approvedPaise / 100}
					required
				/></label
			>
			<button class="btn" name="intent" value="preview">Preview estimate</button>
			<button class="btn btn-primary" name="intent" value="approve">Approve plan</button>
		</form>
		{#if form?.preview}
			<p>
				Preview: {states[form.preview.state] ?? form.preview.state} · Estimated completion: {form
					.preview.estimatedDate ?? 'Unavailable'} · Suggested contribution: {formatRupees(
					form.preview.recommendedPaise
				)}.
			</p>
		{/if}
	</section>
{/if}

<section>
	<h2>Contributions</h2>
	{#if data.plan.goal.status === 'active' && data.plan.projection.remainingPaise > 0}
		{#if accounts.length}
			<form method="POST" action="?/fund" use:enhance>
				<input type="hidden" name="fundingKey" value={form?.fundingKey ?? data.fundingKey} />
				<label
					>Source account<select name="accountId" required>
						<option value="">Choose an account</option>
						{#each accounts as account (account.id)}
							<option value={account.id}
								>{account.name} · {formatRupees(account.availablePaise)} available</option
							>
						{/each}
					</select></label
				>
				<label>Amount (₹)<input name="amount" inputmode="decimal" required /></label>
				<button class="btn btn-primary">Earmark savings</button>
			</form>
		{:else}
			<p>No eligible unallocated funds. <a href={resolve('/finances')}>Review your accounts</a>.</p>
		{/if}
	{/if}
	{#if data.plan.contributions.length}
		<ul>
			{#each data.plan.contributions as contribution (contribution.id)}
				<li>
					{formatRupees(contribution.amountPaise)} · {data.plan.accounts.find(
						(account) => account.id === contribution.accountId
					)?.name ?? 'Account'} · {contribution.createdAt.slice(0, 10)}
				</li>
			{/each}
		</ul>
	{:else}<p>No contributions recorded.</p>{/if}
</section>

{#if data.plan.history.length}
	<section>
		<h2>Plan history</h2>
		<ul>
			{#each data.plan.history as revision (revision.revision)}
				<li>
					Revision {revision.revision} · {revision.mode} · {formatRupees(revision.approvedPaise)}
					{revision.frequency}
				</li>
			{/each}
		</ul>
	</section>
{/if}

{#if canEdit}
	<section>
		<h2>Goal status</h2>
		<form method="POST" action="?/lifecycle" use:enhance>
			<button
				class="btn"
				name="status"
				value={data.plan.goal.status === 'paused' ? 'active' : 'paused'}
				>{data.plan.goal.status === 'paused' ? 'Resume goal' : 'Pause goal'}</button
			>
		</form>
		<form class="cancel" method="POST" action="?/lifecycle" use:enhance>
			<label
				><input type="checkbox" name="confirm" value="yes" required /> Release this goal’s allocations</label
			>
			<button class="btn" name="status" value="cancelled">Cancel goal</button>
		</form>
	</section>
{/if}

<style>
	section {
		margin-block: 1.25rem;
		padding: clamp(1rem, 3vw, 1.5rem);
		border: 1px solid var(--border);
		border-radius: 14px;
		background: var(--surface);
	}
	h2 {
		margin-bottom: 0.75rem;
	}
	form {
		display: flex;
		flex-wrap: wrap;
		align-items: end;
		gap: 0.75rem;
	}
	label {
		display: grid;
		flex: 1 1 10rem;
		gap: 0.3rem;
	}
	input:not([type='checkbox'], [type='hidden']),
	select {
		width: 100%;
		padding: 0.55rem;
		color: var(--foreground);
		background: var(--background);
		border: 1px solid var(--border);
		border-radius: 8px;
	}
	progress {
		width: 100%;
		min-height: 1.1rem;
		accent-color: var(--primary);
	}
	ul {
		padding-inline-start: 1.25rem;
	}
	.cancel {
		margin-top: 0.8rem;
	}
	.cancel label {
		display: flex;
		align-items: center;
		gap: 0.5rem;
	}
	[role='alert'] {
		color: var(--danger-foreground);
	}
</style>
