<script lang="ts">
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import Logo from './Logo.svelte';
	import {
		LayoutDashboard,
		Sparkles,
		Target,
		ShoppingBag,
		Calculator,
		Wallet,
		UserRound,
		Activity,
		X
	} from '@lucide/svelte';

	let {
		isOpen = false,
		onclose
	}: {
		isOpen?: boolean;
		onclose?: () => void;
	} = $props();

	const financeLinks = [
		{ href: resolve('/accounts'), label: 'Accounts' },
		{ href: resolve('/expenses'), label: 'Expenses' },
		{ href: resolve('/budget'), label: 'Budget & Categories' },
		{ href: resolve('/reports'), label: 'Reports' }
	];
	const navItems = [
		{ href: resolve('/'), label: 'Overview', icon: LayoutDashboard, exact: true },
		{ href: resolve('/goals'), label: 'Purchase Goals', icon: Target },
		{ href: resolve('/finances'), label: 'My Finances', icon: Wallet, children: financeLinks },
		{ href: resolve('/products'), label: 'Products & Prices', icon: ShoppingBag },
		{ href: resolve('/simulator'), label: 'Reverse EMI', icon: Calculator },
		{ href: resolve('/activity'), label: 'Activity', icon: Activity },
		{ href: resolve('/ai'), label: 'AI Assist', icon: Sparkles },
		{ href: resolve('/profile'), label: 'Profile', icon: UserRound }
	];

	function isActive(href: string, exact = false) {
		const current = page.url.pathname;
		if (exact) return current === href;
		if (
			href === resolve('/budget') &&
			(current === '/budget' || current === '/budgets' || current === '/categories')
		)
			return true;
		return current === href || (href !== '/' && current.startsWith(href));
	}
</script>

<!-- Backdrop on mobile when drawer is open -->
{#if isOpen}
	<div class="sidebar-backdrop" role="presentation" onclick={onclose}></div>
{/if}

<aside class="sidebar-container" class:open={isOpen}>
	<div class="sidebar-top">
		<a href={resolve('/')} class="brand-link" onclick={onclose}>
			<Logo size="md" showWordmark={true} />
		</a>
		{#if onclose}
			<button type="button" class="mobile-close-btn" onclick={onclose} aria-label="Close menu">
				<X size={20} />
			</button>
		{/if}
	</div>

	<nav class="sidebar-nav">
		<div class="nav-section-title">Navigation</div>
		<ul class="nav-list">
			{#each navItems as item (item.href)}
				{@const active = isActive(item.href, item.exact)}
				<li>
					<a href={item.href} class="nav-link" class:active onclick={onclose}>
						<span class="nav-icon">
							<item.icon size={19} />
						</span>
						<span class="nav-label">{item.label}</span>
						{#if active}
							<span class="active-indicator"></span>
						{/if}
					</a>
					{#if item.children && (active || item.children.some((child) => isActive(child.href)))}<ul
							class="subnav"
						>
							{#each item.children as child (child.href)}<li>
									<a
										class="nav-link"
										class:active={isActive(child.href)}
										href={child.href}
										onclick={onclose}>{child.label}</a
									>
								</li>{/each}
						</ul>{/if}
				</li>
			{/each}
		</ul>
	</nav>

	<div class="sidebar-footer">
		<a href={resolve('/')} class="health-pill" onclick={onclose}>
			<Activity size={16} />
			<span>System Health: Active</span>
		</a>
	</div>
</aside>

<style>
	.subnav {
		list-style: none;
		padding: 0 0 0 1rem;
		margin: 0;
	}

	.sidebar-backdrop {
		position: fixed;
		inset: 0;
		background-color: rgba(11, 18, 32, 0.5);
		backdrop-filter: blur(2px);
		z-index: 39;
	}

	.sidebar-top {
		height: 70px;
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 0 24px;
		border-bottom: 1px solid var(--sidebar-border);
	}

	.brand-link {
		display: flex;
		align-items: center;
		text-decoration: none;
	}

	.mobile-close-btn {
		display: none;
		color: var(--foreground-secondary);
		padding: 6px;
		border-radius: 8px;
	}

	.sidebar-nav {
		flex: 1;
		padding: 24px 16px;
		overflow-y: auto;
	}

	.nav-section-title {
		font-size: 0.72rem;
		font-weight: 600;
		text-transform: uppercase;
		letter-spacing: 0.08em;
		color: var(--foreground-muted);
		padding: 0 12px 10px 12px;
	}

	.nav-list {
		list-style: none;
		display: flex;
		flex-direction: column;
		gap: 4px;
	}

	.nav-link {
		display: flex;
		align-items: center;
		gap: 12px;
		padding: 10px 14px;
		border-radius: 10px;
		color: var(--foreground-secondary);
		font-size: 0.92rem;
		font-weight: 500;
		transition: all 0.15s ease;
		position: relative;
	}

	.nav-link:hover {
		background-color: var(--surface-hover);
		color: var(--foreground);
	}

	.nav-link.active {
		background-color: var(--sidebar-active-bg);
		color: var(--sidebar-active-text);
		font-weight: 600;
	}

	.nav-icon {
		display: flex;
		align-items: center;
		justify-content: center;
		flex-shrink: 0;
	}

	.active-indicator {
		position: absolute;
		right: 12px;
		width: 6px;
		height: 6px;
		border-radius: 50%;
		background-color: var(--primary);
	}

	.sidebar-footer {
		padding: 18px 20px;
		border-top: 1px solid var(--sidebar-border);
	}

	.health-pill {
		display: flex;
		align-items: center;
		gap: 8px;
		font-size: 0.8rem;
		font-weight: 500;
		color: var(--foreground-secondary);
		padding: 8px 12px;
		border-radius: 8px;
		background-color: var(--surface-secondary);
		border: 1px solid var(--border);
		transition: all 0.15s ease;
	}

	.health-pill:hover {
		color: var(--primary);
		border-color: var(--primary);
	}

	@media (max-width: 1023px) {
		.mobile-close-btn {
			display: flex;
		}
	}
</style>
