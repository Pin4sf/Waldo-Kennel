// Types for the main-process settings module.
//
// The module itself is plain JavaScript, because it runs in Electron's main
// process where there is no build step. This declaration exists so the renderer
// test that compares the two sides' defaults — `src/island/settings.test.ts` —
// is type-checked rather than silently `any`. It describes only what that test
// and the main process actually use.

/** One field's description in the schema. */
export type SettingsField =
	| { kind: "boolean"; fallback: boolean }
	| { kind: "integer"; fallback: number; min: number; max: number }
	| { kind: "choice"; fallback: string; values: readonly string[] };

export type SettingsSchema = Readonly<Record<string, Readonly<Record<string, SettingsField>>>>;

export interface StoredSettings {
	version: number;
	[section: string]: unknown;
}

export const SETTINGS_VERSION: number;
export const SETTINGS_SCHEMA: SettingsSchema;

export function defaultSettings(): StoredSettings;
export function normalizeSettings(input: unknown): StoredSettings;
export function mergeSettings(current: unknown, patch: unknown): StoredSettings;
export function sameSettings(left: unknown, right: unknown): boolean;
export function settingsFilePath(userDataPath: string): string;

export interface SettingsStore {
	readonly filePath: string;
	current(): StoredSettings;
	load(): Promise<StoredSettings>;
	update(patch: unknown): Promise<StoredSettings>;
	reset(): Promise<StoredSettings>;
	onChange(listener: (settings: StoredSettings) => void): () => void;
	flush(): Promise<void>;
}

export function createSettingsStore(options: {
	fs: unknown;
	userDataPath: string;
}): SettingsStore;
