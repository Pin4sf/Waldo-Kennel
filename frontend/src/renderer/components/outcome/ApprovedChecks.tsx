import { useTranslation } from "react-i18next";
import type { components } from "../../../api/schema";

export function ApprovedChecks({
	unit,
	criterionText,
}: {
	unit: components["schemas"]["PlanWorkUnitResponse"];
	criterionText?: (id: string) => string | undefined;
}) {
	const { t } = useTranslation();
	return (
		<section className="mt-3 text-xs" aria-label={t("mission.approvedChecks")}>
			<h4 className="font-medium">{t("mission.approvedChecks")}</h4>
			{!unit.approvedChecks?.length ? (
				<p className="text-muted-foreground">{t("mission.noApprovedChecks")}</p>
			) : (
				<ul className="space-y-2">
					{unit.approvedChecks.map((check) => (
						<li key={check.id}>
							<p>{criterionText?.(check.criterionId) ?? t("mission.criterionUnavailable")}</p>
							<pre className="overflow-auto whitespace-pre-wrap break-all rounded border border-border p-2">
								{JSON.stringify(check.argv)}
							</pre>
							<p>{t("mission.checkTimeout", { seconds: check.timeoutSeconds })}</p>
						</li>
					))}
				</ul>
			)}
		</section>
	);
}
