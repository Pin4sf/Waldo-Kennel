import { aoBridge } from "./bridge";
import { usesLiveDaemonPreview } from "./preview-mode";
import { setApiBaseUrl, setApiDaemonStatus } from "./api-client";

export type DaemonStatus = Awaited<ReturnType<typeof aoBridge.daemon.getStatus>>;

export function applyDaemonStatus(nextStatus: DaemonStatus): void {
	setApiDaemonStatus(nextStatus);
	if (nextStatus.state !== "ready") {
		setApiBaseUrl(null);
		return;
	}
	// In live browser preview the daemon is reached through the dev server's
	// proxy, so requests MUST stay same-origin. Pointing them at
	// http://127.0.0.1:<port> would bypass the proxy and hit a daemon that
	// serves no CORS headers, which fails in a way that looks like the daemon
	// being down rather than a misrouted request.
	if (usesLiveDaemonPreview) {
		setApiBaseUrl("");
		return;
	}
	if (nextStatus.port) {
		setApiBaseUrl(`http://127.0.0.1:${nextStatus.port}`);
		return;
	}
	setApiBaseUrl(null);
}

export async function refreshDaemonStatus(): Promise<DaemonStatus> {
	const nextStatus = await readDaemonStatus();
	applyDaemonStatus(nextStatus);
	return nextStatus;
}

export function readDaemonStatus(): Promise<DaemonStatus> {
	return aoBridge.daemon.getStatus();
}
