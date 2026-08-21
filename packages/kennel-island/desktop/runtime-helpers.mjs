import path from "node:path";

const APP_STATE_FILE_NAME = "app-state.json";
const MAX_APP_STATE_BYTES = 64 * 1024;
const MAX_APP_PATH_BYTES = 16 * 1024;

export class KennelLaunchError extends Error {
	constructor(code, message, options = {}) {
		super(message, options.cause ? { cause: options.cause } : undefined);
		this.name = "KennelLaunchError";
		this.code = code;
	}
}

function unavailable(cause) {
	return new KennelLaunchError(
		"KENNEL_APP_UNAVAILABLE",
		"Kennel could not be opened. Launch the Kennel app once, then try again.",
		{ cause },
	);
}

function normalizedAbsolutePath(value) {
	if (
		typeof value !== "string" ||
		value.length === 0 ||
		value !== value.trim() ||
		value.includes("\0") ||
		/[\u0000-\u001f\u007f]/.test(value) ||
		Buffer.byteLength(value, "utf8") > MAX_APP_PATH_BYTES ||
		!path.isAbsolute(value)
	) {
		return null;
	}
	return path.normalize(value);
}

/**
 * Select the attach-only Kennel run file. An absolute AO_RUN_FILE wins. A
 * relative or malformed override is deliberately ignored rather than being
 * resolved against Kennel Island's working directory.
 */
export function resolveKennelRunFilePath({ env = {}, home, dev = false }) {
	const override = normalizedAbsolutePath(env.AO_RUN_FILE);
	if (override) return override;

	const normalizedHome = normalizedAbsolutePath(home);
	if (!normalizedHome) throw new TypeError("A valid absolute home directory is required");
	return path.join(normalizedHome, ".ao", ...(dev ? ["dev"] : []), "running.json");
}

/** The desktop app marker lives beside the run file in every Kennel mode. */
export function appStatePathForRunFile(runFilePath) {
	const normalized = normalizedAbsolutePath(runFilePath);
	if (!normalized) throw new TypeError("A valid absolute run file is required");
	return path.join(path.dirname(normalized), APP_STATE_FILE_NAME);
}

function parseAppPath(contents) {
	const text = typeof contents === "string" ? contents : String(contents);
	if (Buffer.byteLength(text, "utf8") > MAX_APP_STATE_BYTES) throw unavailable();

	let marker;
	try {
		marker = JSON.parse(text.replace(/^\uFEFF/, ""));
	} catch (cause) {
		throw unavailable(cause);
	}
	if (!marker || typeof marker !== "object" || Array.isArray(marker)) throw unavailable();
	const appPath = normalizedAbsolutePath(marker.appPath);
	if (!appPath) throw unavailable();
	return appPath;
}

/**
 * Resolve and validate the local application recorded by Kennel itself. This
 * accepts no renderer-provided path and performs no command execution.
 */
export async function resolveKennelAppPath({ fs, appStatePath, platform }) {
	if (!fs || typeof fs.readFile !== "function" || typeof fs.realpath !== "function" || typeof fs.stat !== "function") {
		throw new TypeError("A filesystem implementation is required");
	}
	if (!normalizedAbsolutePath(appStatePath)) throw new TypeError("A valid absolute app-state path is required");

	let recordedPath;
	try {
		const contents = await fs.readFile(appStatePath, "utf8");
		recordedPath = parseAppPath(contents);
	} catch (error) {
		if (error instanceof KennelLaunchError) throw error;
		throw unavailable(error);
	}

	if (platform === "darwin" && path.extname(recordedPath).toLowerCase() !== ".app") {
		throw unavailable();
	}

	let resolvedPath;
	let info;
	try {
		resolvedPath = await fs.realpath(recordedPath);
		info = await fs.stat(resolvedPath);
	} catch (cause) {
		throw unavailable(cause);
	}

	const normalizedResolvedPath = normalizedAbsolutePath(resolvedPath);
	if (!normalizedResolvedPath) throw unavailable();
	if (platform === "darwin") {
		if (path.extname(normalizedResolvedPath).toLowerCase() !== ".app" || !info.isDirectory()) {
			throw unavailable();
		}
	} else if (!info.isFile()) {
		throw unavailable();
	}

	return normalizedResolvedPath;
}
