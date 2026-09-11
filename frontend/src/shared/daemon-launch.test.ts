import { describe, expect, it } from "vitest";
import { bundledDaemonIdentityError, resolveDaemonLaunch, resolveExpectedDaemonBuildIdentity } from "./daemon-launch";

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
			args: ["run", "./cmd/kennel", "daemon"],
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
				"C:\\Program Files\\Kennel\\resources",
				"C:\\Program Files\\Kennel\\resources\\app.asar",
				"C:\\Users\\alice",
				"win32",
			),
		).toEqual({
			command: "C:\\Program Files\\Kennel\\resources/daemon/kennel-daemon.exe",
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
			buildIdentity: "build-current",
		};
		expect(bundledDaemonIdentityError(probe, launchCommand, appImage, samePath, "build-current")).toBeNull();
	});

	it("rejects a daemon from a different AppImage install", () => {
		const other = "/home/user/Apps/kennel-nightly.AppImage";
		const probe = {
			executablePath: "/tmp/.mount_agent-1Qs4N6/resources/daemon/kennel-daemon",
			appImagePath: other,
			buildIdentity: "build-current",
		};
		expect(bundledDaemonIdentityError(probe, launchCommand, appImage, samePath, "build-current")).toBe(
			`Another Kennel daemon is already running from ${other}; expected ${appImage}. Stop the other daemon before using this app.`,
		);
	});

	it("fails closed under AppImage when the daemon does not report its install identity", () => {
		const probe = {
			executablePath: "/tmp/.mount_agent-1Qs4N6/resources/daemon/kennel-daemon",
			buildIdentity: "build-current",
		};
		expect(bundledDaemonIdentityError(probe, launchCommand, appImage, samePath, "build-current")).toBe(
			"An older Kennel daemon is already running, but it does not report its install identity. Stop it and restart this app.",
		);
	});

	it("compares executable paths outside AppImage", () => {
		const command = "/opt/Kennel/resources/daemon/kennel-daemon";
		expect(
			bundledDaemonIdentityError({ executablePath: command, buildIdentity: "build-current" }, command, undefined, samePath, "build-current"),
		).toBeNull();
		expect(
			bundledDaemonIdentityError(
				{ executablePath: "/other/daemon", buildIdentity: "build-current" },
				command,
				undefined,
				samePath,
				"build-current",
			),
		).toBe(
			`Another Kennel daemon is already running from /other/daemon; expected ${command}. Stop the other daemon before using this app.`,
		);
	});

	it("fails closed outside AppImage when the daemon does not report its binary path", () => {
		expect(
			bundledDaemonIdentityError(
				{ buildIdentity: "build-current" },
				"/opt/Kennel/resources/daemon/kennel-daemon",
				undefined,
				samePath,
				"build-current",
			),
		).toBe(
			"An older Kennel daemon is already running, but it does not report its binary path. Stop it and restart this app.",
		);
	});

	it("rejects a daemon whose process-reported build identity differs", () => {
		expect(
			bundledDaemonIdentityError(
				{ executablePath: "/opt/Kennel/resources/daemon/kennel-daemon", buildIdentity: "build-old" },
				"/opt/Kennel/resources/daemon/kennel-daemon",
				undefined,
				samePath,
				"build-current",
			),
		).toBe(
			"Another Kennel daemon is already running with build identity build-old; expected build-current. Stop the other daemon before using this app.",
		);
	});

	it("fails closed when a packaged daemon omits its build identity", () => {
		expect(
			bundledDaemonIdentityError(
				{ executablePath: "/opt/Kennel/resources/daemon/kennel-daemon" },
				"/opt/Kennel/resources/daemon/kennel-daemon",
				undefined,
				samePath,
				"build-current",
			),
		).toBe(
			"An older Kennel daemon is already running, but it does not report its build identity. Rebuild this app and restart it.",
		);
	});

	it("fails closed when the package has no expected build metadata", () => {
		expect(
			bundledDaemonIdentityError(
				{ executablePath: "/opt/Kennel/resources/daemon/kennel-daemon", buildIdentity: "build-current" },
				"/opt/Kennel/resources/daemon/kennel-daemon",
				undefined,
				samePath,
			),
		).toBe(
			"This Kennel app does not include daemon build identity metadata. Rebuild the app before starting it.",
		);
	});
});

describe("resolveExpectedDaemonBuildIdentity", () => {
	const packaged = {
		command: "/Applications/Kennel.app/Contents/Resources/daemon/kennel-daemon",
		args: ["daemon"],
		cwd: "/Users/alice/.kennel",
		shell: false,
		source: "bundled" as const,
	};

	it("reads the manifest beside the exact packaged daemon path", () => {
		const reads: string[] = [];
		const identity = resolveExpectedDaemonBuildIdentity(packaged, "/app", (manifestPath) => {
			reads.push(manifestPath);
			return manifestPath === "/Applications/Kennel.app/Contents/Resources/daemon/build-identity.json"
				? JSON.stringify({ identity: "build-current" })
				: null;
		});
		expect(identity).toBe("build-current");
		expect(reads).toEqual(["/Applications/Kennel.app/Contents/Resources/daemon/build-identity.json"]);
	});

	it("returns undefined for missing or malformed package metadata", () => {
		expect(resolveExpectedDaemonBuildIdentity(packaged, "/app", () => null)).toBeUndefined();
		expect(resolveExpectedDaemonBuildIdentity(packaged, "/app", () => "not-json")).toBeUndefined();
	});

	it("does not require a package manifest for the non-Windows go-run dev path", () => {
		const dev = resolveDaemonLaunch({}, false, "/resources", "/repo/frontend", "/home/alice", "darwin");
		if (!dev) throw new Error("expected dev launch");
		const readManifest = () => {
			throw new Error("go-run dev must not read a package manifest");
		};
		expect(resolveExpectedDaemonBuildIdentity(dev, "/repo/frontend", readManifest)).toBeUndefined();
	});

	it("uses the dev daemon manifest when Windows dev launches a built binary", () => {
		const dev = resolveDaemonLaunch({}, false, "/resources", "C:\\repo\\frontend", "C:\\Users\\alice", "win32");
		if (!dev) throw new Error("expected dev launch");
		expect(resolveExpectedDaemonBuildIdentity(dev, "C:\\repo\\frontend", (manifestPath) =>
			manifestPath === "C:\\repo\\frontend/daemon/build-identity.json" ? JSON.stringify({ identity: "dev-current" }) : null,
		)).toBe("dev-current");
	});
});
