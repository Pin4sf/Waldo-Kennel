// Reviewers are a narrower vocabulary than worker agents on purpose: a
// reviewer-only tool must not become a valid worker, and the daemon rejects
// anything outside this set.
//
// These are recognized persisted/read identities. Fresh review requests use the
// generated Codex-only request type instead.

const REVIEWER_HARNESS_IDS = [
	"agy",
	"aider",
	"amp",
	"auggie",
	"autohand",
	"claude-code",
	"codex",
	"cline",
	"continue",
	"copilot",
	"crush",
	"cursor",
	"devin",
	"droid",
	"goose",
	"grok",
	"kilocode",
	"kiro",
	"kimi",
	"kimchi",
	"muse",
	"opencode",
	"pi",
	"qwen",
	"vibe",
] as const;

export type ReviewerHarnessId = (typeof REVIEWER_HARNESS_IDS)[number];

export const KNOWN_REVIEWER_HARNESS_IDS: ReadonlySet<string> = new Set(REVIEWER_HARNESS_IDS);

export function toReviewerHarnessId(value?: string): ReviewerHarnessId | undefined {
	return value && KNOWN_REVIEWER_HARNESS_IDS.has(value) ? (value as ReviewerHarnessId) : undefined;
}
