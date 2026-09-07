import { Loader2 } from "lucide-react";
import { useTranslation } from "react-i18next";

import { useAgentsQuery } from "../../hooks/useAgentsQuery";
import type { IntakeAnalysisRequest } from "../../hooks/useIntakeAnalysisRequest";
import { AgentAvatar } from "../AgentAvatar";
import { Button } from "../ui/button";

/**
 * The catalog's display label for a harness, falling back to the stored id.
 *
 * The id is what the daemon records ("codex"); the label is what every other
 * agent control in the app shows ("Codex"). Reading it here keeps this surface
 * from being the one place that spells an agent differently.
 */
function useHarnessLabel(harness: string | undefined): string | undefined {
	const catalog = useAgentsQuery().data;
	if (!harness) return undefined;
	return catalog?.supported?.find((agent) => agent.id === harness)?.label ?? harness;
}

/**
 * What an owner sees while an agent reads the project to propose a Contract.
 *
 * Two things make this different from the indefinite "Understanding the
 * Outcome" message it replaces. It names who is working, because an anonymous
 * spinner gives a person nothing to judge — a spawned agent can take minutes,
 * and knowing which harness is reading is what makes waiting a decision rather
 * than a hope. And it always offers a way out: the deterministic proposal is
 * available at every moment, so nobody is ever stuck behind an agent that
 * stopped answering.
 */
export function IntakeAnalysisWaiting({
	request,
	pending,
	onUseOffline,
	onRelease,
}: {
	/**
	 * Absent until the ask loads, and absent entirely for the brief moment an
	 * offline analysis occupies this state. The surface still renders: the
	 * daemon says analysis is happening, and waiting silently on a blank page
	 * would be worse than waiting on an unnamed one.
	 */
	request?: IntakeAnalysisRequest;
	pending: boolean;
	onUseOffline: () => void;
	onRelease: () => void;
}) {
	const { t } = useTranslation();
	// The avatar is keyed by the stored id; the text shows the catalog label.
	const harnessId = request?.harness?.trim();
	const harness = useHarnessLabel(harnessId);
	return (
		<div
			className="mx-auto flex h-full w-full max-w-xl flex-col items-center justify-center gap-4 px-4 text-center sm:px-8"
			data-testid="intake-analysis-waiting"
		>
			<div className="flex items-center gap-2.5">
				{harnessId ? <AgentAvatar provider={harnessId} className="size-icon-lg" decorative /> : null}
				<Loader2 aria-hidden="true" className="size-4 animate-spin text-muted-foreground" />
			</div>
			<div className="flex flex-col gap-1.5">
				<h2 className="text-lg font-medium text-foreground">
					{harness
						? t("outcome.intake.waiting.titleNamed", { harness })
						: t("outcome.intake.waiting.title")}
				</h2>
				<p className="text-sm leading-body text-muted-foreground">{t("outcome.intake.waiting.body")}</p>
			</div>

			{/* Always present, never behind a disclosure: the whole point is that
			    waiting is optional. */}
			<div className="flex flex-wrap items-center justify-center gap-2">
				<Button disabled={pending} onClick={onUseOffline} size="sm" variant="outline" type="button">
					{t("outcome.intake.waiting.useOffline")}
				</Button>
				<Button disabled={pending} onClick={onRelease} size="sm" variant="ghost" type="button">
					{t("outcome.intake.waiting.release")}
				</Button>
			</div>
			<p className="text-2xs text-passive">{t("outcome.intake.waiting.offlineHint")}</p>
		</div>
	);
}

/**
 * A draft the daemon refused, kept beside the reason it was refused.
 *
 * The draft is retained rather than discarded so the refusal is inspectable:
 * an owner deciding whether to ask again deserves to see what the agent
 * actually proposed and which rule it broke, instead of only being told that
 * something went wrong.
 */
export function IntakeAnalysisRefused({
	request,
	failureCode,
	pending,
	onUseOffline,
	onRetry,
}: {
	/** Absent when the analysis failed without an agent ever answering. */
	request?: IntakeAnalysisRequest;
	failureCode?: string;
	pending: boolean;
	onUseOffline: () => void;
	onRetry: () => void;
}) {
	const { t } = useTranslation();
	const harness = useHarnessLabel(request?.harness?.trim());
	const reason = request?.refusalReason || failureCode;
	return (
		<div className="mx-auto flex w-full max-w-2xl flex-col gap-3 px-4 py-8 sm:px-8" data-testid="intake-analysis-refused">
			<div className="flex flex-col gap-1">
				<h2 className="text-lg font-medium text-foreground">
					{harness
						? t("outcome.intake.refused.titleNamed", { harness })
						: t("outcome.intake.refused.title")}
				</h2>
				<p className="text-sm leading-body text-muted-foreground">{t("outcome.intake.refused.body")}</p>
			</div>
			{reason ? (
				<p className="rounded-group hairline border-border bg-card px-4.5 py-3.5 text-sm leading-body text-warning" role="alert">
					{reason}
				</p>
			) : null}
			{request?.rawProposal ? (
				<details className="rounded-group hairline border-border bg-card px-4.5 py-3.5">
					<summary className="cursor-pointer text-xs font-medium text-muted-foreground">
						{t("outcome.intake.refused.showDraft")}
					</summary>
					<pre className="mt-2 max-h-64 overflow-auto whitespace-pre-wrap break-words text-2xs leading-body text-passive">
						{request.rawProposal}
					</pre>
				</details>
			) : null}
			<div className="flex flex-wrap gap-2">
				<Button disabled={pending} onClick={onUseOffline} size="sm" type="button">
					{t("outcome.intake.waiting.useOffline")}
				</Button>
				<Button disabled={pending} onClick={onRetry} size="sm" variant="outline" type="button">
					{t("outcome.intake.refused.retry")}
				</Button>
			</div>
		</div>
	);
}

/**
 * One line above the Contract saying where it came from.
 *
 * A person about to hand-write four criteria deserves to know whether anything
 * analyzed the first draft. Without this, an offline proposal and an
 * agent-authored one look identical on screen while being worth very different
 * amounts of trust.
 */
export function ProposalProvenanceNote({ kind, harness }: { kind: "agent" | "offline"; harness?: string }) {
	const { t } = useTranslation();
	const label = useHarnessLabel(harness);
	if (kind === "agent") {
		return (
			<p className="flex items-center gap-1.5 text-2xs text-passive" data-testid="proposal-provenance">
				{harness ? <AgentAvatar provider={harness} className="size-icon-sm" decorative /> : null}
				{label
					? t("outcome.intake.provenance.agentNamed", { harness: label })
					: t("outcome.intake.provenance.agent")}
			</p>
		);
	}
	return (
		<p className="text-2xs text-warning" data-testid="proposal-provenance">
			{t("outcome.intake.provenance.offline")}
		</p>
	);
}
