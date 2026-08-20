/** Kennel launches with one English product surface. */
export const APP_LOCALES = ["en"] as const;

// Kept as a permissive IPC compatibility type: persisted pre-launch locale
// values are normalized to English by coerceLocale and are never selected.
export type AppLocale = string;

export const DEFAULT_LOCALE: AppLocale = "en";

export interface UiSettings {
	locale: AppLocale;
}

export const DEFAULT_UI_SETTINGS: UiSettings = { locale: DEFAULT_LOCALE };

/** Normalize an unknown value to a supported UI locale. */
export function coerceLocale(raw: unknown): AppLocale {
	if (typeof raw === "string" && (APP_LOCALES as readonly string[]).includes(raw)) {
		return raw as AppLocale;
	}
	return DEFAULT_LOCALE;
}

/** Normalize unknown persisted or IPC data to the supported UI-settings schema. */
export function coerceUiSettings(raw: unknown): UiSettings {
	const locale =
		typeof raw === "object" && raw !== null ? coerceLocale((raw as Record<string, unknown>).locale) : DEFAULT_LOCALE;
	return { locale };
}
