export type ToastType = 'success' | 'error' | 'info';

export interface ToastItem {
	id: string;
	message: string;
	type: ToastType;
}

class ToastStore {
	items = $state<ToastItem[]>([]);

	show(message: string, type: ToastType = 'info', durationMs = 3500) {
		const id = Math.random().toString(36).substring(2, 9);
		const toast: ToastItem = { id, message, type };
		this.items = [...this.items, toast];

		setTimeout(() => {
			this.dismiss(id);
		}, durationMs);
	}

	success(message: string) {
		this.show(message, 'success');
	}

	error(message: string) {
		this.show(message, 'error', 4500);
	}

	info(message: string) {
		this.show(message, 'info');
	}

	dismiss(id: string) {
		this.items = this.items.filter((item) => item.id !== id);
	}
}

export const toast = new ToastStore();
