<script lang="ts">
	import { resolve } from '$app/paths';
	import { formatRupees } from '#lib/money.js';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();
</script>

<svelte:head>
	<title>Accounts · Self-EMI</title>
</svelte:head>

<main>
	<p><a href={resolve('/')}>Home</a></p>
	<h1>Accounts</h1>

	<form method="POST" action="?/create">
		<label>
			Account name
			<input name="name" required maxlength="200" value={form?.name ?? ''} />
		</label>
		<label>
			Opening balance (₹)
			<input name="opening" inputmode="decimal" placeholder="0" value={form?.opening ?? ''} />
		</label>
		<button type="submit">Add account</button>
	</form>
	{#if form?.error}
		<p role="alert">{form.error}</p>
	{/if}

	{#if data.error}
		<p role="alert">{data.error}</p>
	{:else if data.accounts.length === 0}
		<p>No accounts yet.</p>
	{:else}
		<table>
			<thead>
				<tr><th>Account</th><th>Balance</th></tr>
			</thead>
			<tbody>
				{#each data.accounts as account (account.id)}
					<tr>
						<td><a href={resolve('/accounts/[id]', { id: account.id })}>{account.name}</a></td>
						<td data-testid="balance">{formatRupees(account.balancePaise)}</td>
					</tr>
				{/each}
			</tbody>
		</table>
	{/if}
</main>
