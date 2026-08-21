const { contextBridge, ipcRenderer } = require("electron");

const IPC_SET_INTERACTIVE = "kennel-island:set-interactive";
const IPC_GET_STAGE_GEOMETRY = "kennel-island:get-stage-geometry";
const IPC_STAGE_GEOMETRY_CHANGED = "kennel-island:stage-geometry-changed";
const IPC_GET_MEDIA_ACTIVITY = "kennel-island:get-media-activity";
const IPC_MEDIA_ACTIVITY_CHANGED = "kennel-island:media-activity-changed";
const IPC_SEND_MEDIA_COMMAND = "kennel-island:send-media-command";
const MEDIA_COMMANDS = Object.freeze(["next", "previous", "play-pause"]);
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
const HAPTIC_PATTERNS = Object.freeze(["alignment", "generic", "level"]);

const STAGE_GEOMETRY_FIELDS = [
	"stageWidth",
	"stageHeight",
	"notchWidth",
	"notchHeight",
	"menuBarHeight",
	"scaleFactor",
];

// Main is trusted, but the renderer should still receive a value of a known
// shape rather than whatever arrived on the channel.
function projectedStageGeometry(value) {
	if (!value || typeof value !== "object") return null;

	const geometry = { hasNotch: value.hasNotch === true };
	for (const field of STAGE_GEOMETRY_FIELDS) {
		const measurement = value[field];
		geometry[field] = Number.isFinite(measurement) && measurement >= 0 ? measurement : 0;
	}
	return Object.freeze(geometry);
}

const MAX_TRACK_FIELD_LENGTH = 200;

function projectedText(value) {
	return typeof value === "string" ? value.slice(0, MAX_TRACK_FIELD_LENGTH) : "";
}

/** Artwork the renderer can safely put in an `img` and read back off a canvas. */
const ARTWORK_DATA_URI = /^data:image\/(?:jpeg|png|webp);base64,[A-Za-z0-9+/]+=*$/;
const MAX_ARTWORK_URI_LENGTH = 4 * 1024 * 1024;

// A track name is text from another application. It reaches the renderer as a
// bounded string and nothing else — no object shape it did not ask for, and no
// artwork that is not literally a base64 image.
function projectedArtwork(value) {
	if (typeof value !== "string" || value.length > MAX_ARTWORK_URI_LENGTH) return undefined;
	return ARTWORK_DATA_URI.test(value) ? value : undefined;
}

function projectedMediaActivity(value) {
	if (!value || typeof value !== "object") return { playing: false, track: null };

	const title = projectedText(value.track?.title);
	if (!title) return Object.freeze({ playing: value.playing === true, track: null });

	const artwork = projectedArtwork(value.track?.artwork);
	return Object.freeze({
		playing: value.playing === true,
		track: Object.freeze({
			title,
			artist: projectedText(value.track?.artist),
			...(artwork === undefined ? {} : { artwork }),
		}),
	});
}

// Settings arrive as a nested document of booleans and numbers. The bridge
// rebuilds it field by field from a shape it declares itself, for the same
// reason the stage geometry is rebuilt: the renderer should receive a value of
// a known shape, not whatever happened to be on the channel.
const SETTINGS_SHAPE = Object.freeze({
	notch: Object.freeze({ widthOffset: 0, heightOffset: 0, contentPadding: 12 }),
	hover: Object.freeze({
		peek: true,
		peekWidth: 14,
		peekHeight: 6,
		peekDelayMs: 90,
		openOnHover: false,
		holdOnMouseLeave: false,
		haptics: true,
	}),
	gestures: Object.freeze({
		enabled: true,
		verticalOpenClose: true,
		horizontalMedia: true,
		invertMedia: false,
	}),
	media: Object.freeze({ artwork: true, waveform: true }),
	appearance: Object.freeze({ calibrating: false, demoMode: false }),
});

function projectedSettings(value) {
	const source = value && typeof value === "object" && !Array.isArray(value) ? value : {};
	const settings = {};

	for (const [section, fields] of Object.entries(SETTINGS_SHAPE)) {
		const stored = source[section];
		const incoming = stored && typeof stored === "object" && !Array.isArray(stored) ? stored : {};
		settings[section] = {};
		for (const [name, fallback] of Object.entries(fields)) {
			const candidate = incoming[name];
			if (typeof fallback === "boolean") {
				settings[section][name] = typeof candidate === "boolean" ? candidate : fallback;
			} else {
				settings[section][name] = Number.isFinite(candidate) ? candidate : fallback;
			}
		}
		Object.freeze(settings[section]);
	}

	return Object.freeze(settings);
}

// A patch is renderer-authored, so it is rebuilt the same way on the way out:
// only fields the bridge knows about, and only when the caller named them.
function projectedPatch(value) {
	if (!value || typeof value !== "object" || Array.isArray(value)) return {};
	const patch = {};

	for (const [section, fields] of Object.entries(SETTINGS_SHAPE)) {
		const incoming = value[section];
		if (!incoming || typeof incoming !== "object" || Array.isArray(incoming)) continue;

		const changes = {};
		for (const [name, fallback] of Object.entries(fields)) {
			if (!Object.hasOwn(incoming, name)) continue;
			const candidate = incoming[name];
			if (typeof fallback === "boolean") {
				if (typeof candidate === "boolean") changes[name] = candidate;
			} else if (Number.isFinite(candidate)) {
				changes[name] = candidate;
			}
		}
		if (Object.keys(changes).length > 0) patch[section] = changes;
	}

	return patch;
}

const kennelDesktop = Object.freeze({
	setInteractive(interactive) {
		return ipcRenderer.invoke(IPC_SET_INTERACTIVE, { interactive: interactive === true });
	},
	async getStageGeometry() {
		return projectedStageGeometry(await ipcRenderer.invoke(IPC_GET_STAGE_GEOMETRY));
	},
	onStageGeometry(listener) {
		if (typeof listener !== "function") return () => {};

		const forward = (_event, value) => {
			const geometry = projectedStageGeometry(value);
			if (geometry) listener(geometry);
		};
		ipcRenderer.on(IPC_STAGE_GEOMETRY_CHANGED, forward);
		return () => ipcRenderer.removeListener(IPC_STAGE_GEOMETRY_CHANGED, forward);
	},
	async getMediaActivity() {
		return projectedMediaActivity(await ipcRenderer.invoke(IPC_GET_MEDIA_ACTIVITY));
	},
	onMediaActivity(listener) {
		if (typeof listener !== "function") return () => {};

		const forward = (_event, value) => listener(projectedMediaActivity(value));
		ipcRenderer.on(IPC_MEDIA_ACTIVITY_CHANGED, forward);
		return () => ipcRenderer.removeListener(IPC_MEDIA_ACTIVITY_CHANGED, forward);
	},
	// The command is checked here as well as in the main process: the bridge
	// only ever forwards a value the island itself could have produced.
	sendMediaCommand(command) {
		if (!MEDIA_COMMANDS.includes(command)) return Promise.resolve({ sent: false });
		return ipcRenderer.invoke(IPC_SEND_MEDIA_COMMAND, { command });
	},
	recenter() {
		return ipcRenderer.invoke(IPC_RECENTER);
	},
	getKennelSnapshot() {
		return ipcRenderer.invoke(IPC_GET_KENNEL_SNAPSHOT);
	},
	getKennelConversation(input) {
		return ipcRenderer.invoke(IPC_GET_KENNEL_CONVERSATION, {
			sessionId: input?.sessionId,
		});
	},
	resolveApproval(input) {
		return ipcRenderer.invoke(IPC_RESOLVE_APPROVAL, {
			sessionId: input?.sessionId,
			requestId: input?.requestId,
			decisionId: input?.decisionId,
		});
	},
	resolveInput(input) {
		return ipcRenderer.invoke(IPC_RESOLVE_INPUT, {
			sessionId: input?.sessionId,
			requestId: input?.requestId,
			action: input?.action,
			...(input?.content === undefined ? {} : { content: input.content }),
		});
	},
	steer(input) {
		return ipcRenderer.invoke(IPC_STEER, {
			sessionId: input?.sessionId,
			text: input?.text,
			clientMessageId: input?.clientMessageId,
		});
	},
	interrupt(input) {
		return ipcRenderer.invoke(IPC_INTERRUPT, {
			sessionId: input?.sessionId,
		});
	},
	openKennel(input) {
		return ipcRenderer.invoke(IPC_OPEN_KENNEL, {
			...(input?.projectId === undefined ? {} : { projectId: input.projectId }),
			...(input?.sessionId === undefined ? {} : { sessionId: input.sessionId }),
		});
	},
	async getSettings() {
		return projectedSettings(await ipcRenderer.invoke(IPC_GET_SETTINGS));
	},
	async updateSettings(patch) {
		return projectedSettings(await ipcRenderer.invoke(IPC_UPDATE_SETTINGS, projectedPatch(patch)));
	},
	async resetSettings() {
		return projectedSettings(await ipcRenderer.invoke(IPC_RESET_SETTINGS));
	},
	onSettings(listener) {
		if (typeof listener !== "function") return () => {};

		const forward = (_event, value) => listener(projectedSettings(value));
		ipcRenderer.on(IPC_SETTINGS_CHANGED, forward);
		return () => ipcRenderer.removeListener(IPC_SETTINGS_CHANGED, forward);
	},
	openSettings() {
		return ipcRenderer.invoke(IPC_OPEN_SETTINGS);
	},
	closeSettings() {
		return ipcRenderer.invoke(IPC_CLOSE_SETTINGS);
	},
	// Checked here as well as in the main process, so the bridge only forwards
	// a pattern the island itself could have produced.
	performHaptic(pattern) {
		return ipcRenderer.invoke(IPC_PERFORM_HAPTIC, {
			pattern: HAPTIC_PATTERNS.includes(pattern) ? pattern : "alignment",
		});
	},
});

contextBridge.exposeInMainWorld("kennelDesktop", kennelDesktop);
