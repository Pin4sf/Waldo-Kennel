import { mkdtemp, readFile } from "node:fs/promises";
import os from "node:os";
import path from "node:path";
import { afterEach, expect, test } from "vitest";
import {
	buildTelemetryBootstrap,
	defaultDataDir,
	loadOrCreateTelemetryInstallId,
	parseDisabledEvents,
	rendererTelemetryEnabled,
} from "./telemetry";

const tempDirs: string[] = [];

afterEach(async () => {
	await Promise.all(
		tempDirs
			.splice(0)
			.map((dir) => import("node:fs/promises").then(({ rm }) => rm(dir, { recursive: true, force: true }))),
	);
});

test("defaultDataDir prefers KENNEL_DATA_DIR", () => {
	expect(defaultDataDir("linux", { KENNEL_DATA_DIR: "/tmp/custom" }, "/home/test")).toBe("/tmp/custom");
});

test("loadOrCreateTelemetryInstallId persists a stable install id", async () => {
	const dir = await mkdtemp(path.join(os.tmpdir(), "kennel-telemetry-"));
	tempDirs.push(dir);

	const first = await loadOrCreateTelemetryInstallId(dir);
	const second = await loadOrCreateTelemetryInstallId(dir);
	const stored = (await readFile(path.join(dir, "telemetry_install_id"), "utf8")).trim();

	expect(first).toMatch(/^ins_/);
	expect(second).toBe(first);
	expect(stored).toBe(first);
});

test("buildTelemetryBootstrap returns null when no home dir is available", async () => {
	await expect(buildTelemetryBootstrap({}, "1.2.3", "linux", "")).resolves.toBeNull();
});

test("renderer telemetry is off in every build unless explicitly opted in", () => {
	expect(rendererTelemetryEnabled({}, false)).toBe(false);
	expect(rendererTelemetryEnabled({}, true)).toBe(false);
	expect(rendererTelemetryEnabled({ KENNEL_TELEMETRY_RENDERER: " ON " }, false)).toBe(true);
	expect(rendererTelemetryEnabled({ KENNEL_TELEMETRY_RENDERER: "off" }, true)).toBe(false);
});

// A null bootstrap is the switch itself: initTelemetry bails on it, so an
// unpackaged build never constructs a PostHog client at all.
test("buildTelemetryBootstrap withholds the bootstrap on an unpackaged build", async () => {
	const dir = await mkdtemp(path.join(os.tmpdir(), "kennel-telemetry-"));
	tempDirs.push(dir);

	await expect(buildTelemetryBootstrap({}, "0.11.2", "linux", dir, false)).resolves.toBeNull();
	const optedIn = await buildTelemetryBootstrap({ KENNEL_TELEMETRY_RENDERER: "on" }, "0.11.2", "linux", dir, false);
	expect(optedIn?.appVersion).toBe("0.11.2");
});

test("buildTelemetryBootstrap carries the deny list across the process boundary", async () => {
	const dir = await mkdtemp(path.join(os.tmpdir(), "kennel-telemetry-"));
	tempDirs.push(dir);

	const bootstrap = await buildTelemetryBootstrap(
		{
			KENNEL_TELEMETRY_RENDERER: "on",
			KENNEL_TELEMETRY_DISABLED_EVENTS: "kennel.v2.app.active, kennel.renderer.* ,, ",
		},
		"0.11.2",
		"linux",
		dir,
		true,
	);
	expect(bootstrap?.disabledEvents).toEqual(["kennel.v2.app.active", "kennel.renderer.*"]);
});

test("parseDisabledEvents treats absent or blank policy as no policy", () => {
	expect(parseDisabledEvents(undefined)).toEqual([]);
	expect(parseDisabledEvents(" , , ")).toEqual([]);
});
