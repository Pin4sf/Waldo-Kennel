import { useTranslation } from "react-i18next";
import { useOutcomeRunCommand, useOutcomeRunState } from "../../hooks/useOutcomeRunState";
import { useEventsConnection } from "../../hooks/useEventsConnection";
import { Button } from "../ui/button";

export function OutcomeRunControls({
	outcomeId,
	admissionBlocked = false,
}: {
	outcomeId: string;
	admissionBlocked?: boolean;
}) {
	const { t } = useTranslation();
	const query = useOutcomeRunState(outcomeId);
	const mutation = useOutcomeRunCommand(outcomeId);
	const connection = useEventsConnection();
	const state = query.data;
	const unavailable = query.isError || !state || query.isFetching || connection !== "connected";
	const intent = state?.intent;
	return (
		<section className="rounded-group border border-border bg-card p-4" aria-label={t("mission.run.controls")}>
			<p role="status" className="text-sm">
				{query.error?.message ??
					(!state
						? t("mission.run.loading")
						: intent
							? !intent.acknowledgedAt && intent.desired === "paused"
								? t("mission.run.pausedPending")
								: !intent.acknowledgedAt && intent.desired === "cancelled"
									? t("mission.run.cancelledPending")
									: t(`mission.run.${intent.desired}`)
							: t("mission.run.idle"))}
			</p>
			{state?.blocker && <p className="mt-2 text-sm text-muted-foreground">{state.blocker.message}</p>}
			{unavailable && <p className="text-xs text-muted-foreground">{t("mission.run.unavailable")}</p>}
			<div className="mt-3 flex flex-wrap gap-2">
				{(["start", "pause", "resume", "cancel"] as const).map((action) => {
					const eligibility = state?.eligibleActions.find((item) => item.action === action);
					if (!eligibility?.available) return null;
					return (
						<Button
							key={action}
							data-testid={`outcome-run-${action}`}
							variant={action === "start" || action === "resume" ? "primary" : "outline"}
							disabled={
								unavailable || mutation.isPending || (admissionBlocked && (action === "start" || action === "resume"))
							}
							onClick={() => void mutation.command(action, state!).catch(() => {})}
						>
							{t(`mission.run.action.${action}`)}
						</Button>
					);
				})}
				{query.isError && (
					<Button variant="outline" onClick={() => void query.refetch()}>
						{t("mission.refresh")}
					</Button>
				)}
			</div>
			{mutation.error && (
				<p role="alert" className="mt-2 text-sm text-destructive">
					{mutation.error.message}
				</p>
			)}
		</section>
	);
}
