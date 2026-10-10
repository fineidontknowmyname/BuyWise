<script lang="ts">
	let { provenance }: { provenance: string } = $props();

	const real = $derived(provenance === 'manual' || provenance === 'calculated');
</script>

<span class="badge" class:real data-testid="provenance">{provenance.toUpperCase()}</span>

<style>
	.badge {
		display: inline-block;
		padding: 0 0.4em;
		border-radius: 0.25em;
		font-size: 0.75em;
		font-weight: 600;
		background: #b45309;
		color: #fff;
	}
	.real {
		background: #15803d;
	}
</style>
