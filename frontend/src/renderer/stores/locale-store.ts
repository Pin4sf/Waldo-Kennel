// Compatibility shim for callers compiled against pre-launch settings. Kennel
// no longer loads, detects, persists, or exposes a selectable UI language.
import { create } from "zustand";

type LocaleState = {
	locale: string;
	loaded: boolean;
	saving: boolean;
	saveError: boolean;
	load: () => Promise<void>;
	setLocale: (_locale: string) => Promise<void>;
};

export const useLocaleStore = create<LocaleState>((set) => ({
	locale: "en",
	loaded: true,
	saving: false,
	saveError: false,
	load: async () => {},
	setLocale: async () => set({ locale: "en", loaded: true, saving: false, saveError: false }),
}));

export function useLocale(): string {
	return useLocaleStore((state) => state.locale);
}
