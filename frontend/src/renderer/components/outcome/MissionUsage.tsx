import { useTranslation } from "react-i18next";
import { useOutcomeAttempts } from "../../hooks/useOutcome";
import { useSessionUsageSummaries } from "../../hooks/useSessionUsageSummaries";

export function MissionUsage({ outcomeId, projectId }: { outcomeId: string; projectId: string }) {
	const { t } = useTranslation();
	const attempts = useOutcomeAttempts(outcomeId);
	const usage = useSessionUsageSummaries(projectId);
	const sessions = new Map(
		(attempts.attempts ?? []).flatMap((attempt) =>
			(attempt.sessions ?? []).map((session) => [session.sessionId, session] as const),
		),
	);
	return (
		<section className="mt-4 border-t border-border pt-3">
			<h3 className="text-sm font-medium">{t("mission.usage")}</h3>
			<p className="my-2 text-xs text-muted-foreground">{t("mission.usageScope")}</p>
			{sessions.size ? (
				<ul className="space-y-1 text-xs">
					{[...sessions.values()].map((session) => {
						const recorded = usage.data?.get(session.sessionId);
						return (
							<li key={session.sessionId}>
								{session.harness} ·{" "}
								{recorded
									? t("mission.tokens", { amount: recorded.totalTokens.toLocaleString() })
									: t("mission.unknown")}
								{recorded?.incomplete ? ` · ${t("mission.incomplete")}` : ""}
							</li>
						);
					})}
				</ul>
			) : (
				<p className="text-xs text-muted-foreground">{t("mission.unknown")}</p>
			)}
			{usage.error && (
				<p role="alert" className="text-xs text-destructive">
					{t("mission.usageUnavailable")}
				</p>
			)}
		</section>
	);
}
