<script lang="ts">
	import { page } from '$app/state';
	import { LogOut } from '@lucide/svelte';
	import {
		primaryNavigation,
		supportingNavigation,
		moneyLinks,
		isMoneyPath,
		isNavigationActive,
		isMoneyLinkActive
	} from '#lib/navigation.js';

	let { isOpen = false }: { isOpen?: boolean } = $props();
</script>

<aside id="app-navigation" class="sidebar-container" class:open={isOpen} inert={!isOpen}>
	<p class="navigation-heading">Your workspace</p>
	<nav aria-label="Main navigation">
		<ul class="nav-list">
			{#each primaryNavigation as item (item.href)}
				<li>
					<a
						href={item.href}
						class="nav-link"
						class:active={isNavigationActive(page.url.pathname, item.href)}
						aria-current={page.url.pathname === item.href ? 'page' : undefined}
					>
						<item.icon size={19} aria-hidden="true" />{item.label}
					</a>
					{#if item.shortLabel === 'My Money' && isMoneyPath(page.url.pathname)}
						<ul class="subnav">
							{#each moneyLinks as child (child.href)}
								<li>
									<a
										href={child.href}
										class="nav-link"
										class:active={isMoneyLinkActive(page.url.pathname, child.href)}
										aria-current={page.url.pathname === child.href ? 'page' : undefined}
										>{child.label}</a
									>
								</li>
							{/each}
						</ul>
					{/if}
				</li>
			{/each}
		</ul>
	</nav>
	<div class="supporting-navigation">
		<p class="navigation-heading">Tools & preferences</p>
		<nav aria-label="Tools and preferences">
			<ul class="nav-list">
				{#each supportingNavigation as item (item.href)}
					<li>
						<a
							href={item.href}
							class="nav-link"
							class:active={isNavigationActive(page.url.pathname, item.href)}
							aria-current={page.url.pathname === item.href ? 'page' : undefined}
						>
							<item.icon size={19} aria-hidden="true" />{item.label}
						</a>
					</li>
				{/each}
			</ul>
		</nav>
	</div>
	<form method="POST" action="/sign-out">
		<button type="submit" class="nav-link"><LogOut size={19} aria-hidden="true" />Sign out</button>
	</form>
</aside>

<style>
	.navigation-heading {
		padding: var(--space-2) var(--space-3);
		color: var(--foreground-muted);
		font-size: var(--text-caption);
		font-weight: 600;
	}
	.nav-list,
	.subnav {
		display: grid;
		gap: var(--space-1);
		list-style: none;
	}
	.nav-link {
		display: flex;
		align-items: center;
		gap: var(--space-3);
		min-height: var(--control-height);
		padding: var(--space-2) var(--space-3);
		border: 1px solid transparent;
		border-radius: var(--radius-control);
		color: var(--foreground-secondary);
		font-size: var(--text-small);
		text-align: left;
		transition:
			background-color var(--transition-fast),
			color var(--transition-fast);
	}
	.nav-link:hover {
		background: var(--surface-hover);
		color: var(--foreground);
	}
	.nav-link.active {
		background: var(--surface);
		color: var(--foreground);
		border-color: var(--border);
		box-shadow: inset 3px 0 var(--brand-dot);
		font-weight: 600;
	}
	.nav-link :global(svg) {
		flex: none;
	}
	.subnav {
		margin: var(--space-1) 0 var(--space-2) var(--space-4);
		padding-inline-start: var(--space-2);
		border-inline-start: 1px solid var(--border);
	}
	.supporting-navigation {
		margin-top: var(--space-6);
	}
	form {
		margin-top: auto;
		padding-top: var(--space-6);
	}
	form button {
		width: 100%;
	}
</style>
