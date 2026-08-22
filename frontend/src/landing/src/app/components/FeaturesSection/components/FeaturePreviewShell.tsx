"use client";

import type { CSSProperties, ReactNode } from "react";

export const featurePreviewTokens = {
	// Exact dark-theme values from frontend/src/styles/tokens.css (:root) —
	// the Kennel orchestrator system. Keep in step when those tokens change.
	"--preview-background": "#151515",
	"--preview-foreground": "#fafaf8",
	"--preview-card": "#272725",
	"--preview-card-foreground": "#fafaf8",
	"--preview-primary": "#fafaf8",
	"--preview-primary-foreground": "#272725",
	"--preview-muted": "#1e1e1e",
	"--preview-muted-foreground": "#9a9a96",
	"--preview-accent": "#272725",
	"--preview-border": "rgb(255 255 255 / 8%)",
	"--preview-border-strong": "rgb(255 255 255 / 10%)",
	"--preview-ring": "#2388ff",
	"--preview-divider": "rgb(255 255 255 / 10%)",
	"--preview-input": "rgb(255 255 255 / 10%)",
	"--preview-sidebar": "#161616",
	"--preview-sidebar-foreground": "#fafaf8",
	"--preview-sidebar-accent": "#1e1e1e",
	"--preview-sidebar-hover": "#1e1e1e",
	"--preview-passive": "#6b6b68",
	"--preview-raised": "#353533",
} as CSSProperties;

export const previewStatus = {
	working: "#2388ff", // --color-status-working
	warning: "#fb8404", // --color-status-needs-you
	success: "#00cc6e", // --color-status-ready
	error: "oklch(0.704 0.191 22.216)", // --destructive
	accent: "#2388ff",
} as const;

export function FeaturePreviewShell({
	children,
	className = "",
	title,
	trailing,
}: {
	children: ReactNode;
	className?: string;
	title: string;
	trailing?: ReactNode;
}) {
	return (
		<div
			className={`mx-auto w-full min-w-0 max-w-[570px] select-none overflow-hidden rounded-[20px] border border-[var(--preview-border)] bg-[var(--preview-background)] font-sans text-[var(--preview-foreground)] antialiased shadow-[0_24px_64px_-20px_rgba(0,0,0,0.8)] ${className}`}
			style={featurePreviewTokens}
		>
			<div className="flex h-9 items-center border-b border-[var(--preview-border)] bg-[var(--preview-background)] px-3">
				<div className="flex items-center gap-1.5" aria-hidden="true">
					<span className="size-2.5 rounded-full bg-[#ff5f57]" />
					<span className="size-2.5 rounded-full bg-[#ffbd2e]" />
					<span className="size-2.5 rounded-full bg-[#28c840]" />
				</div>
				<div className="ml-4 flex min-w-0 items-center gap-2">
					<img src="/ao-logo.svg" alt="" className="size-4" draggable="false" />
					<span className="truncate text-[11px] font-semibold tracking-[-0.4px] text-[var(--preview-muted-foreground)]">
						{title}
					</span>
				</div>
				{trailing ? <div className="ml-auto min-w-0 shrink-0">{trailing}</div> : null}
			</div>
			{children}
		</div>
	);
}

export function StatusDot({
	color,
	pulse = false,
}: {
	color: string;
	pulse?: boolean;
}) {
	return (
		<span className="relative flex size-2 shrink-0">
			{pulse ? (
				<span
					className="absolute inline-flex size-full animate-ping rounded-full opacity-40"
					style={{ backgroundColor: color }}
				/>
			) : null}
			<span
				className="relative inline-flex size-2 rounded-full"
				style={{ backgroundColor: color }}
			/>
		</span>
	);
}
