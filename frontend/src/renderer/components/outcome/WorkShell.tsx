import { SessionsViewSwitch } from "@pin4sf/kennel-product-ui";
import { useNavigate } from "@tanstack/react-router";
import { Flag, Search, Terminal, Waypoints } from "lucide-react";
import { useEffect, type ReactNode } from "react";
import { useTranslation } from "react-i18next";

import { useOutcomeAttempts } from "../../hooks/useOutcome";
import { usesBoardActionsInPanel } from "../../lib/platform";
import { useUiStore } from "../../stores/ui-store";
import { NotificationCenter } from "../NotificationCenter";
import { TopbarButton } from "../TopbarButton";
import { Tooltip, TooltipContent, TooltipTrigger } from "../ui/tooltip";
import { OutcomeAttemptTerminalPanel } from "./OutcomeAttemptTerminalPanel";

// Board actions (and, here, the bell) render in-panel only where the framed
// ShellTopbar is hidden (macOS) — Win/Linux already get NotificationCenter
// from ShellTopbar above this surface. Evaluated once at module scope like
// the rest of the platform gates this file's siblings use.
const boardOwnsNotificationCenter = usesBoardActionsInPanel();

type WorkShellProps = {
	/** Absent on the Enter surface, before a project is chosen. */
	projectId?: string;
	/** Absent until an Outcome exists. */
	outcomeId?: string;
	children: ReactNode;
};

/**
 * The persistent chrome for the Work destination (Figma: Board-empty and
 * Enter/describe-outcome screens share this pixel-for-pixel). It wraps every
 * stage branch in `routes/_shell.work.tsx` exactly once, so no stage can ever
 * render with only the stage body and no way to navigate — the bug this
 * component replaces (`OutcomeLifecycleShell`'s inert pill row was
 * previously the only visible chrome above a stage surface).
 *
 * Real navigation is the persistent sidebar (project/outcome tree, untouched
 * here) plus this top-bar cluster: List/Board (governs Act & Observe, present
 * but inert elsewhere — Figma shows it inert on the Enter screen too), the
 * terminal-toggle for the current attempt, and the Outcomes destination.
 */
export function WorkShell({ projectId, outcomeId, children }: WorkShellProps) {
	const { t } = useTranslation();
	const navigate = useNavigate();

	const outcomeRunViewMode = useUiStore((state) => state.outcomeRunViewMode);
	const setOutcomeRunViewMode = useUiStore((state) => state.setOutcomeRunViewMode);
	const isAttemptPanelOpen = useUiStore((state) => state.isOutcomeAttemptPanelOpen);
	const toggleAttemptPanel = useUiStore((state) => state.toggleOutcomeAttemptPanel);
	const closeAttemptPanel = useUiStore((state) => state.closeOutcomeAttemptPanel);
	const setCommandPaletteOpen = useUiStore((state) => state.setCommandPaletteOpen);

	// Attempts only exist once Act & Observe has started, so this is undefined
	// (and the terminal toggle stays disabled) through Enter/Understand/Decide
	// & Authorize. Once an attempt exists, the panel this button opens stays
	// reachable from every later stage too — not just Act & Observe.
	const attemptsQuery = useOutcomeAttempts(outcomeId);
	const attempts = attemptsQuery.attempts ?? [];
	const currentAttempt = attempts.length > 0 ? attempts[attempts.length - 1] : undefined;

	// A stale attempt from a previously viewed Outcome must never linger
	// behind the toggle after the person switches to a different one.
	useEffect(() => {
		closeAttemptPanel();
	}, [outcomeId, closeAttemptPanel]);

	const openOutcomesOverview = () => {
		void navigate({ to: "/work", search: { view: "outcomes" } });
	};

	return (
		<div className="flex h-full min-h-0 flex-col gap-2.5" data-testid="work-shell">
			<div
				className="flex shrink-0 items-center gap-2 border-b border-border-strong px-1 pb-2.5"
				data-testid="work-shell-topbar"
			>
				<Tooltip>
					<TooltipTrigger asChild>
						<TopbarButton
							aria-label={t("shell.search")}
							onClick={() => setCommandPaletteOpen(true)}
							variant="icon"
						>
							<Search aria-hidden="true" className="size-icon-md" />
						</TopbarButton>
					</TooltipTrigger>
					<TooltipContent side="bottom">{t("shell.search")}</TooltipContent>
				</Tooltip>

				<div className="min-w-0 flex-1" />

				<SessionsViewSwitch
					labels={{
						ariaLabel: t("outcome.run.viewSwitchAria"),
						board: t("outcome.run.viewBoard"),
						list: t("outcome.run.viewList"),
					}}
					onChange={setOutcomeRunViewMode}
					value={outcomeRunViewMode}
				/>

				<div className="min-w-0 flex-1" />

				<div className="flex shrink-0 items-center gap-1.5">
					<Tooltip>
						<TooltipTrigger asChild>
							<span className="inline-flex">
								<TopbarButton
									aria-label={t("work.shell.terminalToggle")}
									aria-pressed={isAttemptPanelOpen}
									data-testid="work-shell-terminal-toggle"
									disabled={!currentAttempt}
									onClick={toggleAttemptPanel}
									variant="icon"
								>
									<Terminal aria-hidden="true" className="size-icon-md" />
								</TopbarButton>
							</span>
						</TooltipTrigger>
						<TooltipContent side="bottom">
							{currentAttempt ? t("work.shell.terminalToggle") : t("work.shell.terminalToggleUnavailable")}
						</TooltipContent>
					</Tooltip>

					<Tooltip>
						<TooltipTrigger asChild>
							<span className="inline-flex">
								<TopbarButton
									aria-label={t("work.shell.relationshipGraph")}
									data-testid="work-shell-graph"
									disabled
									variant="icon"
								>
									<Waypoints aria-hidden="true" className="size-icon-md" />
								</TopbarButton>
							</span>
						</TooltipTrigger>
						<TooltipContent side="bottom">{t("work.shell.relationshipGraphComingSoon")}</TooltipContent>
					</Tooltip>

					<TopbarButton
						aria-label={t("work.shell.outcomesButton")}
						data-testid="work-shell-outcomes"
						onClick={openOutcomesOverview}
						variant="feature"
					>
						<Flag aria-hidden="true" className="size-icon-md" />
						<span>{t("work.shell.outcomesButton")}</span>
					</TopbarButton>

					{boardOwnsNotificationCenter ? <NotificationCenter /> : null}
				</div>
			</div>

			<div className="flex min-h-0 flex-1 gap-2.5 overflow-hidden">
				<div className="min-w-0 min-h-0 flex-1" data-project-id={projectId} data-outcome-id={outcomeId}>
					{children}
				</div>
				{isAttemptPanelOpen && currentAttempt ? (
					<OutcomeAttemptTerminalPanel attempt={currentAttempt} onClose={closeAttemptPanel} />
				) : null}
			</div>
		</div>
	);
}
