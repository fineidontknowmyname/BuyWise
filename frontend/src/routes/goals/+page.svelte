<script lang="ts">
	import { resolve } from '$app/paths';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();
</script>

<svelte:head>
	<title>Goals · Self-EMI</title>
</svelte:head>

<main>
	<p><a href={resolve('/')}>Home</a></p>
	<h1>Purchase goals</h1>

	<form method="POST" action="?/create">
		<label>
			What do you want to buy?
			<input name="name" required maxlength="200" value={form?.name ?? ''} />
		</label>
		<button type="submit">Create goal</button>
	</form>
	{#if form?.error}
		<p role="alert">{form.error}</p>
	{/if}

	{#if data.error}
		<p role="alert">{data.error}</p>
	{:else if data.goals.length === 0}
		<p>No goals yet.</p>
	{:else}
		<ul>
			{#each data.goals as goal (goal.id)}
				<li><a href={resolve('/goals/[id]', { id: goal.id })}>{goal.name}</a></li>
			{/each}
		</ul>
	{/if}
</main>
