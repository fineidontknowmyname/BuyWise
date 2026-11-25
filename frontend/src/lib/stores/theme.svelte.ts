export type ThemePreference = 'light' | 'dark' | 'system';

class ThemeManager {
	preference = $state<ThemePreference>('system');
	isDark = $state(false);

	constructor() {
		if (typeof window !== 'undefined') {
			const stored = localStorage.getItem('buywise-theme') as ThemePreference | null;
			if (stored === 'light' || stored === 'dark' || stored === 'system') {
				this.preference = stored;
			}
			this.apply();

			const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
			mediaQuery.addEventListener('change', () => {
				if (this.preference === 'system') {
					this.apply();
				}
			});
		}
	}

	set(theme: ThemePreference) {
		this.preference = theme;
		if (typeof window !== 'undefined') {
			if (theme === 'system') {
				localStorage.removeItem('buywise-theme');
			} else {
				localStorage.setItem('buywise-theme', theme);
			}
			this.apply();
		}
	}

	toggle() {
		this.set(this.isDark ? 'light' : 'dark');
	}

	private apply() {
		if (typeof window === 'undefined') return;

		let shouldBeDark: boolean;
		if (this.preference === 'system') {
			shouldBeDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
		} else {
			shouldBeDark = this.preference === 'dark';
		}

		this.isDark = shouldBeDark;
		document.documentElement.setAttribute('data-theme', shouldBeDark ? 'dark' : 'light');
	}
}

export const themeManager = new ThemeManager();
