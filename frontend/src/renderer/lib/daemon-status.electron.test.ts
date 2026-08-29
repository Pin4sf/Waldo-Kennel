import { beforeEach, describe, expect, it, vi } from "vitest";

// Inside Electron nothing changes: the supervisor reports a port and requests
// go straight at it.
vi.mock("./preview-mode", () => ({
	runsOutsideElectron: false,
	usesLiveDaemonPreview: false,
	usesPreviewWorkspaceData: false,
	usesWaldoUiPreview: false,
}));

const { setApiBaseUrl, setApiDaemonStatus } = vi.hoisted(() => ({
	setApiBaseUrl: vi.fn(),
	setApiDaemonStatus: vi.fn(),
}));
vi.mock("./api-client", () => ({ setApiBaseUrl, setApiDaemonStatus }));
vi.mock("./bridge", () => ({ aoBridge: { daemon: { getStatus: vi.fn() } } }));

import { applyDaemonStatus } from "./daemon-status";

beforeEach(() => setApiBaseUrl.mockClear());

describe("applyDaemonStatus inside Electron", () => {
	it("points requests at the supervisor's reported port", () => {
		applyDaemonStatus({ state: "ready", port: 3032 });
		expect(setApiBaseUrl).toHaveBeenCalledWith("http://127.0.0.1:3032");
	});

	it("untrusts the base URL when ready arrives without a port", () => {
		applyDaemonStatus({ state: "ready" });
		expect(setApiBaseUrl).toHaveBeenCalledWith(null);
	});
});
