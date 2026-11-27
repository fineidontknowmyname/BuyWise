<script lang="ts">
	import '#lib/styles/workbench.css';
	import { enhance } from '$app/forms';
	import { resolve } from '$app/paths';
	import { formatRupees } from '#lib/money.js';
	import Modal from '#lib/components/Modal.svelte';
	import type { PageProps } from './$types';
	let { data, form }: PageProps = $props();
	let createOpen = $state(false);
	const money = $derived(data.money);
</script>

<svelte:head><title>My Money | BuyWise</title></svelte:head>
<div class="workbench">
	<header class="wb-heading">
		<div>
			<p class="wb-eyebrow">My Money</p>
			<h1>A clear plan for your money.</h1>
			<p class="wb-muted">Cover your spending. Save the rest. Build your goals.</p>
		</div>
		<button class="btn btn-secondary" onclick={() => (createOpen = true)}>Add account</button>
	</header>
	{#if data.error || form?.error}<p class="wb-notice" role="alert">
			{form?.error ?? data.error}
		</p>{/if}
	{#if form?.success}<p class="wb-notice" role="status">{form.success}</p>{/if}
	{#if money}
		{#if money.reviewRequired}<p class="wb-notice" role="status">
				A new month has started. <a href="#month-review">Review unused spending</a> before assigning more
				savings to goals.
			</p>{/if}
		<section class="wb-card">
			<div class="wb-card-head">
				<h2>Money available now</h2>
				<span class="wb-badge">Recorded balances</span>
			</div>
			<div class="wb-metrics">
				<div class="wb-metric">
					<span>Account balance today</span><strong>{formatRupees(money.balancePaise)}</strong>
				</div>
				<div class="wb-metric">
					<span>Protected for spending</span><strong
						>{money.configured ? formatRupees(money.protectedPaise) : 'Set your plan below'}</strong
					>
				</div>
				<div class="wb-metric">
					<span>Free savings for goals</span><strong
						>{money.reviewRequired
							? 'Review new month'
							: money.configured
								? formatRupees(money.savingsNowPaise)
								: 'Not calculated'}</strong
					>
				</div>
			</div>
			<p class="wb-muted">
				{formatRupees(money.reservedPaise)} reserved for goals · {formatRupees(
					money.emergencyPaise
				)} in emergency accounts.
			</p>
			{#if money.cashShortfallPaise > 0}<p class="wb-notice" role="status">
					Your recorded balance is {formatRupees(money.cashShortfallPaise)} short of protected spending
					and goal savings. Review spending before adding to goals.
				</p>{/if}
		</section>
		<section class="wb-card">
			<div class="wb-card-head">
				<h2>Your monthly plan</h2>
				<span class="wb-badge">Expected income → spending → savings</span>
			</div>
			<form
				method="POST"
				action="?/plan"
				use:enhance={() =>
					async ({ update }) =>
						update({ reset: false })}
			>
				<input type="hidden" name="revision" value={money.revision} />
				<div class="wb-fields">
					<label
						>Recurring income each month (₹)<input
							name="income"
							inputmode="decimal"
							required
							value={money.incomePaise / 100}
						/></label
					>
				</div>
				<div class="allocation-grid">
					{#each [true, false] as fixed (fixed)}
						<div class="allocation-group">
							<h3>{fixed ? 'Fixed obligations' : 'Flexible spending'}</h3>
							<p class="wb-muted">
								{fixed
									? 'Rent, bills and other commitments.'
									: 'Your monthly limits for everyday spending.'}
							</p>
							{#each money.categories.filter((c) => c.fixed === fixed && !c.archived) as category (category.categoryId)}
								<label class="allocation-row"
									><span>{category.name}</span><input
										aria-label={`${category.name} monthly amount in rupees`}
										name={`category:${category.categoryId}`}
										inputmode="decimal"
										required
										value={category.monthlyPaise / 100}
									/></label
								>
							{:else}<p class="wb-muted">No categories yet.</p>{/each}
						</div>
					{/each}
				</div>
				<details class="plan-preferences">
					<summary>Reserve & repayment settings</summary>
					<div class="wb-fields">
						<label
							>Emergency reserve target (₹)<input
								name="emergencyTarget"
								inputmode="decimal"
								value={money.emergencyTargetPaise / 100}
							/></label
						><label
							>Debt repayments<select name="debtStatus" value={money.debtStatus}
								><option value="">Not confirmed</option><option value="current"
									>No overdue payments</option
								><option value="overdue">Payments overdue</option></select
							></label
						>
					</div>
					<p class="wb-muted">
						Emergency account balances remain protected. These settings help check purchase
						readiness.
					</p>
				</details>
				<div class="wb-row savings-result">
					<div>
						<span class="wb-muted">Planned monthly savings</span>
						<h2>
							{money.configured
								? formatRupees(money.monthlySavingsPaise)
								: 'Save your plan to calculate'}
						</h2>
					</div>
					<button class="btn btn-primary">Save monthly plan</button>
				</div>
				{#if money.configured}<p class="wb-muted arithmetic">
						{formatRupees(money.incomePaise)} income − {formatRupees(money.fixedPaise)} fixed − {formatRupees(
							money.flexiblePaise
						)} flexible = {formatRupees(money.monthlySavingsPaise)} savings.
					</p>{/if}
			</form>
			{#if money.configured}<p class="wb-muted">
					{formatRupees(money.plannedGoalsPaise)} per month committed to goals · {formatRupees(
						money.unassignedMonthlyPaise
					)} left for a new goal.
				</p>{/if}
			{#if money.monthlyShortfallPaise > 0}<p class="wb-notice" role="alert">
					Planned spending exceeds income by {formatRupees(money.monthlyShortfallPaise)}. Reduce
					spending before committing to monthly goal savings.
				</p>{/if}
			<div class="wb-row">
				<a href={resolve('/categories')}>Manage spending categories →</a>
				<p class="wb-muted">Saving this plan does not record income or change your balance.</p>
			</div>
			{#if money.configured}<p class="wb-muted">
					Lower limits apply next month. Current spending stays protected until you explicitly move
					unused flexible money below.
				</p>{/if}
		</section>
		{#if money.configured}
			<section class="wb-card">
				<div class="wb-card-head">
					<h2>Spending left · {money.period}</h2>
					<a href={resolve('/expenses')}>Record an expense →</a>
				</div>
				<div class="spending-list">
					{#each money.categories as category (category.categoryId)}
						<div class="spending-row">
							<div>
								<strong>{category.name}</strong>
								<p class="wb-muted">
									{category.fixed ? 'Fixed · protected' : 'Flexible'} · Spent {formatRupees(
										category.spentPaise
									)}
								</p>
							</div>
							<strong>{formatRupees(category.remainingPaise)} left</strong>
							{#if !category.fixed && category.remainingPaise > 0}<details>
									<summary>Move to savings</summary>
									<form method="POST" action="?/move" use:enhance class="move-form">
										<input type="hidden" name="revision" value={money.revision} /><input
											type="hidden"
											name="categoryId"
											value={category.categoryId}
										/>
										<label>Amount (₹)<input name="amount" inputmode="decimal" required /></label
										><label class="confirm"
											><input type="checkbox" name="confirm" value="yes" required />Reduce this
											month's spending allowance by this amount.</label
										><button class="btn btn-secondary">Confirm move</button>
									</form>
								</details>{/if}
						</div>
					{/each}
				</div>
				<p class="wb-muted">
					Moving flexible money changes its purpose inside your balance. It does not create income
					or transfer money between bank accounts.
				</p>
			</section>
			{#if money.period < money.currentPeriod}<section class="wb-card" id="month-review">
					<h2>Review the new month</h2>
					<p class="wb-muted">
						Unpaid fixed obligations carry forward. Choose what happens to unused flexible spending.
					</p>
					<form method="POST" action="?/nextMonth" use:enhance>
						<input type="hidden" name="revision" value={money.revision} />
						<div class="wb-fields">
							<label
								>Unused flexible money<select name="flexibleChoice" required
									><option value="">Choose an action</option><option value="carry"
										>Carry into this month's spending</option
									><option value="save">Move into savings</option></select
								></label
							>
						</div>
						<button class="btn btn-primary">Confirm and open {money.currentPeriod}</button>
					</form>
				</section>{/if}
			<div class="wb-actions">
				<a class="btn btn-primary" href={resolve('/goals')}>Plan a purchase →</a><a
					href={resolve('/simulator')}>Try the reverse EMI calculator</a
				>
			</div>
		{/if}
	{/if}
	<section class="wb-card">
		<div class="wb-card-head">
			<h2>Your accounts</h2>
			<span class="wb-muted">Transactions stay in their original accounts.</span>
		</div>
		<ul class="wb-list">
			{#each data.accounts as account (account.id)}<li>
					<a class="wb-row" href={resolve('/accounts/[id]', { id: account.id })}
						><strong>{account.name}</strong><span>{formatRupees(account.balancePaise)} →</span></a
					>
				</li>{:else}<li class="wb-muted">
					Add an account with its current balance to start.
				</li>{/each}
		</ul>
	</section>
</div>
<Modal open={createOpen} title="Add account" onclose={() => (createOpen = false)}>
	<form method="POST" action="?/create" class="workbench">
		<div class="wb-fields">
			<label
				>Account name<input name="name" required maxlength="200" value={form?.name ?? ''} /></label
			><label
				>Opening balance (₹)<input
					name="opening"
					inputmode="decimal"
					value={form?.opening ?? ''}
				/></label
			>
		</div>
		<button class="btn btn-primary">Save account</button>
	</form>
</Modal>

<style>
	.allocation-grid {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 24px;
		margin-block: 24px;
	}
	.allocation-group {
		display: grid;
		gap: 12px;
		align-content: start;
	}
	.allocation-row {
		display: grid;
		grid-template-columns: 1fr 120px;
		align-items: center;
		gap: 12px;
	}
	.allocation-row input {
		min-width: 0;
		text-align: right;
	}
	.savings-result {
		border-top: 1px solid var(--border);
		padding-top: 20px;
	}
	.plan-preferences {
		margin-bottom: 20px;
	}
	.plan-preferences summary {
		cursor: pointer;
		font-size: 13px;
		color: var(--foreground-secondary);
		margin-bottom: 12px;
	}
	.arithmetic {
		margin-top: 12px !important;
	}
	.spending-row {
		display: grid;
		grid-template-columns: 1fr auto;
		gap: 12px;
		padding: 16px 0;
		border-bottom: 1px solid var(--border);
	}
	.spending-row details {
		grid-column: 1/-1;
	}
	.move-form {
		display: grid;
		gap: 12px;
		max-width: 420px;
		padding: 16px 0;
	}
	.confirm {
		display: flex !important;
		align-items: center;
		gap: 10px;
	}
	.confirm input {
		width: auto;
	}
	.spending-row summary {
		cursor: pointer;
		font-size: 13px;
	}
	@media (max-width: 640px) {
		.allocation-grid {
			grid-template-columns: 1fr;
		}
		.allocation-row {
			grid-template-columns: 1fr 105px;
		}
	}
</style>
