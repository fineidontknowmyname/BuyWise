<script lang="ts">
	import { resolve } from '$app/paths';
	import type { AssistantAnswer, AssistantSource } from '#lib/assistant.js';
	let { answer }: { answer: AssistantAnswer } = $props();
	const labels = {
		profile: 'Financial profile',
		budget: 'Monthly budget',
		goal: 'Goal',
		plan: 'Savings plan',
		decision: 'Saved assessments',
		product: 'Product price history',
		offers: 'Retailer offers'
	};
	function sourceLink(source: AssistantSource) {
		switch (source.kind) {
			case 'profile':
				return resolve('/finances');
			case 'budget':
				return resolve('/budgets');
			case 'plan':
				return resolve('/goals/[id]/plan', { id: source.id });
			case 'product':
				return resolve('/products/[id]', { id: source.id });
			case 'offers':
				return resolve('/products/[id]/offers', { id: source.id });
			default:
				return resolve('/goals/[id]', { id: source.id });
		}
	}
</script>

<div class="evidence">
	<h3>{answer.title}</h3>
	{#if answer.facts.length}
		<h4>BuyWise records</h4>
		<dl>
			{#each answer.facts as item, index (index)}<div>
					<dt>{item.label}</dt>
					<dd>{item.value}</dd>
				</div>{/each}
		</dl>
	{/if}
	<h4>{answer.mode === 'ai_assisted' ? 'AI-assisted explanation' : 'BuyWise explanation'}</h4>
	{#if answer.fallback}
		<p class="muted">
			{answer.fallback === 'not_configured'
				? 'AI is off. The explanation below uses BuyWise rules.'
				: 'AI is temporarily unavailable. The explanation below uses BuyWise rules.'}
		</p>
	{/if}
	<ul>
		{#each answer.explanation as line, index (index)}<li>{line}</li>{/each}
	</ul>
	<nav aria-label="Source records">
		{#each answer.sources as source (source.kind)}<a href={sourceLink(source)}
				>{labels[source.kind]}</a
			>{/each}
	</nav>
</div>

<style>
	.evidence {
		display: grid;
		gap: 0.7rem;
		overflow-wrap: anywhere;
	}
	h3,
	h4,
	p,
	dl,
	ul {
		margin: 0;
	}
	dl {
		display: grid;
		gap: 0.65rem;
		grid-template-columns: repeat(auto-fit, minmax(min(100%, 13rem), 1fr));
	}
	dl div {
		background: var(--surface-secondary);
		border: 1px solid var(--border-subtle);
		border-radius: var(--radius-md);
		padding: 0.65rem;
	}
	dd {
		margin: 0.25rem 0 0;
		font-weight: 600;
	}
	dt,
	.muted {
		color: var(--text-secondary);
	}
	ul {
		padding-left: 1.25rem;
	}
	li + li {
		margin-top: 0.35rem;
	}
	nav {
		display: flex;
		flex-wrap: wrap;
		gap: 0.85rem;
	}
	a {
		text-decoration: underline;
	}
</style>
