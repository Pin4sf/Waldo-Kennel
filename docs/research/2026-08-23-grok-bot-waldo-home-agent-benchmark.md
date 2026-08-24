# Grok Bot benchmark for Waldo Home and agent interaction

- **Date:** 2026-08-23
- **Status:** Product and interaction research only; no implementation authority
- **Scope:** Current first-party Grok consumer, Grok Bot, and Grok Build behavior relevant to Waldo Home, chat, delegation, and governed agent execution

## Executive finding

“Grok” now names three materially different products:

1. **Grok consumer** is a multimodal assistant with chat, research, voice, creation, connectors, skills, and automations.
2. **Grok Bot** is a roster of persistent named AI teammates, each with a durable conversation and working context, operating through an account-scoped cloud computer.
3. **Grok Build** is a coding-agent surface with coding sessions, subagents, dashboards, and large parallel workflows.

Grok Bot is the closest interaction benchmark for the requested Waldo agent experience. Its strongest pattern is not unrestricted autonomy; it is the legibility of a durable teammate conversation, attention state, artifacts, handoffs, live computer work, and inline approvals. Waldo should adapt those mechanics under one Waldo identity and retain stronger responsibility, provenance, authority, evidence, acceptance, and closure boundaries.

## Evidence discipline

This note uses current first-party xAI/SpaceXAI product pages, documentation, legal pages, and official app-store metadata. Product documentation establishes described behavior, not independent reliability. Testimonials and phrases such as “always on,” “finishes end to end,” and “gets sharper” remain operator claims. Exact pixel styling, animation, narrow desktop behavior, and rollout-dependent controls were not reproduced in an authenticated installation.

## Observed Grok Bot product structure

### Sidebar and durable teammates

- A Bot is a durable named teammate with a job, profile, avatar, primary conversation, working context, routines, and notification state. xAI recommends focused roles rather than one “General Helper.” [Create and manage Bots](https://docs.x.ai/grok-bot/bots)
- `New` creates a Bot or group chat. Bots can be pinned, hidden, duplicated, edited, or deleted. The shell distinguishes attention, unread, and working state. [Create and manage Bots](https://docs.x.ai/grok-bot/bots), [Settings and notifications](https://docs.x.ai/grok-bot/settings-and-notifications)
- An account can have up to 50 Bots and group chats combined. A group contains two to six Bots. [Create and manage Bots](https://docs.x.ai/grok-bot/bots), [Message and collaborate](https://docs.x.ai/grok-bot/chat-and-collaboration)

### Conversation and composer

- The main surface is a teammate-style conversation. The composer accepts text, links, images, local files, `/` skill references, and `@` mentions of Bots, groups, routines, and connectors. It also supports replies, reactions, redirecting current work, and a direct “Stop now” message. [Message and collaborate](https://docs.x.ai/grok-bot/chat-and-collaboration)
- Tool activity, computer use, files, questions, handoffs, errors, and approval requests appear inline with messages. Search can find conversations, messages, files, links, and routines and return to the matching location. [Message and collaborate](https://docs.x.ai/grok-bot/chat-and-collaboration), [Files and results](https://docs.x.ai/grok-bot/files-and-results)
- Bots may hand work to one another asynchronously. Group conversations keep the handoff visible and allow the user to target a specific Bot with `@`. [Message and collaborate](https://docs.x.ai/grok-bot/chat-and-collaboration)

### Computer, tools, and artifacts

- Bots work on a persistent cloud computer with browser, terminal, filesystem, and connected apps. Work may continue after the user's laptop closes. [Grok Bot overview](https://docs.x.ai/grok-bot/overview)
- `Agent Computer` exposes a live view of clicks, typing, navigation, and status. The user can take over for passwords, passkeys, 2FA, CAPTCHAs, payments, identity checks, or a human-only site step. [Use the computer and apps](https://docs.x.ai/grok-bot/computer-and-apps)
- All Bots on an account share the same cloud computer, files, browser sessions, and command-line credentials. Separate screens are work surfaces, not security boundaries. Connectors are also account-wide. [Use the computer and apps](https://docs.x.ai/grok-bot/computer-and-apps)

### Skills, routines, and learning

- A successful workflow can be saved as a reusable skill. A demonstrated browser process of up to ten minutes can become a draft skill for review. [Skills and routines](https://docs.x.ai/grok-bot/skills-routines-and-automations)
- A routine belongs to one Bot and records its schedule, timezone, source, expected result, failure behavior, approval boundary, and recent run history. [Skills and routines](https://docs.x.ai/grok-bot/skills-routines-and-automations)
- Bot memory retains working preferences, facts, and summaries, but xAI explicitly says it is not an authoritative source and recommends reopening current data for consequential decisions. [Create and manage Bots](https://docs.x.ai/grok-bot/bots)

### Approvals and mobile continuity

- Consequential actions may present the proposed operation and inputs inline with `Allow once`, `Deny`, and sometimes `Always allow`. Auto Review can apply model-based approval rules but is not a replacement for least privilege. [Approvals, security, and privacy](https://docs.x.ai/grok-bot/approvals-security-and-privacy)
- iOS uses the same Bots, conversations, routines, connectors, and shared computer. The home screen has search and a `+` action for a new agent or group; drafts persist per conversation. [Grok Bot for iOS](https://docs.x.ai/grok-bot/mobile)

## Adjacent Grok capabilities

- Consumer Grok supports multimodal chat, files, live web/X retrieval, voice and live camera, image/video creation, connectors, skills, and automations. These are supporting capabilities, not evidence of Grok Bot's responsibility model. [Grok overview](https://docs.x.ai/grok/overview), [official App Store listing](https://apps.apple.com/us/app/grok-ai/id6670324846)
- Consumer Automations combine chat-like instructions, files, connectors, skills, a mode, schedules or email triggers, notifications, and run history; each run creates a resumable conversation. [Automations in Grok](https://x.ai/news/grok-automations)
- Grok Build exposes a distinct coding-agent dashboard and parallel coding workflows. It should be compared to Kennel Work provider sessions, not to Personal Home or Waldo identity. [Agent Dashboard](https://x.ai/news/agent-dashboard), [Workflows in Grok Build](https://x.ai/news/workflows)

## Truth, privacy, and product limits

- Grok Bot is an early beta and depends on cloud storage and Cursor authentication/privacy settings. It does not support Legacy Privacy Mode. [Introducing Grok Bot](https://x.ai/news/introducing-grok-bot), [Approvals, security, and privacy](https://docs.x.ai/grok-bot/approvals-security-and-privacy)
- xAI says Grok can hallucinate, mis-summarize, or return incorrect facts and that users should verify consequential results. [Consumer FAQ](https://x.ai/legal/faq)
- Consumer conversations may be used for model improvement unless the user opts out; Private Chat has a separate retention/training treatment. [Consumer FAQ](https://x.ai/legal/faq), [Privacy Policy](https://x.ai/legal/privacy-policy)
- An approval affects the pending action and does not undo earlier effects. A direct stop also does not undo actions already completed. [Message and collaborate](https://docs.x.ai/grok-bot/chat-and-collaboration), [Approvals, security, and privacy](https://docs.x.ai/grok-bot/approvals-security-and-privacy)

## Preliminary Waldo disposition

### Adopt

- One durable conversation with a clear owner and visible attention state.
- Inline artifacts, questions, handoffs, action logs, and approval cards.
- A live execution/computer inspector for work that genuinely needs it.
- Redirect, pause, stop, and exact return while work is active.
- Search across conversations, sources, artifacts, responsibilities, and routines with exact jump-back.
- Mobile/desktop continuity for the same durable relationship when later gates permit it.

### Adapt

- **Bot roster → one Waldo plus bounded specialists.** Waldo remains the user-owned relationship and responsibility governor. Provider agents and specialists are replaceable executors attached to Outcomes/Work Units, not competing assistant identities.
- **Bot conversation → Waldo conversation plus canonical commands.** Messages may ask, capture, correct, propose, explain, or steer. Creating or closing an Open Loop/Outcome still requires the native explicit command and append-only decision record.
- **Needs attention → Needs You.** Show the exact responsibility, evidence, recommendation, consequence, safe default, and approval boundary rather than activity or unread count alone.
- **Memory → inspectable source-bearing context.** Grok's warning that memory is non-authoritative aligns with Waldo, but Waldo adds admission, correction, counter-evidence, scope, expiry, deletion, and retrieval receipts.
- **Routines → governed skills/automations.** Require an editable scope, trigger, authority, expected evidence, failure behavior, version, test history, rollback, and explicit promotion.
- **Shared cloud computer → capability-scoped local Work execution.** Kennel keeps local workspace custody and isolated worktrees. Any future hosted execution requires explicit attachment and must not create dual canonical writers.

### Reject

- Multiple Bots as multiple Waldo identities.
- A primary Home that is only a chat transcript or agent activity roster.
- Treating a message, model memory, tool completion, artifact, run success, or agent handoff as Outcome acceptance or Open Loop closure.
- Account-wide credentials or connectors silently becoming available to every specialist.
- Automatic learning/promotion from observation or demonstration without evaluation and approval.
- “Always on” or “finished end to end” as proof without evidence, reconciliation, and the user's acceptance.

## Design implication for the next brainstorm

Preserve the merged adaptive Home brief and its single expanded Quick Capture. Add Waldo conversation as an on-demand, globally reachable relationship surface rather than replacing Home with chat. In Work, expose bounded specialist/provider sessions and their handoffs under the selected Outcome. The main unresolved product choice is how much of that specialist roster the user should create and manage directly versus letting Waldo recommend and instantiate task-scoped specialists after authorization.
