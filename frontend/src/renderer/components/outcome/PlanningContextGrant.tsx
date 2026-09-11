import type { PlanningContextMode } from "../../hooks/usePlanning";
import { useTranslation } from "react-i18next";

export function PlanningContextGrant({
	value,
	onChange,
	disabled = false,
	locked = false,
	suppliedPacketAvailable = false,
	nativeTools = false,
}: {
	value: PlanningContextMode | null;
	onChange?: (mode: PlanningContextMode) => void;
	disabled?: boolean;
	locked?: boolean;
	suppliedPacketAvailable?: boolean;
	nativeTools?: boolean;
}) {
	const { t } = useTranslation();
	return (
		<fieldset className="flex flex-col gap-2" disabled={disabled || locked} data-testid="planning-context-grant">
			<legend className="text-sm font-medium">{t("planning.contextLegend")}</legend>
			<p className="text-xs text-muted-foreground">{t("planning.contextHelp")}</p>
			<label className={`flex items-start gap-2 rounded-md border px-3 py-2 text-sm ${value === "repository_read" ? "border-ring bg-accent/30" : "border-border"}`}>
				<input checked={value === "repository_read"} name="planning-context" onChange={() => onChange?.("repository_read")} type="radio" />
				<span><span className="font-medium">{t("planning.repositoryLabel")}</span><span className="mt-0.5 block text-xs text-muted-foreground">{t(nativeTools ? "planning.repositoryNativeHelp" : "planning.repositoryHelp")}</span></span>
			</label>
			{suppliedPacketAvailable && <label className={`flex items-start gap-2 rounded-md border px-3 py-2 text-sm ${value === "supplied_packet" ? "border-ring bg-accent/30" : "border-border"}`}>
				<input checked={value === "supplied_packet"} name="planning-context" onChange={() => onChange?.("supplied_packet")} type="radio" />
				<span><span className="font-medium">{t("planning.packetLabel")}</span><span className="mt-0.5 block text-xs text-muted-foreground">{t("planning.packetHelp")}</span></span>
			</label>}
			{locked && <p className="text-xs text-muted-foreground">{t("planning.scopeFrozen")}</p>}
		</fieldset>
	);
}
