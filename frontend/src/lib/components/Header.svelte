<script lang="ts">
	import { page } from '$app/state';
	import ThemeToggle from './ThemeToggle.svelte';
	import { Menu, Search, ShieldCheck } from '@lucide/svelte';

	let {
		ontoggleMobileMenu
	}: {
		ontoggleMobileMenu: () => void;
	} = $props();

	const pageTitles: Record<string, string> = {
		'/': 'Overview',
		'/ai': 'AI Manager',
		'/goals': 'Purchase Goals',
		'/products': 'Products & Prices',
		'/simulator': 'Affordability Simulator',
		'/reports': 'Financial Reports',
		'/finances': 'My finances',
		'/accounts': 'Accounts',
		'/budget': 'Budget & Categories',
		'/categories': 'Budget & Categories',
		'/budgets': 'Budget & Categories',
		'/health': 'System Health'
	};

	const currentTitle = $derived.by(() => {
		const path = page.url.pathname;
		if (pageTitles[path]) return pageTitles[path];
		if (path.startsWith('/goals/')) return 'Goal Decision';
		if (path.startsWith('/products/')) return 'Product Details';
		if (path.startsWith('/accounts/')) return 'Account Details';
		return 'Dashboard';
	});
</script>

<header class="topbar">
	<div class="topbar-left">
		<button
			type="button"
			class="btn-icon mobile-menu-btn"
			onclick={ontoggleMobileMenu}
			aria-label="Toggle navigation menu"
		>
			<Menu size={20} />
		</button>
		<div class="page-title-group">
			<span class="page-title">{currentTitle}</span>
		</div>
	</div>

	<div class="topbar-right">
		<div class="search-wrapper">
			<Search size={16} class="search-icon" />
			<input
				type="search"
				class="search-input"
				placeholder="Search finances..."
				aria-label="Quick search"
			/>
		</div>

		<div class="status-indicator-badge">
			<ShieldCheck size={16} class="status-shield" />
			<span class="status-text">BuyWise Core</span>
		</div>

		<ThemeToggle />
		<form method="POST" action="/sign-out">
			<button type="submit" class="btn-icon">Sign out</button>
		</form>
	</div>
</header>

<style>
	.topbar-left {
		display: flex;
		align-items: center;
		gap: 16px;
	}

	.mobile-menu-btn {
		display: none;
	}

	.page-title {
		font-size: 1.18rem;
		font-weight: 700;
		color: var(--foreground);
		letter-spacing: -0.02em;
	}

	.topbar-right {
		display: flex;
		align-items: center;
		gap: 14px;
	}

	.search-wrapper {
		position: relative;
		display: flex;
		align-items: center;
	}

	:global(.search-icon) {
		position: absolute;
		left: 12px;
		color: var(--foreground-muted);
		pointer-events: none;
	}

	.search-input {
		height: 38px;
		padding: 0 14px 0 36px;
		border-radius: 10px;
		background-color: var(--surface-secondary);
		border: 1px solid var(--border);
		font-size: 0.85rem;
		color: var(--foreground);
		width: 200px;
		transition: all 0.2s ease;
	}

	.search-input:focus {
		outline: none;
		border-color: var(--primary);
		width: 250px;
		background-color: var(--surface);
		box-shadow: 0 0 0 3px var(--primary-subtle);
	}

	.status-indicator-badge {
		display: flex;
		align-items: center;
		gap: 6px;
		padding: 6px 12px;
		border-radius: 8px;
		background-color: var(--surface-secondary);
		border: 1px solid var(--border);
		font-size: 0.8rem;
		font-weight: 600;
		color: var(--foreground-secondary);
	}

	:global(.status-shield) {
		color: var(--success);
	}

	@media (max-width: 1023px) {
		.mobile-menu-btn {
			display: inline-flex;
		}
	}

	@media (max-width: 768px) {
		.search-wrapper,
		.status-indicator-badge {
			display: none;
		}
	}
</style>
