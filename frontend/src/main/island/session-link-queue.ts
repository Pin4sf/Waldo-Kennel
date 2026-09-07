import type { IslandSessionTarget } from "./session-link";

const DEFAULT_RETRY_INITIAL_MS = 250;
const DEFAULT_RETRY_MAX_MS = 5_000;
const RETRYABLE_CODES = new Set([
	"DAEMON_NOT_RUNNING",
	"DAEMON_NOT_READY",
	"DAEMON_UNREACHABLE",
	"DAEMON_TIMEOUT",
]);

type SessionLinkQueueOptions = {
	isReady: () => boolean;
	focusSession: (target: IslandSessionTarget) => Promise<boolean>;
	log?: (message: string, error?: unknown) => void;
	retryInitialMs?: number;
	retryMaxMs?: number;
};

export type SessionLinkQueue = {
	enqueue(target: IslandSessionTarget): void;
	readinessChanged(): void;
	dispose(): void;
	readonly pending: IslandSessionTarget | null;
};

function isRecord(value: unknown): value is Record<string, unknown> {
	return typeof value === "object" && value !== null && !Array.isArray(value);
}

function retryable(error: unknown): boolean {
	if (!isRecord(error)) return false;
	if (error.retryable === true) return true;
	return typeof error.code === "string" && RETRYABLE_CODES.has(error.code);
}

function sameTarget(left: IslandSessionTarget | null, right: IslandSessionTarget): boolean {
	return left?.projectId === right.projectId && left.sessionId === right.sessionId;
}

function positiveDelay(value: number | undefined, fallback: number, field: string): number {
	const resolved = value ?? fallback;
	if (!Number.isInteger(resolved) || resolved < 1 || resolved > 60_000) {
		throw new TypeError(`${field} must be an integer between 1 and 60000`);
	}
	return resolved;
}

/** Retains one latest external session intent until it is routed or rejected. */
export function createSessionLinkQueue(options: SessionLinkQueueOptions): SessionLinkQueue {
	const initialRetryMs = positiveDelay(options.retryInitialMs, DEFAULT_RETRY_INITIAL_MS, "retryInitialMs");
	const maximumRetryMs = positiveDelay(options.retryMaxMs, DEFAULT_RETRY_MAX_MS, "retryMaxMs");
	if (maximumRetryMs < initialRetryMs) throw new TypeError("retryMaxMs must be at least retryInitialMs");
	const log = options.log ?? (() => undefined);

	let pending: IslandSessionTarget | null = null;
	let disposed = false;
	let attemptRunning = false;
	let retryTimer: ReturnType<typeof setTimeout> | null = null;
	let retryDelayMs = initialRetryMs;

	const clearRetry = () => {
		if (retryTimer === null) return;
		clearTimeout(retryTimer);
		retryTimer = null;
	};

	const scheduleRetry = () => {
		if (disposed || retryTimer !== null || !pending || !options.isReady()) return;
		const delay = retryDelayMs;
		retryDelayMs = Math.min(retryDelayMs * 2, maximumRetryMs);
		retryTimer = setTimeout(() => {
			retryTimer = null;
			void flush();
		}, delay);
		retryTimer.unref?.();
	};

	async function flush(): Promise<void> {
		if (disposed || attemptRunning || !pending || !options.isReady()) return;
		const target = pending;
		attemptRunning = true;
		try {
			const focused = await options.focusSession(target);
			if (!sameTarget(pending, target)) return;
			pending = null;
			retryDelayMs = initialRetryMs;
			if (!focused) log("Kennel session deep link was rejected: session was not found");
		} catch (error) {
			if (!sameTarget(pending, target)) return;
			if (retryable(error)) {
				log("Kennel session deep link is waiting for the daemon; it will retry", error);
				scheduleRetry();
			} else {
				pending = null;
				log("Kennel session deep link was rejected", error);
			}
		} finally {
			attemptRunning = false;
			if (pending && retryTimer === null && options.isReady() && !sameTarget(pending, target)) {
				void flush();
			}
		}
	}

	return {
		enqueue(target) {
			if (disposed) return;
			pending = { projectId: target.projectId, sessionId: target.sessionId };
			retryDelayMs = initialRetryMs;
			clearRetry();
			void flush();
		},
		readinessChanged() {
			if (disposed) return;
			clearRetry();
			if (!options.isReady()) return;
			retryDelayMs = initialRetryMs;
			void flush();
		},
		dispose() {
			if (disposed) return;
			disposed = true;
			pending = null;
			clearRetry();
		},
		get pending() {
			return pending;
		},
	};
}
