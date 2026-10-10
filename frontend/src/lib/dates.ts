const indiaZone = 'Asia/Kolkata';

export function todayInIndia(now: Date = new Date()): string {
	return new Intl.DateTimeFormat('en-CA', {
		timeZone: indiaZone,
		year: 'numeric',
		month: '2-digit',
		day: '2-digit'
	}).format(now);
}
