#!/usr/bin/env node

import * as fs from "node:fs/promises";
import http from "node:http";
import os from "node:os";
import path from "node:path";
import { fileURLToPath } from "node:url";

const LOOPBACK = "127.0.0.1";
const SERVICE_NAME = "agent-orchestrator-daemon";
const MAX_BODY_BYTES = 64 * 1024;
const scriptDirectory = path.dirname(fileURLToPath(import.meta.url));
const projectRoot = path.resolve(scriptDirectory, "..");

function fixtureTimestamp(offsetSeconds = 0) {
	return new Date(Date.now() + offsetSeconds * 1_000).toISOString();
}

function projectFixture() {
	return {
		id: "kennel-island-fixture",
		name: "Kennel Island",
		path: projectRoot,
		kind: "single_repo",
		sessionPrefix: "island",
	};
}

function sessionFixture({ id, displayName, branch, status, activity, updatedOffsetSeconds }) {
	return {
		id,
		projectId: "kennel-island-fixture",
		displayName,
		kind: "worker",
		mode: "chat",
		harness: "codex",
		branch,
		status,
		activity: {
			state: activity,
			lastActivityAt: fixtureTimestamp(updatedOffsetSeconds),
		},
		isTerminated: false,
		createdAt: fixtureTimestamp(-3_600),
		updatedAt: fixtureTimestamp(updatedOffsetSeconds),
		prs: [],
	};
}

function conversationFixture({ sessionId, controller, turnState, activities = [], rateLimits, capabilities = [] }) {
	return {
		conversationId: `conversation-${sessionId}`,
		sessionId,
		mode: "chat",
		controller,
		latestSequence: activities.length,
		oldestSequence: activities.length > 0 ? 1 : 0,
		hasMoreBefore: false,
		turns: [{ id: `turn-${sessionId}`, state: turnState }],
		messages: [],
		activities,
		settings: {},
		capabilities,
		...(rateLimits
			? {
				account: { planLabel: rateLimits.planLabel },
				rateLimits,
				usage: {
					contextUsed: 31_200,
					contextWindow: 200_000,
					inputTokens: 26_300,
					outputTokens: 4_900,
					cachedTokens: 12_400,
					totalTokens: 31_200,
					cost: 0.42,
					currency: "USD",
				},
			}
			: {}),
	};
}

function buildFixtureState() {
	const longCommand = [
		"npm run verify -- --scope kennel-island",
		...Array.from({ length: 300 }, (_, index) => `--fixture-guard-${String(index + 1).padStart(3, "0")}=enabled`),
	].join(" ");

	const sessions = [
		sessionFixture({
			id: "approval-normal",
			displayName: "Review native shell access",
			branch: "codex/native-shell",
			status: "needs_input",
			activity: "waiting_input",
			updatedOffsetSeconds: -5,
		}),
		sessionFixture({
			id: "input-enum",
			displayName: "Choose deployment target",
			branch: "codex/release-flow",
			status: "needs_input",
			activity: "waiting_input",
			updatedOffsetSeconds: -15,
		}),
		sessionFixture({
			id: "approval-interrupt",
			displayName: "Stop a stalled agent turn",
			branch: "codex/interrupt-flow",
			status: "needs_input",
			activity: "waiting_input",
			updatedOffsetSeconds: -20,
		}),
		sessionFixture({
			id: "approval-truncated",
			displayName: "Review a long command",
			branch: "codex/fixture-guard",
			status: "needs_input",
			activity: "blocked",
			updatedOffsetSeconds: -25,
		}),
		sessionFixture({
			id: "running-chat",
			displayName: "Polish the Kennel Island",
			branch: "codex/island-polish",
			status: "working",
			activity: "active",
			updatedOffsetSeconds: -35,
		}),
	];

	const conversations = new Map([
		[
			"approval-normal",
			conversationFixture({
				sessionId: "approval-normal",
				controller: "waiting_input",
				turnState: "running",
				capabilities: ["interrupt"],
				activities: [
					{
						kind: "activity",
						id: "activity-approval-normal",
						sequence: 1,
						revision: 1,
						activityKind: "approval",
						status: "pending",
						requestId: "request-approval-normal",
						summary: "Allow the native QA command?",
						detail: {
							reason: "The fixture is asking for an explicit provider decision before it runs native QA.",
							command: "npm run test && npm run build",
							cwd: projectRoot,
							decisions: [
								{ id: "deny", label: "Deny" },
								{ id: "allow_once", label: "Allow once" },
								{ id: "allow_session", label: "Allow for session" },
							],
						},
						createdAt: fixtureTimestamp(-5),
					},
				],
			}),
		],
		[
			"input-enum",
			conversationFixture({
				sessionId: "input-enum",
				controller: "waiting_input",
				turnState: "running",
				capabilities: ["interrupt"],
				activities: [
					{
						kind: "activity",
						id: "activity-input-enum",
						sequence: 1,
						revision: 1,
						activityKind: "user_input",
						status: "pending",
						requestId: "request-input-enum",
						summary: "Choose where to deploy the fixture",
						detail: {
							inputMode: "form",
							message: "Where should the release candidate be deployed?",
							elicitationId: "elicitation-deployment-target",
							schema: {
								title: "Deployment target",
								type: "object",
								required: ["deploymentTarget"],
								properties: {
									deploymentTarget: {
										type: "string",
										default: "staging",
										oneOf: [
											{ const: "staging", title: "Staging" },
											{ const: "production", title: "Production" },
										],
									},
								},
							},
						},
						createdAt: fixtureTimestamp(-15),
					},
				],
			}),
		],
		[
			"approval-interrupt",
			conversationFixture({
				sessionId: "approval-interrupt",
				controller: "waiting_input",
				turnState: "waiting_input",
				capabilities: ["interrupt"],
				activities: [
					{
						kind: "activity",
						id: "activity-approval-interrupt",
						sequence: 1,
						revision: 1,
						activityKind: "approval",
						status: "pending",
						requestId: "request-approval-interrupt",
						summary: "Allow the stalled turn to continue?",
						detail: {
							reason: "Use the island interrupt confirmation to discard this controlled fixture turn.",
							command: "npm run controlled-fixture-turn",
							cwd: projectRoot,
							decisions: [
								{ id: "deny", label: "Deny" },
								{ id: "allow_once", label: "Allow once" },
							],
						},
						createdAt: fixtureTimestamp(-20),
					},
				],
			}),
		],
		[
			"approval-truncated",
			conversationFixture({
				sessionId: "approval-truncated",
				controller: "waiting_input",
				turnState: "running",
				capabilities: ["interrupt"],
				activities: [
					{
						kind: "activity",
						id: "activity-approval-truncated",
						sequence: 1,
						revision: 1,
						activityKind: "approval",
						status: "pending",
						requestId: "request-approval-truncated",
						summary: "Review the full command in Kennel",
						detail: {
							reason: "This command intentionally exceeds the island disclosure limit and must not be approved inline.",
							command: longCommand,
							cwd: projectRoot,
							decisions: [
								{ id: "deny", label: "Deny" },
								{ id: "allow_once", label: "Allow once" },
							],
						},
						createdAt: fixtureTimestamp(-25),
					},
				],
			}),
		],
		[
			"running-chat",
			conversationFixture({
				sessionId: "running-chat",
				controller: "busy",
				turnState: "running",
				capabilities: ["steer", "interrupt"],
				rateLimits: {
					title: "Five-hour window",
					planLabel: "Codex Plus",
					primaryUsedPercent: 38,
					primaryResetsInSeconds: 5_400,
					secondaryUsedPercent: 71,
					secondaryResetsInSeconds: 248_400,
				},
			}),
		],
	]);

	const notifications = [
		{
			id: "notification-approval-normal",
			projectId: "kennel-island-fixture",
			sessionId: "approval-normal",
			status: "unread",
			createdAt: fixtureTimestamp(-5),
		},
		{
			id: "notification-input-enum",
			projectId: "kennel-island-fixture",
			sessionId: "input-enum",
			status: "unread",
			createdAt: fixtureTimestamp(-15),
		},
		{
			id: "notification-approval-interrupt",
			projectId: "kennel-island-fixture",
			sessionId: "approval-interrupt",
			status: "unread",
			createdAt: fixtureTimestamp(-20),
		},
		{
			id: "notification-approval-truncated",
			projectId: "kennel-island-fixture",
			sessionId: "approval-truncated",
			status: "unread",
			createdAt: fixtureTimestamp(-25),
		},
	];

	return {
		project: projectFixture(),
		sessions,
		conversations,
		notifications,
		nextMutationId: 1,
	};
}

function sendJson(response, status, body) {
	const payload = JSON.stringify(body);
	response.writeHead(status, {
		"cache-control": "no-store",
		"content-length": Buffer.byteLength(payload),
		"content-type": "application/json; charset=utf-8",
	});
	response.end(payload);
}

function sendNoContent(response) {
	response.writeHead(204, { "cache-control": "no-store" });
	response.end();
}

function sendError(response, status, code, message) {
	sendJson(response, status, { code, message });
}

async function readJsonBody(request) {
	const chunks = [];
	let byteLength = 0;
	for await (const chunk of request) {
		byteLength += chunk.byteLength;
		if (byteLength > MAX_BODY_BYTES) {
			const error = new Error("Request body is too large");
			error.status = 413;
			error.code = "REQUEST_TOO_LARGE";
			throw error;
		}
		chunks.push(chunk);
	}
	if (byteLength === 0) return {};
	try {
		return JSON.parse(Buffer.concat(chunks, byteLength).toString("utf8"));
	} catch (cause) {
		const error = new Error("Request body must be valid JSON", { cause });
		error.status = 400;
		error.code = "INVALID_JSON";
		throw error;
	}
}

function decodedSegment(value) {
	try {
		return decodeURIComponent(value);
	} catch {
		return null;
	}
}

function findSession(state, sessionId) {
	return state.sessions.find((session) => session.id === sessionId);
}

function touchSession(session, { status, activity }) {
	const now = fixtureTimestamp();
	session.status = status;
	session.activity = { state: activity, lastActivityAt: now };
	session.updatedAt = now;
}

function resolveNotification(state, sessionId) {
	const now = fixtureTimestamp();
	for (const notification of state.notifications) {
		if (notification.sessionId === sessionId && !notification.resolvedAt) {
			notification.status = "read";
			notification.resolvedAt = now;
		}
	}
}

function resolvePendingActivity(state, sessionId, activity) {
	activity.status = "resolved";
	activity.revision = Number.isInteger(activity.revision) ? activity.revision + 1 : 2;
	const conversation = state.conversations.get(sessionId);
	const session = findSession(state, sessionId);
	if (conversation) {
		conversation.controller = "busy";
		for (const turn of conversation.turns) {
			if (turn.state === "waiting_input") turn.state = "running";
		}
		if (!conversation.capabilities.includes("steer")) conversation.capabilities.push("steer");
	}
	if (session) touchSession(session, { status: "working", activity: "active" });
	resolveNotification(state, sessionId);
}

function pendingActivity(conversation, activityKind, requestId) {
	return conversation?.activities.find((activity) =>
		activity.activityKind === activityKind && activity.requestId === requestId && activity.status === "pending",
	);
}

function publicNotifications(state) {
	const notifications = state.notifications.filter((notification) => !notification.resolvedAt);
	return {
		notifications,
		unreadCount: notifications.filter((notification) => notification.status === "unread").length,
		unresolvedCount: notifications.length,
	};
}

function logMutation(kind, detail) {
	process.stdout.write(`KENNEL_FIXTURE_MUTATION ${JSON.stringify({ kind, ...detail })}\n`);
}

async function handleRequest(request, response, state) {
	const requestUrl = new URL(request.url ?? "/", `http://${LOOPBACK}`);
	const method = request.method ?? "GET";
	const pathname = requestUrl.pathname;

	if (method === "GET" && pathname === "/readyz") {
		sendJson(response, 200, {
			status: "ready",
			service: SERVICE_NAME,
			pid: process.pid,
		});
		return;
	}

	if (method === "GET" && pathname === "/api/v1/projects") {
		sendJson(response, 200, { projects: [state.project] });
		return;
	}

	if (method === "GET" && pathname === "/api/v1/sessions") {
		const projectId = requestUrl.searchParams.get("project");
		const activeOnly = requestUrl.searchParams.get("active") === "true";
		const sessions = state.sessions.filter((session) =>
			(!projectId || session.projectId === projectId) && (!activeOnly || !session.isTerminated),
		);
		sendJson(response, 200, { sessions });
		return;
	}

	if (method === "GET" && pathname === "/api/v1/notifications") {
		sendJson(response, 200, publicNotifications(state));
		return;
	}

	const conversationMatch = pathname.match(/^\/api\/v1\/sessions\/([^/]+)\/conversation$/);
	if (method === "GET" && conversationMatch) {
		const sessionId = decodedSegment(conversationMatch[1]);
		const conversation = sessionId ? state.conversations.get(sessionId) : null;
		if (!conversation) {
			sendError(response, 404, "SESSION_NOT_FOUND", "Fixture session was not found");
			return;
		}
		sendJson(response, 200, conversation);
		return;
	}

	const approvalMatch = pathname.match(
		/^\/api\/v1\/sessions\/([^/]+)\/conversation\/approvals\/([^/]+)\/resolve$/,
	);
	if (method === "POST" && approvalMatch) {
		const sessionId = decodedSegment(approvalMatch[1]);
		const requestId = decodedSegment(approvalMatch[2]);
		const conversation = sessionId ? state.conversations.get(sessionId) : null;
		const activity = requestId ? pendingActivity(conversation, "approval", requestId) : null;
		if (!sessionId || !requestId || !conversation || !activity) {
			sendError(response, 409, "APPROVAL_NOT_PENDING", "The fixture approval is no longer pending");
			return;
		}
		const body = await readJsonBody(request);
		const offered = Array.isArray(activity.detail?.decisions)
			? activity.detail.decisions.some((decision) => decision.id === body.decisionId)
			: false;
		if (!offered) {
			sendError(response, 422, "DECISION_NOT_OFFERED", "The provider did not offer this fixture decision");
			return;
		}
		resolvePendingActivity(state, sessionId, activity);
		logMutation("approval", { sessionId, requestId, decisionId: body.decisionId });
		sendNoContent(response);
		return;
	}

	const inputMatch = pathname.match(
		/^\/api\/v1\/sessions\/([^/]+)\/conversation\/inputs\/([^/]+)\/resolve$/,
	);
	if (method === "POST" && inputMatch) {
		const sessionId = decodedSegment(inputMatch[1]);
		const requestId = decodedSegment(inputMatch[2]);
		const conversation = sessionId ? state.conversations.get(sessionId) : null;
		const activity = requestId ? pendingActivity(conversation, "user_input", requestId) : null;
		if (!sessionId || !requestId || !conversation || !activity) {
			sendError(response, 409, "INPUT_NOT_PENDING", "The fixture input is no longer pending");
			return;
		}
		const body = await readJsonBody(request);
		if (!["accept", "decline", "cancel"].includes(body.action)) {
			sendError(response, 422, "INVALID_INPUT_ACTION", "Fixture input action must be accept, decline, or cancel");
			return;
		}
		if (
			body.action === "accept" &&
			(!body.content || !["staging", "production"].includes(body.content.deploymentTarget))
		) {
			sendError(response, 422, "INVALID_INPUT_CONTENT", "Choose staging or production");
			return;
		}
		resolvePendingActivity(state, sessionId, activity);
		logMutation("input", {
			sessionId,
			requestId,
			action: body.action,
			...(body.content ? { content: body.content } : {}),
		});
		sendNoContent(response);
		return;
	}

	const steerMatch = pathname.match(/^\/api\/v1\/sessions\/([^/]+)\/conversation\/steer$/);
	if (method === "POST" && steerMatch) {
		const sessionId = decodedSegment(steerMatch[1]);
		const conversation = sessionId ? state.conversations.get(sessionId) : null;
		const session = sessionId ? findSession(state, sessionId) : null;
		if (!sessionId || !conversation || !session) {
			sendError(response, 404, "SESSION_NOT_FOUND", "Fixture session was not found");
			return;
		}
		if (!conversation.turns.some((turn) => turn.state === "running")) {
			sendError(response, 409, "TURN_NOT_RUNNING", "The fixture turn is not running");
			return;
		}
		const body = await readJsonBody(request);
		if (typeof body.text !== "string" || body.text.trim() === "") {
			sendError(response, 422, "INVALID_STEER_TEXT", "Steer text must not be empty");
			return;
		}
		const mutationId = state.nextMutationId++;
		const activityId = `activity-steer-${mutationId}`;
		conversation.latestSequence += 2;
		conversation.messages.push({
			kind: "message",
			id: `message-steer-${mutationId}`,
			sequence: conversation.latestSequence - 1,
			role: "user",
			text: body.text,
			clientMessageId: typeof body.clientMessageId === "string" ? body.clientMessageId : undefined,
			createdAt: fixtureTimestamp(),
		});
		conversation.activities.push({
			kind: "activity",
			id: activityId,
			sequence: conversation.latestSequence,
			revision: 1,
			activityKind: "system",
			status: "completed",
			summary: "Steering guidance accepted by the fixture",
			createdAt: fixtureTimestamp(),
		});
		touchSession(session, { status: "working", activity: "active" });
		logMutation("steer", { sessionId, text: body.text, clientMessageId: body.clientMessageId });
		sendJson(response, 202, {
			providerTurnId: conversation.turns.find((turn) => turn.state === "running")?.id,
			activityId,
		});
		return;
	}

	const interruptMatch = pathname.match(/^\/api\/v1\/sessions\/([^/]+)\/conversation\/interrupt$/);
	if (method === "POST" && interruptMatch) {
		const sessionId = decodedSegment(interruptMatch[1]);
		const conversation = sessionId ? state.conversations.get(sessionId) : null;
		const session = sessionId ? findSession(state, sessionId) : null;
		if (!sessionId || !conversation || !session) {
			sendError(response, 404, "SESSION_NOT_FOUND", "Fixture session was not found");
			return;
		}
		for (const turn of conversation.turns) {
			if (turn.state === "running" || turn.state === "waiting_input") turn.state = "interrupted";
		}
		for (const activity of conversation.activities) {
			if (activity.status === "pending") activity.status = "cancelled";
		}
		conversation.controller = "ready";
		touchSession(session, { status: "idle", activity: "idle" });
		resolveNotification(state, sessionId);
		logMutation("interrupt", { sessionId });
		sendNoContent(response);
		return;
	}

	sendError(response, 404, "FIXTURE_ROUTE_NOT_FOUND", `${method} ${pathname} is not implemented by the fixture`);
}

async function firstExistingAppPath() {
	const requested = process.env.KENNEL_FIXTURE_APP_PATH;
	if (requested !== undefined) {
		if (!path.isAbsolute(requested)) {
			throw new Error("KENNEL_FIXTURE_APP_PATH must be an absolute path");
		}
		if (path.extname(requested).toLowerCase() !== ".app") {
			throw new Error("KENNEL_FIXTURE_APP_PATH must identify a macOS .app bundle");
		}
		const info = await fs.stat(requested);
		if (!info.isDirectory()) throw new Error("KENNEL_FIXTURE_APP_PATH must identify a macOS app directory");
		return path.normalize(requested);
	}

	const candidates = [
		path.resolve(projectRoot, "../Waldo-Kennel/frontend/out/Kennel-darwin-arm64/Kennel.app"),
		path.resolve(projectRoot, "../Waldo-Kennel/frontend/node_modules/electron/dist/Electron.app"),
		path.resolve(projectRoot, "node_modules/electron/dist/Electron.app"),
	];
	for (const candidate of candidates) {
		try {
			const info = await fs.stat(candidate);
			if (info.isDirectory()) return candidate;
		} catch (error) {
			if (error?.code !== "ENOENT") throw error;
		}
	}
	throw new Error("No local Kennel-compatible .app was found for app-state.json");
}

async function main() {
	const state = buildFixtureState();
	const appPath = await firstExistingAppPath();
	const temporaryDirectory = await fs.mkdtemp(path.join(os.tmpdir(), "kennel-island-fixture-"));
	const runFilePath = path.join(temporaryDirectory, "running.json");
	const appStatePath = path.join(temporaryDirectory, "app-state.json");
	const startedAt = fixtureTimestamp();

	const server = http.createServer((request, response) => {
		void handleRequest(request, response, state).catch((error) => {
			if (response.headersSent) {
				response.destroy(error);
				return;
			}
			sendError(
				response,
				Number.isInteger(error?.status) ? error.status : 500,
				typeof error?.code === "string" ? error.code : "FIXTURE_INTERNAL_ERROR",
				typeof error?.message === "string" ? error.message : "Fixture request failed",
			);
		});
	});

	await new Promise((resolve, reject) => {
		server.once("error", reject);
		server.listen(0, LOOPBACK, () => {
			server.off("error", reject);
			resolve();
		});
	});
	const address = server.address();
	if (!address || typeof address === "string") throw new Error("Fixture did not acquire a TCP port");

	await Promise.all([
		fs.writeFile(runFilePath, `${JSON.stringify({
			pid: process.pid,
			port: address.port,
			startedAt,
			owner: "kennel-island-fixture",
		}, null, 2)}\n`, { mode: 0o600 }),
		fs.writeFile(appStatePath, `${JSON.stringify({ appPath }, null, 2)}\n`, { mode: 0o600 }),
	]);

	process.stdout.write(`KENNEL_FIXTURE_PORT=${address.port}\n`);
	process.stdout.write(`KENNEL_FIXTURE_RUN_FILE=${runFilePath}\n`);
	process.stdout.write(`KENNEL_FIXTURE_APP_STATE=${appStatePath}\n`);
	process.stdout.write(`KENNEL_FIXTURE_APP_PATH=${appPath}\n`);
	process.stdout.write("KENNEL_FIXTURE_READY\n");

	let shuttingDown = false;
	const shutdown = async (signal) => {
		if (shuttingDown) return;
		shuttingDown = true;
		await new Promise((resolve) => server.close(resolve));
		await fs.rm(temporaryDirectory, { recursive: true, force: true });
		process.stdout.write(`KENNEL_FIXTURE_STOPPED=${signal}\n`);
	};

	for (const signal of ["SIGINT", "SIGTERM"]) {
		process.once(signal, () => {
			void shutdown(signal).then(
				() => process.exit(0),
				(error) => {
					process.stderr.write(`${error?.stack ?? error}\n`);
					process.exit(1);
				},
			);
		});
	}
}

main().catch((error) => {
	process.stderr.write(`${error?.stack ?? error}\n`);
	process.exitCode = 1;
});
