<script lang="ts">
	import { resolve } from '$app/paths';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();

	const labels = { up: 'Up', down: 'Down', unknown: 'Unknown' };
</script>

<svelte:head>
	<title>Self-EMI</title>
</svelte:head>

<main>
	<h1>Self-EMI</h1>
	<p><a href={resolve('/goals')}>Purchase goals</a></p>
	<h2>System status</h2>
	<ul>
		<li>API: <strong data-testid="api-status">{labels[data.health.api]}</strong></li>
		<li>
			Database: <strong data-testid="database-status">{labels[data.health.database]}</strong>
		</li>
	</ul>
</main>
