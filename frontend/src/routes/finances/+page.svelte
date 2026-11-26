<script lang="ts">
	import '#lib/styles/workbench.css';
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
<div class="workbench">
	<header class="wb-heading">
		<div>
			<p class="wb-eyebrow">My money</p>
			<h1>Financial setup</h1>
			<p class="wb-muted">One clear plan for what you earn, spend and save.</p>
		</div>
		<a class="btn btn-secondary" href={resolve('/accounts')}>Manage accounts</a>
	</header>
	{#if data.error || form?.error}<p class="wb-notice" role="alert">
			{form?.error ?? data.error}
		</p>{/if}
	{#if form?.success}<p class="wb-notice" role="status">{form.success}</p>{/if}
	{#if data.foundation}
		{@const foundation = data.foundation}{@const profile = foundation.profile}{@const estimates =
			foundation.capacity}
		<div class="wb-columns">
			<div class="wb-stack">
				<form
					class="wb-card"
					method="POST"
					action="?/profile"
					aria-busy={savingProfile}
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
				>
					<div class="wb-card-head">
						<div>
							<h2>Income & commitments</h2>
							<p class="wb-muted">Add your expected income and regular costs.</p>
						</div>
						<button
							type="button"
							class="btn btn-secondary"
							disabled={rows.length >= 100}
							onclick={() =>
								rows.push({
									name: '',
									kind: 'fixed_income',
									frequency: 'monthly',
									amount: '',
									budgetCategory: ''
								})}>+ Add entry</button
						>
					</div>
					<input type="hidden" name="revision" value={profile.revision} />
					<div>
						{#each rows as row, index (row)}
							<div class="wb-entry">
								<label
									>Name<input
										name="name"
										maxlength="200"
										required
										bind:value={row.name}
										placeholder="Salary, rent, insurance…"
									/></label
								>
								<label
									>Type<select
										name="kind"
										bind:value={row.kind}
										onchange={() =>
											(row.frequency = row.kind === 'irregular_income' ? 'once' : 'monthly')}
										>{#each kinds as [value, label] (value)}<option {value}>{label}</option
											>{/each}</select
									></label
								>
								<label
									>Amount (₹)<input
										name="amount"
										inputmode="decimal"
										required
										bind:value={row.amount}
									/></label
								>
								<label
									>Frequency<select name="frequency" bind:value={row.frequency}
										>{#if row.kind === 'irregular_income'}<option value="once">One-off</option
											>{:else}{#each frequencies as [value, label] (value)}<option {value}
													>{label}</option
												>{/each}{/if}</select
									></label
								>
								<button
									type="button"
									class="btn btn-ghost"
									aria-label={'Remove ' + (row.name || 'entry ' + (index + 1))}
									onclick={() => rows.splice(index, 1)}>×</button
								>
								{#if !row.kind.includes('income')}<details class="wb-wide">
										<summary class="entry-options">Budget mapping</summary>
										<div class="wb-fields entry-extra">
											{#if !row.kind.includes('income')}<label
													>Covered by budget<select
														name="budgetCategory"
														bind:value={row.budgetCategory}
														><option value="">Additional cost</option
														>{#each foundation.connections.budgets as budget (budget.categoryId)}<option
																value={budget.categoryId}>{budget.name}</option
															>{/each}{#if row.budgetCategory && !foundation.connections.budgets.some((b) => b.categoryId === row.budgetCategory)}<option
																value={row.budgetCategory}
																>Removed budget · use manual amount</option
															>{/if}</select
													></label
												>{:else}<input type="hidden" name="budgetCategory" value="" />{/if}
										</div>
									</details>{:else}<input type="hidden" name="budgetCategory" value="" />{/if}
								{#if useCategoryBudgets && foundation.connections.budgets.length && row.kind === 'discretionary'}<p
										class="wb-muted wb-wide"
									>
										This allowance is replaced by your category budgets.
									</p>{/if}
							</div>
						{:else}<div class="wb-empty">
								Start with your monthly income. Add commitments as needed.
							</div>{/each}
					</div>
					<div class="budget-setting">
						<label class="wb-check"
							><input
								type="checkbox"
								name="useCategoryBudgets"
								value="yes"
								bind:checked={useCategoryBudgets}
							/>Use my category budgets</label
						><strong
							>{formatRupees(
								foundation.connections.budgets.reduce((sum, b) => sum + b.amountPaise, 0)
							)}<span class="wb-muted"> / month</span></strong
						>
					</div>
					<details class="inline-help">
						<summary>View spending plan</summary>
						<div class="entry-extra">
							<p class="wb-muted">
								Budgets replace discretionary allowances. Link an existing expense to its category
								to count it once.
							</p>
							<dl class="wb-lines">
								{#each foundation.connections.budgets as budget (budget.categoryId)}<div>
										<dt>{budget.name}</dt>
										<dd>{formatRupees(budget.amountPaise)}</dd>
									</div>{:else}<div>
										<dt>No budgets yet</dt>
										<dd>Manual costs apply</dd>
									</div>{/each}
							</dl>
							<a href={resolve('/budget')}>Edit category budgets →</a>
						</div>
					</details>
					<details class="inline-help">
						<summary
							>Suggestions from your records <span class="wb-badge"
								>{foundation.connections.suggestions.length}</span
							></summary
						>
						<div class="entry-extra">
							<p class="wb-muted">
								Monthly averages from {foundation.connections.historyFrom} to {foundation
									.connections.historyThrough}. Review one-off receipts before adding.
							</p>
							<ul class="wb-list">
								{#each foundation.connections.suggestions as suggestion (suggestion.categoryId)}<li
										class="wb-row"
									>
										<div>
											<strong>{suggestion.name}</strong>
											<p class="wb-muted">
												{formatRupees(suggestion.amountPaise)} / month · {suggestion.monthsObserved} months
												recorded
											</p>
										</div>
										<button
											type="button"
											class="btn btn-secondary"
											disabled={rows.length >= 100 ||
												rows.some(
													(r) => r.name.trim().toLowerCase() === suggestion.name.toLowerCase()
												)}
											onclick={() =>
												rows.push({
													name: suggestion.name,
													kind: suggestion.kind,
													frequency: 'monthly',
													amount: String(suggestion.amountPaise / 100),
													budgetCategory: ''
												})}>Add to plan</button
										>
									</li>{:else}<li class="wb-muted">
										More history is needed before we can suggest regular income or spending.
									</li>{/each}
							</ul>
						</div>
					</details>
					{#key profile.revision}<details class="inline-help">
							<summary>Safety preferences</summary>
							<div class="wb-fields entry-extra">
								<label
									>Variable income counted (%)<input
										name="variableIncomePercent"
										type="number"
										min="0"
										max="100"
										value={profile.variableIncomePercent}
										required
									/></label
								>
								<label
									>Buffer rate (%)<input
										name="bufferPercent"
										type="number"
										min="0"
										max="100"
										value={profile.bufferPercent}
										required
									/></label
								>
								<label
									>Minimum buffer (₹)<input
										name="bufferMinimum"
										inputmode="decimal"
										value={profile.bufferMinimumPaise / 100}
										required
									/></label
								>
								<label
									>Emergency target (₹)<input
										name="emergencyTarget"
										inputmode="decimal"
										value={profile.emergencyTargetPaise / 100}
										required
									/></label
								>
								<label
									>Debt repayments<select name="debtStatus" value={profile.debtStatus}
										><option value="">Not confirmed / none recorded</option><option value="current"
											>All current</option
										><option value="overdue">Overdue</option></select
									></label
								>
							</div>
						</details>{/key}
					<div class="wb-divider"></div>
					<div class="wb-row">
						<span class="wb-muted">Changes apply to your saving plans.</span><button
							class="btn btn-primary"
							disabled={savingProfile}>{savingProfile ? 'Saving…' : 'Save financial setup'}</button
						>
					</div>
				</form>
				<details class="wb-fold">
					<summary>Transfers & activity</summary>
					<div class="wb-fold-body">
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
										{foundation.accounts.find((a) => a.id === entry.fromAccountId)?.name ??
											'Account'}
										→ {foundation.accounts.find((a) => a.id === entry.toAccountId)?.name ??
											'Account'}
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
					</div>
				</details>
			</div>
			<aside class="wb-stack wb-sticky">
				<section class="wb-card">
					<div class="wb-card-head">
						<p class="wb-eyebrow">Monthly saving estimate</p>
						<span class="wb-badge">Plan estimate</span>
					</div>
					<p class="wb-amount">
						{profile.revision ? formatRupees(estimates.safeSavingsPaise) : '—'}
					</p>
					<p class="wb-muted">
						{profile.revision
							? 'Before allocation across your goals.'
							: 'Add your income and save to see your estimate.'}
					</p>
					<dl class="wb-lines">
						<div>
							<dt>Income counted</dt>
							<dd>{formatRupees(estimates.fixedIncomePaise + estimates.countedVariablePaise)}</dd>
						</div>
						<div>
							<dt>Planned spending</dt>
							<dd>{formatRupees(estimates.committedPaise + estimates.discretionaryPaise)}</dd>
						</div>
						<div>
							<dt>Safety buffer</dt>
							<dd>{formatRupees(estimates.bufferPaise)}</dd>
						</div>
					</dl>
					<a href={resolve('/goals')}>View purchase goals →</a>
				</section>
				<section class="wb-card">
					<div class="wb-card-head">
						<h2>Your accounts</h2>
						<span class="wb-badge">Recorded funds</span>
					</div>
					<ul class="wb-list">
						{#each foundation.accounts as account (account.id)}<li>
								<div class="wb-row">
									<strong>{account.name}</strong><strong
										>{formatRupees(account.balancePaise)}</strong
									>
								</div>
								<p class="wb-muted">
									{account.purpose === 'emergency_reserve'
										? 'Emergency reserve'
										: foundation.reservationsAvailable
											? formatRupees(account.availablePaise) + ' after goal reservations'
											: 'Reservations unavailable'}
								</p>
								<details class="inline-help">
									<summary>Account role</summary>
									<form method="POST" action="?/purpose" class="account-role" use:enhance>
										<input type="hidden" name="accountId" value={account.id} /><label
											>Role<select name="purpose" value={account.purpose}
												>{#each purposes as [value, label] (value)}<option {value}>{label}</option
													>{/each}</select
											></label
										><button class="btn btn-secondary">Update</button>
									</form>
								</details>
							</li>{:else}<li class="wb-muted">No accounts added.</li>{/each}
					</ul>
					<p class="wb-muted">Balances are recorded cash, separate from your monthly estimate.</p>
					<details class="inline-help">
						<summary>Reserve breakdown</summary>
						<dl class="wb-lines">
							<div>
								<dt>Emergency balance</dt>
								<dd>{formatRupees(foundation.emergencySavingsPaise)}</dd>
							</div>
							<div>
								<dt>Emergency gap</dt>
								<dd>{formatRupees(foundation.emergencyGapPaise)}</dd>
							</div>
							<div>
								<dt>General savings</dt>
								<dd>{formatRupees(foundation.generalSavingsPaise)}</dd>
							</div>
						</dl>
					</details>
				</section>
			</aside>
		</div>
	{/if}
</div>

<style>
	.budget-setting {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 14px;
		flex-wrap: wrap;
		padding: 16px;
		background: var(--surface-secondary);
		border-radius: 12px;
	}
	.budget-setting strong {
		font-size: 14px;
		font-variant-numeric: tabular-nums;
	}
	.inline-help > summary,
	.entry-options {
		cursor: pointer;
		font-size: 12px;
		font-weight: 500;
		color: var(--foreground-secondary);
	}
	.entry-extra {
		margin-top: 14px;
		display: grid;
		gap: 14px;
	}
	.account-role {
		display: grid;
		grid-template-columns: minmax(0, 1fr) auto;
		gap: 10px;
		align-items: end;
		margin-top: 12px;
	}
	.workbench :global(.form-grid) {
		display: grid;
		grid-template-columns: repeat(2, minmax(0, 1fr));
		gap: 14px;
		margin-bottom: 14px;
	}
	@media (max-width: 600px) {
		.workbench :global(.form-grid) {
			grid-template-columns: 1fr;
		}
	}
</style>
