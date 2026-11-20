<script lang="ts">
	import { enhance } from '$app/forms';
	import { untrack } from 'svelte';
	import { resolve } from '$app/paths';
	import { formatRupees } from '#lib/money.js';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();
	const kinds = [
		['fixed_income', 'Fixed income'],
		['variable_income', 'Predictable variable income'],
		['irregular_income', 'One-off income'],
		['fixed_expense', 'Fixed expense'],
		['essential_expense', 'Essential variable expense'],
		['discretionary', 'Discretionary allowance'],
		['debt', 'Debt repayment or EMI']
	];
	const frequencies = [
		['weekly', 'Weekly'],
		['biweekly', 'Every two weeks'],
		['monthly', 'Monthly'],
		['annual', 'Yearly']
	];
	const purposes = [
		['everyday', 'Everyday'],
		['general_savings', 'General savings'],
		['emergency_reserve', 'Emergency reserve']
	];
	let rows = $state(untrack(readRows));
	let loadedRevision = $state(untrack(() => data.foundation?.profile.revision));
	function readRows() {
		return (data.foundation?.profile.items ?? []).map((item) => ({
			name: item.name,
			kind: item.kind,
			frequency: item.frequency,
			amount: String(item.amountPaise / 100)
		}));
	}
	$effect(() => {
		const profile = data.foundation?.profile;
		if (profile && profile.revision !== loadedRevision) {
			loadedRevision = profile.revision;
			rows = readRows();
		}
	});
</script>

<svelte:head><title>My finances | BuyWise</title></svelte:head>

<h1>My finances</h1>
<p>Enter your financial assumptions manually and record transfers between your accounts.</p>
{#if data.error || form?.error}<p role="alert">{form?.error ?? data.error}</p>{/if}
{#if form?.success}<p role="status">{form.success}</p>{/if}

{#if data.foundation}
	{@const foundation = data.foundation}
	{@const profile = foundation.profile}
	{@const estimates = foundation.capacity}
	<section>
		<h2>Safe monthly savings</h2>
		{#if profile.revision === 0}
			<p>Save your financial profile to calculate an estimate.</p>
		{:else}
			<p class="total">{formatRupees(estimates.safeSavingsPaise)}</p>
			<dl>
				<dt>Fixed income</dt>
				<dd>{formatRupees(estimates.fixedIncomePaise)}</dd>
				<dt>Variable income counted at {profile.variableIncomePercent}%</dt>
				<dd>{formatRupees(estimates.countedVariablePaise)}</dd>
				<dt>Committed expenses and debt</dt>
				<dd>{formatRupees(estimates.committedPaise)}</dd>
				<dt>Discretionary allowance</dt>
				<dd>{formatRupees(estimates.discretionaryPaise)}</dd>
				<dt>Operating safety buffer</dt>
				<dd>{formatRupees(estimates.bufferPaise)}</dd>
			</dl>
			<p>
				One-off income is excluded. Estimate {estimates.version}, profile revision {profile.revision}.
			</p>
		{/if}
	</section>

	<section>
		<h2>Financial profile</h2>
		<form
			method="POST"
			action="?/profile"
			use:enhance={() =>
				async ({ update }) => {
					await update({ reset: false });
				}}
		>
			<input type="hidden" name="revision" value={profile.revision} />
			{#each rows as row, index (row)}
				<fieldset class="form-grid">
					<legend>Entry {index + 1}</legend>
					<label>Name<input name="name" maxlength="200" required bind:value={row.name} /></label>
					<label
						>Type<select
							name="kind"
							bind:value={row.kind}
							onchange={() => {
								row.frequency = row.kind === 'irregular_income' ? 'once' : 'monthly';
							}}
						>
							{#each kinds as [value, label] (value)}<option {value}>{label}</option>{/each}
						</select></label
					>
					<label
						>Frequency<select name="frequency" bind:value={row.frequency}>
							{#if row.kind === 'irregular_income'}<option value="once">One-off</option>
							{:else}{#each frequencies as [value, label] (value)}<option {value}>{label}</option
									>{/each}{/if}
						</select></label
					>
					<label
						>Amount (₹)<input
							name="amount"
							inputmode="decimal"
							required
							bind:value={row.amount}
						/></label
					>
					<button
						type="button"
						class="btn"
						onclick={() => {
							rows.splice(index, 1);
						}}>Remove entry</button
					>
				</fieldset>
			{/each}
			<button
				type="button"
				class="btn"
				disabled={rows.length >= 100}
				onclick={() => {
					rows.push({ name: '', kind: 'fixed_income', frequency: 'monthly', amount: '' });
				}}>Add entry</button
			>
			{#key profile.revision}
				<div class="form-grid">
					<label>
						Debt repayment status
						<select name="debtStatus" value={profile.debtStatus}>
							<option value="">Unconfirmed or no debt recorded</option>
							<option value="current">All repayments current</option>
							<option value="overdue">Repayment overdue</option>
						</select>
					</label>
					<label
						>Variable income counted (%)<input
							name="variableIncomePercent"
							type="number"
							min="0"
							max="100"
							required
							value={profile.variableIncomePercent}
						/></label
					>
					<label
						>Operating buffer (%)<input
							name="bufferPercent"
							type="number"
							min="0"
							max="100"
							required
							value={profile.bufferPercent}
						/></label
					>
					<label
						>Minimum operating buffer (₹)<input
							name="bufferMinimum"
							inputmode="decimal"
							required
							value={profile.bufferMinimumPaise / 100}
						/></label
					>
					<label
						>Emergency reserve target (₹)<input
							name="emergencyTarget"
							inputmode="decimal"
							required
							value={profile.emergencyTargetPaise / 100}
						/></label
					>
				</div>
			{/key}
			<button class="btn btn-primary">Save profile</button>
		</form>
	</section>

	<section>
		<h2>Accounts and reserves</h2>
		<dl>
			<dt>Emergency reserve balance</dt>
			<dd>{formatRupees(foundation.emergencySavingsPaise)}</dd>
			<dt>Remaining to emergency target</dt>
			<dd>{formatRupees(foundation.emergencyGapPaise)}</dd>
			<dt>General savings balance</dt>
			<dd>{formatRupees(foundation.generalSavingsPaise)}</dd>
		</dl>
		<a href={resolve('/accounts')}>Manage accounts and opening balances</a>
		{#each foundation.accounts as account (account.id)}
			<form method="POST" action="?/purpose" class="form-grid" use:enhance>
				<input type="hidden" name="accountId" value={account.id} />
				<label
					>{account.name} · {formatRupees(account.balancePaise)}
					<select name="purpose" value={account.purpose}>
						{#each purposes as [value, label] (value)}<option {value}>{label}</option>{/each}
					</select>
				</label>
				<button class="btn">Save purpose</button>
			</form>
		{/each}
	</section>

	<section>
		<h2>Account transfers</h2>
		{#if foundation.accounts.length < 2}
			<p>Add at least two accounts to record a transfer.</p>
		{:else}
			<form method="POST" action="?/transfer" use:enhance>
				<input
					type="hidden"
					name="idempotencyKey"
					value={form?.idempotencyKey ?? data.idempotencyKey}
				/>
				<div class="form-grid">
					<label
						>From<select name="fromAccountId" required
							><option value="">Choose an account</option>
							{#each foundation.accounts as account (account.id)}<option value={account.id}
									>{account.name}</option
								>{/each}
						</select></label
					>
					<label
						>To<select name="toAccountId" required
							><option value="">Choose an account</option>
							{#each foundation.accounts as account (account.id)}<option value={account.id}
									>{account.name}</option
								>{/each}
						</select></label
					>
					<label>Amount (₹)<input name="amount" inputmode="decimal" required /></label>
					<label
						>Date<input
							type="date"
							name="occurredOn"
							required
							value={data.today}
							max={data.today}
						/></label
					>
					<label>Note<input name="note" maxlength="500" /></label>
				</div>
				<button class="btn btn-primary">Record transfer</button>
			</form>
		{/if}
		{#if data.transfers.length}
			<h3>Recent transfers</h3>
			<ul class="transfers">
				{#each data.transfers as entry (entry.id)}
					<li>
						<strong>{formatRupees(entry.amountPaise)}</strong>
						{foundation.accounts.find((a) => a.id === entry.fromAccountId)?.name ?? 'Account'}
						→ {foundation.accounts.find((a) => a.id === entry.toAccountId)?.name ?? 'Account'}
						· {entry.occurredOn}
						{#if entry.note}<span>{entry.note}</span>{/if}
						{#if entry.reversesId}<span>Reversal</span>
						{:else if entry.reversed}<span>Reversed</span>
						{:else if entry.purchaseManaged}<span>Purchase transfer</span>
						{:else}
							<form method="POST" action="?/reverse" use:enhance>
								<input type="hidden" name="transferId" value={entry.id} />
								<button class="btn">Reverse transfer</button>
							</form>
						{/if}
					</li>
				{/each}
			</ul>
		{/if}
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
	h2,
	h3 {
		margin-bottom: 0.75rem;
	}
	.form-grid {
		display: flex;
		align-items: end;
		flex-wrap: wrap;
		gap: 0.75rem;
		margin-block: 0.75rem;
	}
	fieldset {
		border: 1px solid var(--border);
		border-radius: 8px;
		padding: 0.75rem;
		min-inline-size: 0;
	}
	label {
		display: grid;
		gap: 0.3rem;
		flex: 1 1 10rem;
	}
	input:not([type='hidden']),
	select {
		color: var(--foreground);
		background: var(--background);
		border: 1px solid var(--border);
		padding: 0.55rem;
		border-radius: 8px;
		width: 100%;
	}
	dl {
		display: grid;
		grid-template-columns: minmax(0, 1fr) auto;
		gap: 0.5rem 1rem;
		max-width: 33rem;
	}
	dd {
		margin: 0;
		text-align: right;
	}
	.total {
		font-weight: 700;
		font-size: 1.75rem;
	}
	.transfers {
		display: grid;
		gap: 0.75rem;
		padding-inline-start: 1.25rem;
	}
	.transfers form {
		display: inline;
	}
	[role='alert'] {
		color: var(--danger-foreground);
	}
</style>
