<script lang="ts">
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import {
		LayoutDashboard,
		Sparkles,
		Target,
		ShoppingBag,
		Calculator,
		Wallet,
		UserRound,
		Activity,
		LogOut
	} from '@lucide/svelte';

	let {
		isOpen = false,
		onfollow
	}: {
		isOpen?: boolean;
		onfollow?: () => void;
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
		return current === href || (href !== '/' && current.startsWith(href + '/'));
	}
</script>

<aside id="app-navigation" class="sidebar-container" class:open={isOpen} inert={!isOpen}>
	<a href={resolve('/')} class="sidebar-brand" onclick={onfollow}>
		BuyWise<span>.</span>
	</a>
	<nav aria-label="Main navigation">
		<ul class="nav-list">
			{#each navItems as item (item.href)}
				{@const active = isActive(item.href, item.exact)}
				<li>
					<a
						href={item.href}
						class="nav-link"
						class:active
						aria-current={active ? 'page' : undefined}
						onclick={onfollow}
					>
						<span class="nav-icon"><item.icon size={16} /></span>
						{item.label}
					</a>
					{#if item.children && (active || item.children.some((child) => isActive(child.href)))}
						<ul class="subnav">
							{#each item.children as child (child.href)}
								<li>
									<a
										class="nav-link"
										class:active={isActive(child.href)}
										aria-current={isActive(child.href) ? 'page' : undefined}
										href={child.href}
										onclick={onfollow}
									>
										{child.label}
									</a>
								</li>
							{/each}
						</ul>
					{/if}
				</li>
			{/each}
		</ul>
	</nav>

	<form method="POST" action="/sign-out">
		<button type="submit" class="nav-link">
			<span class="nav-icon"><LogOut size={16} /></span>
			Sign out
		</button>
	</form>
</aside>

<style>
	.sidebar-brand {
		display: inline-flex;
		padding: 3px 10px 16px;
		font: 700 1.35rem var(--font-mono);
	}
	.sidebar-brand span {
		color: var(--brand-dot);
	}
	.nav-list,
	.subnav {
		display: grid;
		gap: 3px;
		list-style: none;
	}
	.subnav {
		padding: 3px 0 3px 36px;
	}
	.nav-link {
		display: flex;
		align-items: center;
		gap: 10px;
		min-height: 40px;
		padding: 6px 8px;
		border: 1px solid transparent;
		border-radius: 12px;
		color: var(--foreground-secondary);
		text-align: left;
		transition:
			background-color 0.2s,
			color 0.2s,
			border-color 0.2s;
	}
	.nav-link:hover,
	.nav-link.active {
		background: var(--surface);
		color: var(--foreground);
		border-color: var(--border);
	}
	.nav-link.active {
		font-weight: 600;
	}
	.nav-icon {
		display: grid;
		place-items: center;
		width: 29px;
		height: 29px;
		flex: none;
		border: 1px solid #333;
		border-radius: 9px;
		background: #0d0d0d;
		color: var(--accent);
	}
	form {
		margin-top: auto;
		padding-top: 18px;
	}
	form button {
		width: 100%;
	}
</style>
