<script lang="ts">
	import { resolve } from '$app/paths';
	import { formatRupees } from '#lib/money.js';
	import Modal from '#lib/components/Modal.svelte';
	import type { PageProps } from './$types';
	import { ArrowLeft, Plus, CreditCard, FileText, Ban, AlertCircle } from '@lucide/svelte';

	let { data, form }: PageProps = $props();

	let showAddModal = $state(false);
	let voidTargetId = $state<string | null>(null);

	const incomeCategories = $derived(data.categories.filter((c) => c.kind === 'income'));
	const expenseCategories = $derived(data.categories.filter((c) => c.kind === 'expense'));
	const categoryName = $derived(new Map(data.categories.map((c) => [c.id, c.name])));

	// Calculate total credits and debits
	const totalIncomePaise = $derived(
		data.transactions
			.filter((t) => !t.voided && t.kind === 'income')
			.reduce((sum, t) => sum + t.amountPaise, 0)
	);

	const totalExpensePaise = $derived(
		data.transactions
			.filter((t) => !t.voided && t.kind === 'expense')
			.reduce((sum, t) => sum + t.amountPaise, 0)
	);
</script>

<svelte:head>
	<title>{data.account.name} · BuyWise</title>
</svelte:head>

<div class="account-detail-page">
	<!-- Back Navigation -->
	<div>
		<a href={resolve('/accounts')} class="back-link">
			<ArrowLeft size={16} />
			<span>Back to Accounts</span>
		</a>
	</div>

	<!-- Account Header Card -->
	<div class="account-hero card">
		<div class="hero-top">
			<div class="hero-title-group">
				<div class="hero-icon-box">
					<CreditCard size={26} />
				</div>
				<div>
					<span class="hero-subtitle">Account Ledger</span>
					<h1 class="hero-account-name">{data.account.name}</h1>
				</div>
			</div>
			<div class="hero-action-group">
				<button type="button" class="btn btn-primary" onclick={() => (showAddModal = true)}>
					<Plus size={18} />
					<span>Add Transaction</span>
				</button>
			</div>
		</div>

		<div class="hero-stats-row">
			<div class="hero-stat-item">
				<span class="hero-stat-label">Current Balance</span>
				<div class="hero-stat-value tabular-nums" data-testid="balance">
					{formatRupees(data.account.balancePaise)}
				</div>
			</div>
			<div class="hero-stat-divider"></div>
			<div class="hero-stat-item">
				<span class="hero-stat-label">Total Inflow</span>
				<div class="hero-stat-subvalue text-success tabular-nums">
					+{formatRupees(totalIncomePaise)}
				</div>
			</div>
			<div class="hero-stat-divider"></div>
			<div class="hero-stat-item">
				<span class="hero-stat-label">Total Outflow</span>
				<div class="hero-stat-subvalue text-secondary tabular-nums">
					-{formatRupees(totalExpensePaise)}
				</div>
			</div>
		</div>
	</div>

	<!-- Error Alert -->
	{#if form?.error}
		<div class="alert-box alert-error" role="alert">
			<AlertCircle size={18} />
			<span>{form.error}</span>
		</div>
	{/if}

	<!-- Transactions Ledger Table -->
	<div class="transactions-section card">
		<div class="card-header">
			<div>
				<h3 class="card-title">Transaction History</h3>
				<p class="card-desc">Complete record of posted deposits and withdrawals</p>
			</div>
			<span class="badge badge-neutral">
				{data.transactions.length}
				{data.transactions.length === 1 ? 'record' : 'records'}
			</span>
		</div>

		{#if data.transactions.length === 0}
			<div class="empty-state">
				<div class="empty-icon-box">
					<FileText size={32} />
				</div>
				<h4 class="empty-title">No transactions recorded</h4>
				<p class="empty-text">Add your first deposit or expense using the button above.</p>
				<button type="button" class="btn btn-primary btn-sm" onclick={() => (showAddModal = true)}>
					<Plus size={16} />
					<span>Add Transaction</span>
				</button>
			</div>
		{:else}
			<div class="table-container">
				<table class="table">
					<thead>
						<tr>
							<th>Date</th>
							<th>Category</th>
							<th>Note / Description</th>
							<th class="amount">Amount</th>
							<th>Status</th>
							<th style="width: 80px;"></th>
						</tr>
					</thead>
					<tbody>
						{#each data.transactions as tx (tx.id)}
							<tr class:void-row={tx.voided} data-testid="transaction">
								<td class="tabular-nums">{tx.occurredOn}</td>
								<td>
									<span class="badge badge-neutral">
										{categoryName.get(tx.categoryId) ?? 'Unknown'}
									</span>
								</td>
								<td class="font-medium text-secondary">{tx.note || '—'}</td>
								<td class="amount">
									<span
										class="amount-badge"
										class:income={tx.kind === 'income'}
										class:expense={tx.kind === 'expense'}
									>
										{tx.kind === 'expense' ? '-' : '+'}{formatRupees(tx.amountPaise)}
									</span>
								</td>
								<td>
									{#if tx.voided}
										<span class="badge badge-danger">Voided</span>
									{:else}
										<span class="badge badge-success">Posted</span>
									{/if}
								</td>
								<td style="text-align: right;">
									{#if !tx.voided && !tx.purchaseManaged}
										<button
											type="button"
											class="btn-icon btn-sm text-danger-btn"
											title="Void transaction"
											onclick={() => (voidTargetId = tx.id)}
										>
											<Ban size={15} />
										</button>
									{/if}
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		{/if}
	</div>
</div>

<!-- Modal Dialog: Add Transaction -->
<Modal open={showAddModal} title="Record Transaction" onclose={() => (showAddModal = false)}>
	<form method="POST" action="?/add" class="transaction-form">
		<input type="hidden" name="idempotencyKey" value={data.idempotencyKey} />

		<div class="form-group">
			<label for="tx-category" class="label">Category</label>
			<select id="tx-category" name="categoryId" required class="select">
				<option value="">Select a category...</option>
				<optgroup label="Expenses">
					{#each expenseCategories as category (category.id)}
						<option value={category.id} selected={form?.categoryId === category.id}>
							{category.name}{category.committed ? ' (committed)' : ''}
						</option>
					{/each}
				</optgroup>
				<optgroup label="Income">
					{#each incomeCategories as category (category.id)}
						<option value={category.id} selected={form?.categoryId === category.id}>
							{category.name}
						</option>
					{/each}
				</optgroup>
			</select>
		</div>

		<div class="form-row">
			<div class="form-group">
				<label for="tx-amount" class="label">Amount (₹)</label>
				<input
					id="tx-amount"
					name="amount"
					required
					inputmode="decimal"
					placeholder="e.g. 1500 or 1500.50"
					class="input"
					value={form?.amount ?? ''}
				/>
			</div>

			<div class="form-group">
				<label for="tx-date" class="label">Date</label>
				<input
					id="tx-date"
					name="occurredOn"
					type="date"
					required
					class="input"
					value={form?.occurredOn ?? data.today}
				/>
			</div>
		</div>

		<div class="form-group">
			<label for="tx-note" class="label">Note (Optional)</label>
			<input
				id="tx-note"
				name="note"
				maxlength="500"
				placeholder="e.g. Grocery trip, Monthly dividend"
				class="input"
				value={form?.note ?? ''}
			/>
		</div>

		<div class="modal-footer">
			<button type="button" class="btn btn-secondary" onclick={() => (showAddModal = false)}>
				Cancel
			</button>
			<button type="submit" class="btn btn-primary"> Post Transaction </button>
		</div>
	</form>
</Modal>

<!-- Modal Dialog: Confirm Void -->
<Modal
	open={voidTargetId !== null}
	title="Confirm Void Transaction"
	onclose={() => (voidTargetId = null)}
>
	<div class="void-confirm-content">
		<p class="text-secondary">
			Are you sure you want to void this transaction? Voided transactions are retained for audit
			records but excluded from balance calculations.
		</p>
		<form method="POST" action="?/void">
			<input type="hidden" name="transactionId" value={voidTargetId} />
			<div class="modal-footer">
				<button type="button" class="btn btn-secondary" onclick={() => (voidTargetId = null)}>
					Cancel
				</button>
				<button type="submit" class="btn btn-danger"> Void Transaction </button>
			</div>
		</form>
	</div>
</Modal>

<style>
	.account-detail-page {
		display: flex;
		flex-direction: column;
		gap: 24px;
	}

	.back-link {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		font-size: 0.88rem;
		font-weight: 500;
		color: var(--foreground-secondary);
		transition: color 0.15s ease;
	}

	.back-link:hover {
		color: var(--primary);
	}

	.account-hero {
		padding: 28px;
		display: flex;
		flex-direction: column;
		gap: 24px;
	}

	.hero-top {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 20px;
		flex-wrap: wrap;
	}

	.hero-title-group {
		display: flex;
		align-items: center;
		gap: 16px;
	}

	.hero-icon-box {
		width: 52px;
		height: 52px;
		border-radius: 14px;
		background-color: var(--primary-subtle);
		color: var(--primary);
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.hero-subtitle {
		font-size: 0.8rem;
		font-weight: 600;
		text-transform: uppercase;
		letter-spacing: 0.05em;
		color: var(--foreground-muted);
	}

	.hero-account-name {
		font-size: 1.8rem;
		font-weight: 700;
		color: var(--foreground);
		letter-spacing: -0.03em;
		line-height: 1.15;
	}

	.hero-stats-row {
		display: flex;
		align-items: center;
		gap: 32px;
		padding-top: 20px;
		border-top: 1px solid var(--border-subtle);
		flex-wrap: wrap;
	}

	.hero-stat-item {
		display: flex;
		flex-direction: column;
		gap: 4px;
	}

	.hero-stat-label {
		font-size: 0.78rem;
		text-transform: uppercase;
		font-weight: 600;
		letter-spacing: 0.04em;
		color: var(--foreground-muted);
	}

	.hero-stat-value {
		font-size: 1.7rem;
		font-weight: 700;
		color: var(--foreground);
		letter-spacing: -0.02em;
	}

	.hero-stat-subvalue {
		font-size: 1.15rem;
		font-weight: 600;
	}

	.text-success {
		color: var(--success);
	}

	.hero-stat-divider {
		width: 1px;
		height: 36px;
		background-color: var(--border);
	}

	.alert-box {
		display: flex;
		align-items: center;
		gap: 10px;
		padding: 12px 16px;
		border-radius: 12px;
		font-size: 0.9rem;
	}

	.alert-error {
		background-color: var(--danger-surface);
		color: var(--danger-foreground);
		border: 1px solid var(--danger);
	}

	.amount-badge.income {
		color: var(--success);
		font-weight: 600;
	}

	.amount-badge.expense {
		color: var(--foreground);
		font-weight: 600;
	}

	.void-row {
		text-decoration: line-through;
		opacity: 0.55;
	}

	.text-danger-btn {
		color: var(--danger);
	}

	.text-danger-btn:hover {
		background-color: var(--danger-surface);
	}

	.void-confirm-content {
		display: flex;
		flex-direction: column;
		gap: 16px;
	}

	@media (max-width: 640px) {
		.hero-top {
			flex-direction: column;
			align-items: flex-start;
		}
		.hero-stats-row {
			gap: 20px;
		}
		.hero-stat-divider {
			display: none;
		}
	}
</style>
