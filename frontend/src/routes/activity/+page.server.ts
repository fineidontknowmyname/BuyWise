import { apiRequest } from '#lib/server/api.js';
import { apiUrl } from '#lib/server/config.js';
import type { ActivityPage } from '#lib/workspace.js';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ fetch, url }) => {
	const query = new URLSearchParams();
	for (const field of ['kind', 'before']) {
		const value = url.searchParams.get(field);
		if (value) query.set(field, value);
	}
	const result = await apiRequest<ActivityPage>(fetch, apiUrl(), `/activity?${query}`);
	return {
		activity: result.ok ? result.data : null,
		error: result.ok ? null : result.error.message,
		kind: query.get('kind') ?? '',
		paged: query.has('before')
	};
};
