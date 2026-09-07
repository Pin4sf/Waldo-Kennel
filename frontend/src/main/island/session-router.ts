// @ts-expect-error Canonical Island service is a tested JavaScript package.
import { createKennelService } from "../../../../packages/kennel-island/desktop/kennel-service.mjs";

export type IslandSessionRouteTarget = { projectId?: string; sessionId: string };
export type IslandDaemonConnection = { port: number; pid?: number } | null;

type SnapshotService = {
	getSnapshot(options: { activeOnly: false }): Promise<unknown>;
};

type IslandSessionRouterOptions = {
	getConnection: () => IslandDaemonConnection;
	fetch: (input: string, init: RequestInit) => Promise<Response>;
	openSession: (target: { projectId: string; sessionId: string }) => void;
	/** Test seam; production constructs the allowlisted Kennel service. */
	service?: SnapshotService;
};

export type IslandSessionRouter = {
	focusSession(target: IslandSessionRouteTarget): Promise<boolean>;
};

function isRecord(value: unknown): value is Record<string, unknown> {
	return typeof value === "object" && value !== null && !Array.isArray(value);
}

function identifier(value: unknown, field: string): string {
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

/**
 * Main-app owner for validated session routing.
 *
 * This intentionally does not depend on the Island controller. External links
 * can still recreate/focus the workspace if Island initialization fails or the
 * current machine does not support an Island window.
 */
export function createIslandSessionRouter(options: IslandSessionRouterOptions): IslandSessionRouter {
	const service: SnapshotService = options.service ?? createKennelService({
		fetch: options.fetch,
		getConnection: options.getConnection,
	});

	return {
		async focusSession(target) {
			if (!isRecord(target)) throw new TypeError("session target is invalid");
			const sessionId = identifier(target.sessionId, "sessionId");
			const requestedProjectId = target.projectId === undefined
				? undefined
				: identifier(target.projectId, "projectId");
			const snapshot = await service.getSnapshot({ activeOnly: false });
			if (!isRecord(snapshot) || !Array.isArray(snapshot.sessions)) {
				throw new Error("Kennel returned an invalid session snapshot");
			}
			const candidate = snapshot.sessions.find((entry) =>
				isRecord(entry) && entry.id === sessionId &&
				(requestedProjectId === undefined || entry.projectId === requestedProjectId));
			if (!candidate || !isRecord(candidate)) return false;
			const projectId = identifier(candidate.projectId, "projectId");
			options.openSession({ projectId, sessionId });
			return true;
		},
	};
}
