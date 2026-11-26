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
	let savingProfile = $state(false);
	let useCategoryBudgets = $state(
		untrack(() =>
			data.foundation?.profile.revision ? data.foundation.profile.useCategoryBudgets : true
		)
	);
	function readRows() {
		return (data.foundation?.profile.items ?? []).map((item) => ({
			name: item.name,
			kind: item.kind,
			frequency: item.frequency,
			budgetCategory:
				data.foundation?.profile.budgetCategoryLinks?.[item.name] ??
				data.foundation?.connections.budgets.find(
					(b) => b.name.trim().toLowerCase() === item.name.trim().toLowerCase()
				)?.categoryId ??
				'',
			amount: String(item.amountPaise / 100)
		}));
	}
	$effect(() => {
		const profile = data.foundation?.profile;
		if (profile && profile.revision !== loadedRevision) {
			loadedRevision = profile.revision;
			rows = readRows();
			useCategoryBudgets = profile.useCategoryBudgets;
		}
	});
</script>

<svelte:head><title>Financial setup | BuyWise</title></svelte:head>

<div class="setup-heading">
	<p class="eyebrow">My Money</p>
	<h1>Financial setup</h1>
	<p>
		Use your existing accounts and budgets, then review income and spending assumptions for your
		plan.
	</p>
</div>
<div class="money-guide" aria-label="How My Money works">
	<div>
		<strong>Setup</strong>
		<p>What you expect to earn and spend.</p>
	</div>
	<div>
		<strong>Accounts & expenses</strong>
		<p>The balances and spending you record.</p>
	</div>
	<div>
		<strong>Budgets</strong>
		<p>How much you intend to spend by category.</p>
	</div>
</div>
{#if data.error || form?.error}<p role="alert">{form?.error ?? data.error}</p>{/if}
{#if form?.success}<p role="status">{form.success}</p>{/if}

{#if data.foundation}
	{@const foundation = data.foundation}
	{@const profile = foundation.profile}
	{@const estimates = foundation.capacity}
	<section class="capacity-card" aria-labelledby="capacity-title">
		<div class="section-header">
			<h2 id="capacity-title">Monthly saving estimate</h2>
			<span class="estimate-label">Planning estimate</span>
		</div>
		{#if profile.revision === 0}
			<p>Save your financial profile to calculate an estimate.</p>
		{:else}
			<p class="total">{formatRupees(estimates.safeSavingsPaise)}</p>
			<p class="estimate-note">
				Per month, before goal allocations. This is an estimate, not money available in your
				accounts.
			</p>
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
		<h2>Connected accounts and budgets</h2>
		<p class="section-description">
			Account balances update from your records. Monthly budgets can supply your spending plan
			without entering the same allowance again.
		</p>
		<div class="connected-grid">
			{#each foundation.accounts as account (account.id)}
				<div class="connected-card">
					<strong>{account.name}</strong>
					<p>{formatRupees(account.balancePaise)} recorded</p>
					<p>
						{account.purpose === 'emergency_reserve'
							? 'Protected emergency reserve'
							: foundation.reservationsAvailable
								? `${formatRupees(account.availablePaise)} after goal reservations`
								: 'Goal reservations unavailable'}
					</p>
					{#if account.reservedPaise}<p>
							{formatRupees(account.reservedPaise)} reserved for goals
						</p>{/if}
				</div>
			{:else}<p>Add an account to see recorded funds here.</p>{/each}
		</div>
		<p class="section-description">
			Unreserved balances are before your operating buffer and other obligations. They are not
			monthly income.
		</p>
		<a href={resolve('/accounts')}>Manage accounts</a>
		<h3>Category spending plan</h3>
		<dl>
			{#each foundation.connections.budgets as budget (budget.categoryId)}<dt>{budget.name}</dt>
				<dd>{formatRupees(budget.amountPaise)} / month</dd>{:else}<dt>No category budgets saved</dt>
				<dd>Not configured</dd>{/each}
		</dl>
		<p>
			{profile.useCategoryBudgets
				? foundation.connections.budgets.length
					? 'These budgets are included once in your saved monthly estimate.'
					: 'No category budgets exist; manual spending allowances still apply.'
				: 'Enable category budgets below and save setup to use this spending plan.'}
		</p>
		<a href={resolve('/budgets')}>Manage budgets and categories</a>
	</section>

	<section>
		<h2>Income & planned outflows</h2>
		<p class="section-description">
			Edit an amount below or add an entry. Saving updates your current assumptions; it does not
			record a payment or schedule a future change.
		</p>
		<form
			method="POST"
			action="?/profile"
			use:enhance={() => {
				savingProfile = true;
				return async ({ update }) => {
					try {
						await update({ reset: false });
					} finally {
						savingProfile = false;
					}
				};
			}}
			aria-busy={savingProfile}
		>
			<input type="hidden" name="revision" value={profile.revision} />
			<label class="budget-choice"
				><input
					type="checkbox"
					name="useCategoryBudgets"
					value="yes"
					bind:checked={useCategoryBudgets}
				/> Use category budgets for my monthly spending plan</label
			>
			{#if useCategoryBudgets}
				<p class="section-description">
					Category budgets replace discretionary allowance entries. For other expenses already
					covered by a budget, select that category below so it is counted once. Unlinked
					obligations remain additional costs. Deleting a linked budget restores that entry's manual
					amount.
				</p>
				{#if !foundation.connections.budgets.length}<p role="status">
						No category budgets exist yet. Manual allowances continue to apply until you add a
						budget.
					</p>{/if}
			{/if}
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
					{#if !row.kind.includes('income')}
						<label
							>Covered by category budget<select
								name="budgetCategory"
								bind:value={row.budgetCategory}
							>
								<option value="">Additional obligation / not covered</option>
								{#each foundation.connections.budgets as budget (budget.categoryId)}<option
										value={budget.categoryId}>{budget.name}</option
									>{/each}
								{#if row.budgetCategory && !foundation.connections.budgets.some((b) => b.categoryId === row.budgetCategory)}<option
										value={row.budgetCategory}>Budget removed: manual amount applies</option
									>{/if}
							</select></label
						>
					{:else}<input type="hidden" name="budgetCategory" value="" />{/if}
					{#if useCategoryBudgets && row.kind === 'discretionary'}<p>
							Replaced by category budgets while linked.
						</p>{/if}
					<button
						type="button"
						class="btn btn-secondary"
						aria-label={'Remove ' + (row.name || `entry ${index + 1}`)}
						onclick={() => {
							rows.splice(index, 1);
						}}>Remove entry</button
					>
				</fieldset>
			{/each}
			<button
				type="button"
				class="btn btn-secondary"
				disabled={rows.length >= 100}
				onclick={() => {
					rows.push({
						name: '',
						kind: 'fixed_income',
						frequency: 'monthly',
						amount: '',
						budgetCategory: ''
					});
				}}>Add entry</button
			>
			<h3>Suggestions from recorded transactions</h3>
			<p class="section-description">
				Three complete months: {foundation.connections.historyFrom} to {foundation.connections
					.historyThrough}. Amounts are monthly averages, including months with no entries. Income
				needs records in at least two months. Review one-off receipts before accepting; no
				suggestion changes your plan until you save.
			</p>
			<div class="connected-grid">
				{#each foundation.connections.suggestions as suggestion (suggestion.categoryId)}
					<div class="connected-card">
						<strong>{suggestion.name}</strong>
						<p>
							{suggestion.kind === 'variable_income'
								? 'Income to review'
								: 'Spending without a category budget'}: {formatRupees(suggestion.amountPaise)} / month
						</p>
						<p>Recorded in {suggestion.monthsObserved} of 3 months.</p>
						<button
							type="button"
							class="btn btn-secondary"
							disabled={rows.length >= 100 ||
								rows.some((row) => row.name.trim().toLowerCase() === suggestion.name.toLowerCase())}
							onclick={() =>
								rows.push({
									name: suggestion.name,
									kind: suggestion.kind,
									frequency: 'monthly',
									amount: String(suggestion.amountPaise / 100),
									budgetCategory: ''
								})}>Add to review</button
						>
					</div>
				{:else}<p>
						No suggestions yet. Record more history or add your expected income and obligations
						manually.
					</p>{/each}
			</div>
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
			<button class="btn btn-primary" disabled={savingProfile}
				>{savingProfile ? 'Saving setup…' : 'Save financial setup'}</button
			>
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
	.connected-grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(min(100%, 240px), 1fr));
		gap: 12px;
		margin-block: 12px;
	}
	.connected-card {
		padding: 14px;
		border: 1px solid var(--border);
		border-radius: 12px;
	}
	.connected-card p {
		margin-block: 8px;
		color: var(--foreground-secondary);
	}
	.budget-choice {
		display: flex;
		align-items: center;
		gap: 10px;
		margin-block: 16px;
	}
	.budget-choice input[type='checkbox'] {
		width: 18px;
		min-height: 18px;
	}
	.setup-heading {
		display: grid;
		gap: var(--space-2);
		margin-bottom: var(--space-6);
	}
	.setup-heading > p,
	.section-description,
	.estimate-note {
		color: var(--foreground-secondary);
	}
	.eyebrow {
		font-size: var(--text-small);
		font-weight: 600;
	}
	.money-guide {
		display: grid;
		grid-template-columns: repeat(3, minmax(0, 1fr));
		gap: var(--space-4);
	}
	.money-guide > div {
		padding: var(--space-4);
		border: 1px solid var(--border);
		border-radius: var(--radius-control);
	}
	.money-guide p {
		margin-top: var(--space-1);
		font-size: var(--text-small);
		color: var(--foreground-secondary);
	}
	.section-header {
		display: flex;
		align-items: baseline;
		justify-content: space-between;
		gap: var(--space-3);
		flex-wrap: wrap;
	}
	.estimate-label {
		font-size: var(--text-caption);
		padding: var(--space-1) var(--space-2);
		border-radius: 999px;
		background: var(--accent-subtle);
	}
	.estimate-note,
	.section-description {
		margin-bottom: var(--space-4);
	}
	.capacity-card {
		border-inline-start: 4px solid var(--brand-dot);
	}
	@media (max-width: 640px) {
		.money-guide {
			grid-template-columns: minmax(0, 1fr);
		}
	}
	section {
		margin-block: 1.25rem;
		padding: clamp(1rem, 3vw, 1.5rem);
		border: 1px solid var(--border);
		border-radius: var(--radius-card);
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
		min-height: var(--control-height);
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
		font-variant-numeric: tabular-nums;
		overflow-wrap: anywhere;
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
