import { Badge } from "../ui/badge";
import type { PlanningCandidate } from "../../hooks/usePlanning";
import { useTranslation } from "react-i18next";

export function PlanningAgentPicker({
	candidates,
	selectedId,
	onChange,
	onRetry,
	loading = false,
	failure,
	disabled = false,
}: {
	candidates: PlanningCandidate[];
	selectedId: string;
	onChange: (candidateId: string) => void;
	onRetry?: () => void;
	loading?: boolean;
	failure?: string;
	disabled?: boolean;
}) {
	const { t } = useTranslation();
	return (
		<fieldset className="flex flex-col gap-2" disabled={disabled} data-testid="planning-agent-picker">
			<legend className="text-sm font-medium">{t("planning.providerLegend")}</legend>
			<p className="text-xs text-muted-foreground">{t("planning.providerHelp")}</p>
			{loading ? (
				<p className="rounded-md border border-border px-3 py-2 text-xs text-muted-foreground">{t("planning.checkingCandidates")}</p>
			) : failure ? (
				<div className="flex flex-wrap items-center gap-2 rounded-md border border-warning/40 px-3 py-2 text-xs text-muted-foreground"><span>{failure}</span>{onRetry && <button className="underline" onClick={onRetry} type="button">{t("mission.refresh")}</button>}</div>
			) : candidates.length === 0 ? (
				<p className="rounded-md border border-border px-3 py-2 text-xs text-muted-foreground" data-testid="planning-no-candidates">
					{t("planning.noCandidates")}
				</p>
			) : (
				<div className="flex flex-col gap-2">
					{candidates.map((candidate) => {
						const binding = candidate.binding;
						const identity = binding.modelSelection === "explicit" ? binding.model ?? t("planning.explicitModel") : t("planning.providerDefault");
						return (
							<label
								className={`flex cursor-pointer items-start gap-2 rounded-md border px-3 py-2 text-sm ${selectedId === candidate.id ? "border-ring bg-accent/30" : "border-border"} ${!candidate.ready ? "cursor-not-allowed opacity-60" : ""}`}
								key={candidate.id}
							>
								<input
									checked={selectedId === candidate.id}
									disabled={!candidate.ready}
									name="planning-candidate"
									onChange={() => onChange(candidate.id)}
									type="radio"
								/>
								<span className="min-w-0 flex-1">
									<span className="flex flex-wrap items-center gap-2 font-medium">
										{binding.provider} · {identity} · {binding.mode === "native_harness" ? t("planning.subscriptionAgent") : t("planning.apiReasoner")}
										<Badge variant={candidate.ready ? "success" : "outline"}>{candidate.ready ? t("planning.available") : t("planning.unavailable")}</Badge>
									</span>
									{!candidate.ready && (
										<span className="mt-1 block text-xs text-muted-foreground">
											{candidate.unavailableCode ?? t("planning.unavailableCode")}: {candidate.unavailableDetail ?? t("planning.unavailableDetail")}
										</span>
									)}
								</span>
							</label>
						);
					})}
				</div>
			)}
		</fieldset>
	);
}
