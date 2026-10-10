import { defineEnvVars } from '@sveltejs/kit/env';

export const defaultApiUrl = 'http://127.0.0.1:8080';

export const variables = defineEnvVars({
	API_URL: {
		description: 'Base URL of the private Go API, called only from the server.',
		schema: (value) => value || defaultApiUrl
	}
});
