<script lang="ts">
	import { resolve } from '$app/paths';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();

	const income = $derived(data.categories.filter((c) => c.kind === 'income'));
	const expense = $derived(data.categories.filter((c) => c.kind === 'expense'));
</script>

<svelte:head>
	<title>Categories · Self-EMI</title>
</svelte:head>

<main>
	<p><a href={resolve('/')}>Home</a></p>
	<h1>Categories</h1>
	<p>
		A committed expense is one you cannot easily skip, like rent. The rest count as discretionary.
	</p>

	<form method="POST" action="?/create">
		<label>
			Name
			<input name="name" required maxlength="200" value={form?.name ?? ''} />
		</label>
		<label>
			Kind
			<select name="kind" required>
				<option value="expense" selected={form?.kind !== 'income'}>Expense</option>
				<option value="income" selected={form?.kind === 'income'}>Income</option>
			</select>
		</label>
		<label>
			<input type="checkbox" name="committed" checked={form?.committed ?? false} />
			Committed (expenses only)
		</label>
		<button type="submit">Add category</button>
	</form>
	{#if form && 'error' in form && form.error}
		<p role="alert">{form.error}</p>
	{/if}

	{#if data.error}
		<p role="alert">{data.error}</p>
	{:else}
		<h2>Expenses</h2>
		<ul>
			{#each expense as category (category.id)}
				<li>{category.name}{category.committed ? ' (committed)' : ''}</li>
			{/each}
		</ul>
		<h2>Income</h2>
		<ul>
			{#each income as category (category.id)}
				<li>{category.name}</li>
			{/each}
		</ul>
	{/if}
</main>
