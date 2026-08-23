import { Loader2, Plus } from "lucide-react";
import { useTranslation } from "react-i18next";
import { useShell } from "../lib/shell-context";
import { CreateProjectFlow } from "./CreateProjectFlow";
import { TopbarButton } from "./TopbarButton";
import { WelcomePanel } from "./WelcomePanel";

// Board empty states: first-launch welcome (`BoardWelcome`) and project board
// with no worker sessions yet (`ProjectBoardEmpty`).
export function BoardWelcome() {
	const { createProject, initializeProjectRepository } = useShell();
	return (
		<WelcomePanel>
			<div
				className="flex h-full min-h-0 items-center justify-center overflow-y-auto px-6 py-8"
				data-testid="board-welcome"
			>
				<CreateProjectFlow
					embedded
					mode="choose"
					onCreateProject={createProject}
					onInitializeProject={initializeProjectRepository}
				/>
			</div>
		</WelcomePanel>
	);
}

// Project board with a registered project but no worker sessions yet: a clear
// starting point instead of four empty Kanban columns.
export function ProjectBoardEmpty({
	isProjectRestarting,
	isInstallingTmux,
	isSpawning,
	onNewTask,
	onInstallTmux,
	isExploring = false,
	tmuxInstallMessage,
	spawnError,
}: {
	isProjectRestarting: boolean;
	isInstallingTmux?: boolean;
	isSpawning: boolean;
	onNewTask: () => void;
	onInstallTmux?: () => void;
	isExploring?: boolean;
	tmuxInstallMessage?: string | null;
	spawnError?: string | null;
}) {
	const { t } = useTranslation();
	const agentLabel = t("shell.orchestrator");
	return (
		<div className="flex h-full min-h-0 items-center justify-center overflow-y-auto px-6">
			<div className="flex w-full max-w-preview-content flex-col items-center rounded-xl border border-accent/20 bg-raised/60 px-8 py-10 text-center shadow-sm">
				{isExploring ? (
					<>
						<Loader2 className="mb-3 size-6 animate-spin text-accent" aria-hidden="true" />
						<h2 className="text-subtitle font-semibold tracking-tight text-foreground">{t("board.exploring.title")}</h2>
						<p className="mt-2 max-w-lg text-md-sm leading-relaxed text-muted-foreground">{t("board.exploring.body", { agent: agentLabel })}</p>
					</>
				) : (
					<>
						<h2 className="text-subtitle font-semibold tracking-tight text-foreground">{t("board.empty.title")}</h2>
						<p className="mt-2 max-w-lg text-md-sm leading-relaxed text-muted-foreground">{t("board.empty.body")}</p>
					</>
				)}
				<div className="mt-5 flex items-center gap-2">
					<TopbarButton
						aria-label={t("shell.newTask")}
						className="outcome-primary-action"
						disabled={isProjectRestarting}
						onClick={onNewTask}
						variant="primary"
					>
						<Plus className="size-icon-md" aria-hidden="true" />
						{t("shell.newTask")}
					</TopbarButton>
				</div>
				{spawnError && (
					<div className="mt-3 flex flex-col items-center gap-2">
						<p className="text-caption leading-body text-error" role="status">
							{spawnError}
						</p>
						{onInstallTmux ? (
							<TopbarButton disabled={isSpawning || isInstallingTmux || isProjectRestarting} onClick={onInstallTmux}>
								{isInstallingTmux ? "Installing tmux…" : "Install tmux via Homebrew"}
							</TopbarButton>
						) : null}
						{tmuxInstallMessage ? (
							<p className="text-caption leading-body text-muted-foreground" role="status">
								{tmuxInstallMessage}
							</p>
						) : null}
					</div>
				)}
			</div>
		</div>
	);
}
