<script lang="ts">
	import { enhance } from '$app/forms';
	import { resolve } from '$app/paths';
	import { formatRupees } from '#lib/money.js';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();
	type Draft = {
		accountId?: string;
		categoryId?: string;
		amount?: string;
		occurredOn?: string;
		note?: string;
	};
</script>

<svelte:head><title>Expenses | BuyWise</title></svelte:head>

{#snippet fields(draft: Draft)}
	<div class="fields">
		<label
			>Amount in ₹ <input
				class="input"
				name="amount"
				inputmode="decimal"
				required
				value={draft.amount ?? ''}
			/></label
		>
		<label
			>Date <input
				class="input"
				type="date"
				name="occurredOn"
				required
				max={data.today}
				value={draft.occurredOn ?? data.today}
			/></label
		>
		<label
			>Paid from
			<select class="input" name="accountId" required value={draft.accountId ?? ''}>
				<option value="">Choose an account</option>
				{#each data.accounts ?? [] as account (account.id)}<option value={account.id}
						>{account.name}</option
					>{/each}
			</select>
		</label>
		<label
			>Expense category
			<select class="input" name="categoryId" required value={draft.categoryId ?? ''}>
				<option value="">Choose a category</option>
				{#each data.categories ?? [] as category (category.id)}<option value={category.id}
						>{category.name}</option
					>{/each}
			</select>
		</label>
	</div>
	<label
		>Note (optional) <input
			class="input"
			name="note"
			maxlength="500"
			value={draft.note ?? ''}
		/></label
	>
{/snippet}

<div class="expenses">
	<header><h1>Expenses</h1></header>
	<section class="card" aria-labelledby="manual-expense">
		<h2 id="manual-expense">Record an expense</h2>
		{#if form?.manualSaved}<p role="status">Expense recorded.</p>{/if}
		{#if form?.manualError}<p role="alert">{form.manualError}</p>{/if}
		{#if data.accounts === null || data.categories === null}
			<p>Accounts or categories couldn’t load. Reload the page to try again.</p>
		{:else if !data.accounts.length || !data.categories.length}
			<p>
				Add an <a href={resolve('/accounts')}>account</a> and an active
				<a href={resolve('/categories')}>expense category</a> to record expenses.
			</p>
		{:else}
			<form method="POST" action="?/manual" use:enhance>
				<input type="hidden" name="key" value={form?.requestKey ?? data.requestKey} />
				{@render fields(form?.manualDraft ?? {})}
				<div><button class="btn btn-primary">Record expense</button></div>
			</form>
		{/if}
	</section>

	<section class="card" aria-labelledby="expense-inbox">
		<h2 id="expense-inbox">Expense suggestions</h2>
		{#if form?.candidateSaved}<p role="status">Suggestion confirmed and recorded.</p>{/if}
		{#if form?.candidateRejected}<p role="status">Suggestion dismissed.</p>{/if}
		{#if form?.candidateError && !data.candidates?.some((item) => item.id === form?.candidateId)}<p
				role="alert"
			>
				{form.candidateError}
			</p>{/if}
		{#if data.candidates === null}
			<p>Suggestions couldn’t load. Reload the page to try again.</p>
		{:else if !data.candidates.length}
			<p>No expense suggestions to review.</p>
		{:else}
			{#each data.candidates as item (item.id)}
				{@const feedback = form?.candidateId === item.id ? form : null}
				<details open={!!feedback?.candidateError}>
					<summary>
						<span>{item.merchant || 'Expense suggestion'}</span>
						<strong>{formatRupees(item.amountPaise)}</strong>
					</summary>
					<p>Android notification · {item.sourceApp} · {item.occurredOn}</p>
					{#if item.categorySuggestion}<p>
							Suggested category {item.categorySuggestion.name} ({item.categorySuggestion
								.confidence} confidence)
						</p>{/if}
					{#if item.possibleDuplicate}<p>
							A recorded expense or another suggestion has the same date and amount. Check whether
							they’re separate purchases.
						</p>{/if}
					{#if feedback?.candidateError}<p role="alert">{feedback.candidateError}</p>{/if}
					{#if data.accounts?.length && data.categories?.length}
						<form method="POST" action="?/confirm" use:enhance>
							<input type="hidden" name="candidateId" value={item.id} />
							{@render fields(
								feedback?.candidateDraft ?? {
									amount: (item.amountPaise / 100).toFixed(2),
									occurredOn: item.occurredOn,
									categoryId: item.categorySuggestion?.id,
									note: item.merchant
								}
							)}
							{#if item.possibleDuplicate || feedback?.knownDuplicate}
								<label class="choice"
									><input type="checkbox" name="allowDuplicate" value="yes" /> This is a separate expense.</label
								>
							{/if}
							<label class="choice"
								><input type="checkbox" name="confirmed" value="yes" required /> I’ve reviewed these details.</label
							>
							<div><button class="btn btn-primary">Confirm expense</button></div>
						</form>
					{:else}<p>An account and an active expense category are required to confirm.</p>{/if}
					<form method="POST" action="?/reject" use:enhance>
						<input type="hidden" name="candidateId" value={item.id} />
						<div><button class="btn btn-secondary">Dismiss suggestion</button></div>
					</form>
				</details>
			{/each}
		{/if}
	</section>
</div>

<style>
	.expenses {
		display: grid;
		gap: 1.25rem;
		max-width: 60rem;
	}
	section {
		display: grid;
		gap: 0.85rem;
		padding: clamp(1rem, 3vw, 1.5rem);
	}
	h1,
	h2,
	p {
		margin: 0;
	}
	form,
	label {
		display: grid;
		gap: 0.5rem;
	}
	form {
		gap: 0.9rem;
	}
	.fields {
		display: grid;
		gap: 0.9rem;
		grid-template-columns: repeat(auto-fit, minmax(min(100%, 15rem), 1fr));
	}
	.input {
		width: 100%;
	}
	.choice {
		display: flex;
		align-items: baseline;
	}
	details {
		border: 1px solid var(--border-subtle);
		border-radius: var(--radius-md);
		padding: 0.85rem;
		overflow-wrap: anywhere;
	}
	summary {
		cursor: pointer;
	}
	summary strong {
		margin-left: 0.5rem;
	}
	details p,
	details form {
		margin-top: 0.8rem;
	}
	[role='alert'] {
		color: var(--danger);
	}
</style>
