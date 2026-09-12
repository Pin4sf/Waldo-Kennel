import { useEffect, useMemo, useState, type ReactNode } from "react";
import * as Dialog from "@radix-ui/react-dialog";
import { useTranslation } from "react-i18next";
import { ArrowLeft, ArrowRight, Check, FolderPlus, Sparkles, Target, X } from "lucide-react";
import { agentLabel } from "../lib/agent-options";
import { cn } from "../lib/utils";
import { refreshAgentsIfStale, useAgentsQuery } from "../hooks/useAgentsQuery";
import { useSettings, useUpdateReasoning } from "../hooks/useSettings";
import { useUiStore } from "../stores/ui-store";
import { AgentAvatar } from "./AgentAvatar";
import { Button } from "./ui/button";

/**
 * First-run setup tour.
 *
 * Three steps, each one decision or next action, in the order a person needs
 * them: which providers are ready and how to enter the first Outcome. Every
 * setting remains reachable from Settings afterwards, so nothing
 * and every one of them is reachable again from Settings afterwards, so nothing
 * here is a one-shot choice a person can regret.
 */

const STEP_IDS = ["welcome", "agent", "outcome"] as const;
type StepId = (typeof STEP_IDS)[number];

// Spelled out rather than built from the step id: the typed `t` only accepts
// literal keys, and a template string would trade that check for nothing.
const STEP_TITLE_KEYS = {
	welcome: "onboarding.step.welcome.title",
	agent: "onboarding.step.agent.title",
	outcome: "onboarding.step.outcome.title",
} as const satisfies Record<StepId, string>;

export function OnboardingTour({ daemonReady }: { daemonReady: boolean }) {
	const { t } = useTranslation();
	const isOpen = useUiStore((state) => state.isOnboardingOpen);
	const hasCompleted = useUiStore((state) => state.hasCompletedOnboarding);
	const openOnboarding = useUiStore((state) => state.openOnboarding);
	const closeOnboarding = useUiStore((state) => state.closeOnboarding);
	const [stepIndex, setStepIndex] = useState(0);

	// A person meets the tour once, on the launch where they have never finished
	// it — and not before the daemon is up, because the agent step asks the daemon
	// what is installed and a setup dialog over a startup spinner teaches nothing.
	// Re-running it later is a deliberate act from Settings.
	useEffect(() => {
		if (!hasCompleted && daemonReady) openOnboarding();
	}, [daemonReady, hasCompleted, openOnboarding]);

	useEffect(() => {
		if (isOpen) setStepIndex(0);
	}, [isOpen]);

	const stepId = STEP_IDS[stepIndex];
	const isLast = stepIndex === STEP_IDS.length - 1;

	return (
		<Dialog.Root open={isOpen} onOpenChange={(next) => !next && closeOnboarding()}>
			<Dialog.Portal>
				<Dialog.Overlay className="dialog-overlay data-[state=open]:animate-overlay-in" />
				<Dialog.Content
					aria-describedby={undefined}
					className="fixed left-1/2 top-1/2 z-overlay flex w-[min(680px,calc(100vw-32px))] -translate-x-1/2 -translate-y-1/2 flex-col overflow-hidden rounded-panel hairline border-border bg-card shadow-[var(--shadow-import-modal)] outline-none data-[state=open]:animate-modal-in"
					data-testid="onboarding-tour"
				>
					<header className="flex shrink-0 items-center justify-between gap-3 border-b border-border px-4.5 py-3">
						<Dialog.Title className="flex min-w-0 items-baseline gap-1.5">
							<span className="truncate text-sm font-medium text-foreground">
								{t(STEP_TITLE_KEYS[stepId])}
							</span>
							<span className="shrink-0 text-2xs text-passive">
								{t("onboarding.stepCount", { current: stepIndex + 1, total: STEP_IDS.length })}
							</span>
						</Dialog.Title>
						<div className="flex shrink-0 items-center gap-2.5">
							<StepPager
								current={stepIndex}
								label={t("onboarding.progressAria", {
									current: stepIndex + 1,
									total: STEP_IDS.length,
								})}
								total={STEP_IDS.length}
							/>
							{/* Named apart from "Skip tour" in the footer: two controls that do
							    the same thing may share behaviour, never an accessible name. */}
							<Dialog.Close
								aria-label={t("onboarding.close")}
								className="grid size-control-chip place-items-center rounded-md text-passive transition-colors hover:bg-muted hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/60"
							>
								<X aria-hidden="true" className="size-icon-md" />
							</Dialog.Close>
						</div>
					</header>

					{/* One fixed height for every step, not a minimum: a tour that grows
					    with its content moves its own footer buttons while a person is
					    reaching for them. Long steps scroll inside this box instead. */}
					<div className="board-scrollbar h-onboarding-body shrink-0 overflow-y-auto px-4.5 py-5">
						{stepId === "welcome" ? <WelcomeStep /> : null}
						{stepId === "agent" ? <AgentStep /> : null}
						{stepId === "outcome" ? <OutcomeStep /> : null}
					</div>

					<footer className="flex shrink-0 items-center justify-between gap-3 border-t border-border px-4.5 py-3">
						<Button
							className="gap-1.5"
							disabled={stepIndex === 0}
							onClick={() => setStepIndex((current) => Math.max(0, current - 1))}
							size="sm"
							variant="ghost"
						>
							<ArrowLeft aria-hidden="true" className="size-icon-sm" />
							{t("onboarding.back")}
						</Button>
						<Button onClick={closeOnboarding} size="sm" variant="ghost">
							{t("onboarding.skip")}
						</Button>
						<Button
							className="gap-1.5"
							onClick={() =>
								isLast ? closeOnboarding() : setStepIndex((current) => current + 1)
							}
							size="sm"
							variant="primary"
						>
							{isLast ? <Check aria-hidden="true" className="size-icon-sm" /> : null}
							{stepIndex === 0
								? t("onboarding.start")
								: isLast
									? t("onboarding.finish")
									: t("onboarding.next")}
							{isLast ? null : <ArrowRight aria-hidden="true" className="size-icon-sm" />}
						</Button>
					</footer>
				</Dialog.Content>
			</Dialog.Portal>
		</Dialog.Root>
	);
}

/**
 * Progress reads in neutral greys, not the lane hues. Orange and green mean
 * "this session needs you" and "this session is ready" everywhere else in the
 * app; spending them on a step counter would make them mean nothing.
 */
function StepPager({ current, label, total }: { current: number; label: string; total: number }) {
	return (
		<div aria-label={label} className="flex items-center gap-1" role="progressbar">
			{Array.from({ length: total }, (_, index) => (
				<span
					aria-hidden="true"
					className={cn(
						"size-1.5 rounded-full transition-colors",
						index === current
							? "bg-foreground"
							: index < current
								? "bg-muted-foreground"
								: "bg-border-strong",
					)}
					key={index}
				/>
			))}
		</div>
	);
}

function StepHeading({
	description,
	icon,
	title,
}: {
	description: string;
	icon?: ReactNode;
	title: string;
}) {
	return (
		<div className="flex flex-col gap-2.5">
			<h2 className="flex items-center gap-2 text-brand font-medium leading-snug text-foreground">
				{icon}
				{title}
			</h2>
			<p className="text-xs leading-body text-foreground opacity-60">{description}</p>
		</div>
	);
}

function WelcomeStep() {
	const { t } = useTranslation();
	const items: { body: string; icon: ReactNode; title: string }[] = [
		{
			body: t("onboarding.welcome.agentBody"),
			icon: <Sparkles aria-hidden="true" className="size-icon-md" />,
			title: t("onboarding.welcome.agentTitle"),
		},
		{
			body: t("onboarding.welcome.outcomeBody", { defaultValue: "Register a Project, then describe your first Outcome in Work." }),
			icon: <Target aria-hidden="true" className="size-icon-md" />,
			title: t("onboarding.welcome.outcomeTitle", { defaultValue: "Start with an Outcome" }),
		},
	];
	return (
		<div className="flex flex-col items-center gap-5 text-center">
			<div className="flex flex-col gap-2.5">
				<h2 className="text-heading-sm font-medium leading-snug text-foreground">
					{t("onboarding.welcome.heading")}
				</h2>
				<p className="text-xs leading-body text-foreground opacity-60">
					{t("onboarding.welcome.body")}
				</p>
			</div>
			{/* The two steps ahead, named before they arrive: a tour that says how
			    long it is up front is one a person will actually finish. */}
			<ol className="flex w-full flex-col gap-1 text-left">
				{items.map((item) => (
					<li className="flex items-start gap-3 rounded-md px-2 py-2" key={item.title}>
						<span className="grid size-control-md shrink-0 place-items-center rounded-full hairline border-border bg-popover text-muted-foreground">
							{item.icon}
						</span>
						<span className="flex min-w-0 flex-col gap-0.5">
							<span className="text-xs font-medium text-foreground">{item.title}</span>
							<span className="text-2xs text-passive">{item.body}</span>
						</span>
					</li>
				))}
			</ol>
		</div>
	);
}

function AgentStep() {
	const { t } = useTranslation();
	const defaultAgentId = useUiStore((state) => state.defaultAgentId);
	const setDefaultAgentId = useUiStore((state) => state.setDefaultAgentId);
	const agentsQuery = useAgentsQuery();
	const { settings } = useSettings();
	const { update: updateReasoning } = useUpdateReasoning();
	const [reasoningNotice, setReasoningNotice] = useState<string | null>(null);

	// The daemon probes agent binaries at boot, so an agent installed after launch
	// is invisible until something re-probes. Asking a person to pick is exactly
	// the moment to freshen the inventory.
	useEffect(() => {
		void refreshAgentsIfStale();
	}, []);

	const agents = useMemo(() => {
		const installed = agentsQuery.data?.installed ?? [];
		const authorized = new Set((agentsQuery.data?.authorized ?? []).map((agent) => agent.id));
		return installed.map((agent) => ({
			id: agent.id,
			isAuthorized: authorized.has(agent.id),
			label: agent.label || agentLabel(agent.id),
		}));
	}, [agentsQuery.data]);

	const selectAgent = (agentId: string, label: string, isSelected: boolean) => {
		const nextAgentId = isSelected ? "" : agentId;
		setDefaultAgentId(nextAgentId);
		if (isSelected) {
			setReasoningNotice(null);
			return;
		}

		if (agentId !== "codex") {
			setReasoningNotice(t("onboarding.agent.reasoningUnavailable", { agent: label }));
			return;
		}

		if (settings?.reasoning.provider === "codex") {
			setReasoningNotice(
				settings.reasoning.ready
					? t("onboarding.agent.reasoningReady")
					: t("onboarding.agent.reasoningNotReady"),
			);
			return;
		}

		void updateReasoning({
			provider: "codex",
			model: settings?.reasoning.model ?? "",
			effort: settings?.reasoning.effort ?? "",
		}).then((status) => {
			setReasoningNotice(
				status?.ready ? t("onboarding.agent.reasoningReady") : t("onboarding.agent.reasoningNotReady"),
			);
		}).catch(() => {
			setReasoningNotice(t("onboarding.agent.reasoningSyncFailed"));
		});
	};

	return (
		<div className="flex flex-col gap-4.5">
			<StepHeading
				description={t("onboarding.agent.body")}
				icon={<Sparkles aria-hidden="true" className="size-icon-base text-muted-foreground" />}
				title={t("onboarding.agent.heading")}
			/>
			{agentsQuery.isPending ? (
				<p className="text-xs text-passive">{t("onboarding.agent.detecting")}</p>
			) : agents.length === 0 ? (
				<p className="rounded-md hairline border-border bg-popover px-3 py-2.5 text-xs leading-body text-passive">
					{t("onboarding.agent.noneFound")}
				</p>
			) : (
				<div className="grid grid-cols-2 gap-1.5">
					{agents.map((agent) => {
						const isSelected = defaultAgentId === agent.id;
						return (
							<button
								aria-pressed={isSelected}
								className={cn(
									"flex items-center gap-2.5 rounded-md hairline px-3 py-2.5 text-left transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/60",
									isSelected
										? "border-border-strong bg-popover text-foreground"
										: "border-border bg-transparent text-muted-foreground hover:bg-popover hover:text-foreground",
								)}
								key={agent.id}
								onClick={() => selectAgent(agent.id, agent.label, isSelected)}
								type="button"
							>
								<AgentAvatar provider={agent.id} />
								<span className="min-w-0 flex-1 truncate text-xs font-medium">{agent.label}</span>
								{/* A check means the local auth probe passed, not that this is the
								    pick — selection is the plate and the stronger hairline. */}
								{agent.isAuthorized ? (
									<Check
										aria-label={t("onboarding.agent.signedIn")}
										className="size-icon-sm shrink-0 text-status-ready"
									/>
								) : null}
							</button>
						);
					})}
				</div>
			)}
			{reasoningNotice ? <p className="text-2xs text-passive" role="status">{reasoningNotice}</p> : null}
			<p className="text-2xs text-passive">{t("onboarding.agent.perSessionHint")}</p>
		</div>
	);
}

function OutcomeStep() {
	const { t } = useTranslation();
	const requestCreateProject = useUiStore((state) => state.requestCreateProject);
	const closeOnboarding = useUiStore((state) => state.closeOnboarding);

	return (
		<div className="flex flex-col gap-4.5">
			<StepHeading
				description={t("onboarding.outcome.body", { defaultValue: "Kennel keeps the responsibility with you. Register a Project, describe what you want to be true, then review the Contract and Plan before authorizing any work." })}
				title={t("onboarding.outcome.heading", { defaultValue: "Create your first Outcome" })}
				icon={<Target aria-hidden="true" className="size-icon-base text-muted-foreground" />}
			/>
			<div className="flex flex-col gap-2 rounded-md hairline border-border bg-popover px-3 py-3">
				<p className="text-xs leading-body text-passive">{t("onboarding.outcome.projectHint", { defaultValue: "A Project keeps repository context and provider preferences together. It does not start work." })}</p>
				<Button className="self-start gap-1.5" onClick={() => { closeOnboarding(); requestCreateProject(); }} size="sm" variant="primary">
					<FolderPlus aria-hidden="true" className="size-icon-sm" />
					{t("onboarding.outcome.createProject", { defaultValue: "Create a Project" })}
				</Button>
			</div>
		</div>
	);
}
