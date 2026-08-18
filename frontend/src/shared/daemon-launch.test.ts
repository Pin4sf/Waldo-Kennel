import { describe, expect, it } from "vitest";
import { bundledDaemonIdentityError, resolveDaemonLaunch } from "./daemon-launch";

describe("resolveDaemonLaunch", () => {
	it("uses KENNEL_DAEMON_COMMAND when configured", () => {
		expect(
			resolveDaemonLaunch({ KENNEL_DAEMON_COMMAND: "/tmp/ao daemon" }, false, "/resources", "/app", "/home/user", "darwin"),
		).toEqual({
			command: "/tmp/ao daemon",
			args: [],
			cwd: "/app",
			shell: true,
			source: "configured",
		});
	});

	it("runs the backend daemon from source in non-Windows dev without an explicit command", () => {
		expect(resolveDaemonLaunch({}, false, "/resources", "/repo/frontend", "/home/user", "darwin")).toEqual({
			command: "go",
			args: ["run", "./cmd/ao", "daemon"],
			cwd: "/repo/frontend/../backend",
			shell: false,
			source: "dev",
		});
	});

	it("uses the prebuilt daemon exe in Windows dev", () => {
		expect(resolveDaemonLaunch({}, false, "/resources", "C:\\repo\\frontend", "C:\\Users\\alice", "win32")).toEqual({
			command: "C:\\repo\\frontend/daemon/kennel-daemon.exe",
			args: ["daemon"],
			cwd: "C:\\repo\\frontend",
			shell: false,
			source: "dev",
		});
	});

	it("uses the versioned daemon exe in Windows dev when build-daemon wrote one", () => {
		expect(
			resolveDaemonLaunch(
				{ KENNEL_DEV_DAEMON_BINARY: "C:\\repo\\frontend\\daemon\\dev-123\\kennel.exe" },
				false,
				"/resources",
				"C:\\repo\\frontend",
				"C:\\Users\\alice",
				"win32",
			),
		).toEqual({
			command: "C:\\repo\\frontend\\daemon\\dev-123\\kennel.exe",
			args: ["daemon"],
			cwd: "C:\\repo\\frontend",
			shell: false,
			source: "dev",
		});
	});

	it("uses the bundled daemon binary for packaged macOS/Linux builds", () => {
		expect(
			resolveDaemonLaunch(
				{},
				true,
				"/Applications/Kennel.app/Contents/Resources",
				"/app",
				"/Users/alice",
				"darwin",
			),
		).toEqual({
			command: "/Applications/Kennel.app/Contents/Resources/daemon/kennel-daemon",
			args: ["daemon"],
			cwd: "/Users/alice/.kennel",
			shell: false,
			source: "bundled",
		});
	});

	it("uses the bundled daemon exe for packaged Windows builds", () => {
		expect(
			resolveDaemonLaunch(
				{},
				true,
				"C:\\Program Files\\AO\\resources",
				"C:\\Program Files\\AO\\resources\\app.asar",
				"C:\\Users\\alice",
				"win32",
			),
		).toEqual({
			command: "C:\\Program Files\\AO\\resources/daemon/kennel-daemon.exe",
			args: ["daemon"],
			cwd: "C:\\Users\\alice/.kennel",
			shell: false,
			source: "bundled",
		});
	});
});

describe("bundledDaemonIdentityError", () => {
	const samePath = (a: string, b: string): boolean => a === b;
	const appImage = "/home/user/Apps/kennel.AppImage";
	// The bundled command under AppImage: a random FUSE mount, different per launch.
	const launchCommand = "/tmp/.mount_agent-mDQfUL/resources/daemon/kennel-daemon";

	it("accepts the same install across two AppImage mounts (relaunch-to-update)", () => {
		const probe = {
			executablePath: "/tmp/.mount_agent-1Qs4N6/resources/daemon/kennel-daemon",
			appImagePath: appImage,
		};
		expect(bundledDaemonIdentityError(probe, launchCommand, appImage, samePath)).toBeNull();
	});

	it("rejects a daemon from a different AppImage install", () => {
		const other = "/home/user/Apps/kennel-nightly.AppImage";
		const probe = { executablePath: "/tmp/.mount_agent-1Qs4N6/resources/daemon/kennel-daemon", appImagePath: other };
		expect(bundledDaemonIdentityError(probe, launchCommand, appImage, samePath)).toBe(
			`Another Kennel daemon is already running from ${other}; expected ${appImage}. Stop the other daemon before using this app.`,
		);
	});

	it("fails closed under AppImage when the daemon does not report its install identity", () => {
		const probe = { executablePath: "/tmp/.mount_agent-1Qs4N6/resources/daemon/kennel-daemon" };
		expect(bundledDaemonIdentityError(probe, launchCommand, appImage, samePath)).toBe(
			"An older Kennel daemon is already running, but it does not report its install identity. Stop it and restart this app.",
		);
	});

	it("compares executable paths outside AppImage", () => {
		const command = "/opt/Kennel/resources/daemon/kennel-daemon";
		expect(bundledDaemonIdentityError({ executablePath: command }, command, undefined, samePath)).toBeNull();
		expect(bundledDaemonIdentityError({ executablePath: "/other/ao" }, command, undefined, samePath)).toBe(
			`Another Kennel daemon is already running from /other/ao; expected ${command}. Stop the other daemon before using this app.`,
		);
	});

	it("fails closed outside AppImage when the daemon does not report its binary path", () => {
		expect(bundledDaemonIdentityError({}, "/opt/ao/resources/daemon/kennel-daemon", undefined, samePath)).toBe(
			"An older Kennel daemon is already running, but it does not report its binary path. Stop it and restart this app.",
		);
	});
});
