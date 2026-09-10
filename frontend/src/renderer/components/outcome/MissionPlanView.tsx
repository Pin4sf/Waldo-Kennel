import { useState } from "react";
import { useTranslation } from "react-i18next";
import type { components } from "../../../api/schema";
import { MissionWorkUnitGraph } from "./MissionWorkUnitGraph";
import { layerByDependency } from "../../lib/dependency-layers";
import { Button } from "../ui/button";

type Unit = components["schemas"]["PlanWorkUnitResponse"];
type Schedule = components["schemas"]["ScheduleResponse"];

/** Graph and Table inspect the same durable units; selecting never admits work. */
export function MissionPlanView({
	workUnits,
	schedule,
	criterionText,
}: {
	workUnits: Unit[];
	schedule?: Schedule;
	criterionText?: (id: string) => string | undefined;
}) {
	const { t } = useTranslation();
	const [view, setView] = useState<"graph" | "table">("graph");
	const [selectedId, setSelectedId] = useState<string>();
	const units = layerByDependency(
		workUnits.map((unit) => ({ ...unit, upstream: unit.dependsOn ?? [] })).sort((a, b) => a.id.localeCompare(b.id)),
	).flat();
	const selected = units.find((unit) => unit.id === selectedId);
	const entry = schedule?.workUnits.find((item) => item.workUnit.id === selectedId);
	const title = (id: string) => units.find((unit) => unit.id === id)?.title ?? id;
	return (
		<section className="flex min-w-0 flex-col gap-3" data-testid="mission-plan-view">
			<div className="flex gap-1" role="group" aria-label={t("mission.plan")}>
				{(["graph", "table"] as const).map((mode) => (
					<Button key={mode} size="sm" variant="ghost" aria-pressed={view === mode} onClick={() => setView(mode)}>
						{t(`mission.${mode}`)}
					</Button>
				))}
			</div>
			{/* Keep both views mounted to preserve graph zoom, focus and scroll on refresh. */}
			<div hidden={view !== "graph"} className="overflow-auto p-1">
				<MissionWorkUnitGraph
					workUnits={units}
					schedule={schedule}
					criterionText={criterionText}
					selectedWorkUnitId={selected?.id}
					onSelectWorkUnit={setSelectedId}
				/>
			</div>
			<div hidden={view !== "table"} className="overflow-auto">
				<table className="w-full border-collapse text-left text-xs">
					<caption className="pb-2 text-left text-muted-foreground">
						{t(schedule ? "outcome.missionGraph.serialNote" : "outcome.missionGraph.proposedNote")}
					</caption>
					<thead>
						<tr>
							{(["workUnit", "state", "dependencies", "output", "agent", "evidence"] as const).map((label) => (
								<th key={label} className="border-b border-border px-2 py-2 font-medium">
									{t(`mission.${label}`)}
								</th>
							))}
						</tr>
					</thead>
					<tbody>
						{units.map((unit) => {
							const item = schedule?.workUnits.find((item) => item.workUnit.id === unit.id);
							return (
								<tr
									key={unit.id}
									className="align-top aria-selected:bg-interactive-hover"
									aria-selected={selected?.id === unit.id}
								>
									<td className="border-b border-border p-2">
										<button
											type="button"
											className="text-left font-medium hover:underline focus-visible:ring-2 focus-visible:ring-ring"
											onClick={() => setSelectedId(unit.id)}
										>
											{unit.title}
										</button>
									</td>
									<td className="border-b border-border p-2">
										{item ? t(`outcome.missionGraph.state.${item.state}`) : t("outcome.missionGraph.state.proposed")}
									</td>
									<td className="border-b border-border p-2">
										{(unit.dependsOn ?? []).map(title).join(", ") || t("outcome.decide.dependsOnNone")}
										{item?.blockedReason && (
											<p className="text-warning">
												{t(`outcome.missionGraph.blocked.${item.blockedReason}`, {
													units: (item.blockingDependencies ?? []).map(title).join(", "),
												})}
											</p>
										)}
									</td>
									<td className="border-b border-border p-2">{unit.outputSummary || t("mission.unknown")}</td>
									<td className="border-b border-border p-2">
										{unit.provider || t("outcome.missionGraph.bindingUnset")} ·{" "}
										{unit.modelSelection === "provider_default"
											? t("outcome.missionGraph.providerDefault")
											: unit.model || t("mission.unknown")}
									</td>
									<td className="border-b border-border p-2">
										{(unit.evidenceChecks ?? []).join(" · ") || t("mission.none")}
									</td>
								</tr>
							);
						})}
					</tbody>
				</table>
			</div>
			{selected ? (
				<section className="border-t border-border pt-3" aria-label={selected.title} data-testid="mission-unit-detail">
					<h3 className="text-sm font-medium break-words">{selected.title}</h3>
					<dl className="mt-2 grid grid-cols-[auto_1fr] gap-x-4 gap-y-2 text-xs">
						<dt>{t("mission.output")}</dt>
						<dd className="whitespace-pre-wrap break-words">{selected.outputSummary || t("mission.unknown")}</dd>
						<dt>{t("mission.criteria")}</dt>
						<dd>
							{(selected.criterionIds ?? []).map((id) => criterionText?.(id) ?? id).join(" · ") || t("mission.none")}
						</dd>
						<dt>{t("mission.evidence")}</dt>
						<dd>
							{(selected.evidenceChecks ?? []).join(" · ")}
							<p>{selected.verificationRequirement}</p>
							{(selected.criterionIds ?? []).map((id) => (
								<p key={id}>
									{criterionText?.(id) ?? id}:{" "}
									{entry?.criterionReady?.[id] === undefined
										? t("mission.proofUnknown")
										: entry.criterionReady[id]
											? t("mission.proofReady")
											: t("mission.proofMissing")}
								</p>
							))}
						</dd>
						<dt>{t("mission.capabilities")}</dt>
						<dd>{(selected.requiredCapabilities ?? []).join(", ") || t("mission.none")}</dd>
						<dt>{t("mission.stops")}</dt>
						<dd>{(selected.stopConditions ?? []).join(" · ") || t("mission.none")}</dd>
						<dt>{t("mission.attempts")}</dt>
						<dd>
							{entry?.attempts.length ? (
								<ul>
									{entry.attempts.map((attempt) => (
										<li key={attempt.id}>
											{attempt.status} ·{" "}
											<time dateTime={attempt.updatedAt}>{new Date(attempt.updatedAt).toLocaleString()}</time>
											<details>
												<summary>{t("mission.attempts")}</summary>
												<code>{attempt.id}</code>
											</details>
										</li>
									))}
								</ul>
							) : (
								t("mission.noAttempts")
							)}
						</dd>
					</dl>
				</section>
			) : (
				<p className="text-xs text-muted-foreground">{t("mission.selectUnit")}</p>
			)}
		</section>
	);
}
