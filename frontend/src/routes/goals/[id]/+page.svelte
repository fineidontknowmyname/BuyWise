<script lang="ts">
	import { resolve } from '$app/paths';
	import ProvenanceBadge from '#lib/ProvenanceBadge.svelte';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();

	const stateLabels = { BUY: 'Buy', WAIT: 'Wait', SAVE_MORE: 'Save more', REASSESS: 'Reassess' };
	const decision = $derived(data.decision);
</script>

<svelte:head>
	<title>{decision.goalName} · Self-EMI</title>
</svelte:head>

<main>
	<p><a href={resolve('/goals')}>All goals</a></p>
	<h1>{decision.goalName}</h1>

	{#if !decision.allInputsReal}
		<p role="note" data-testid="stub-warning">
			This decision uses placeholder data. It is not real advice yet.
		</p>
	{/if}

	<h2>Decision: <span data-testid="decision-state">{stateLabels[decision.state]}</span></h2>
	<p>Rules version: {decision.rulesVersion}</p>

	<h3>Inputs used</h3>
	<table>
		<thead>
			<tr><th>Input</th><th>Value</th><th>Source</th></tr>
		</thead>
		<tbody>
			{#each decision.inputs as input (input.key)}
				<tr>
					<td>{input.label}</td>
					<td>{input.value}</td>
					<td><ProvenanceBadge provenance={input.provenance} /></td>
				</tr>
			{/each}
		</tbody>
	</table>
</main>
