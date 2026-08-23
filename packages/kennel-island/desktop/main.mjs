import {
	app,
	BrowserWindow,
	globalShortcut,
	ipcMain,
	Menu,
	net,
	protocol,
	screen,
	session,
	shell,
} from "electron";
import { execFile, spawn } from "node:child_process";
import * as nodeFs from "node:fs/promises";
import path from "node:path";
import { isIP } from "node:net";
import os from "node:os";
import { pathToFileURL } from "node:url";
import { createHaptics, hapticsHelperPath } from "./haptics.mjs";
import { createKennelService, KennelServiceError } from "./kennel-service.mjs";
import { readArtwork, sameTrack } from "./media-artwork.mjs";
import {
	focusApplication,
	isMediaCommand,
	readMediaActivity,
	seekMedia,
	sendMediaCommand,
} from "./media-activity.mjs";
import { measureNotch, notchHelperPath } from "./notch-measure.mjs";
import { notchGeometryFor, notchOptionsFromSettings } from "./notch-geometry.mjs";
import { createSettingsStore, defaultSettings } from "./settings.mjs";
import {
	appStatePathForRunFile,
	KennelLaunchError,
	resolveKennelAppPath,
	resolveKennelRunFilePath,
} from "./runtime-helpers.mjs";

const APP_SCHEME = "kennel";
const APP_HOST = "island";
// Electron's accelerator grammar names the physical Backquote key with the
// punctuation character itself.
const TOGGLE_ACCELERATOR = "CommandOrControl+`";

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
const IPC_HIDE_ISLAND = "kennel-island:hide-island";
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
const ISLAND_IPC_CHANNELS = Object.freeze([
	IPC_SET_INTERACTIVE,
	IPC_SET_FOCUSABLE,
	IPC_GET_MEDIA_ACTIVITY,
	IPC_SEND_MEDIA_COMMAND,
	IPC_FOCUS_MEDIA_APP,
	IPC_SEEK_MEDIA,
	IPC_GET_STAGE_GEOMETRY,
	IPC_RECENTER,
	IPC_HIDE_ISLAND,
	IPC_GET_KENNEL_SNAPSHOT,
	IPC_GET_KENNEL_CONVERSATION,
	IPC_RESOLVE_APPROVAL,
	IPC_RESOLVE_INPUT,
	IPC_STEER,
	IPC_INTERRUPT,
	IPC_OPEN_KENNEL,
	IPC_GET_SETTINGS,
	IPC_UPDATE_SETTINGS,
	IPC_RESET_SETTINGS,
	IPC_OPEN_SETTINGS,
	IPC_CLOSE_SETTINGS,
	IPC_PERFORM_HAPTIC,
]);

/** Query the renderer reads to decide which of the two surfaces it is. */
const SETTINGS_WINDOW_QUERY = "window=settings";

// The stage is a fixed transparent canvas pinned over the notch. It is never
// resized per surface: an OS-level resize cannot be animated, and a window that
// tracks content is always a frame behind the content it is tracking. The
// island morphs inside this canvas instead, and the canvas stays click-through
// everywhere the island is not drawn.
const STAGE_MAX_WIDTH = 820;
const STAGE_MAX_HEIGHT = 460;
// Room for the resting shape plus its concave fillets when a display is too
// narrow to grant the full stage.
const STAGE_MIN_WIDTH = 380;
const STAGE_MIN_HEIGHT = 160;

// The settings window is an ordinary window with an ordinary frame. Nothing
// about it is notch-shaped: it is a form, and a form pretending to be hardware
// is a form nobody can resize when a slider label wraps.
const SETTINGS_WIDTH = 620;
const SETTINGS_HEIGHT = 640;
const SETTINGS_MIN_HEIGHT = 420;

let islandWindow = null;
let settingsWindow = null;
let isQuitting = false;
let islandIpcRegistered = false;
let islandInteractive = false;
// Tracks what setIslandFocusable last granted, so the defensive `focus`
// listener below can tell "the steer composer legitimately asked for a caret"
// apart from "the OS handed us key status we never wanted".
let islandFocusable = false;

// Settings are only reachable once `app.getPath("userData")` is, which is after
// readiness. Until then the defaults stand in, so anything that reads settings
// during startup gets a valid document rather than a null check.
let settingsStore = null;
let haptics = null;

// The notch as AppKit reports it, or null when the helper could not be asked.
// Null is not "no notch": it is "measure unavailable", and the geometry falls
// back to deriving the housing from the menu bar.
let measuredNotch = null;

function currentSettings() {
	return settingsStore?.current() ?? defaultSettings();
}

/**
 * Re-reads the hardware and republishes the stage if anything moved.
 *
 * Called at startup and on every display change, because "the built-in display"
 * is not a fixed thing: closing the lid, changing the scaled resolution, or
 * docking an external monitor all change which panel the island is on and how
 * many points its housing spans.
 */
async function refreshNotchMeasurement() {
	const next = await measureNotch({
		execFile,
		helperPath: notchHelperPath(app.getAppPath()),
	});
	if (next === null) return;

	const changed =
		measuredNotch?.hasNotch !== next.hasNotch ||
		measuredNotch?.notchWidth !== next.notchWidth ||
		measuredNotch?.notchHeight !== next.notchHeight;
	measuredNotch = next;
	if (changed) recenterWindow();
}

// Media has no change notification available to us, so it is sampled. The
// interval is a compromise: fast enough that the waveform appears with the
// first bar of a song, slow enough that a `pmset` read every few seconds costs
// nothing measurable.
const MEDIA_POLL_INTERVAL_MS = 2_500;
let mediaActivity = { playing: false, owner: null, track: null };
let mediaPollTimer = null;

function validatedDevServerUrl(rawValue) {
	if (!rawValue) return null;

	const candidate = new URL(rawValue);
	const hostname = candidate.hostname.replace(/^\[|\]$/g, "").toLowerCase();
	const isLoopback =
		hostname === "localhost" ||
		hostname === "::1" ||
		(isIP(hostname) !== 0 && hostname.startsWith("127."));
	if ((candidate.protocol !== "http:" && candidate.protocol !== "https:") || !isLoopback) {
		throw new Error("VITE_DEV_SERVER_URL must use http(s) on a loopback host");
	}

	candidate.hash = "";
	candidate.search = "";
	return candidate.toString().replace(/\/+$/, "");
}

const DEV_SERVER_URL = validatedDevServerUrl(process.env.VITE_DEV_SERVER_URL);
const KENNEL_RUN_FILE_PATH = resolveKennelRunFilePath({
	env: process.env,
	home: os.homedir(),
	dev: Boolean(DEV_SERVER_URL),
});
const KENNEL_APP_STATE_PATH = appStatePathForRunFile(KENNEL_RUN_FILE_PATH);
const kennelService = createKennelService({ runFilePath: KENNEL_RUN_FILE_PATH });

// This must run before app readiness so Chromium treats the packaged renderer
// as a normal, secure origin rather than an opaque custom protocol.
protocol.registerSchemesAsPrivileged([
	{
		scheme: APP_SCHEME,
		privileges: {
			standard: true,
			secure: true,
			supportFetchAPI: true,
			corsEnabled: false,
		},
	},
]);

function distRoot() {
	return path.resolve(app.getAppPath(), "dist");
}

function packagedRendererUrl() {
	return `${APP_SCHEME}://${APP_HOST}/index.html`;
}

function notFoundResponse() {
	return new Response("Not found", {
		status: 404,
		headers: { "content-type": "text/plain; charset=utf-8" },
	});
}

function registerPackagedRendererProtocol() {
	protocol.handle(APP_SCHEME, async (request) => {
		if (request.method !== "GET") return notFoundResponse();

		let requestUrl;
		try {
			requestUrl = new URL(request.url);
		} catch {
			return notFoundResponse();
		}

		if (requestUrl.host !== APP_HOST) return notFoundResponse();

		let pathname;
		try {
			pathname = decodeURIComponent(requestUrl.pathname);
		} catch {
			return notFoundResponse();
		}

		const root = distRoot();
		const relativePath = pathname === "/" ? "index.html" : pathname.replace(/^\/+/, "");
		const target = path.resolve(root, relativePath);
		const rootPrefix = `${root}${path.sep}`;
		if (target !== root && !target.startsWith(rootPrefix)) return notFoundResponse();

		try {
			return await net.fetch(pathToFileURL(target).toString());
		} catch {
			return notFoundResponse();
		}
	});
}

function installRendererSecurityPolicy() {
	const rendererSession = session.defaultSession;

	rendererSession.setPermissionCheckHandler(() => false);
	rendererSession.setPermissionRequestHandler((_webContents, _permission, callback) => callback(false));
	if (typeof rendererSession.setDevicePermissionHandler === "function") {
		rendererSession.setDevicePermissionHandler(() => false);
	}
	rendererSession.on("will-download", (event) => event.preventDefault());

	// Vite owns its development headers so HMR continues to work. The packaged
	// renderer has no network needs and receives a strict policy at the boundary.
	rendererSession.webRequest.onHeadersReceived((details, callback) => {
		if (!details.url.startsWith(`${APP_SCHEME}://${APP_HOST}/`)) {
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
	});
}

function internalDisplay() {
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

/**
 * The stage sits flush with the top of the built-in display and is centred on
 * it, so the notch is centred in the stage and the renderer can anchor the
 * island to the stage centre without knowing where the display is.
 */
function stageBounds(display) {
	const available = display.bounds;
	const width = Math.max(
		Math.min(STAGE_MAX_WIDTH, available.width),
		Math.min(STAGE_MIN_WIDTH, available.width),
	);
	const height = Math.max(
		Math.min(STAGE_MAX_HEIGHT, available.height),
		Math.min(STAGE_MIN_HEIGHT, available.height),
	);

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

/**
 * Re-pins the stage and republishes its geometry. Called on every display
 * change so a resolution switch, a lid open, or an external monitor arriving
 * moves the island rather than stranding it off-centre.
 */
function recenterWindow(window = islandWindow) {
	if (!window || window.isDestroyed()) return null;
	const display = internalDisplay();
	const nextBounds = stageBounds(display);
	window.setBounds(nextBounds, false);

	const geometry = stageGeometry(display);
	if (!window.webContents.isDestroyed()) {
		window.webContents.send(IPC_STAGE_GEOMETRY_CHANGED, geometry);
	}
	if (DEV_SERVER_URL) console.info("Kennel Island stage", nextBounds, geometry);
	return geometry;
}

/**
 * The stage ignores the mouse by default so the menu bar and the desktop below
 * it stay usable. `forward: true` keeps mousemove flowing to the renderer while
 * ignoring, which is how the renderer knows the pointer reached the island and
 * can ask for the clicks back.
 */
function setIslandInteractive(interactive) {
	if (!islandWindow || islandWindow.isDestroyed()) return { interactive: islandInteractive };
	if (interactive === islandInteractive) return { interactive: islandInteractive };

	islandInteractive = interactive;
	if (interactive) {
		islandWindow.setIgnoreMouseEvents(false);
	} else {
		islandWindow.setIgnoreMouseEvents(true, { forward: true });
	}
	return { interactive: islandInteractive };
}

/**
 * Lets the island take the key window, or takes that ability away again.
 *
 * Dropping focusability alone leaves an already-key window key, so the caret
 * would stay stranded here. Blurring hands it back to the app that had it.
 */
function setIslandFocusable(focusable) {
	islandFocusable = focusable;
	if (!islandWindow || islandWindow.isDestroyed()) return { focusable: false };
	islandWindow.setFocusable(focusable);
	if (!focusable && islandWindow.isFocused()) islandWindow.blur();
	return { focusable };
}

function sameMediaActivity(left, right) {
	return (
		left.playing === right.playing &&
		left.owner === right.owner &&
		sameTrack(left.track, right.track) &&
		(left.track?.artwork ?? null) === (right.track?.artwork ?? null)
	);
}

/**
 * Album art for the track currently in `mediaActivity`.
 *
 * Read once per track rather than once per poll: the poll runs every couple of
 * seconds, and re-exporting a JPEG — or re-fetching one — to discover it is the
 * same JPEG would be the most expensive thing this process does.
 */
async function resolveArtwork(activity) {
	if (!activity.track) return activity;

	const artwork = await readArtwork({
		source: activity.track.source,
		execFile,
		fs: nodeFs,
		fetchImpl: (url, options) => net.fetch(url, options),
		temporaryDirectory: app.getPath("temp"),
		allowNetwork: currentSettings().media.artwork === true,
	});
	if (!artwork) return activity;

	return { ...activity, track: { ...activity.track, artwork } };
}

/**
 * Brings whatever is playing to the front.
 *
 * The application comes from this process's own last sample, never from the
 * renderer: the island asks to focus "the player", and letting it name one
 * would turn this into a way to launch anything.
 */
/** Drags the playhead, then re-samples so the bar lands where it was dropped. */
async function runSeek(positionSeconds) {
	try {
		const result = await seekMedia(positionSeconds, { execFile });
		if (result.sought) void sampleMediaActivity();
		return { sought: result.sought === true };
	} catch {
		return { sought: false };
	}
}

async function focusMediaApp() {
	const owner = mediaActivity.owner;
	if (!owner) return { focused: false };
	try {
		const result = await focusApplication(owner, { execFile });
		return { focused: result.focused === true };
	} catch {
		return { focused: false };
	}
}

async function sampleMediaActivity() {
	let next;
	try {
		next = await readMediaActivity({ execFile, platform: process.platform });
	} catch {
		// A failed sample means "we do not know", and the honest rendering of
		// not knowing is silence rather than a stuck waveform.
		next = { playing: false, owner: null, track: null };
	}

	// Artwork survives a poll that found the same track, so a repeat sample
	// does not drop the picture and take the accent colour with it.
	if (sameTrack(next.track, mediaActivity.track) && mediaActivity.track?.artwork) {
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

/**
 * Runs a transport command, then resamples so the island stops showing the
 * track that was playing a moment ago. The poll is the only thing that knows
 * what a player did with the instruction, so it is asked rather than assumed.
 */
async function runMediaCommand(command) {
	if (!isMediaCommand(command)) return { sent: false };

	let result;
	try {
		result = await sendMediaCommand(command, { execFile, platform: process.platform });
	} catch {
		return { sent: false };
	}

	if (result.sent) void sampleMediaActivity();
	return { sent: result.sent === true };
}

function startMediaPolling() {
	if (mediaPollTimer) return;
	void sampleMediaActivity();
	mediaPollTimer = setInterval(() => void sampleMediaActivity(), MEDIA_POLL_INTERVAL_MS);
	// The island is a background panel; sampling media must never be the reason
	// the process stays alive.
	mediaPollTimer.unref?.();
}

function stopMediaPolling() {
	if (!mediaPollTimer) return;
	clearInterval(mediaPollTimer);
	mediaPollTimer = null;
}

function parseInteractiveRequest(payload) {
	if (!payload || typeof payload !== "object" || Array.isArray(payload)) {
		throw new TypeError("Interactive request must be an object");
	}
	if (typeof payload.interactive !== "boolean") {
		throw new TypeError("interactive must be a boolean");
	}
	return payload.interactive;
}

/**
 * Windows this process will answer IPC from.
 *
 * The two surfaces are not interchangeable. The island holds the Kennel session
 * — approvals, steering, interrupts — and the settings window is a form the
 * user opened; a form has no business resolving an approval, so most channels
 * name the island alone and only the shared ones accept both.
 */
function assertTrustedSender(event, allowed) {
	const trusted = allowed.some(
		(window) => window && !window.isDestroyed() && event.sender === window.webContents,
	);
	if (!trusted) throw new Error("Untrusted window sender");
	if (event.senderFrame !== event.sender.mainFrame) {
		throw new Error("Island IPC may only be used by the main frame");
	}
}

const SAFE_SERVICE_ERROR_MESSAGES = Object.freeze({
	INVALID_ARGUMENT: "Kennel Island sent an invalid request.",
	DAEMON_NOT_RUNNING: "Kennel is not running.",
	RUN_FILE_INVALID: "Kennel connection information is invalid.",
	RUN_FILE_UNREADABLE: "Kennel connection information is unavailable.",
	DAEMON_TIMEOUT: "Kennel took too long to respond.",
	DAEMON_UNREACHABLE: "Could not reach Kennel.",
	DAEMON_RESPONSE_TOO_LARGE: "Kennel returned an invalid response.",
	DAEMON_RESPONSE_INVALID: "Kennel returned an invalid response.",
	DAEMON_NOT_READY: "Kennel is still starting.",
	DAEMON_IDENTITY_MISMATCH: "Could not verify the running Kennel daemon.",
	APPROVAL_NOT_PENDING: "This approval is no longer pending.",
	DECISION_NOT_OFFERED: "This approval option is no longer available.",
	INPUT_NOT_PENDING: "This input request is no longer pending.",
	CHAT_NO_ACTIVE_TURN: "There is no active turn to steer.",
	CHAT_TURN_NOT_STEERABLE: "This turn cannot be steered right now.",
	CHAT_STEER_UNSUPPORTED: "This agent does not support steering.",
	CHAT_STEER_TEXT_REQUIRED: "Enter guidance before steering the running turn.",
	CHAT_NO_ACTIVE_CONVERSATION: "This session has no active conversation.",
});

function normalizedErrorCode(value, fallback = "UNKNOWN") {
	return typeof value === "string" && /^[A-Z][A-Z0-9_]{0,63}$/.test(value) ? value : fallback;
}

function safeIpcError(error) {
	let code = "ISLAND_OPERATION_FAILED";
	let message = "Kennel could not complete the request.";

	if (error instanceof KennelLaunchError) {
		code = normalizedErrorCode(error.code, "KENNEL_APP_OPEN_FAILED");
		message = error.message;
	} else if (error instanceof KennelServiceError) {
		code = normalizedErrorCode(error.code, "DAEMON_ERROR");
		message = (Object.hasOwn(SAFE_SERVICE_ERROR_MESSAGES, code)
			? SAFE_SERVICE_ERROR_MESSAGES[code]
			: null)
			?? (error.status === 409
				? "Kennel changed while this action was running. Refresh and try again."
				: "Kennel could not complete the action. Refresh and try again.");
	} else if (error instanceof TypeError) {
		code = "INVALID_ARGUMENT";
		message = SAFE_SERVICE_ERROR_MESSAGES.INVALID_ARGUMENT;
	}

	// Electron invokes only need a renderer-safe message. Replace the generated
	// stack so absolute main-process paths and nested causes never cross IPC.
	const normalized = new Error(message);
	normalized.name = "KennelIslandError";
	normalized.code = code;
	normalized.stack = `${normalized.name}: ${message}`;
	return normalized;
}

function guardedHandler(operation, handler, senders) {
	return async (event, payload) => {
		try {
			assertTrustedSender(event, senders());
			return await handler(payload);
		} catch (error) {
			const code = normalizedErrorCode(error?.code, normalizedErrorCode(error?.name));
			console.warn(`Kennel Island ${operation} failed [${code}]`);
			throw safeIpcError(error);
		}
	};
}

/** Channels only the island may use. */
function trustedHandler(operation, handler) {
	return guardedHandler(operation, handler, () => [islandWindow]);
}

/** Channels both surfaces share: settings, and asking which settings there are. */
function sharedHandler(operation, handler) {
	return guardedHandler(operation, handler, () => [islandWindow, settingsWindow]);
}

function validateOpenRequest(payload) {
	if (payload === undefined) return;
	if (!payload || typeof payload !== "object" || Array.isArray(payload)) {
		throw new TypeError("Open request must be an object");
	}
	for (const field of ["projectId", "sessionId"]) {
		const value = payload[field];
		if (value === undefined) continue;
		if (
			typeof value !== "string" ||
			value.length === 0 ||
			value !== value.trim() ||
			Buffer.byteLength(value, "utf8") > 512 ||
			/[\u0000-\u001f\u007f]/.test(value)
		) {
			throw new TypeError(`${field} is invalid`);
		}
	}
}

function conversationSessionId(payload) {
	if (!payload || typeof payload !== "object" || Array.isArray(payload)) {
		throw new TypeError("Conversation request must be an object");
	}
	const sessionId = payload.sessionId;
	if (
		typeof sessionId !== "string" ||
		sessionId.length === 0 ||
		sessionId !== sessionId.trim() ||
		Buffer.byteLength(sessionId, "utf8") > 512 ||
		/[\u0000-\u001f\u007f]/.test(sessionId)
	) {
		throw new TypeError("sessionId is invalid");
	}
	return sessionId;
}

async function openKennelFromMarker(payload) {
	validateOpenRequest(payload);
	const appPath = await resolveKennelAppPath({
		fs: nodeFs,
		appStatePath: KENNEL_APP_STATE_PATH,
		platform: process.platform,
	});

	let openError;
	try {
		openError = await shell.openPath(appPath);
	} catch (cause) {
		throw new KennelLaunchError(
			"KENNEL_APP_OPEN_FAILED",
			"The system could not open Kennel. Launch Kennel manually, then try again.",
			{ cause },
		);
	}
	if (openError) {
		throw new KennelLaunchError(
			"KENNEL_APP_OPEN_FAILED",
			"The system could not open Kennel. Launch Kennel manually, then try again.",
		);
	}

	// Kennel currently publishes no project/session deep-link contract. Opening
	// the validated app is real; claiming that the requested session was focused
	// would not be.
	return { ok: true, targeted: false };
}

/* --------------------------------------------------------------------------
   Settings
   -------------------------------------------------------------------------- */

function broadcastSettings(settings) {
	for (const window of [islandWindow, settingsWindow]) {
		if (!window || window.isDestroyed() || window.webContents.isDestroyed()) continue;
		window.webContents.send(IPC_SETTINGS_CHANGED, settings);
	}
}

/**
 * Applies settings that the main process, rather than a renderer, has to act
 * on. Today that is the notch fine tune: the housing's size decides the stage's
 * geometry, so a slider moving has to reach the window before it can reach the
 * shape drawn inside it.
 */
function applySettings(settings) {
	broadcastSettings(settings);
	recenterWindow();
}

async function updateSettings(patch) {
	if (!settingsStore) return currentSettings();
	if (patch !== undefined && (!patch || typeof patch !== "object" || Array.isArray(patch))) {
		throw new TypeError("Settings patch must be an object");
	}
	return settingsStore.update(patch ?? {});
}

async function resetSettings() {
	if (!settingsStore) return currentSettings();
	return settingsStore.reset();
}

function settingsRendererUrl() {
	if (DEV_SERVER_URL) return `${DEV_SERVER_URL}/?${SETTINGS_WINDOW_QUERY}`;
	return `${packagedRendererUrl()}?${SETTINGS_WINDOW_QUERY}`;
}

async function openSettingsWindow() {
	if (settingsWindow && !settingsWindow.isDestroyed()) {
		settingsWindow.show();
		settingsWindow.focus();
		// An accessory app owns no menu bar, so nothing has made it frontmost.
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
			devTools: Boolean(DEV_SERVER_URL),
			nodeIntegration: false,
			nodeIntegrationInSubFrames: false,
			nodeIntegrationInWorker: false,
			preload: path.join(app.getAppPath(), "desktop", "preload.cjs"),
			safeDialogs: true,
			sandbox: true,
			spellcheck: false,
			webSecurity: true,
			webviewTag: false,
		},
	});

	settingsWindow.webContents.setWindowOpenHandler(() => ({ action: "deny" }));
	settingsWindow.webContents.on("will-navigate", (event, targetUrl) => {
		if (!rendererUrlIsAllowed(targetUrl)) event.preventDefault();
	});
	settingsWindow.webContents.on("will-attach-webview", (event) => event.preventDefault());

	settingsWindow.once("ready-to-show", () => {
		if (!settingsWindow || settingsWindow.isDestroyed()) return;
		settingsWindow.show();
		app.focus({ steal: true });
	});
	settingsWindow.on("closed", () => {
		settingsWindow = null;
		// The calibration outline is a tool for the window that is now gone. It
		// must not outlive it and sit lit over the notch.
		void updateSettings({ appearance: { calibrating: false } }).catch(() => {});
	});

	await settingsWindow.loadURL(settingsRendererUrl());
	return { open: true };
}

function closeSettingsWindow() {
	if (settingsWindow && !settingsWindow.isDestroyed()) settingsWindow.close();
	return { open: false };
}

async function performHaptic(payload) {
	// A tap the user has switched off is not an error; it is a tap that does
	// not happen.
	if (!haptics || currentSettings().hover.haptics !== true) return { performed: false };
	return haptics.perform(typeof payload?.pattern === "string" ? payload.pattern : "alignment");
}

function registerIslandIpc() {
	if (islandIpcRegistered) return;
	islandIpcRegistered = true;

	ipcMain.handle(IPC_SET_INTERACTIVE, trustedHandler("interactivity", (payload) =>
		setIslandInteractive(parseInteractiveRequest(payload))));
	ipcMain.handle(IPC_GET_SETTINGS, sharedHandler("settings read", () => currentSettings()));
	ipcMain.handle(IPC_UPDATE_SETTINGS, sharedHandler("settings write", (payload) =>
		updateSettings(payload)));
	ipcMain.handle(IPC_RESET_SETTINGS, sharedHandler("settings reset", () => resetSettings()));
	ipcMain.handle(IPC_OPEN_SETTINGS, sharedHandler("open settings", () => openSettingsWindow()));
	ipcMain.handle(IPC_CLOSE_SETTINGS, sharedHandler("close settings", () => closeSettingsWindow()));
	ipcMain.handle(IPC_PERFORM_HAPTIC, trustedHandler("haptic", (payload) => performHaptic(payload)));
	ipcMain.handle(IPC_GET_STAGE_GEOMETRY, trustedHandler("stage geometry", () => stageGeometry()));
	ipcMain.handle(IPC_GET_MEDIA_ACTIVITY, trustedHandler("media activity", () => mediaActivity));
	ipcMain.handle(IPC_SEND_MEDIA_COMMAND, trustedHandler("media command", (payload) =>
		runMediaCommand(payload?.command)));
	ipcMain.handle(IPC_FOCUS_MEDIA_APP, trustedHandler("focus media app", () => focusMediaApp()));
	ipcMain.handle(IPC_SEEK_MEDIA, trustedHandler("seek media", (payload) =>
		runSeek(payload?.positionSeconds)));
	ipcMain.handle(IPC_SET_FOCUSABLE, trustedHandler("set focusable", (payload) =>
		setIslandFocusable(payload?.focusable === true)));
	ipcMain.handle(IPC_RECENTER, trustedHandler("recenter", () => recenterWindow()));
	ipcMain.handle(IPC_HIDE_ISLAND, trustedHandler("hide island", () => hideIsland()));
	ipcMain.handle(IPC_GET_KENNEL_SNAPSHOT, trustedHandler("snapshot", () =>
		kennelService.getSnapshot({
			includePendingConversations: true,
			includeActiveConversations: true,
		})));
	ipcMain.handle(IPC_GET_KENNEL_CONVERSATION, trustedHandler("conversation", (payload) =>
		kennelService.getConversation(conversationSessionId(payload))));
	ipcMain.handle(IPC_RESOLVE_APPROVAL, trustedHandler("approval resolution", (payload) =>
		kennelService.resolveApproval(payload)));
	ipcMain.handle(IPC_RESOLVE_INPUT, trustedHandler("input resolution", (payload) =>
		kennelService.resolveInput(payload)));
	ipcMain.handle(IPC_STEER, trustedHandler("steer", (payload) => kennelService.steer(payload)));
	ipcMain.handle(IPC_INTERRUPT, trustedHandler("interrupt", (payload) => kennelService.interrupt(payload)));
	ipcMain.handle(IPC_OPEN_KENNEL, trustedHandler("open Kennel", openKennelFromMarker));
}

function unregisterIslandIpc() {
	if (!islandIpcRegistered) return;
	islandIpcRegistered = false;
	for (const channel of ISLAND_IPC_CHANNELS) ipcMain.removeHandler(channel);
}

function rendererUrlIsAllowed(targetUrl) {
	if (targetUrl.startsWith(`${APP_SCHEME}://${APP_HOST}/`)) return true;
	if (!DEV_SERVER_URL) return false;

	try {
		return new URL(targetUrl).origin === new URL(DEV_SERVER_URL).origin;
	} catch {
		return false;
	}
}

async function loadRenderer(window) {
	if (DEV_SERVER_URL) {
		await window.loadURL(DEV_SERVER_URL);
		return;
	}
	await window.loadURL(packagedRendererUrl());
}

async function createIslandWindow() {
	if (islandWindow && !islandWindow.isDestroyed()) return islandWindow;

	const display = internalDisplay();
	const initialBounds = stageBounds(display);
	islandWindow = new BrowserWindow({
		...initialBounds,
		acceptFirstMouse: true,
		// A heads-up display, not an app you switch to: clicking a chip must
		// never take the key window away from whatever is being typed in. The
		// steer composer asks for focus while it is open.
		focusable: false,
		alwaysOnTop: true,
		backgroundColor: "#00000000",
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
			devTools: Boolean(DEV_SERVER_URL),
			nodeIntegration: false,
			nodeIntegrationInSubFrames: false,
			nodeIntegrationInWorker: false,
			preload: path.join(app.getAppPath(), "desktop", "preload.cjs"),
			safeDialogs: true,
			sandbox: true,
			spellcheck: false,
			webSecurity: true,
			webviewTag: false,
		},
	});

	islandInteractive = true;
	setIslandInteractive(false);
	islandWindow.setAlwaysOnTop(true, "screen-saver");
	islandWindow.setVisibleOnAllWorkspaces(true, {
		visibleOnFullScreen: true,
		skipTransformProcessType: true,
	});
	islandWindow.setHiddenInMissionControl(true);
	islandWindow.setMenuBarVisibility(false);
	if (process.platform === "darwin") islandWindow.setWindowButtonVisibility(false);
	// `focusable: false` at construction is the documented way to make a panel
	// non-activating on macOS, but AppKit can still hand a window key status on
	// click regardless of the constructor option on some Electron builds — the
	// moment that happens, whatever the user was typing into resigns key and
	// the caret goes with it. Re-asserting after creation covers builds where
	// the constructor option alone does not stick; the `focus` listener is the
	// backstop, surrendering key status in the same tick it is granted.
	islandWindow.setFocusable(false);
	islandWindow.on("focus", () => {
		if (!islandWindow.isDestroyed() && !islandFocusable) islandWindow.blur();
	});

	islandWindow.webContents.setWindowOpenHandler(() => ({ action: "deny" }));
	islandWindow.webContents.on("will-navigate", (event, targetUrl) => {
		if (!rendererUrlIsAllowed(targetUrl)) event.preventDefault();
	});
	islandWindow.webContents.on("will-attach-webview", (event) => event.preventDefault());

	islandWindow.once("ready-to-show", () => {
		if (!islandWindow || islandWindow.isDestroyed()) return;
		recenterWindow(islandWindow);
		islandWindow.showInactive();
	});
	islandWindow.on("closed", () => {
		islandWindow = null;
		islandInteractive = false;
		stopMediaPolling();
	});

	await loadRenderer(islandWindow);
	return islandWindow;
}

function showIsland({ activate = false } = {}) {
	if (!islandWindow || islandWindow.isDestroyed()) return;
	recenterWindow(islandWindow);
	if (activate) {
		islandWindow.show();
		islandWindow.focus();
		return;
	}
	islandWindow.showInactive();
}

function hideIsland() {
	if (!islandWindow || islandWindow.isDestroyed()) return { visible: false };
	// A hidden stage must not keep the pointer, or the next show would start by
	// swallowing clicks over the menu bar.
	setIslandInteractive(false);
	islandWindow.hide();
	return { visible: false };
}

function toggleIsland() {
	if (!islandWindow || islandWindow.isDestroyed()) {
		void createIslandWindow()
			.then(() => showIsland({ activate: true }))
			.catch((error) => console.error("Kennel Island could not be shown", error));
		return;
	}

	if (islandWindow.isVisible()) {
		hideIsland();
		return;
	}
	showIsland({ activate: true });
}

function registerToggleShortcut() {
	const registered = globalShortcut.register(TOGGLE_ACCELERATOR, toggleIsland);
	if (!registered) {
		console.warn(
			`Kennel Island could not register ${TOGGLE_ACCELERATOR}; another app may already own it`,
		);
	}
}

function hideMacDesktopChrome() {
	if (process.platform !== "darwin") return;
	app.setActivationPolicy("accessory");
	app.dock?.hide();
}

function recenterOnDisplayChange() {
	if (!islandWindow || islandWindow.isDestroyed()) return;
	recenterWindow(islandWindow);
	// The panel may have changed, so the housing may have too. Re-measuring
	// recenters again if it did, which is cheaper than being wrong until relaunch.
	void refreshNotchMeasurement();
}

const hasSingleInstanceLock = app.requestSingleInstanceLock();

if (!hasSingleInstanceLock) {
	app.quit();
} else {
	app.on("second-instance", () => {
		if (!islandWindow || islandWindow.isDestroyed()) {
			void createIslandWindow().catch((error) => console.error("Kennel Island could not reopen", error));
			return;
		}
		showIsland({ activate: true });
	});

	app.whenReady().then(async () => {
		app.setName("Kennel Island");
		hideMacDesktopChrome();
		Menu.setApplicationMenu(null);
		registerPackagedRendererProtocol();
		installRendererSecurityPolicy();

		// Settings are loaded before the first window, because the notch fine
		// tune decides the stage's geometry and a stage that opens at the wrong
		// size and corrects itself a frame later is a visible flinch.
		settingsStore = createSettingsStore({
			fs: nodeFs,
			userDataPath: app.getPath("userData"),
		});
		await settingsStore.load();
		settingsStore.onChange(applySettings);

		haptics = createHaptics({
			spawn,
			fs: nodeFs,
			helperPath: hapticsHelperPath(app.getAppPath()),
		});

		// Measured before the first window for the same reason settings are: a
		// stage that opens at a derived size and snaps to the real one a frame
		// later is a visible flinch on every launch.
		await refreshNotchMeasurement();

		registerIslandIpc();
		registerToggleShortcut();

		screen.on("display-added", recenterOnDisplayChange);
		screen.on("display-removed", recenterOnDisplayChange);
		screen.on("display-metrics-changed", recenterOnDisplayChange);

		await createIslandWindow();
		startMediaPolling();
	}).catch((error) => {
		console.error("Kennel Island failed to start", error);
		app.quit();
	});

	app.on("activate", () => {
		if (isQuitting) return;
		if (islandWindow && !islandWindow.isDestroyed()) {
			showIsland({ activate: true });
			return;
		}
		void createIslandWindow().catch((error) => console.error("Kennel Island could not reopen", error));
	});

	// The island is the app. Closing the settings window is closing a form, not
	// quitting — and the island itself is never "closed" while running.
	app.on("window-all-closed", () => {
		if (islandWindow && !islandWindow.isDestroyed()) return;
		app.quit();
	});
	app.on("before-quit", () => {
		isQuitting = true;
	});
	app.on("will-quit", () => {
		stopMediaPolling();
		haptics?.stop();
		globalShortcut.unregisterAll();
		unregisterIslandIpc();
		screen.removeListener("display-added", recenterOnDisplayChange);
		screen.removeListener("display-removed", recenterOnDisplayChange);
		screen.removeListener("display-metrics-changed", recenterOnDisplayChange);
		protocol.unhandle(APP_SCHEME);
	});
}
