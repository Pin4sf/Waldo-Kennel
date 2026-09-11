export type DaemonLaunchSpec = {
	command: string;
	args: string[];
	cwd: string;
	shell: boolean;
	source: "configured" | "bundled" | "dev";
};

function joinPath(...segments: string[]): string {
	return segments.map((segment) => segment.replace(/[/\\]+$/, "")).join("/");
}

export function bundledDaemonBinaryName(platform: NodeJS.Platform): string {
	return platform === "win32" ? "kennel-daemon.exe" : "kennel-daemon";
}

export function resolveDaemonLaunch(
	env: Record<string, string | undefined>,
	isPackaged: boolean,
	resourcesPath: string,
	appPath: string,
	homeDir: string,
	platform: NodeJS.Platform,
): DaemonLaunchSpec | null {
	const configuredCommand = env.KENNEL_DAEMON_COMMAND?.trim();
	if (configuredCommand) {
		return {
			command: configuredCommand,
			args: [],
			cwd: appPath,
			shell: true,
			source: "configured",
		};
	}

	if (!isPackaged) {
		if (platform === "win32") {
			return {
				command: env.KENNEL_DEV_DAEMON_BINARY?.trim() || joinPath(appPath, "daemon", bundledDaemonBinaryName(platform)),
				args: ["daemon"],
				cwd: appPath,
				shell: false,
				source: "dev",
			};
		}
		return {
			command: "go",
			args: ["run", "./cmd/kennel", "daemon"],
			cwd: joinPath(appPath, "..", "backend"),
			shell: false,
			source: "dev",
		};
	}

	return {
		command: joinPath(resourcesPath, "daemon", bundledDaemonBinaryName(platform)),
		args: ["daemon"],
		cwd: joinPath(homeDir, ".kennel"),
		shell: false,
		source: "bundled",
	};
}

export type BundledDaemonProbe = {
	executablePath?: string;
	appImagePath?: string;
	buildIdentity?: string;
};

/**
 * Identity check for a bundled daemon. The process-reported build identity is
 * checked first; executable/install paths remain compatibility checks after a
 * build match (for AppImage relaunches). Legacy packages without expected build
 * metadata fail closed rather than claiming path equality proves process identity.
 *
 * Returns an error message, or null when the probed daemon belongs to this
 * install. A probe that cannot prove its identity (missing field) fails closed.
 */
export function bundledDaemonIdentityError(
	probe: BundledDaemonProbe,
	expectedCommand: string,
	appImagePath: string | undefined,
	samePath: (a: string, b: string) => boolean,
	expectedBuildIdentity?: string,
): string | null {
	if (expectedBuildIdentity === undefined) {
		return "This Kennel app does not include daemon build identity metadata. Rebuild the app before starting it.";
	}
	if (!probe.buildIdentity) {
		return "An older Kennel daemon is already running, but it does not report its build identity. Rebuild this app and restart it.";
	}
	if (probe.buildIdentity !== expectedBuildIdentity) {
		return `Another Kennel daemon is already running with build identity ${probe.buildIdentity}; expected ${expectedBuildIdentity}. Stop the other daemon before using this app.`;
	}
	if (appImagePath) {
		if (!probe.appImagePath) {
			return "An older Kennel daemon is already running, but it does not report its install identity. Stop it and restart this app.";
		}
		if (!samePath(probe.appImagePath, appImagePath)) {
			return `Another Kennel daemon is already running from ${probe.appImagePath}; expected ${appImagePath}. Stop the other daemon before using this app.`;
		}
		return null;
	}
	if (!probe.executablePath) {
		return "An older Kennel daemon is already running, but it does not report its binary path. Stop it and restart this app.";
	}
	if (!samePath(probe.executablePath, expectedCommand)) {
		return `Another Kennel daemon is already running from ${probe.executablePath}; expected ${expectedCommand}. Stop the other daemon before using this app.`;
	}
	return null;
}
