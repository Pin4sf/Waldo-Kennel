const INVALIDATE_DELAY_MS = 150;
const RECONNECT_INITIAL_MS = 500;
const RECONNECT_MAX_MS = 10_000;
const MAX_SSE_LINE_CHARS = 1 << 20;
const MAX_EVENT_ID_CHARS = 128;

const STREAMS = [
	{ path: "/api/v1/events", durable: true },
	{ path: "/api/v1/notifications/stream", durable: false },
] as const;

type StreamDefinition = (typeof STREAMS)[number];

type StreamState = {
	definition: StreamDefinition;
	controller: AbortController | null;
	reader: ReadableStreamDefaultReader<Uint8Array> | null;
	retryTimer: ReturnType<typeof setTimeout> | null;
	backoffMs: number;
	lastEventId: string | null;
};

type FetchLike = (input: string, init?: RequestInit) => Promise<Response>;

export type IslandEventStreamOptions = {
	onInvalidate: () => void;
	fetch?: FetchLike;
	log?: (message: string) => void;
	/** Test seam; production uses 150 ms. */
	invalidateDelayMs?: number;
	/** Test seam for deterministic reconnect tests. */
	reconnectInitialMs?: number;
	/** Test seam for deterministic reconnect tests. */
	reconnectMaxMs?: number;
};

export interface IslandEventStreamHandle {
	readonly port: number | null;
	/**
	 * Bind both daemon streams to a supervisor-verified loopback port. Passing
	 * null disconnects without disposing the handle, so a later daemon can bind.
	 */
	rebind(port: number | null): void;
	/** Permanently abort all reads and timers. Safe to call more than once. */
	stop(): void;
}

/**
 * Main-process SSE transport for the Island.
 *
 * Callers provide only the port that the Electron supervisor already trusted.
 * This module constructs the loopback origin itself, never accepts a renderer
 * URL, and never exposes event payloads across the Electron boundary. Both
 * streams collapse into one bounded invalidation signal.
 */
export function createIslandEventStream(options: IslandEventStreamOptions): IslandEventStreamHandle {
	if (!options || typeof options.onInvalidate !== "function") {
		throw new TypeError("onInvalidate is required");
	}
	const fetchImpl = options.fetch ?? globalThis.fetch;
	if (typeof fetchImpl !== "function") throw new TypeError("fetch is required");
	const log = options.log ?? (() => undefined);
	const invalidateDelayMs = positiveDelay(options.invalidateDelayMs, INVALIDATE_DELAY_MS, "invalidateDelayMs");
	const reconnectInitialMs = positiveDelay(
		options.reconnectInitialMs,
		RECONNECT_INITIAL_MS,
		"reconnectInitialMs",
	);
	const reconnectMaxMs = positiveDelay(options.reconnectMaxMs, RECONNECT_MAX_MS, "reconnectMaxMs");
	if (reconnectMaxMs < reconnectInitialMs) {
		throw new TypeError("reconnectMaxMs must be greater than or equal to reconnectInitialMs");
	}

	let stopped = false;
	let currentPort: number | null = null;
	let generation = 0;
	let invalidateTimer: ReturnType<typeof setTimeout> | null = null;
	let streams: StreamState[] = [];

	const clearInvalidation = () => {
		if (invalidateTimer === null) return;
		clearTimeout(invalidateTimer);
		invalidateTimer = null;
	};

	const scheduleInvalidation = () => {
		if (stopped || currentPort === null) return;
		if (invalidateTimer !== null) clearTimeout(invalidateTimer);
		invalidateTimer = setTimeout(() => {
			invalidateTimer = null;
			if (!stopped && currentPort !== null) options.onInvalidate();
		}, invalidateDelayMs);
	};

	const closeStream = (state: StreamState) => {
		if (state.retryTimer !== null) {
			clearTimeout(state.retryTimer);
			state.retryTimer = null;
		}
		state.controller?.abort();
		state.controller = null;
		const reader = state.reader;
		state.reader = null;
		if (reader) void reader.cancel().catch(() => undefined);
	};

	const closeBinding = () => {
		generation += 1;
		for (const state of streams) closeStream(state);
		streams = [];
		clearInvalidation();
	};

	const isCurrent = (state: StreamState, boundGeneration: number, port: number) =>
		!stopped && generation === boundGeneration && currentPort === port && streams.includes(state);

	const scheduleReconnect = (state: StreamState, boundGeneration: number, port: number) => {
		if (!isCurrent(state, boundGeneration, port) || state.retryTimer !== null) return;
		const delay = state.backoffMs;
		state.backoffMs = Math.min(state.backoffMs * 2, reconnectMaxMs);
		state.retryTimer = setTimeout(() => {
			state.retryTimer = null;
			if (isCurrent(state, boundGeneration, port)) connect(state, boundGeneration, port);
		}, delay);
	};

	const consume = async (
		state: StreamState,
		response: Response,
		boundGeneration: number,
		port: number,
	): Promise<void> => {
		if (!response.ok) throw new Error(`HTTP ${response.status}`);
		const contentType = response.headers.get("content-type") ?? "";
		if (!contentType.toLowerCase().startsWith("text/event-stream")) {
			throw new Error(`unexpected content type ${contentType || "<missing>"}`);
		}
		if (!response.body || typeof response.body.getReader !== "function") {
			throw new Error("response body is not a readable stream");
		}

		const reader = response.body.getReader();
		state.reader = reader;
		const decoder = new TextDecoder();
		const parser = createSseParser((eventId) => {
			if (!isCurrent(state, boundGeneration, port)) return;
			if (state.definition.durable && eventId !== undefined) {
				state.lastEventId = eventId === "" ? null : eventId;
			}
			state.backoffMs = reconnectInitialMs;
			scheduleInvalidation();
		});

		// A successful (re)open covers changes that occurred while either stream
		// was unavailable. The two opens and any immediate replay frames coalesce.
		scheduleInvalidation();
		try {
			for (;;) {
				const { done, value } = await reader.read();
				if (done) break;
				if (!isCurrent(state, boundGeneration, port)) return;
				parser.push(decoder.decode(value, { stream: true }));
			}
			parser.push(decoder.decode());
			parser.finish();
		} finally {
			if (state.reader === reader) state.reader = null;
			reader.releaseLock?.();
		}
	};

	function connect(state: StreamState, boundGeneration: number, port: number): void {
		if (!isCurrent(state, boundGeneration, port) || state.controller !== null) return;
		const controller = new AbortController();
		state.controller = controller;
		const headers: Record<string, string> = {
			Accept: "text/event-stream",
			"Cache-Control": "no-cache",
		};
		if (state.definition.durable && state.lastEventId !== null) {
			headers["Last-Event-ID"] = state.lastEventId;
		}
		const url = `http://127.0.0.1:${port}${state.definition.path}`;

		void fetchImpl(url, {
			method: "GET",
			headers,
			cache: "no-store",
			redirect: "error",
			signal: controller.signal,
		})
			.then(async (response) => {
				if (!isCurrent(state, boundGeneration, port)) {
					await response.body?.cancel().catch(() => undefined);
					return;
				}
				await consume(state, response, boundGeneration, port);
			})
			.catch((error: unknown) => {
				if (!controller.signal.aborted && isCurrent(state, boundGeneration, port)) {
					log(`island-event-stream: ${state.definition.path}: ${errorMessage(error)}`);
				}
			})
			.finally(() => {
				if (state.controller === controller) state.controller = null;
				if (!controller.signal.aborted && isCurrent(state, boundGeneration, port)) {
					scheduleReconnect(state, boundGeneration, port);
				}
			});
	}

	return {
		get port() {
			return currentPort;
		},
		rebind(port) {
			if (stopped) return;
			if (port !== null) validatePort(port);
			if (port === currentPort) return;
			closeBinding();
			currentPort = port;
			if (port === null) return;

			const boundGeneration = generation;
			streams = STREAMS.map((definition) => ({
				definition,
				controller: null,
				reader: null,
				retryTimer: null,
				backoffMs: reconnectInitialMs,
				lastEventId: null,
			}));
			for (const state of streams) connect(state, boundGeneration, port);
		},
		stop() {
			if (stopped) return;
			stopped = true;
			closeBinding();
			currentPort = null;
		},
	};
}

function validatePort(port: number): void {
	if (!Number.isInteger(port) || port < 1 || port > 65_535) {
		throw new TypeError("port must be an integer between 1 and 65535");
	}
}

function positiveDelay(value: number | undefined, fallback: number, name: string): number {
	const resolved = value ?? fallback;
	if (!Number.isInteger(resolved) || resolved < 1 || resolved > 60_000) {
		throw new TypeError(`${name} must be an integer between 1 and 60000`);
	}
	return resolved;
}

function errorMessage(error: unknown): string {
	return error instanceof Error ? error.message : String(error);
}

type SseParser = {
	push(chunk: string): void;
	finish(): void;
};

/** Parse only framing and event ids; payload bytes are deliberately discarded. */
function createSseParser(onEvent: (eventId: string | undefined) => void): SseParser {
	let line = "";
	let pendingCarriageReturn = false;
	let hasData = false;
	let eventId: string | undefined;

	const completeLine = () => {
		if (line === "") {
			if (hasData) onEvent(eventId);
			hasData = false;
			eventId = undefined;
			return;
		}
		if (line.startsWith(":")) {
			line = "";
			return;
		}
		const separator = line.indexOf(":");
		const field = separator < 0 ? line : line.slice(0, separator);
		let value = separator < 0 ? "" : line.slice(separator + 1);
		if (value.startsWith(" ")) value = value.slice(1);
		if (field === "data") hasData = true;
		if (field === "id" && !value.includes("\0") && value.length <= MAX_EVENT_ID_CHARS) eventId = value;
		line = "";
	};

	const append = (character: string) => {
		line += character;
		if (line.length > MAX_SSE_LINE_CHARS) throw new Error("SSE line is too large");
	};

	return {
		push(chunk) {
			for (const character of chunk) {
				if (pendingCarriageReturn) {
					pendingCarriageReturn = false;
					completeLine();
					if (character === "\n") continue;
				}
				if (character === "\r") {
					pendingCarriageReturn = true;
				} else if (character === "\n") {
					completeLine();
				} else {
					append(character);
				}
			}
		},
		finish() {
			if (pendingCarriageReturn) {
				pendingCarriageReturn = false;
				completeLine();
			}
			// Per the SSE grammar, an event is dispatched only after a blank line.
		},
	};
}
