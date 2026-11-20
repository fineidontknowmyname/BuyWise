import { apiRequest } from '#lib/server/api.js';
import { apiUrl } from '#lib/server/config.js';
import type { Preferences } from '#lib/workspace.js';
import type { LayoutServerLoad } from './$types';

export const load: LayoutServerLoad = async ({ locals, fetch, depends }) => {
	depends('buywise:preferences');
	if (!locals.userId) return { workspacePreferences: null };
	const result = await apiRequest<Preferences>(fetch, apiUrl(), '/profile/preferences');
	return { workspacePreferences: result.ok ? result.data : null };
};
