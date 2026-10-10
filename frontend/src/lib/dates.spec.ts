import { describe, expect, it } from 'vitest';
import { todayInIndia } from './dates';

describe('todayInIndia', () => {
	it('is the next calendar day once UTC passes 18:30', () => {
		expect(todayInIndia(new Date('2026-10-09T20:00:00Z'))).toBe('2026-10-10');
		expect(todayInIndia(new Date('2026-10-31T19:00:00Z'))).toBe('2026-11-01');
	});

	it('is the same calendar day before UTC reaches 18:30', () => {
		expect(todayInIndia(new Date('2026-10-31T18:29:59Z'))).toBe('2026-10-31');
		expect(todayInIndia(new Date('2026-10-09T00:00:00Z'))).toBe('2026-10-09');
	});

	it('handles the year end and leap day', () => {
		expect(todayInIndia(new Date('2026-12-31T19:00:00Z'))).toBe('2027-01-01');
		expect(todayInIndia(new Date('2028-02-28T19:00:00Z'))).toBe('2028-02-29');
	});
});
