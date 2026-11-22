<script lang="ts">
	import { resolve } from '$app/paths';
	import { formatRupees } from '#lib/money.js';
	import Modal from '#lib/components/Modal.svelte';
	import type { PageProps } from './$types';
	import { Wallet, Plus, ArrowRight, CreditCard, TrendingUp, AlertCircle } from '@lucide/svelte';

	let { data, form }: PageProps = $props();

	let showCreateModal = $state(false);

	const totalNetBalancePaise = $derived(
		data.accounts.reduce((sum, acc) => sum + acc.balancePaise, 0)
	);
</script>

<svelte:head>
	<title>Accounts · BuyWise</title>
</svelte:head>

<div class="accounts-page">
	<!-- Page Header -->
	<div class="page-header">
		<div>
			<h1 class="page-headline">Accounts</h1>
			<p class="page-subheadline">Manage your cash, bank accounts, and savings balances.</p>
		</div>
		<div class="header-actions">
			<button type="button" class="btn btn-primary" onclick={() => (showCreateModal = true)}>
				<Plus size={18} />
				<span>Add Account</span>
			</button>
		</div>
	</div>

	<!-- Total Liquidity Hero Banner -->
	{#if data.accounts.length > 0}
		<div class="net-worth-banner card">
			<div class="banner-left">
				<div class="banner-icon">
					<TrendingUp size={24} />
				</div>
				<div>
					<span class="banner-label">Total Liquid Balance</span>
					<div class="banner-value tabular-nums">{formatRupees(totalNetBalancePaise)}</div>
				</div>
			</div>
			<div class="banner-right">
				<span class="account-stat-pill">
					{data.accounts.length}
					{data.accounts.length === 1 ? 'account' : 'accounts'} registered
				</span>
			</div>
		</div>
	{/if}

	<!-- Error Alert -->
	{#if data.error}
		<div class="alert-box alert-error" role="alert">
			<AlertCircle size={18} />
			<span>{data.error}</span>
		</div>
	{/if}

	<!-- Accounts List / Cards -->
	{#if data.accounts.length === 0}
		<div class="card empty-state">
			<div class="empty-icon-box">
				<Wallet size={32} />
			</div>
			<h3 class="empty-title">No accounts added yet</h3>
			<p class="empty-text">
				Add your primary bank account or cash wallet to begin tracking balances and transactions.
			</p>
			<button type="button" class="btn btn-primary" onclick={() => (showCreateModal = true)}>
				<Plus size={18} />
				<span>Add your first account</span>
			</button>
		</div>
	{:else}
		<div class="accounts-grid">
			{#each data.accounts as account (account.id)}
				<a href={resolve('/accounts/[id]', { id: account.id })} class="account-card">
					<div class="acc-top">
						<div class="acc-icon-box">
							<CreditCard size={20} />
						</div>
						<span class="badge badge-neutral">Active</span>
					</div>

					<div class="acc-info">
						<h3 class="acc-name">{account.name}</h3>
						<span class="acc-type">Personal Account</span>
					</div>

					<div class="acc-balance-group">
						<span class="balance-label">Current Balance</span>
						<div class="acc-balance tabular-nums" data-testid="balance">
							{formatRupees(account.balancePaise)}
						</div>
					</div>

					<div class="acc-footer">
						<span class="view-ledger">View Ledger</span>
						<ArrowRight size={16} class="acc-arrow" />
					</div>
				</a>
			{/each}
		</div>
	{/if}
</div>

<!-- Modal Dialog: Add Account -->
<Modal open={showCreateModal} title="Add New Account" onclose={() => (showCreateModal = false)}>
	<form method="POST" action="?/create">
		<div class="form-group">
			<label for="account-name" class="label">Account Name</label>
			<input
				id="account-name"
				name="name"
				required
				maxlength="200"
				placeholder="e.g. HDFC Salary, Emergency Fund, Cash Wallet"
				class="input"
				value={form?.name ?? ''}
			/>
		</div>

		<div class="form-group">
			<label for="account-opening" class="label">Opening Balance (₹)</label>
			<input
				id="account-opening"
				name="opening"
				inputmode="decimal"
				placeholder="0 or 25000"
				class="input"
				value={form?.opening ?? ''}
			/>
			<span class="text-xs text-muted">Initial balance in rupees. Leave empty for 0.</span>
		</div>

		{#if form?.error}
			<p class="form-error" role="alert">{form.error}</p>
		{/if}

		<div class="modal-footer">
			<button type="button" class="btn btn-secondary" onclick={() => (showCreateModal = false)}>
				Cancel
			</button>
			<button type="submit" class="btn btn-primary"> Save Account </button>
		</div>
	</form>
</Modal>

<style>
	.accounts-page {
		display: flex;
		flex-direction: column;
		gap: 24px;
	}

	.page-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 20px;
	}

	.page-headline {
		font-size: 1.85rem;
		font-weight: 700;
		color: var(--foreground);
		letter-spacing: -0.03em;
	}

	.page-subheadline {
		font-size: 0.92rem;
		color: var(--foreground-secondary);
		margin-top: 4px;
	}

	.net-worth-banner {
		padding: 22px 26px;
		display: flex;
		align-items: center;
		justify-content: space-between;
		background: linear-gradient(135deg, var(--surface) 0%, var(--surface-secondary) 100%);
	}

	.banner-left {
		display: flex;
		align-items: center;
		gap: 16px;
	}

	.banner-icon {
		width: 48px;
		height: 48px;
		border-radius: 14px;
		background-color: var(--primary-subtle);
		color: var(--primary);
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.banner-label {
		font-size: 0.82rem;
		font-weight: 600;
		text-transform: uppercase;
		letter-spacing: 0.05em;
		color: var(--foreground-muted);
	}

	.banner-value {
		font-size: 1.85rem;
		font-weight: 700;
		color: var(--foreground);
		letter-spacing: -0.03em;
		line-height: 1.1;
		margin-top: 2px;
	}

	.account-stat-pill {
		font-size: 0.85rem;
		font-weight: 500;
		padding: 6px 14px;
		border-radius: 999px;
		background-color: var(--surface);
		border: 1px solid var(--border);
		color: var(--foreground-secondary);
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

	.accounts-grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
		gap: 20px;
	}

	.account-card {
		background-color: var(--surface);
		border: 1px solid var(--border);
		border-radius: 16px;
		padding: 22px;
		display: flex;
		flex-direction: column;
		text-decoration: none;
		box-shadow: var(--shadow-sm);
		transition: all 0.2s ease;
	}

	.account-card:hover {
		box-shadow: var(--shadow-md);
		border-color: var(--primary);
		transform: translateY(-2px);
	}

	.acc-top {
		display: flex;
		align-items: center;
		justify-content: space-between;
		margin-bottom: 16px;
	}

	.acc-icon-box {
		width: 42px;
		height: 42px;
		border-radius: 12px;
		background-color: var(--primary-subtle);
		color: var(--primary);
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.acc-name {
		font-size: 1.15rem;
		font-weight: 600;
		color: var(--foreground);
	}

	.acc-type {
		font-size: 0.8rem;
		color: var(--foreground-muted);
	}

	.acc-balance-group {
		margin: 20px 0 16px 0;
	}

	.balance-label {
		font-size: 0.78rem;
		text-transform: uppercase;
		letter-spacing: 0.04em;
		color: var(--foreground-muted);
		display: block;
		margin-bottom: 2px;
	}

	.acc-balance {
		font-size: 1.6rem;
		font-weight: 700;
		color: var(--foreground);
		letter-spacing: -0.02em;
	}

	.acc-footer {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding-top: 14px;
		border-top: 1px solid var(--border-subtle);
		margin-top: auto;
	}

	.view-ledger {
		font-size: 0.85rem;
		font-weight: 600;
		color: var(--primary);
	}

	:global(.acc-arrow) {
		color: var(--foreground-muted);
		transition: transform 0.15s ease;
	}

	.account-card:hover :global(.acc-arrow) {
		transform: translateX(3px);
		color: var(--primary);
	}

	@media (max-width: 640px) {
		.page-header {
			flex-direction: column;
			align-items: flex-start;
		}
		.net-worth-banner {
			flex-direction: column;
			align-items: flex-start;
			gap: 16px;
		}
		.accounts-grid {
			grid-template-columns: 1fr;
		}
	}
</style>
