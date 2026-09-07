/**
 * Two different questions the renderer used to answer with one flag.
 *
 * "Am I outside an Electron shell?" and "should I use demo fixtures?" are not
 * the same question, but VITE_NO_ELECTRON answered both. That made the dev
 * server's KENNEL_DEV_API_TARGET proxy — which exists specifically so the
 * renderer can be driven against a real daemon from a browser — unreachable:
 * the flag that enabled browser mode also forced fixtures, so nothing ever
 * called the proxy.
 */

/** Running outside an Electron shell, so the preload bridge is stubbed. */
export const runsOutsideElectron = import.meta.env.VITE_NO_ELECTRON === "1";

/**
 * Browser preview pointed at a REAL daemon through the dev server's proxy.
 * Opt-in, so `dev:web` keeps its fixture behaviour unchanged.
 */
export const usesLiveDaemonPreview = runsOutsideElectron && import.meta.env.VITE_KENNEL_LIVE_PREVIEW === "1";

/**
 * Serve built-in demo fixtures instead of a daemon. Still the default outside
 * Electron so existing `dev:web` usage and its tests behave exactly as before.
 */
export const usesPreviewWorkspaceData = runsOutsideElectron && !usesLiveDaemonPreview;

export const usesWaldoUiPreview =
	import.meta.env.VITE_NO_ELECTRON === "1" ||
	import.meta.env.VITE_WALDO_UI_PREVIEW === "1";
