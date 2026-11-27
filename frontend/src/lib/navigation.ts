import { resolve } from '$app/paths';
import {
	Activity,
	Calculator,
	LayoutDashboard,
	Settings,
	ShoppingBag,
	Sparkles,
	Target,
	Wallet
} from '@lucide/svelte';

export const moneyLinks = [
	{ href: resolve('/accounts'), label: 'Money plan & accounts' },
	{ href: resolve('/expenses'), label: 'Expenses' },
	{ href: resolve('/budget'), label: 'Spending & categories' },
	{ href: resolve('/finances'), label: 'Transfers & reserves' },
	{ href: resolve('/reports'), label: 'Reports' }
];

export const primaryNavigation = [
	{ href: resolve('/'), label: 'Overview', shortLabel: 'Overview', icon: LayoutDashboard },
	{ href: resolve('/goals'), label: 'Purchase goals', shortLabel: 'Goals', icon: Target },
	{ href: resolve('/accounts'), label: 'My Money', shortLabel: 'My Money', icon: Wallet },
	{
		href: resolve('/products'),
		label: 'Products & prices',
		shortLabel: 'Products',
		icon: ShoppingBag
	},
	{ href: resolve('/activity'), label: 'Activity', shortLabel: 'Activity', icon: Activity }
];

export const supportingNavigation = [
	{ href: resolve('/ai'), label: 'AI Assist', icon: Sparkles },
	{ href: resolve('/simulator'), label: 'Reverse EMI calculator', icon: Calculator },
	{ href: resolve('/profile'), label: 'Settings & profile', icon: Settings }
];

export function isCurrentPath(pathname: string, href: string): boolean {
	const current = pathname.replace(/\/+$/, '') || '/';
	const target = href.replace(/\/+$/, '') || '/';
	return current === target || (target !== '/' && current.startsWith(target + '/'));
}

export function isMoneyLinkActive(pathname: string, href: string): boolean {
	return (
		isCurrentPath(pathname, href) ||
		(href === resolve('/budget') &&
			(isCurrentPath(pathname, resolve('/budgets')) ||
				isCurrentPath(pathname, resolve('/categories'))))
	);
}

export function isMoneyPath(pathname: string): boolean {
	return moneyLinks.some((item) => isMoneyLinkActive(pathname, item.href));
}

export function isNavigationActive(pathname: string, href: string): boolean {
	return href === resolve('/accounts') ? isMoneyPath(pathname) : isCurrentPath(pathname, href);
}
