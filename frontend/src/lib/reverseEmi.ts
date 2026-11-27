import { parseRupees } from './money.js';

export function addSavingMonths(date: string, months: number): string | null {
	if (
		!/^\d{4}-\d{2}-\d{2}$/.test(date) ||
		!Number.isSafeInteger(months) ||
		months < 0 ||
		months > 1200
	)
		return null;
	const [year, month, day] = date.split('-').map(Number);
	const original = new Date(Date.UTC(year, month - 1, day));
	if (original.toISOString().slice(0, 10) !== date) return null;
	const target = new Date(Date.UTC(year, month - 1 + months, 1));
	if (target.getUTCFullYear() > 9999) return null;
	const last = new Date(
		Date.UTC(target.getUTCFullYear(), target.getUTCMonth() + 1, 0)
	).getUTCDate();
	target.setUTCDate(Math.min(day, last));
	return target.toISOString().slice(0, 10);
}

export function reverseEmiDraft(
	target: string,
	saved: string,
	contribution: string,
	firstDate: string
) {
	const targetPaise = parseRupees(target);
	const savedPaise = parseRupees(saved || '0');
	const contributionPaise = parseRupees(contribution);
	if (!targetPaise || savedPaise === null)
		return {
			targetPaise: targetPaise ?? 0,
			remainingPaise: 0,
			contributions: null,
			completionDate: null,
			error: 'Enter a purchase target and valid existing savings.'
		};
	const remainingPaise = Math.max(0, targetPaise - savedPaise);
	if (remainingPaise === 0)
		return { targetPaise, remainingPaise, contributions: 0, completionDate: null, error: null };
	if (!contributionPaise)
		return {
			targetPaise,
			remainingPaise,
			contributions: null,
			completionDate: null,
			error: 'Enter a monthly saving amount above zero.'
		};
	const contributions = Math.ceil(remainingPaise / contributionPaise);
	const completionDate = addSavingMonths(firstDate, contributions - 1);
	return {
		targetPaise,
		remainingPaise,
		contributions,
		completionDate,
		error: completionDate ? null : 'Choose a valid date and a plan within 100 years.'
	};
}
