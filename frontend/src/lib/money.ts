export const maxPaise = 1_000_000_000_000;

const rupeeAmount = /^(\d{1,12})(?:\.(\d{1,2}))?$/;
const rupeeFormat = new Intl.NumberFormat('en-IN');

export function parseRupees(input: string): number | null {
	const match = rupeeAmount.exec(input.trim().replaceAll(',', ''));
	if (!match) return null;
	const rupees = Number(match[1]);
	const paise = Number((match[2] ?? '').padEnd(2, '0'));
	const total = rupees * 100 + paise;
	return total <= maxPaise ? total : null;
}

export function formatRupees(paise: number): string {
	const sign = paise < 0 ? '-' : '';
	const absolute = Math.abs(paise);
	const rupees = Math.trunc(absolute / 100);
	const remainder = String(absolute % 100).padStart(2, '0');
	return `${sign}₹${rupeeFormat.format(rupees)}.${remainder}`;
}

export function parseSignedRupees(input: string): number | null {
	const trimmed = input.trim();
	if (trimmed.startsWith('-')) {
		const value = parseRupees(trimmed.slice(1));
		return value === null ? null : -value;
	}
	return parseRupees(trimmed);
}
