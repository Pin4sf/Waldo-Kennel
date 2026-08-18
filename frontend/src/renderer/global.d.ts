import type { AoBridge } from "../preload";

declare global {
	interface Window {
		kennel?: AoBridge;
	}

	interface ImportMetaEnv {
		readonly VITE_KENNEL_POSTHOG_KEY?: string;
		readonly VITE_KENNEL_POSTHOG_HOST?: string;
	}
}

export {};
