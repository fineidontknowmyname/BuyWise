<script lang="ts">
	import { resolve } from '$app/paths';
	import { formatRupees } from '#lib/money.js';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();

	const incomeCategories = $derived(data.categories.filter((c) => c.kind === 'income'));
	const expenseCategories = $derived(data.categories.filter((c) => c.kind === 'expense'));
	const categoryName = $derived(new Map(data.categories.map((c) => [c.id, c.name])));
</script>

<svelte:head>
	<title>{data.account.name} · Self-EMI</title>
</svelte:head>

<main>
	<p><a href={resolve('/accounts')}>All accounts</a></p>
	<h1>{data.account.name}</h1>
	<p>Balance: <strong data-testid="balance">{formatRupees(data.account.balancePaise)}</strong></p>

	<h2>Add a transaction</h2>
	<form method="POST" action="?/add">
		<input type="hidden" name="idempotencyKey" value={data.idempotencyKey} />
		<label>
			Category
			<select name="categoryId" required>
				<option value="">Choose…</option>
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
		</label>
		<label>
			Amount (₹)
			<input name="amount" required inputmode="decimal" value={form?.amount ?? ''} />
		</label>
		<label>
			Date
			<input name="occurredOn" type="date" required value={form?.occurredOn ?? data.today} />
		</label>
		<label>
			Note
			<input name="note" maxlength="500" value={form?.note ?? ''} />
		</label>
		<button type="submit">Add</button>
	</form>
	{#if form?.error}
		<p role="alert">{form.error}</p>
	{/if}

	<h2>Transactions</h2>
	{#if data.transactions.length === 0}
		<p>No transactions yet.</p>
	{:else}
		<table>
			<thead>
				<tr><th>Date</th><th>Category</th><th>Note</th><th>Amount</th><th></th></tr>
			</thead>
			<tbody>
				{#each data.transactions as tx (tx.id)}
					<tr class:void={tx.voided} data-testid="transaction">
						<td>{tx.occurredOn}</td>
						<td>{categoryName.get(tx.categoryId) ?? 'Unknown'}</td>
						<td>{tx.note}</td>
						<td>
							{tx.kind === 'expense' ? '-' : '+'}{formatRupees(tx.amountPaise)}
						</td>
						<td>
							{#if tx.voided}
								<em>Void</em>
							{:else}
								<form method="POST" action="?/void">
									<input type="hidden" name="transactionId" value={tx.id} />
									<button type="submit">Void</button>
								</form>
							{/if}
						</td>
					</tr>
				{/each}
			</tbody>
		</table>
	{/if}
</main>

<style>
	.void {
		text-decoration: line-through;
		opacity: 0.6;
	}
</style>
