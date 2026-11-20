import { registerPlugin } from '@capacitor/core';

export type Response<T> = { status: number; data: T; cached?: boolean };
type CaptureState = {
	granted: boolean;
	enabled: boolean;
	packages: string[];
	queued: number;
	rejected?: number;
};

export const native = registerPlugin<{
	status(): Promise<{ signedIn: boolean }>;
	signIn(input: { email: string; password: string }): Promise<void>;
	signOut(): Promise<void>;
	request<T>(input: {
		path: string;
		method?: 'GET' | 'POST';
		body?: object;
		key?: string;
	}): Promise<Response<T>>;
	captureStatus(): Promise<CaptureState>;
	configureCapture(input: { enabled: boolean; packages: string[] }): Promise<void>;
	openNotificationSettings(): Promise<void>;
	sync(): Promise<CaptureState>;
	openWeb(input: { path: string }): Promise<void>;
}>('BuyWise');
