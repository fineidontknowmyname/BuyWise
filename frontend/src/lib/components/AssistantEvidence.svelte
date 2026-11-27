<script lang="ts">
	import { resolve } from '$app/paths';
	import type { AssistantAnswer, AssistantSource } from '#lib/assistant.js';
	let { answer }: { answer: AssistantAnswer } = $props();
	const labels = {
		profile: 'Money plan',
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
				return resolve('/accounts');
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
	{#if answer.fallback}
		<p class="muted">
			{answer.fallback === 'not_configured'
				? 'A summary from your saved BuyWise records.'
				: 'AI is temporarily unavailable. Here is a summary from your saved records.'}
		</p>
	{/if}
	<ul>
		{#each answer.explanation as line, index (index)}<li>{line}</li>{/each}
	</ul>
	{#if answer.facts.length}<details class="supporting-figures">
			<summary>View supporting figures <span>{answer.facts.length}</span></summary>
			<dl>
				{#each answer.facts as item, index (index)}<div>
						<dt>{item.label}</dt>
						<dd>{item.value}</dd>
					</div>{/each}
			</dl>
		</details>{/if}
	<nav aria-label="Source records">
		{#each answer.sources as source (source.kind + sourceLink(source))}<a href={sourceLink(source)}
				>{labels[source.kind]}</a
			>{/each}
	</nav>
</div>

<style>
	.evidence {
		display: grid;
		gap: 1rem;
		overflow-wrap: anywhere;
	}
	h3,
	p,
	dl,
	ul {
		margin: 0;
	}
	h3 {
		font-size: 17px;
		font-weight: 600;
		letter-spacing: -0.025em;
		line-height: 1.4;
	}
	summary {
		display: flex;
		align-items: center;
		gap: 8px;
		cursor: pointer;
		font-size: 12px;
		font-weight: 500;
		color: var(--foreground-secondary);
		padding-block: 10px;
	}
	summary span {
		font-size: 10px;
		border: 1px solid var(--border);
		border-radius: 5px;
		padding: 0 5px;
	}
	.supporting-figures {
		border-block: 1px solid var(--border);
	}
	.supporting-figures[open] dl {
		padding-bottom: 16px;
	}
	dl {
		display: grid;
		gap: 0.65rem;
		grid-template-columns: repeat(auto-fit, minmax(min(100%, 13rem), 1fr));
	}
	dl div {
		background: var(--surface-secondary);
		border: 1px solid var(--border);
		border-radius: var(--radius-md);
		padding: 0.65rem;
	}
	dd {
		margin: 0.25rem 0 0;
		font-weight: 600;
		font-size: 14px;
		font-variant-numeric: tabular-nums;
	}
	dt,
	.muted {
		color: var(--foreground-secondary);
		font-size: 12px;
	}
	ul {
		padding-left: 1.25rem;
		font-size: 14px;
		line-height: 1.75;
	}
	li + li {
		margin-top: 0.35rem;
	}
	nav {
		display: flex;
		flex-wrap: wrap;
		gap: 0.5rem;
	}
	a {
		text-decoration: none;
		font-size: 11px;
		font-weight: 500;
		color: var(--foreground-secondary);
		padding: 5px 10px;
		border: 1px solid var(--border);
		border-radius: 8px;
	}
	a:hover {
		color: var(--foreground);
		border-color: var(--foreground-secondary);
	}
	a:focus-visible,
	summary:focus-visible {
		outline: 2px solid var(--brand-dot);
		outline-offset: 3px;
	}
</style>
