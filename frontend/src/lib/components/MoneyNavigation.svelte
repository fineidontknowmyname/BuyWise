<script lang="ts">
	import { page } from '$app/state';
	import { moneyLinks, isMoneyLinkActive } from '#lib/navigation.js';
</script>

<nav class="money-navigation" aria-label="My Money sections">
	{#each moneyLinks as item (item.href)}
		<a
			href={item.href}
			class:active={isMoneyLinkActive(page.url.pathname, item.href)}
			aria-current={isMoneyLinkActive(page.url.pathname, item.href) ? 'page' : undefined}
			>{item.label}</a
		>
	{/each}
</nav>

<style>
	.money-navigation {
		display: flex;
		gap: var(--space-2);
		flex-wrap: wrap;
		margin-bottom: var(--space-6);
		padding-bottom: var(--space-4);
		border-bottom: 1px solid var(--border);
	}
	a {
		display: inline-flex;
		align-items: center;
		min-height: var(--control-height);
		padding: var(--space-2) var(--space-3);
		border: 1px solid transparent;
		border-radius: var(--radius-control);
		font-size: var(--text-small);
		color: var(--foreground-secondary);
	}
	a:hover {
		background: var(--surface-hover);
		color: var(--foreground);
	}
	a.active {
		background: var(--surface);
		border-color: var(--border);
		color: var(--foreground);
		font-weight: 600;
		box-shadow: inset 0 -2px var(--brand-dot);
	}
</style>
