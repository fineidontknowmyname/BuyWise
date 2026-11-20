<script lang="ts">
	import { page } from '$app/state';
	import ThemeToggle from './ThemeToggle.svelte';
	import { Menu, ShieldCheck } from '@lucide/svelte';

	let {
		ontoggleMobileMenu
	}: {
		ontoggleMobileMenu: () => void;
	} = $props();

	const pageTitles: Record<string, string> = {
		'/': 'Overview',
		'/ai': 'Financial Assistant',
		'/activity': 'Activity',
		'/profile': 'Profile',
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
		.status-indicator-badge {
			display: none;
		}
	}
</style>
