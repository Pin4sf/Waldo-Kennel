import * as nodeFs from "node:fs/promises";
import os from "node:os";
import path from "node:path";

// Keep in sync with frontend/src/shared/daemon-attach.ts.
const DAEMON_SERVICE_NAME = "kennel-daemon";
const DEFAULT_TIMEOUT_MS = 2_000;
const MAX_RUN_FILE_BYTES = 64 * 1024;
const MAX_RESPONSE_BYTES = 8 * 1024 * 1024;
const MAX_IDENTIFIER_BYTES = 512;
const MAX_CONVERSATIONS_PER_SNAPSHOT = 16;
const MAX_INPUT_CONTENT_BYTES = 64 * 1024;
const MAX_STEER_TEXT_BYTES = 64 * 1024;
const MAX_APPROVAL_CONTEXT_CHARS = 4_096;
const INPUT_ACTIONS = new Set(["accept", "decline", "cancel"]);

export class KennelServiceError extends Error {
	constructor(code, message, options = {}) {
		super(message, options.cause ? { cause: options.cause } : undefined);
		this.name = "KennelServiceError";
		this.code = code;
		this.status = options.status ?? null;
		this.retryable = options.retryable ?? false;
		this.details = options.details ?? null;
		this.requestId = options.requestId ?? null;
	}

	toJSON() {
		return {
			code: this.code,
			message: this.message,
			status: this.status,
			retryable: this.retryable,
			...(this.details ? { details: this.details } : {}),
			...(this.requestId ? { requestId: this.requestId } : {}),
		};
	}
}

function fail(code, message, options) {
	throw new KennelServiceError(code, message, options);
}

function isRecord(value) {
	return typeof value === "object" && value !== null && !Array.isArray(value);
}

function isPlainObject(value) {
	if (!isRecord(value)) return false;
	const prototype = Object.getPrototypeOf(value);
	return prototype === Object.prototype || prototype === null;
}

function utf8Bytes(value) {
	return Buffer.byteLength(value, "utf8");
}

function normalizeIdentifier(value, field) {
	if (
		typeof value !== "string" ||
		value.length === 0 ||
		value !== value.trim() ||
		/[\u0000-\u001f\u007f]/.test(value) ||
		utf8Bytes(value) > MAX_IDENTIFIER_BYTES
	) {
		fail("INVALID_ARGUMENT", `${field} is invalid`, { details: { field } });
	}
	return value;
}

function encodeSegment(value, field) {
	return encodeURIComponent(normalizeIdentifier(value, field));
}

function parseRunFile(contents) {
	const text = typeof contents === "string" ? contents : String(contents);
	if (utf8Bytes(text) > MAX_RUN_FILE_BYTES) {
		fail("RUN_FILE_INVALID", "Kennel running.json is too large");
	}

	let raw;
	try {
		raw = JSON.parse(text.replace(/^\uFEFF/, ""));
	} catch (cause) {
		fail("RUN_FILE_INVALID", "Kennel running.json is not valid JSON", { cause });
	}

	if (
		!isRecord(raw) ||
		!Number.isInteger(raw.pid) ||
		raw.pid <= 0 ||
		!Number.isInteger(raw.port) ||
		raw.port < 1 ||
		raw.port > 65_535
	) {
		fail("RUN_FILE_INVALID", "Kennel running.json has an invalid pid or port");
	}

	const startedAt = typeof raw.startedAt === "string" && Number.isFinite(Date.parse(raw.startedAt))
		? raw.startedAt
		: null;
	return {
		pid: raw.pid,
		port: raw.port,
		startedAt,
		owner: typeof raw.owner === "string" ? raw.owner : null,
	};
}

function normalizeInjectedConnection(value) {
	if (value === null || value === undefined) {
		fail("DAEMON_NOT_RUNNING", "Kennel daemon is not running", { retryable: true });
	}
	const raw = typeof value === "number" ? { port: value } : value;
	if (!isRecord(raw)) {
		fail("DAEMON_CONNECTION_INVALID", "Injected Kennel daemon connection is invalid");
	}
	if (raw.state !== undefined && raw.state !== "ready") {
		fail("DAEMON_NOT_READY", "Kennel daemon is not ready", { retryable: true });
	}
	if (!Number.isInteger(raw.port) || raw.port < 1 || raw.port > 65_535) {
		fail("DAEMON_CONNECTION_INVALID", "Injected Kennel daemon connection has an invalid port");
	}
	if (raw.pid !== undefined && (!Number.isInteger(raw.pid) || raw.pid <= 0)) {
		fail("DAEMON_CONNECTION_INVALID", "Injected Kennel daemon connection has an invalid pid");
	}

	return {
		port: raw.port,
		pid: raw.pid,
		startedAt: typeof raw.startedAt === "string" && Number.isFinite(Date.parse(raw.startedAt))
			? raw.startedAt
			: null,
		owner: typeof raw.owner === "string" ? raw.owner : null,
	};
}

function publicDaemon(connection) {
	return {
		pid: connection.pid,
		port: connection.port,
		startedAt: connection.startedAt,
		owner: connection.owner,
	};
}

async function readResponseText(response) {
	const declaredLength = Number(response.headers?.get?.("content-length"));
	if (Number.isFinite(declaredLength) && declaredLength > MAX_RESPONSE_BYTES) {
		fail("DAEMON_RESPONSE_TOO_LARGE", "Kennel daemon response is too large");
	}

	if (!response.body || typeof response.body.getReader !== "function") {
		const text = await response.text();
		if (utf8Bytes(text) > MAX_RESPONSE_BYTES) {
			fail("DAEMON_RESPONSE_TOO_LARGE", "Kennel daemon response is too large");
		}
		return text;
	}

	const reader = response.body.getReader();
	const chunks = [];
	let totalBytes = 0;
	try {
		while (true) {
			const { done, value } = await reader.read();
			if (done) break;
			totalBytes += value.byteLength;
			if (totalBytes > MAX_RESPONSE_BYTES) {
				try {
					await reader.cancel();
				} catch {
					// The stable size error below is more useful than cancellation noise.
				}
				fail("DAEMON_RESPONSE_TOO_LARGE", "Kennel daemon response is too large");
			}
			chunks.push(Buffer.from(value));
		}
	} finally {
		reader.releaseLock?.();
	}
	return Buffer.concat(chunks, totalBytes).toString("utf8");
}

function parseResponseJSON(text, route) {
	if (text.trim() === "") return null;
	try {
		return JSON.parse(text);
	} catch (cause) {
		fail("DAEMON_RESPONSE_INVALID", `Kennel returned invalid JSON for ${route}`, { cause });
	}
}

function remoteError(response, payload) {
	const body = isRecord(payload) ? payload : {};
	const code = typeof body.code === "string" && body.code ? body.code : "DAEMON_HTTP_ERROR";
	const message = typeof body.message === "string" && body.message
		? body.message
		: `Kennel request failed with HTTP ${response.status}`;
	return new KennelServiceError(code, message, {
		status: response.status,
		retryable: response.status >= 500,
		details: isRecord(body.details) ? body.details : null,
		requestId: typeof body.requestId === "string" ? body.requestId : null,
	});
}

function networkError(error) {
	if (error instanceof KennelServiceError) return error;
	const timedOut = error?.name === "AbortError" || error?.name === "TimeoutError";
	return new KennelServiceError(
		timedOut ? "DAEMON_TIMEOUT" : "DAEMON_UNREACHABLE",
		timedOut ? "Kennel daemon request timed out" : "Could not reach the Kennel daemon",
		{ cause: error, retryable: true },
	);
}

function validateEnvelope(payload, key, route) {
	if (!isRecord(payload) || !Array.isArray(payload[key])) {
		fail("DAEMON_RESPONSE_INVALID", `Kennel returned an invalid ${route} response`);
	}
	return payload[key];
}

function validateConversation(payload, sessionId) {
	if (
		!isRecord(payload) ||
		payload.sessionId !== sessionId ||
		!Array.isArray(payload.activities) ||
		!Array.isArray(payload.messages) ||
		!Array.isArray(payload.turns)
	) {
		fail("DAEMON_RESPONSE_INVALID", "Kennel returned an invalid conversation response");
	}
	return payload;
}

function validateSteerResponse(payload) {
	if (
		!isRecord(payload) ||
		typeof payload.providerTurnId !== "string" ||
		payload.providerTurnId.length === 0 ||
		(payload.activityId !== undefined && typeof payload.activityId !== "string")
	) {
		fail("DAEMON_RESPONSE_INVALID", "Kennel returned an invalid steer response");
	}
	return {
		providerTurnId: payload.providerTurnId,
		...(typeof payload.activityId === "string" && payload.activityId !== ""
			? { activityId: payload.activityId }
			: {}),
	};
}

function providerDecisionOptions(detail) {
	if (!isRecord(detail) || !Array.isArray(detail.decisions)) return [];
	const decisions = [];
	for (const value of detail.decisions) {
		// Never expose a partial provider decision set: omitting one action can
		// materially change the meaning of the remaining approval choices.
		if (!isRecord(value) || typeof value.id !== "string" || value.id.length === 0) return [];
		decisions.push({
			id: value.id,
			label: typeof value.label === "string" && value.label ? value.label : value.id,
		});
	}
	return decisions;
}

function approvalContext(detail) {
	if (!isRecord(detail)) return null;
	const context = {};
	let truncated = false;
	for (const field of ["reason", "command", "cwd"]) {
		const value = detail[field];
		if (typeof value === "string" && value.trim() !== "") {
			if (value.length > MAX_APPROVAL_CONTEXT_CHARS) truncated = true;
			context[field] = value.slice(0, MAX_APPROVAL_CONTEXT_CHARS);
		}
	}
	return Object.keys(context).length > 0 ? { ...context, truncated } : null;
}

function finiteNumberField(source, field, target) {
	const value = source[field];
	if (typeof value === "number" && Number.isFinite(value)) target[field] = value;
}

function stringField(source, field, target) {
	const value = source[field];
	if (typeof value === "string") target[field] = value;
}

/**
 * Project the daemon's rich conversation timeline to the narrow island
 * contract. In particular, activities/messages/settings never cross the
 * Electron boundary: provider command details are represented only by the
 * bounded approval context above.
 */
function publicConversation(conversation) {
	const result = { sessionId: conversation.sessionId };
	stringField(conversation, "controller", result);

	if (Array.isArray(conversation.turns)) {
		result.turns = conversation.turns
			.filter(isRecord)
			.map((turn) => {
				const projected = {};
				stringField(turn, "id", projected);
				stringField(turn, "state", projected);
				return projected;
			});
	}
	if (Array.isArray(conversation.capabilities)) {
		result.capabilities = conversation.capabilities.filter((value) => typeof value === "string");
	}
	if (isRecord(conversation.account)) {
		const account = {};
		stringField(conversation.account, "planLabel", account);
		if (Object.keys(account).length > 0) result.account = account;
	}
	if (isRecord(conversation.rateLimits)) {
		const rateLimits = {};
		for (const field of ["planLabel", "title"]) stringField(conversation.rateLimits, field, rateLimits);
		for (const field of [
			"primaryUsedPercent",
			"primaryResetsInSeconds",
			"secondaryUsedPercent",
			"secondaryResetsInSeconds",
		]) {
			finiteNumberField(conversation.rateLimits, field, rateLimits);
		}
		if (Object.keys(rateLimits).length > 0) result.rateLimits = rateLimits;
	}
	if (isRecord(conversation.usage)) {
		const usage = {};
		for (const field of [
			"cachedTokens",
			"contextUsed",
			"contextWindow",
			"cost",
			"inputTokens",
			"outputTokens",
			"totalTokens",
		]) {
			finiteNumberField(conversation.usage, field, usage);
		}
		stringField(conversation.usage, "currency", usage);
		if (Object.keys(usage).length > 0) result.usage = usage;
	}
	return result;
}

function pendingConversationActions(conversation) {
	const approvals = [];
	const inputs = [];

	for (const activity of conversation.activities) {
		if (!isRecord(activity) || activity.status !== "pending" || typeof activity.requestId !== "string") continue;
		if (activity.activityKind === "approval") {
			const context = approvalContext(activity.detail);
			approvals.push({
				...(typeof activity.id === "string" && activity.id !== "" ? { activityId: activity.id } : {}),
				requestId: activity.requestId,
				summary: typeof activity.summary === "string" ? activity.summary : "Approval required",
				decisions: providerDecisionOptions(activity.detail),
				...(context ? { context } : {}),
			});
			continue;
		}
		if (activity.activityKind === "user_input") {
			const detail = isRecord(activity.detail) ? activity.detail : {};
			inputs.push({
				...(typeof activity.id === "string" && activity.id !== "" ? { activityId: activity.id } : {}),
				requestId: activity.requestId,
				summary: typeof activity.summary === "string" ? activity.summary : "Input required",
				detail: {
					...(typeof detail.inputMode === "string" ? { inputMode: detail.inputMode } : {}),
					...(typeof detail.message === "string" ? { message: detail.message } : {}),
					...(isRecord(detail.schema) ? { schema: detail.schema } : {}),
					...(typeof detail.url === "string" ? { url: detail.url } : {}),
					...(typeof detail.elicitationId === "string" ? { elicitationId: detail.elicitationId } : {}),
				},
			});
		}
	}

	return { approvals, inputs };
}

function normalizedInputContent(action, content) {
	if (!INPUT_ACTIONS.has(action)) {
		fail("INVALID_ARGUMENT", "action must be accept, decline, or cancel", { details: { field: "action" } });
	}
	if (content === undefined) return undefined;
	if (action !== "accept") {
		fail("INVALID_ARGUMENT", "content is only allowed when action is accept", { details: { field: "content" } });
	}
	if (!isPlainObject(content)) {
		fail("INVALID_ARGUMENT", "content must be a plain object", { details: { field: "content" } });
	}

	let encoded;
	try {
		encoded = JSON.stringify(content);
	} catch (cause) {
		fail("INVALID_ARGUMENT", "content must be JSON serializable", { cause, details: { field: "content" } });
	}
	if (typeof encoded !== "string") {
		fail("INVALID_ARGUMENT", "content must encode a JSON object", { details: { field: "content" } });
	}
	if (utf8Bytes(encoded) > MAX_INPUT_CONTENT_BYTES) {
		fail("INVALID_ARGUMENT", "content is too large", { details: { field: "content" } });
	}
	const normalized = JSON.parse(encoded);
	if (!isPlainObject(normalized)) {
		fail("INVALID_ARGUMENT", "content must encode a JSON object", { details: { field: "content" } });
	}
	return normalized;
}

function needsPendingConversation(session) {
	if (!isRecord(session) || session.mode !== "chat") return false;
	if (session.status === "needs_input") return true;
	const activityState = isRecord(session.activity) ? session.activity.state : null;
	return activityState === "blocked" || activityState === "waiting_input";
}

function hasActiveConversation(session) {
	if (!isRecord(session) || session.mode !== "chat") return false;
	const activityState = isRecord(session.activity) ? session.activity.state : null;
	return activityState === "active";
}

function normalizeSteerText(value) {
	if (typeof value !== "string" || value.trim() === "" || utf8Bytes(value) > MAX_STEER_TEXT_BYTES) {
		fail("INVALID_ARGUMENT", "text must be a non-empty string of at most 64 KiB", {
			details: { field: "text" },
		});
	}
	return value;
}

/**
 * Attach-only client for an already-running local Kennel daemon.
 *
 * There is deliberately no generic request method and no daemon lifecycle API.
 * Standalone mode discovers the daemon from an injected/standard run file.
 * Unified-app mode injects `getConnection`; that path never resolves a home or
 * reads the filesystem, and accepts only a loopback port (plus optional trusted
 * metadata). Both modes independently verify `/readyz` before any API request.
 */
export function createKennelService(options = {}) {
	const getConnection = options.getConnection;
	if (getConnection !== undefined && typeof getConnection !== "function") {
		throw new TypeError("getConnection must be a function");
	}
	const fs = getConnection ? null : options.fs ?? nodeFs;
	const fetchImpl = options.fetch ?? globalThis.fetch;
	const timeoutMs = options.timeoutMs ?? DEFAULT_TIMEOUT_MS;

	if (!getConnection && (!fs || typeof fs.readFile !== "function")) throw new TypeError("fs.readFile is required");
	if (typeof fetchImpl !== "function") throw new TypeError("fetch is required");
	if (!Number.isInteger(timeoutMs) || timeoutMs < 1 || timeoutMs > 30_000) {
		throw new TypeError("timeoutMs must be an integer between 1 and 30000");
	}

	let runFilePath = null;
	if (getConnection) {
		// Deliberately bypass every run-file/home option in unified-app mode. The
		// supervisor already owns daemon discovery and is the only trusted source.
	} else if (options.runFilePath !== undefined) {
		if (
			typeof options.runFilePath !== "string" ||
			options.runFilePath.includes("\0") ||
			!path.isAbsolute(options.runFilePath)
		) {
			throw new TypeError("runFilePath must be an absolute path");
		}
		runFilePath = path.normalize(options.runFilePath);
	} else {
		const injectedHome = options.home ?? os.homedir;
		const home = typeof injectedHome === "function" ? injectedHome() : injectedHome;
		if (typeof home !== "string" || home.length === 0) throw new TypeError("home is required");
		runFilePath = path.join(home, ".kennel", "running.json");
	}

	async function requestJSON(connection, route, requestOptions = {}) {
		const controller = new AbortController();
		const timeout = setTimeout(() => controller.abort(), timeoutMs);
		timeout.unref?.();
		const method = requestOptions.method ?? "GET";
		const headers = {
			accept: "application/json",
			...(requestOptions.body === undefined ? {} : { "content-type": "application/json" }),
		};

		let response;
		let text;
		try {
			response = await fetchImpl(`${connection.baseUrl}${route}`, {
				method,
				headers,
				redirect: "error",
				signal: controller.signal,
				...(requestOptions.body === undefined ? {} : { body: JSON.stringify(requestOptions.body) }),
			});
			if (!response || typeof response.status !== "number" || typeof response.text !== "function") {
				fail("DAEMON_RESPONSE_INVALID", "Kennel returned an invalid HTTP response");
			}
			// Keep the abort timer active while the body is consumed; receiving headers
			// alone does not mean a local daemon response has completed.
			text = await readResponseText(response);
		} catch (error) {
			throw networkError(error);
		} finally {
			clearTimeout(timeout);
		}

		const payload = parseResponseJSON(text, route);
		if (!response.ok) throw remoteError(response, payload);
		if (response.status === 204) return null;
		if (payload === null) fail("DAEMON_RESPONSE_INVALID", `Kennel returned an empty response for ${route}`);
		return payload;
	}

	async function discoverConnection() {
		if (getConnection) {
			let injected;
			try {
				injected = await getConnection();
			} catch (cause) {
				fail("DAEMON_NOT_RUNNING", "Kennel daemon connection is unavailable", { cause, retryable: true });
			}
			const connection = normalizeInjectedConnection(injected);
			return {
				...connection,
				// Never accept a host/base URL from the callback. The local control plane
				// remains pinned to IPv4 loopback even if extra fields are supplied.
				baseUrl: `http://127.0.0.1:${connection.port}`,
			};
		}

		let contents;
		try {
			contents = await fs.readFile(runFilePath, "utf8");
		} catch (cause) {
			if (cause?.code === "ENOENT") {
				fail("DAEMON_NOT_RUNNING", "Kennel daemon is not running", { cause, retryable: true });
			}
			fail("RUN_FILE_UNREADABLE", "Could not read Kennel running.json", { cause, retryable: true });
		}

		const runInfo = parseRunFile(contents);
		return {
			...runInfo,
			// Never accept a host from disk. Kennel's local control plane is loopback-only.
			baseUrl: `http://127.0.0.1:${runInfo.port}`,
		};
	}

	async function attachDaemon() {
		const connection = await discoverConnection();

		let probe;
		try {
			probe = await requestJSON(connection, "/readyz");
		} catch (error) {
			if (error instanceof KennelServiceError && error.status !== null) {
				throw new KennelServiceError("DAEMON_NOT_READY", "Kennel daemon is not ready", {
					cause: error,
					status: error.status,
					retryable: true,
				});
			}
			throw error;
		}
		if (
			!isRecord(probe) ||
			probe.service !== DAEMON_SERVICE_NAME ||
			!Number.isInteger(probe.pid) ||
			probe.pid <= 0 ||
			(connection.pid !== undefined && probe.pid !== connection.pid)
		) {
			fail("DAEMON_IDENTITY_MISMATCH", "The trusted connection does not identify the daemon answering on this port");
		}
		if (probe.status !== "ready") {
			fail("DAEMON_NOT_READY", "Kennel daemon is not ready", { retryable: true });
		}
		return { ...connection, pid: probe.pid };
	}

	async function loadConversation(connection, rawSessionId) {
		const sessionId = normalizeIdentifier(rawSessionId, "sessionId");
		const payload = await requestJSON(
			connection,
			`/api/v1/sessions/${encodeSegment(sessionId, "sessionId")}/conversation`,
		);
		const conversation = validateConversation(payload, sessionId);
		return {
			conversation: publicConversation(conversation),
			pending: pendingConversationActions(conversation),
		};
	}

	async function attach() {
		return publicDaemon(await attachDaemon());
	}

	async function getConversation(sessionId) {
		const connection = await attachDaemon();
		return loadConversation(connection, sessionId);
	}

	async function getSnapshot(snapshotOptions = {}) {
		if (!isRecord(snapshotOptions)) fail("INVALID_ARGUMENT", "snapshot options must be an object");
		const projectId = snapshotOptions.projectId === undefined
			? null
			: normalizeIdentifier(snapshotOptions.projectId, "projectId");
		const activeOnly = snapshotOptions.activeOnly ?? true;
		if (typeof activeOnly !== "boolean") {
			fail("INVALID_ARGUMENT", "activeOnly must be a boolean", { details: { field: "activeOnly" } });
		}
		const includePendingConversations = snapshotOptions.includePendingConversations ?? false;
		if (typeof includePendingConversations !== "boolean") {
			fail("INVALID_ARGUMENT", "includePendingConversations must be a boolean", {
				details: { field: "includePendingConversations" },
			});
		}
		const includeActiveConversations = snapshotOptions.includeActiveConversations ?? false;
		if (typeof includeActiveConversations !== "boolean") {
			fail("INVALID_ARGUMENT", "includeActiveConversations must be a boolean", {
				details: { field: "includeActiveConversations" },
			});
		}
		const requestedConversationIds = snapshotOptions.conversationSessionIds ?? [];
		if (!Array.isArray(requestedConversationIds) || requestedConversationIds.length > MAX_CONVERSATIONS_PER_SNAPSHOT) {
			fail("INVALID_ARGUMENT", `conversationSessionIds must contain at most ${MAX_CONVERSATIONS_PER_SNAPSHOT} ids`);
		}
		const explicitConversationIds = [...new Set(
			requestedConversationIds.map((value) => normalizeIdentifier(value, "conversationSessionIds")),
		)];

		const connection = await attachDaemon();
		const sessionQuery = new URLSearchParams();
		if (projectId) sessionQuery.set("project", projectId);
		sessionQuery.set("active", String(activeOnly));
		const [projectPayload, sessionPayload, notificationPayload] = await Promise.all([
			requestJSON(connection, "/api/v1/projects"),
			requestJSON(connection, `/api/v1/sessions?${sessionQuery.toString()}`),
			requestJSON(connection, "/api/v1/notifications?status=unresolved&limit=100"),
		]);

		const projects = validateEnvelope(projectPayload, "projects", "projects");
		const allSessions = validateEnvelope(sessionPayload, "sessions", "sessions");
		const allNotifications = validateEnvelope(notificationPayload, "notifications", "notifications");
		const sessions = projectId
			? allSessions.filter((session) => isRecord(session) && session.projectId === projectId)
			: allSessions;
		const notifications = projectId
			? allNotifications.filter((notification) => isRecord(notification) && notification.projectId === projectId)
			: allNotifications;
		const sessionById = new Map(
			sessions.filter(isRecord).map((session) => [session.id, session]),
		);

		for (const sessionId of explicitConversationIds) {
			const session = sessionById.get(sessionId);
			if (!session) {
				fail("INVALID_ARGUMENT", `conversation session ${sessionId} is not present in this snapshot`);
			}
			if (session.mode !== "chat") {
				fail("INVALID_ARGUMENT", `conversation session ${sessionId} is not a chat session`);
			}
		}

		const selectedConversationIds = new Set(explicitConversationIds);
		const pendingCandidateIds = includePendingConversations
			? sessions
				.filter(needsPendingConversation)
				.map((session) => session.id)
				.filter((sessionId) => typeof sessionId === "string" && !selectedConversationIds.has(sessionId))
			: [];
		const remainingConversationSlots = MAX_CONVERSATIONS_PER_SNAPSHOT - selectedConversationIds.size;
		for (const sessionId of pendingCandidateIds.slice(0, remainingConversationSlots)) {
			selectedConversationIds.add(sessionId);
		}
		const activeCandidateIds = includeActiveConversations
			? sessions
				.filter(hasActiveConversation)
				.map((session) => session.id)
				.filter((sessionId) => typeof sessionId === "string" && !selectedConversationIds.has(sessionId))
			: [];
		const remainingActiveSlots = MAX_CONVERSATIONS_PER_SNAPSHOT - selectedConversationIds.size;
		for (const sessionId of activeCandidateIds.slice(0, remainingActiveSlots)) {
			selectedConversationIds.add(sessionId);
		}
		const conversationIds = [...selectedConversationIds];
		const pendingConversationsTruncated = includePendingConversations &&
			pendingCandidateIds.length > remainingConversationSlots;
		const activeConversationsTruncated = includeActiveConversations &&
			activeCandidateIds.length > remainingActiveSlots;

		const conversationEntries = await Promise.all(
			conversationIds.map(async (sessionId) => [sessionId, await loadConversation(connection, sessionId)]),
		);
		const unreadCount = projectId
			? notifications.filter((notification) => isRecord(notification) && notification.status === "unread").length
			: Number.isInteger(notificationPayload.unreadCount) ? notificationPayload.unreadCount : 0;
		const unresolvedCount = projectId
			? notifications.filter((notification) => isRecord(notification) && !notification.resolvedAt).length
			: Number.isInteger(notificationPayload.unresolvedCount) ? notificationPayload.unresolvedCount : notifications.length;

		return {
			daemon: publicDaemon(connection),
			projects,
			project: projectId
				? projects.find((project) => isRecord(project) && project.id === projectId) ?? null
				: null,
			sessions,
			notifications,
			notificationCounts: { unread: unreadCount, unresolved: unresolvedCount },
			notificationsTruncated: typeof notificationPayload.nextCursor === "string" && notificationPayload.nextCursor !== "",
			pendingConversationsTruncated,
			activeConversationsTruncated,
			conversations: Object.fromEntries(conversationEntries),
		};
	}

	async function resolveApproval(input) {
		if (!isRecord(input)) fail("INVALID_ARGUMENT", "approval input must be an object");
		const sessionId = normalizeIdentifier(input.sessionId, "sessionId");
		const requestId = normalizeIdentifier(input.requestId, "requestId");
		const decisionId = normalizeIdentifier(input.decisionId, "decisionId");
		const connection = await attachDaemon();
		const state = await loadConversation(connection, sessionId);
		const approval = state.pending.approvals.find((value) => value.requestId === requestId);
		if (!approval) {
			fail("APPROVAL_NOT_PENDING", "The provider approval is no longer pending");
		}
		if (!approval.decisions.some((decision) => decision.id === decisionId)) {
			fail("DECISION_NOT_OFFERED", "The provider did not offer this approval decision");
		}

		await requestJSON(
			connection,
			`/api/v1/sessions/${encodeSegment(sessionId, "sessionId")}/conversation/approvals/${encodeSegment(requestId, "requestId")}/resolve`,
			{ method: "POST", body: { decisionId } },
		);
		return { ok: true, sessionId, requestId, decisionId };
	}

	async function resolveInput(input) {
		if (!isRecord(input)) fail("INVALID_ARGUMENT", "input resolution must be an object");
		const sessionId = normalizeIdentifier(input.sessionId, "sessionId");
		const requestId = normalizeIdentifier(input.requestId, "requestId");
		const action = normalizeIdentifier(input.action, "action");
		const content = normalizedInputContent(action, input.content);
		const connection = await attachDaemon();
		const state = await loadConversation(connection, sessionId);
		if (!state.pending.inputs.some((value) => value.requestId === requestId)) {
			fail("INPUT_NOT_PENDING", "The provider input request is no longer pending");
		}

		const body = { action, ...(content === undefined ? {} : { content }) };
		await requestJSON(
			connection,
			`/api/v1/sessions/${encodeSegment(sessionId, "sessionId")}/conversation/inputs/${encodeSegment(requestId, "requestId")}/resolve`,
			{ method: "POST", body },
		);
		return { ok: true, sessionId, requestId, action };
	}

	async function steer(input) {
		if (!isRecord(input)) fail("INVALID_ARGUMENT", "steer input must be an object");
		const sessionId = normalizeIdentifier(input.sessionId, "sessionId");
		const text = normalizeSteerText(input.text);
		const clientMessageId = input.clientMessageId === undefined
			? undefined
			: normalizeIdentifier(input.clientMessageId, "clientMessageId");
		const connection = await attachDaemon();
		const payload = await requestJSON(
			connection,
			`/api/v1/sessions/${encodeSegment(sessionId, "sessionId")}/conversation/steer`,
			{
				method: "POST",
				body: { text, ...(clientMessageId === undefined ? {} : { clientMessageId }) },
			},
		);
		return validateSteerResponse(payload);
	}

	async function interrupt(input) {
		if (!isRecord(input)) fail("INVALID_ARGUMENT", "interrupt input must be an object");
		const sessionId = normalizeIdentifier(input.sessionId, "sessionId");
		const connection = await attachDaemon();
		await requestJSON(
			connection,
			`/api/v1/sessions/${encodeSegment(sessionId, "sessionId")}/conversation/interrupt`,
			{ method: "POST" },
		);
		return { ok: true, sessionId };
	}

	return Object.freeze({
		runFilePath,
		attach,
		getSnapshot,
		getConversation,
		resolveApproval,
		resolveInput,
		steer,
		interrupt,
	});
}
