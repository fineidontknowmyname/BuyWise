const indiaZone = 'Asia/Kolkata';

export function todayInIndia(now: Date = new Date()): string {
	return new Intl.DateTimeFormat('en-CA', {
		timeZone: indiaZone,
		year: 'numeric',
		month: '2-digit',
		day: '2-digit'
	}).format(now);
}

export function currentMonth(now: Date = new Date()): string {
	return todayInIndia(now).slice(0, 7);
}

export function marketDate(now = new Date()): string {
	return new Intl.DateTimeFormat('en-CA', {
		timeZone: 'Asia/Kolkata',
		dateStyle: 'short'
	}).format(now);
}
