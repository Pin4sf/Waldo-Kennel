import {
	app,
	BrowserWindow,
	dialog,
	globalShortcut,
	ipcMain,
	Menu,
	net,
	screen,
	session,
	type Display,
	type IpcMainInvokeEvent,
	type HeadersReceivedResponse,
	type OnHeadersReceivedListenerDetails,
	type WebContents,
} from "electron";
import { execFile, spawn } from "node:child_process";
import * as nodeFs from "node:fs/promises";
import path from "node:path";
import { pathToFileURL } from "node:url";
import type { DaemonStatus } from "../../shared/daemon-status";
import {
	ISLAND_GET_STATE_CHANNEL,
	ISLAND_OPEN_SETTINGS_CHANNEL,
	ISLAND_SET_VISIBLE_CHANNEL,
	ISLAND_STATE_CHANNEL,
	type IslandVisibilityState,
} from "../../shared/island";
import { createIslandEventStream } from "./event-stream";
// These modules remain the canonical, heavily tested desktop boundary while
// the renderer is shared by the standalone visual lab and the unified app.
// @ts-expect-error Canonical Island desktop modules are JavaScript packages.
import { createHaptics } from "../../../../packages/kennel-island/desktop/haptics.mjs";
// @ts-expect-error Canonical Island desktop modules are JavaScript packages.
import { createKennelService } from "../../../../packages/kennel-island/desktop/kennel-service.mjs";
// @ts-expect-error Canonical Island desktop modules are JavaScript packages.
import { readArtwork, sameTrack } from "../../../../packages/kennel-island/desktop/media-artwork.mjs";
// @ts-expect-error Canonical Island desktop modules are JavaScript packages.
import { focusApplication, isMediaCommand, readMediaActivity, seekMedia, sendMediaCommand } from "../../../../packages/kennel-island/desktop/media-activity.mjs";
// @ts-expect-error Canonical Island desktop modules are JavaScript packages.
import { measureNotch } from "../../../../packages/kennel-island/desktop/notch-measure.mjs";
// @ts-expect-error Canonical Island desktop modules are JavaScript packages.
import { notchGeometryFor, notchOptionsFromSettings } from "../../../../packages/kennel-island/desktop/notch-geometry.mjs";
import { createSettingsStore } from "../../../../packages/kennel-island/desktop/settings.mjs";

const ISLAND_PARTITION = "kennel-island";
const ISLAND_HOST = "island";
const ISLAND_ORIGIN = `app://${ISLAND_HOST}`;
const TOGGLE_ACCELERATOR = "CommandOrControl+`";
const SHORTCUT_LABEL = "Command+`";

const IPC_SET_INTERACTIVE = "kennel-island:set-interactive";
const IPC_SET_FOCUSABLE = "kennel-island:set-focusable";
const IPC_GET_MEDIA_ACTIVITY = "kennel-island:get-media-activity";
const IPC_MEDIA_ACTIVITY_CHANGED = "kennel-island:media-activity-changed";
const IPC_SEND_MEDIA_COMMAND = "kennel-island:send-media-command";
const IPC_FOCUS_MEDIA_APP = "kennel-island:focus-media-app";
const IPC_SEEK_MEDIA = "kennel-island:seek-media";
const IPC_GET_STAGE_GEOMETRY = "kennel-island:get-stage-geometry";
const IPC_STAGE_GEOMETRY_CHANGED = "kennel-island:stage-geometry-changed";
const IPC_RECENTER = "kennel-island:recenter";
const IPC_GET_KENNEL_SNAPSHOT = "kennel-island:get-snapshot";
const IPC_GET_KENNEL_CONVERSATION = "kennel-island:get-conversation";
const IPC_RESOLVE_APPROVAL = "kennel-island:resolve-approval";
const IPC_RESOLVE_INPUT = "kennel-island:resolve-input";
const IPC_STEER = "kennel-island:steer";
const IPC_INTERRUPT = "kennel-island:interrupt";
const IPC_OPEN_KENNEL = "kennel-island:open-kennel";
const IPC_GET_SETTINGS = "kennel-island:get-settings";
const IPC_UPDATE_SETTINGS = "kennel-island:update-settings";
const IPC_RESET_SETTINGS = "kennel-island:reset-settings";
const IPC_SETTINGS_CHANGED = "kennel-island:settings-changed";
const IPC_OPEN_SETTINGS = "kennel-island:open-settings";
const IPC_CLOSE_SETTINGS = "kennel-island:close-settings";
const IPC_PERFORM_HAPTIC = "kennel-island:perform-haptic";
const IPC_HIDE_ISLAND = "kennel-island:hide-island";
export const IPC_SNAPSHOT_INVALIDATED = "kennel-island:snapshot-invalidated";

const STAGE_MAX_WIDTH = 820;
const STAGE_MAX_HEIGHT = 460;
const STAGE_MIN_WIDTH = 380;
const STAGE_MIN_HEIGHT = 160;
const SETTINGS_WIDTH = 620;
const SETTINGS_HEIGHT = 640;
const SETTINGS_MIN_HEIGHT = 420;
const MEDIA_POLL_INTERVAL_MS = 2_500;
const VISIBILITY_FILE = "island-visibility.json";

type KennelConnection = { port: number; pid?: number };
type IslandControllerOptions = {
	rendererUrl: () => string;
	rendererName: string;
	preloadPath: string;
	getMainContents: () => WebContents | null;
	focusMainWindow: () => void;
	focusSession: (target: { projectId?: string; sessionId: string }) => Promise<boolean>;
};

export type IslandController = {
	setDaemonStatus(status: DaemonStatus): void;
	getState(): IslandVisibilityState;
	setVisible(visible: boolean): Promise<IslandVisibilityState>;
	openSettings(): Promise<{ open: boolean }>;
	focusSession(target: { projectId: string; sessionId: string }): Promise<boolean>;
	invalidateSnapshot(): void;
	dispose(): Promise<void>;
};

type MeasuredNotch = { hasNotch: boolean; notchWidth?: number; notchHeight?: number } | null;
type MediaTrack = {
	title: string;
	artist: string;
	source?: string;
	artwork?: string;
	/** Playhead, for the players that answer. See KennelMediaTrack for why. */
	positionSeconds?: number;
	durationSeconds?: number;
	sampledAt?: number;
	seekable?: boolean;
};
type MediaActivity = { playing: boolean; owner: string | null; track: MediaTrack | null };
type DetailedSettings = {
	notch: { widthOffset: number; heightOffset: number };
	hover: { haptics: boolean };
	media: { artwork: boolean };
};

function validPort(value: unknown): value is number {
	return Number.isInteger(value) && Number(value) >= 1 && Number(value) <= 65_535;
}

function helperPath(name: "kennel-haptics" | "kennel-notch"): string {
	return app.isPackaged
		? path.join(process.resourcesPath, name)
		: path.resolve(app.getAppPath(), "../packages/kennel-island/desktop/helpers", name);
}

function visibilityPath(): string {
	return path.join(app.getPath("userData"), VISIBILITY_FILE);
}

async function readVisibilityPreference(): Promise<boolean> {
	try {
		const value: unknown = JSON.parse(await nodeFs.readFile(visibilityPath(), "utf8"));
		return typeof value === "object" && value !== null && "enabled" in value
			? (value as { enabled?: unknown }).enabled !== false
			: true;
	} catch {
		return true;
	}
}

async function writeVisibilityPreference(enabled: boolean): Promise<void> {
	const target = visibilityPath();
	const temporary = `${target}.${process.pid}.tmp`;
	try {
		await nodeFs.mkdir(path.dirname(target), { recursive: true });
		await nodeFs.writeFile(temporary, `${JSON.stringify({ version: 1, enabled }, null, "\t")}\n`, {
			encoding: "utf8",
			mode: 0o600,
		});
		await nodeFs.rename(temporary, target);
	} catch {
		await nodeFs.rm(temporary, { force: true }).catch(() => undefined);
	}
}

function isRecord(value: unknown): value is Record<string, unknown> {
	return typeof value === "object" && value !== null && !Array.isArray(value);
}

function validateIdentifier(value: unknown, field: string): string {
	if (
		typeof value !== "string" ||
		value.length === 0 ||
		value !== value.trim() ||
		Buffer.byteLength(value, "utf8") > 512 ||
		/[\u0000-\u001f\u007f]/.test(value)
	) {
		throw new TypeError(`${field} is invalid`);
	}
	return value;
}

function conversationSessionId(payload: unknown): string {
	if (!isRecord(payload)) throw new TypeError("Conversation request must be an object");
	return validateIdentifier(payload.sessionId, "sessionId");
}

function parseOpenRequest(payload: unknown): { projectId?: string; sessionId?: string } {
	if (payload === undefined) return {};
	if (!isRecord(payload)) throw new TypeError("Open request must be an object");
	return {
		...(payload.projectId === undefined ? {} : { projectId: validateIdentifier(payload.projectId, "projectId") }),
		...(payload.sessionId === undefined ? {} : { sessionId: validateIdentifier(payload.sessionId, "sessionId") }),
	};
}

function safeIpcError(error: unknown): Error {
	let message = "Kennel could not complete the request.";
	let code = "ISLAND_OPERATION_FAILED";
	if (error instanceof TypeError) {
		message = "Kennel Island sent an invalid request.";
		code = "INVALID_ARGUMENT";
	} else if (isRecord(error) && typeof error.code === "string") {
		code = error.code;
		if (code === "DAEMON_NOT_RUNNING" || code === "DAEMON_NOT_READY") {
			message = "Kennel is still connecting.";
		} else if (code === "DAEMON_UNREACHABLE" || code === "DAEMON_TIMEOUT") {
			message = "Could not reach Kennel.";
		} else if (error.status === 409) {
			message = "Kennel changed while this action was running. Refresh and try again.";
		}
	}
	const normalized = new Error(message);
	normalized.name = "KennelIslandError";
	Object.assign(normalized, { code });
	normalized.stack = `${normalized.name}: ${message}`;
	return normalized;
}

export async function createIslandController(options: IslandControllerOptions): Promise<IslandController> {
	let islandWindow: BrowserWindow | null = null;
	let settingsWindow: BrowserWindow | null = null;
	let islandInteractive = false;
	// Tracks what setIslandFocusable last granted, so the defensive `focus`
	// listener created below can tell "the steer composer legitimately asked
	// for a caret" apart from "the OS handed us key status we never wanted".
	let islandFocusable = false;
	let measuredNotch: MeasuredNotch = null;
	let enabled = await readVisibilityPreference();
	let supported = false;
	let disposed = false;
	let mediaPollTimer: NodeJS.Timeout | null = null;
	let mediaActivity: MediaActivity = { playing: false, owner: null, track: null };
	let daemonConnection: KennelConnection | null = null;
	let shortcutRegistered = false;
	let displayRefresh: Promise<void> | null = null;
	let displayRefreshQueued = false;
	let islandWindowPromise: Promise<BrowserWindow | null> | null = null;
	let settingsWindowPromise: Promise<{ open: boolean }> | null = null;
	let visibilityOperations: Promise<void> = Promise.resolve();
	let visibilityWrites: Promise<void> = Promise.resolve();
	let protocolRegistered = false;
	let sessionSecurityInstalled = false;
	let displayListenersRegistered = false;
	const registeredIpcChannels = new Set<string>();

	const islandSession = session.fromPartition(ISLAND_PARTITION);
	const settingsStore = createSettingsStore({
		fs: nodeFs,
		userDataPath: path.join(app.getPath("userData"), "island"),
	});
	await settingsStore.load();
	const haptics = createHaptics({
		spawn,
		fs: nodeFs,
		helperPath: helperPath("kennel-haptics"),
	});
	const service = createKennelService({
		fetch: (url: string, init: RequestInit) => net.fetch(url, init),
		getConnection: () => daemonConnection,
	});
	const eventStream = createIslandEventStream({
		fetch: (url, init) => net.fetch(url, init),
		onInvalidate: invalidateSnapshot,
		log: (message) => console.warn(message),
	});

	function currentSettings(): DetailedSettings {
		return settingsStore.current() as unknown as DetailedSettings;
	}

	function internalDisplay(): Display {
		return screen.getAllDisplays().find((display) => display.internal) ?? screen.getPrimaryDisplay();
	}

	function currentNotchGeometry(display = internalDisplay()) {
		return notchGeometryFor(
			display,
			notchOptionsFromSettings(currentSettings(), {
				overrideWidth: process.env.KENNEL_ISLAND_NOTCH_WIDTH ?? null,
				measured: measuredNotch,
			}),
		);
	}

	function stageBounds(display = internalDisplay()): Electron.Rectangle {
		const available = display.bounds;
		const width = Math.max(Math.min(STAGE_MAX_WIDTH, available.width), Math.min(STAGE_MIN_WIDTH, available.width));
		const height = Math.max(Math.min(STAGE_MAX_HEIGHT, available.height), Math.min(STAGE_MIN_HEIGHT, available.height));
		return {
			x: available.x + Math.round((available.width - width) / 2),
			y: available.y,
			width,
			height,
		};
	}

	function stageGeometry(display = internalDisplay()) {
		const bounds = stageBounds(display);
		const notch = currentNotchGeometry(display);
		return {
			stageWidth: bounds.width,
			stageHeight: bounds.height,
			hasNotch: notch.hasNotch,
			notchWidth: notch.notchWidth,
			notchHeight: notch.notchHeight,
			menuBarHeight: notch.menuBarHeight,
			scaleFactor: notch.scaleFactor,
		};
	}

	function state(): IslandVisibilityState {
		return {
			supported,
			enabled: supported && enabled,
			visible: Boolean(islandWindow && !islandWindow.isDestroyed() && islandWindow.isVisible()),
			shortcut: SHORTCUT_LABEL,
		};
	}

	function updateDockMenu(): void {
		if (process.platform !== "darwin" || !app.dock) return;
		const current = state();
		app.dock.setMenu(Menu.buildFromTemplate([
			{
				label: current.visible ? "Hide Island" : "Show Island",
				enabled: current.supported,
				click: () => {
					void setVisible(!state().visible).catch((error) => {
						console.error("Kennel Island visibility could not be changed:", error);
					});
				},
			},
			{
				label: "Island Settings…",
				enabled: current.supported,
				click: () => {
					void openSettingsWindow().catch((error) => {
						console.error("Kennel Island settings could not be opened:", error);
					});
				},
			},
		]));
	}

	function broadcastState(): IslandVisibilityState {
		const next = state();
		const mainContents = options.getMainContents();
		if (mainContents && !mainContents.isDestroyed()) mainContents.send(ISLAND_STATE_CHANNEL, next);
		updateDockMenu();
		return next;
	}

	function recenterWindow(window = islandWindow) {
		if (!window || window.isDestroyed()) return null;
		const display = internalDisplay();
		window.setBounds(stageBounds(display), false);
		const geometry = stageGeometry(display);
		if (!window.webContents.isDestroyed()) window.webContents.send(IPC_STAGE_GEOMETRY_CHANGED, geometry);
		return geometry;
	}

	function setIslandInteractive(interactive: boolean): { interactive: boolean } {
		if (!islandWindow || islandWindow.isDestroyed()) return { interactive: islandInteractive };
		if (interactive === islandInteractive) return { interactive: islandInteractive };
		islandInteractive = interactive;
		if (interactive) islandWindow.setIgnoreMouseEvents(false);
		else islandWindow.setIgnoreMouseEvents(true, { forward: true });
		return { interactive: islandInteractive };
	}

	function rendererUrlAllowed(target: string): boolean {
		try {
			const candidate = new URL(target);
			const expected = new URL(options.rendererUrl());
			if (candidate.origin !== expected.origin || candidate.pathname !== expected.pathname) return false;
			for (const [name, value] of candidate.searchParams) {
				if (name !== "window" || value !== "settings") return false;
			}
			return true;
		} catch {
			return false;
		}
	}

	function secureWindow(window: BrowserWindow): void {
		window.webContents.setWindowOpenHandler(() => ({ action: "deny" }));
		window.webContents.on("will-navigate", (event, target) => {
			if (!rendererUrlAllowed(target)) event.preventDefault();
		});
		window.webContents.on("will-attach-webview", (event) => event.preventDefault());
	}

	function broadcastSettings(settings: unknown): void {
		for (const window of [islandWindow, settingsWindow]) {
			if (!window || window.isDestroyed() || window.webContents.isDestroyed()) continue;
			window.webContents.send(IPC_SETTINGS_CHANGED, settings);
		}
	}

	function applySettings(settings: unknown): void {
		broadcastSettings(settings);
		recenterWindow();
	}
	settingsStore.onChange(applySettings);

	async function resolveArtwork(activity: MediaActivity): Promise<MediaActivity> {
		if (!activity.track) return activity;
		const artwork = await readArtwork({
			source: activity.track.source,
			execFile,
			fs: nodeFs,
			fetchImpl: (url: string, init: RequestInit) => net.fetch(url, init),
			temporaryDirectory: app.getPath("temp"),
			allowNetwork: currentSettings().media.artwork === true,
		});
		return artwork ? { ...activity, track: { ...activity.track, artwork } } : activity;
	}

	function sameMediaActivity(left: MediaActivity, right: MediaActivity): boolean {
		return left.playing === right.playing && left.owner === right.owner &&
			sameTrack(left.track, right.track) &&
			(left.track?.artwork ?? null) === (right.track?.artwork ?? null);
	}

	async function sampleMediaActivity(): Promise<MediaActivity> {
		let next: MediaActivity;
		try {
			next = await readMediaActivity({ execFile, platform: process.platform });
		} catch {
			next = { playing: false, owner: null, track: null };
		}
		if (sameTrack(next.track, mediaActivity.track) && mediaActivity.track?.artwork && next.track) {
			next = { ...next, track: { ...next.track, artwork: mediaActivity.track.artwork } };
		} else if (next.track) {
			next = await resolveArtwork(next);
		}
		if (sameMediaActivity(next, mediaActivity)) return mediaActivity;
		mediaActivity = next;
		if (islandWindow && !islandWindow.isDestroyed() && !islandWindow.webContents.isDestroyed()) {
			islandWindow.webContents.send(IPC_MEDIA_ACTIVITY_CHANGED, mediaActivity);
		}
		return mediaActivity;
	}

	async function runMediaCommand(command: unknown): Promise<{ sent: boolean }> {
		if (!isMediaCommand(command)) return { sent: false };
		try {
			const result = await sendMediaCommand(command, { execFile, platform: process.platform });
			if (result.sent) void sampleMediaActivity();
			return { sent: result.sent === true };
		} catch {
			return { sent: false };
		}
	}

	/**
	 * Brings whatever is playing to the front.
	 *
	 * The application is taken from the host's own last sample rather than
	 * from the renderer: the island asks to focus "the player", and naming one
	 * across the bridge would turn this into a way to launch anything.
	 */
	/** Drags the playhead, then re-samples so the bar lands where it was dropped. */
	async function runSeek(positionSeconds: unknown): Promise<{ sought: boolean }> {
		try {
			const result = await seekMedia(positionSeconds, { execFile, platform: process.platform });
			if (result.sought) void sampleMediaActivity();
			return { sought: result.sought === true };
		} catch {
			return { sought: false };
		}
	}

	async function focusMediaApp(): Promise<{ focused: boolean }> {
		const owner = mediaActivity.owner;
		if (!owner) return { focused: false };
		try {
			const result = await focusApplication(owner, { execFile, platform: process.platform });
			return { focused: result.focused === true };
		} catch {
			return { focused: false };
		}
	}

	/**
	 * Lets the island take the key window, or takes that ability away again.
	 *
	 * Dropping focusability is not enough on its own: a window that is already
	 * key stays key, so the caret would remain stranded here until the user
	 * clicked elsewhere. Blurring hands the key window back to whatever had it
	 * before, which is the app the person was actually working in.
	 */
	function setIslandFocusable(focusable: boolean): { focusable: boolean } {
		islandFocusable = focusable;
		const window = islandWindow;
		if (!window || window.isDestroyed()) return { focusable: false };
		window.setFocusable(focusable);
		if (!focusable && window.isFocused()) window.blur();
		return { focusable };
	}

	function startMediaPolling(): void {
		if (mediaPollTimer || !supported) return;
		void sampleMediaActivity();
		mediaPollTimer = setInterval(() => void sampleMediaActivity(), MEDIA_POLL_INTERVAL_MS);
		mediaPollTimer.unref();
	}

	function stopMediaPolling(): void {
		if (!mediaPollTimer) return;
		clearInterval(mediaPollTimer);
		mediaPollTimer = null;
	}

	async function createIslandWindowInternal(): Promise<BrowserWindow | null> {
		if (!supported || disposed) return null;
		if (islandWindow && !islandWindow.isDestroyed()) return islandWindow;
		islandWindow = new BrowserWindow({
			...stageBounds(),
			acceptFirstMouse: true,
			alwaysOnTop: true,
			// The island is a heads-up display, not an app you switch to. A
			// focusable overlay takes the key window on the first click, which
			// drops the caret out of whatever was being typed in and leaves the
			// user clicking back into their own chat box. The steer composer
			// turns this on for as long as it is open and off again after.
			focusable: false,
			backgroundColor: "#00000000",
			closable: false,
			frame: false,
			fullscreenable: false,
			hasShadow: false,
			maximizable: false,
			minimizable: false,
			movable: false,
			resizable: false,
			roundedCorners: false,
			skipTaskbar: true,
			show: false,
			transparent: true,
			type: "panel",
			useContentSize: true,
			webPreferences: {
				backgroundThrottling: false,
				contextIsolation: true,
				devTools: !app.isPackaged,
				nodeIntegration: false,
				nodeIntegrationInSubFrames: false,
				nodeIntegrationInWorker: false,
				partition: ISLAND_PARTITION,
				preload: options.preloadPath,
				safeDialogs: true,
				sandbox: true,
				spellcheck: false,
				webSecurity: true,
				webviewTag: false,
			},
		});
		const created = islandWindow;
		islandInteractive = true;
		setIslandInteractive(false);
		created.setAlwaysOnTop(true, "screen-saver");
		created.setVisibleOnAllWorkspaces(true, { visibleOnFullScreen: true });
		created.setHiddenInMissionControl(true);
		created.setMenuBarVisibility(false);
		created.setWindowButtonVisibility(false);
		// `focusable: false` at construction is the documented way to make an
		// Electron panel non-activating on macOS, but it is not the only lever:
		// AppKit can still hand the window key status on a click in some Electron
		// builds regardless of the constructor option, and the moment that
		// happens the previously-key window (whatever the user was typing into)
		// resigns key and the caret goes with it — which is exactly the friction
		// reported (clicking the island drops focus out of another app's text
		// field). `setFocusable(false)` again after creation covers builds where
		// the constructor option alone does not stick, and the `focus` listener
		// is the belt-and-suspenders backstop: if key status is granted despite
		// both of those, it is surrendered in the same tick, before a person
		// could perceive the island as focused at all. `blur()` does not need to
		// know what to hand key status back to — the window server does that on
		// its own, returning it to whatever was key immediately before.
		created.setFocusable(false);
		created.on("focus", () => {
			if (!created.isDestroyed() && !islandFocusable) created.blur();
		});
		secureWindow(created);
		created.once("ready-to-show", () => {
			if (created.isDestroyed() || !enabled || !supported) return;
			recenterWindow(created);
			created.showInactive();
			broadcastState();
		});
		created.on("closed", () => {
			if (islandWindow === created) islandWindow = null;
			islandInteractive = false;
			broadcastState();
		});
		try {
			await created.loadURL(options.rendererUrl());
		} catch (error) {
			if (!created.isDestroyed()) created.destroy();
			throw error;
		}
		startMediaPolling();
		return created;
	}

	function createIslandWindow(): Promise<BrowserWindow | null> {
		if (islandWindowPromise) return islandWindowPromise;
		const pending = createIslandWindowInternal();
		islandWindowPromise = pending;
		const clearPending = () => {
			if (islandWindowPromise === pending) islandWindowPromise = null;
		};
		void pending.then(clearPending, clearPending);
		return pending;
	}

	async function showIsland(activate = false): Promise<void> {
		const window = await createIslandWindow();
		if (!window || window.isDestroyed() || !enabled || !supported || disposed) return;
		recenterWindow(window);
		if (activate) {
			window.show();
			window.focus();
		} else {
			window.showInactive();
		}
	}

	async function applyVisibility(visible: boolean): Promise<IslandVisibilityState> {
		if (disposed || !supported) return state();
		const previousEnabled = enabled;
		enabled = visible;
		broadcastState();
		try {
			if (visible) await showIsland(false);
			else if (islandWindow && !islandWindow.isDestroyed()) {
				setIslandInteractive(false);
				islandWindow.hide();
			}
		} catch (error) {
			enabled = previousEnabled;
			broadcastState();
			throw error;
		}
		if (disposed) return state();
		const nextEnabled = enabled;
		const persist = () => writeVisibilityPreference(nextEnabled);
		visibilityWrites = visibilityWrites.then(persist, persist);
		return broadcastState();
	}

	function setVisible(visible: boolean): Promise<IslandVisibilityState> {
		const operation = visibilityOperations.then(
			() => applyVisibility(visible),
			() => applyVisibility(visible),
		);
		visibilityOperations = operation.then(
			() => undefined,
			() => undefined,
		);
		return operation;
	}

	async function confirmHide(): Promise<IslandVisibilityState> {
		if (!islandWindow || islandWindow.isDestroyed()) return state();
		const result = await dialog.showMessageBox(islandWindow, {
			type: "question",
			buttons: ["Hide Island", "Cancel"],
			defaultId: 0,
			cancelId: 1,
			title: "Hide Kennel Island?",
			message: "Hide Kennel Island?",
			detail: `Bring it back anytime with ${SHORTCUT_LABEL}, Kennel Settings, or the Dock menu.`,
			noLink: true,
		});
		return result.response === 0 ? setVisible(false) : state();
	}

	async function openSettingsWindowInternal(): Promise<{ open: boolean }> {
		if (!supported || disposed) return { open: false };
		if (settingsWindow && !settingsWindow.isDestroyed()) {
			settingsWindow.show();
			settingsWindow.focus();
			app.focus({ steal: true });
			return { open: true };
		}
		settingsWindow = new BrowserWindow({
			width: SETTINGS_WIDTH,
			height: SETTINGS_HEIGHT,
			minWidth: SETTINGS_WIDTH,
			maxWidth: SETTINGS_WIDTH,
			minHeight: SETTINGS_MIN_HEIGHT,
			backgroundColor: "#111111",
			fullscreenable: false,
			maximizable: false,
			minimizable: true,
			resizable: true,
			show: false,
			title: "Kennel Island Settings",
			titleBarStyle: "hiddenInset",
			trafficLightPosition: { x: 16, y: 18 },
			webPreferences: {
				contextIsolation: true,
				devTools: !app.isPackaged,
				nodeIntegration: false,
				nodeIntegrationInSubFrames: false,
				nodeIntegrationInWorker: false,
				partition: ISLAND_PARTITION,
				preload: options.preloadPath,
				safeDialogs: true,
				sandbox: true,
				spellcheck: false,
				webSecurity: true,
				webviewTag: false,
			},
		});
		const created = settingsWindow;
		secureWindow(created);
		created.once("ready-to-show", () => {
			if (created.isDestroyed()) return;
			created.show();
			app.focus({ steal: true });
		});
		created.on("closed", () => {
			if (settingsWindow === created) settingsWindow = null;
			void settingsStore.update({ appearance: { calibrating: false } });
		});
		const renderer = new URL(options.rendererUrl());
		renderer.searchParams.set("window", "settings");
		try {
			await created.loadURL(renderer.toString());
		} catch (error) {
			if (!created.isDestroyed()) created.destroy();
			if (settingsWindow === created) settingsWindow = null;
			throw error;
		}
		return { open: true };
	}

	function openSettingsWindow(): Promise<{ open: boolean }> {
		if (settingsWindowPromise) return settingsWindowPromise;
		const pending = openSettingsWindowInternal();
		settingsWindowPromise = pending;
		const clearPending = () => {
			if (settingsWindowPromise === pending) settingsWindowPromise = null;
		};
		void pending.then(clearPending, clearPending);
		return pending;
	}

	function closeSettingsWindow(): { open: boolean } {
		if (settingsWindow && !settingsWindow.isDestroyed()) settingsWindow.close();
		return { open: false };
	}

	async function openKennel(payload: unknown): Promise<{ ok: true; targeted: boolean }> {
		const target = parseOpenRequest(payload);
		if (!target.sessionId) {
			options.focusMainWindow();
			return { ok: true, targeted: false };
		}
		const targeted = await options.focusSession({
			...(target.projectId === undefined ? {} : { projectId: target.projectId }),
			sessionId: target.sessionId,
		});
		if (!targeted) throw new TypeError("sessionId is not in the current Kennel snapshot");
		return { ok: true, targeted: true };
	}

	function assertSender(event: IpcMainInvokeEvent, allowed: Array<BrowserWindow | null>): void {
		if (!allowed.some((window) => window && !window.isDestroyed() && event.sender === window.webContents)) {
			throw new Error("Untrusted Island IPC sender");
		}
		if (event.senderFrame !== event.sender.mainFrame) throw new Error("Island IPC requires the main frame");
	}

	function assertMainSender(event: IpcMainInvokeEvent): void {
		const contents = options.getMainContents();
		if (!contents || contents.isDestroyed() || event.sender !== contents || event.senderFrame !== event.sender.mainFrame) {
			throw new Error("Untrusted Kennel IPC sender");
		}
	}

	function handle(
		channel: string,
		allowed: () => Array<BrowserWindow | null>,
		operation: (payload: unknown) => unknown | Promise<unknown>,
	): void {
		ipcMain.handle(channel, async (event, payload) => {
			try {
				assertSender(event, allowed());
				return await operation(payload);
			} catch (error) {
				console.warn(`Kennel Island IPC ${channel} failed`);
				throw safeIpcError(error);
			}
		});
		registeredIpcChannels.add(channel);
	}

	function registerIpc(): void {
		handle(IPC_SET_INTERACTIVE, () => [islandWindow], (payload) => {
			if (!isRecord(payload) || typeof payload.interactive !== "boolean") throw new TypeError("interactive is invalid");
			return setIslandInteractive(payload.interactive);
		});
		handle(IPC_GET_SETTINGS, () => [islandWindow, settingsWindow], () => currentSettings());
		handle(IPC_UPDATE_SETTINGS, () => [islandWindow, settingsWindow], (payload) => settingsStore.update(payload ?? {}));
		handle(IPC_RESET_SETTINGS, () => [islandWindow, settingsWindow], () => settingsStore.reset());
		handle(IPC_OPEN_SETTINGS, () => [islandWindow, settingsWindow], () => openSettingsWindow());
		handle(IPC_CLOSE_SETTINGS, () => [islandWindow, settingsWindow], () => closeSettingsWindow());
		handle(IPC_PERFORM_HAPTIC, () => [islandWindow], (payload) =>
			currentSettings().hover.haptics === true
				? haptics.perform(isRecord(payload) && typeof payload.pattern === "string" ? payload.pattern : "alignment")
				: { performed: false });
		handle(IPC_GET_STAGE_GEOMETRY, () => [islandWindow], () => stageGeometry());
		handle(IPC_GET_MEDIA_ACTIVITY, () => [islandWindow], () => mediaActivity);
		handle(IPC_SEND_MEDIA_COMMAND, () => [islandWindow], (payload) => runMediaCommand(isRecord(payload) ? payload.command : undefined));
		handle(IPC_FOCUS_MEDIA_APP, () => [islandWindow], () => focusMediaApp());
		handle(IPC_SEEK_MEDIA, () => [islandWindow], (payload) =>
			runSeek(isRecord(payload) ? payload.positionSeconds : undefined));
		handle(IPC_SET_FOCUSABLE, () => [islandWindow], (payload) =>
			setIslandFocusable(isRecord(payload) && payload.focusable === true));
		handle(IPC_RECENTER, () => [islandWindow], () => recenterWindow());
		handle(IPC_GET_KENNEL_SNAPSHOT, () => [islandWindow], () => service.getSnapshot({
			includePendingConversations: true,
			includeActiveConversations: true,
		}));
		handle(IPC_GET_KENNEL_CONVERSATION, () => [islandWindow], (payload) => service.getConversation(conversationSessionId(payload)));
		handle(IPC_RESOLVE_APPROVAL, () => [islandWindow], (payload) => service.resolveApproval(payload));
		handle(IPC_RESOLVE_INPUT, () => [islandWindow], (payload) => service.resolveInput(payload));
		handle(IPC_STEER, () => [islandWindow], (payload) => service.steer(payload));
		handle(IPC_INTERRUPT, () => [islandWindow], (payload) => service.interrupt(payload));
		handle(IPC_OPEN_KENNEL, () => [islandWindow], openKennel);
		handle(IPC_HIDE_ISLAND, () => [islandWindow], () => confirmHide());

		ipcMain.handle(ISLAND_GET_STATE_CHANNEL, (event) => {
			assertMainSender(event);
			return state();
		});
		registeredIpcChannels.add(ISLAND_GET_STATE_CHANNEL);
		ipcMain.handle(ISLAND_SET_VISIBLE_CHANNEL, async (event, visible) => {
			assertMainSender(event);
			if (typeof visible !== "boolean") throw new TypeError("visible must be a boolean");
			return setVisible(visible);
		});
		registeredIpcChannels.add(ISLAND_SET_VISIBLE_CHANNEL);
		ipcMain.handle(ISLAND_OPEN_SETTINGS_CHANNEL, (event) => {
			assertMainSender(event);
			return openSettingsWindow();
		});
		registeredIpcChannels.add(ISLAND_OPEN_SETTINGS_CHANNEL);
	}

	function unregisterIpc(): void {
		for (const channel of registeredIpcChannels) ipcMain.removeHandler(channel);
		registeredIpcChannels.clear();
	}

	async function registerIslandProtocol(): Promise<void> {
		if (!app.isPackaged) return;
		const root = path.join(__dirname, `../renderer/${options.rendererName}`);
		await islandSession.protocol.handle("app", async (request) => {
			if (request.method !== "GET") return new Response("Not found", { status: 404 });
			let url: URL;
			try {
				url = new URL(request.url);
			} catch {
				return new Response("Not found", { status: 404 });
			}
			if (url.host !== ISLAND_HOST) return new Response("Not found", { status: 404 });
			let pathname: string;
			try {
				pathname = decodeURIComponent(url.pathname);
			} catch {
				return new Response("Not found", { status: 404 });
			}
			const relative = pathname === "/" ? "index.html" : pathname.replace(/^\/+/, "");
			const target = path.resolve(root, relative);
			if (target !== root && !target.startsWith(`${root}${path.sep}`)) {
				return new Response("Forbidden", { status: 403 });
			}
			try {
				return await net.fetch(pathToFileURL(target).toString());
			} catch {
				return new Response("Not found", { status: 404 });
			}
		});
		protocolRegistered = true;
	}

	const preventIslandDownload = (event: Electron.Event) => event.preventDefault();
	const enforceIslandHeaders = (
		details: OnHeadersReceivedListenerDetails,
		callback: (response: HeadersReceivedResponse) => void,
	) => {
			if (!details.url.startsWith(`${ISLAND_ORIGIN}/`)) {
				callback({ responseHeaders: details.responseHeaders });
				return;
			}
			callback({
				responseHeaders: {
					...details.responseHeaders,
					"Content-Security-Policy": [
						"default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self' data:; connect-src 'none'; object-src 'none'; base-uri 'none'; form-action 'none'; frame-ancestors 'none'",
					],
				},
			});
	};

	function uninstallSessionSecurity(): void {
		if (!sessionSecurityInstalled) return;
		islandSession.setPermissionCheckHandler(null);
		islandSession.setPermissionRequestHandler(null);
		islandSession.setDevicePermissionHandler?.(null);
		islandSession.removeListener("will-download", preventIslandDownload);
		islandSession.webRequest.onHeadersReceived(null);
		sessionSecurityInstalled = false;
	}

	function installSessionSecurity(): void {
		if (sessionSecurityInstalled) return;
		sessionSecurityInstalled = true;
		try {
			islandSession.setPermissionCheckHandler(() => false);
			islandSession.setPermissionRequestHandler((_contents, _permission, callback) => callback(false));
			islandSession.setDevicePermissionHandler?.(() => false);
			islandSession.on("will-download", preventIslandDownload);
			islandSession.webRequest.onHeadersReceived(enforceIslandHeaders);
		} catch (error) {
			uninstallSessionSecurity();
			throw error;
		}
	}

	async function refreshNotch(): Promise<void> {
		if (process.platform !== "darwin") {
			measuredNotch = null;
			supported = false;
			return;
		}
		measuredNotch = await measureNotch({ execFile, helperPath: helperPath("kennel-notch") });
		const geometry = currentNotchGeometry();
		supported = process.env.KENNEL_ISLAND_FORCE === "1" || geometry.hasNotch === true;
	}

	async function reconcileEligibility(): Promise<void> {
		if (disposed) return;
		await refreshNotch();
		if (disposed) return;
		if (!supported) {
			eventStream.rebind(null);
			if (shortcutRegistered) {
				globalShortcut.unregister(TOGGLE_ACCELERATOR);
				shortcutRegistered = false;
			}
			stopMediaPolling();
			settingsWindow?.close();
			if (islandWindow && !islandWindow.isDestroyed()) islandWindow.destroy();
			broadcastState();
			return;
		}
		if (!shortcutRegistered) {
			shortcutRegistered = globalShortcut.register(TOGGLE_ACCELERATOR, () => {
				void setVisible(!state().visible).catch((error) => {
					console.error("Kennel Island shortcut could not change visibility:", error);
				});
			});
			if (!shortcutRegistered) console.warn(`Kennel Island could not register ${TOGGLE_ACCELERATOR}`);
		}
		if (enabled) {
			try {
				await showIsland(false);
			} catch (error) {
				// A renderer failure must not take down the host controller: session
				// routing and a later Show Island retry should remain available.
				console.error("Kennel Island could not open on this display:", error);
			}
		}
		else recenterWindow();
		eventStream.rebind(daemonConnection?.port ?? null);
		startMediaPolling();
		broadcastState();
	}

	function onDisplayChange(): void {
		if (displayRefresh) {
			displayRefreshQueued = true;
			return;
		}
		const refreshUntilCurrent = async () => {
			do {
				displayRefreshQueued = false;
				await reconcileEligibility().catch((error) => {
					console.error("Kennel Island could not follow the display change:", error);
				});
			} while (displayRefreshQueued && !disposed);
		};
		displayRefresh = refreshUntilCurrent().finally(() => {
			displayRefresh = null;
		});
	}

	function invalidateSnapshot(): void {
		if (!islandWindow || islandWindow.isDestroyed() || islandWindow.webContents.isDestroyed()) return;
		islandWindow.webContents.send(IPC_SNAPSHOT_INVALIDATED);
	}

	function setDaemonStatus(status: DaemonStatus): void {
		const next = status.state === "ready" && validPort(status.port)
			? { port: status.port, ...(Number.isInteger(status.pid) ? { pid: status.pid } : {}) }
			: null;
		const changed = next?.port !== daemonConnection?.port || next?.pid !== daemonConnection?.pid;
		daemonConnection = next;
		eventStream.rebind(supported ? next?.port ?? null : null);
		if (changed) invalidateSnapshot();
	}

	async function dispose(): Promise<void> {
		if (disposed) return;
		disposed = true;
		stopMediaPolling();
		eventStream.stop();
		haptics.stop();
		if (shortcutRegistered) globalShortcut.unregister(TOGGLE_ACCELERATOR);
		shortcutRegistered = false;
		if (displayListenersRegistered) {
			screen.removeListener("display-added", onDisplayChange);
			screen.removeListener("display-removed", onDisplayChange);
			screen.removeListener("display-metrics-changed", onDisplayChange);
			displayListenersRegistered = false;
		}
		const pendingDisplayRefresh = displayRefresh;
		if (pendingDisplayRefresh) await pendingDisplayRefresh.catch(() => undefined);
		unregisterIpc();
		uninstallSessionSecurity();
		settingsWindow?.destroy();
		islandWindow?.destroy();
		settingsWindow = null;
		islandWindow = null;
		let protocolCleanupError: unknown;
		if (protocolRegistered) {
			try {
				await islandSession.protocol.unhandle("app");
			} catch (error) {
				protocolCleanupError = error;
			} finally {
				protocolRegistered = false;
			}
		}
		await Promise.all([settingsStore.flush(), visibilityWrites]);
		if (protocolCleanupError) throw protocolCleanupError;
	}

	try {
		await registerIslandProtocol();
		installSessionSecurity();
		registerIpc();
		screen.on("display-added", onDisplayChange);
		screen.on("display-removed", onDisplayChange);
		screen.on("display-metrics-changed", onDisplayChange);
		displayListenersRegistered = true;
		await reconcileEligibility();
	} catch (error) {
		await dispose().catch((cleanupError) => {
			console.error("Kennel Island initialization cleanup failed:", cleanupError);
		});
		throw error;
	}

	return {
		setDaemonStatus,
		getState: state,
		setVisible,
		openSettings: openSettingsWindow,
		focusSession: async (target) => (await openKennel(target)).targeted,
		invalidateSnapshot,
		dispose,
	};
}
