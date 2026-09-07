import { beforeEach, describe, expect, it, vi } from "vitest";

// Live browser preview reaches the daemon through the dev server's proxy, so
// this pins the one rule that makes that work.
vi.mock("./preview-mode", () => ({
	runsOutsideElectron: true,
	usesLiveDaemonPreview: true,
	usesPreviewWorkspaceData: false,
	usesWaldoUiPreview: true,
}));

const { setApiBaseUrl, setApiDaemonStatus } = vi.hoisted(() => ({
	setApiBaseUrl: vi.fn(),
	setApiDaemonStatus: vi.fn(),
}));
vi.mock("./api-client", () => ({ setApiBaseUrl, setApiDaemonStatus }));
vi.mock("./bridge", () => ({ aoBridge: { daemon: { getStatus: vi.fn() } } }));

import { applyDaemonStatus } from "./daemon-status";

beforeEach(() => {
	setApiBaseUrl.mockClear();
	setApiDaemonStatus.mockClear();
});

describe("applyDaemonStatus in live browser preview", () => {
	// Pointing at http://127.0.0.1:<port> would bypass the proxy and hit a
	// daemon that serves no CORS headers — failing in a way that looks like the
	// daemon being down rather than a misrouted request.
	it("keeps requests same-origin even when the status carries a port", () => {
		applyDaemonStatus({ state: "ready", port: 3032 });
		expect(setApiBaseUrl).toHaveBeenCalledWith("");
		expect(setApiBaseUrl).not.toHaveBeenCalledWith("http://127.0.0.1:3032");
	});

	it("is same-origin when the status carries no port at all", () => {
		applyDaemonStatus({ state: "ready" });
		expect(setApiBaseUrl).toHaveBeenCalledWith("");
	});

	it("still untrusts the base URL when the daemon is not ready", () => {
		for (const state of ["starting", "stopped", "error"] as const) {
			setApiBaseUrl.mockClear();
			applyDaemonStatus({ state });
			expect(setApiBaseUrl).toHaveBeenCalledWith(null);
		}
	});
});
