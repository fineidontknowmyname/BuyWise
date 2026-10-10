import { describe, expect, it } from 'vitest';
import { formatRupees, parseRupees, parseSignedRupees } from './money';

describe('parseRupees', () => {
	it('converts rupees to whole paise without floating point', () => {
		expect(parseRupees('600')).toBe(60000);
		expect(parseRupees('600.5')).toBe(60050);
		expect(parseRupees('600.05')).toBe(60005);
		expect(parseRupees('0.01')).toBe(1);
		expect(parseRupees('0.29')).toBe(29);
		expect(parseRupees('19.99')).toBe(1999);
		expect(parseRupees('1,23,456.50')).toBe(12345650);
		expect(parseRupees('  42 ')).toBe(4200);
	});

	it('accepts zero, leaving the "must be positive" rule to the API', () => {
		expect(parseRupees('0')).toBe(0);
	});

	it('rejects anything that is not a plain amount', () => {
		for (const bad of ['', ' ', 'abc', '1.234', '1.', '.5', '-5', '1e3', '₹5', '5 0', '1..2']) {
			expect(parseRupees(bad), bad).toBeNull();
		}
	});

	it('enforces the same upper limit as the API', () => {
		expect(parseRupees('10000000000')).toBe(1_000_000_000_000);
		expect(parseRupees('10000000000.01')).toBeNull();
		expect(parseRupees('999999999999')).toBeNull();
	});
});

describe('parseSignedRupees', () => {
	it('allows a leading minus for opening balances', () => {
		expect(parseSignedRupees('-2500.50')).toBe(-250050);
		expect(parseSignedRupees('2500.50')).toBe(250050);
		expect(parseSignedRupees('-')).toBeNull();
		expect(parseSignedRupees('--5')).toBeNull();
	});
});

describe('formatRupees', () => {
	it('shows rupees with Indian digit grouping and two decimals', () => {
		expect(formatRupees(0)).toBe('₹0.00');
		expect(formatRupees(5)).toBe('₹0.05');
		expect(formatRupees(60050)).toBe('₹600.50');
		expect(formatRupees(12345650)).toBe('₹1,23,456.50');
	});

	it('shows negatives with a leading minus', () => {
		expect(formatRupees(-250050)).toBe('-₹2,500.50');
		expect(formatRupees(-5)).toBe('-₹0.05');
	});

	it('round-trips with parseRupees', () => {
		for (const paise of [1, 99, 100, 101, 123456789, 1_000_000_000_000]) {
			expect(parseRupees(formatRupees(paise).replace('₹', ''))).toBe(paise);
		}
	});
});
