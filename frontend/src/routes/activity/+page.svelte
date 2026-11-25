<script lang="ts">
	import { resolve } from '$app/paths';
	import ActivityList from '#lib/components/ActivityList.svelte';
	import type { PageProps } from './$types';
	let { data }: PageProps = $props();
	const groups = [
		['', 'All activity'],
		['ledger', 'Ledger'],
		['goals', 'Goals and plans'],
		['decisions', 'Decisions'],
		['prices', 'Prices'],
		['purchases', 'Purchases and surplus']
	];
	const next = $derived(
		new URLSearchParams({ kind: data.kind, before: data.activity?.next ?? '' }).toString()
	);
	const newest = $derived(new URLSearchParams({ kind: data.kind }).toString());
</script>

<svelte:head><title>Activity | BuyWise</title></svelte:head>
<h1>Activity</h1>
<section class="card">
	<form method="GET" action={resolve('/activity')}>
		<label
			>Activity type <select class="input" name="kind" value={data.kind}
				>{#each groups as [value, label] (value)}<option {value}>{label}</option>{/each}</select
			></label
		>
		<button class="btn btn-secondary">Apply filter</button>
	</form>
	{#if data.activity}<ActivityList events={data.activity.events} />{:else}<p role="alert">
			{data.error}
		</p>{/if}
	<nav aria-label="Activity pages">
		{#if data.paged}<a href="{resolve('/activity')}?{newest}">Newest activity</a
			>{/if}{#if data.activity?.next}<a href="{resolve('/activity')}?{next}">Older activity</a>{/if}
	</nav>
</section>

<style>
	section {
		max-width: 62rem;
		padding: clamp(1rem, 3vw, 1.5rem);
		display: grid;
		gap: 1rem;
	}
	form,
	nav {
		display: flex;
		align-items: end;
		flex-wrap: wrap;
		gap: 0.75rem;
	}
	label {
		display: grid;
		gap: 0.35rem;
	}
	nav a {
		text-decoration: underline;
	}
</style>
