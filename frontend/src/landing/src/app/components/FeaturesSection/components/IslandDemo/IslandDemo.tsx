"use client";

import { useEffect, useMemo, useRef, useState } from "react";
import "./island-demo.css";
import { useKennelIsland } from "./island/adapter";
import { KennelIsland } from "./island/KennelIsland";
import { usePointerDwell, useKennelSettings } from "./island/useIslandStage";
import { createDemoIslandAdapter, type DemoScenario } from "./fixtures/island";

/**
 * Ported from packages/kennel-island's own browser "state lab"
 * (src/App.tsx PrototypeLab): the real KennelIsland component, driven by the
 * same demo adapter, staged on a mock MacBook menu bar. Only the desktop-only
 * wiring (Electron IPC, haptics, real media transport) is left behind — none
 * of it runs in the lab either.
 */

const STAGE_W = 520;
const STAGE_H = 360;

// Cycles the same states kennel-island's own lab exposes as scenario buttons,
// so a visitor who never hovers still sees the island do something.
const SCENARIOS: DemoScenario[] = [
	"compact",
	"queue",
	"choice",
	"permission",
	"usage",
];
const SCENARIO_INTERVAL_MS = 4200;

export function IslandDemo() {
	const adapter = useMemo(() => createDemoIslandAdapter(), []);
	const { model, dispatch } = useKennelIsland(adapter);
	const [hovered, setHovered] = useState(false);
	const settings = useKennelSettings();
	const settled = usePointerDwell(hovered, settings.hover.peekDelayMs);

	const stepRef = useRef(0);
	useEffect(() => {
		// A hovering visitor is driving the island by hand; the auto-cycle would
		// only fight them for it.
		if (hovered) return;
		const id = window.setInterval(() => {
			stepRef.current = (stepRef.current + 1) % SCENARIOS.length;
			adapter.setScenario(SCENARIOS[stepRef.current]);
		}, SCENARIO_INTERVAL_MS);
		return () => window.clearInterval(id);
	}, [adapter, hovered]);

	return (
		<div className="relative flex h-[300px] w-full items-center justify-center sm:h-[380px] lg:h-[420px]">
			<div
				aria-hidden="true"
				className="pointer-events-none absolute size-[260px] rounded-full bg-[#fb8404] opacity-[0.13] blur-[70px]"
			/>

			<div
				className="kennel-island-demo relative shrink-0 origin-center scale-[0.5] sm:scale-[0.68] lg:scale-[0.78]"
				style={{ width: STAGE_W, height: STAGE_H }}
			>
				<section
					aria-label="Kennel Island preview"
					className="desktop-stage"
					style={{ height: STAGE_H, minHeight: STAGE_H }}
				>
					<div className="desktop-stage__grain" />
					<div
						className="desktop-stage__island"
						onMouseEnter={() => setHovered(true)}
						onMouseLeave={() => setHovered(false)}
					>
						<KennelIsland
							hovered={hovered}
							model={model}
							onAction={dispatch}
							settings={settings}
							settled={settled}
						/>
					</div>
				</section>
			</div>
		</div>
	);
}
