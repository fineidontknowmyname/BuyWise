import { fail } from '@sveltejs/kit';
import { apiRequest } from '#lib/server/api.js';
import { apiUrl } from '#lib/server/config.js';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ fetch }) => {
	const result = await apiRequest<{ email: string }>(fetch, apiUrl(), '/auth/session');
	return { email: result.ok ? result.data.email : null };
};

export const actions: Actions = {
	preferences: async ({ fetch, request }) => {
		const form = await request.formData();
		const result = await apiRequest(fetch, apiUrl(), '/profile/preferences', {
			method: 'PUT',
			body: JSON.stringify({
				displayName: String(form.get('displayName') ?? ''),
				priceUpdates: form.get('priceUpdates') === 'yes',
				reducedMotion: form.get('reducedMotion') === 'yes'
			})
		});
		return result.ok
			? { message: 'Preferences saved.' }
			: fail(result.status, { error: result.error.message });
	},
	revoke: async ({ fetch, request }) => {
		if ((await request.formData()).get('confirm') !== 'yes')
			return fail(400, { error: 'Confirm signing out other sessions.' });
		const result = await apiRequest<{ revoked: number }>(fetch, apiUrl(), '/auth/revoke-others', {
			method: 'POST'
		});
		return result.ok
			? {
					message: result.data.revoked
						? 'Other sessions signed out.'
						: 'No other sessions to sign out.'
				}
			: fail(result.status, { error: result.error.message });
	}
};
