<script lang="ts">
	import { resolve } from '$app/paths';
	import { formatRupees } from '#lib/money.js';
	import type { ActivityEvent } from '#lib/workspace.js';
	let { events }: { events: ActivityEvent[] } = $props();
	const clock = new Intl.DateTimeFormat('en-IN', {
		dateStyle: 'medium',
		timeStyle: 'short',
		timeZone: 'Asia/Kolkata'
	});
</script>

{#if !events.length}<p>No activity recorded here yet.</p>{:else}
	<ul>
		{#each events as event (event.key)}
			<li>
				<div>
					{#if event.resource === 'account'}
						<a
							href="{resolve('/accounts/[id]', {
								id: event.resourceId
							})}#transaction-{event.key.split(':')[1]}">{event.title}</a
						>
					{:else if event.resource === 'plan'}
						<a href={resolve('/goals/[id]/plan', { id: event.resourceId })}>{event.title}</a>
					{:else if event.resource === 'goal'}
						<a
							href="{resolve('/goals/[id]', { id: event.resourceId })}#{event.kind === 'decisions'
								? 'decision-history'
								: event.key.startsWith('allocation:')
									? 'allocation-history'
									: 'purchase-complete'}">{event.title}</a
						>
					{:else if event.resource === 'product'}
						<a href={resolve('/products/[id]', { id: event.resourceId })}>{event.title}</a>
					{:else if event.resource === 'offers'}
						<a href={resolve('/products/[id]/offers', { id: event.resourceId })}>{event.title}</a>
					{:else}
						<a href={resolve('/finances')}>{event.title}</a>
					{/if}
					<p>{event.detail}</p>
					<small
						>{event.kind === 'ledger' ? 'Ledger' : 'History'} · {clock.format(
							new Date(event.at)
						)}</small
					>
				</div>
				{#if event.paise !== null}<strong>{formatRupees(event.paise)}</strong>{/if}
			</li>
		{/each}
	</ul>
{/if}

<style>
	ul {
		list-style: none;
		margin: 0;
		padding: 0;
	}
	li {
		display: flex;
		justify-content: space-between;
		gap: 1rem;
		border-bottom: 1px solid var(--border-subtle);
		padding: 0.7rem 0;
		overflow-wrap: anywhere;
	}
	p {
		margin: 0.15rem 0;
	}
	small {
		color: var(--text-secondary);
	}
	a {
		font-weight: 600;
		text-decoration: underline;
	}
	strong {
		white-space: nowrap;
	}
</style>
