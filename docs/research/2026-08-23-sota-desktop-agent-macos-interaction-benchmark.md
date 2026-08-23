# SOTA desktop-agent and premium macOS interaction benchmark

**Date:** 2026-08-23  
**Status:** product and interaction research only; no implementation authority  
**Scope:** OpenAI ChatGPT/Codex desktop, Anthropic Claude Desktop/Claude Code Desktop, Cursor, Raycast, Linear, Windsurf, Dayflow, and Apple macOS interaction guidance  
**Question:** Which current agent utilities and mature macOS conventions should Waldo Kennel adopt, adapt, or reject before the Personal Agency surface is considered complete?

## Executive finding

The strongest current products do not treat an agent as one endless transcript. They separate a fast conversational entry from inspectable work, keep active context visible, let the person interrupt or redirect work, and put approvals, progress, artifacts, diffs, errors, and recovery in native structures around the conversation.

The coherent pattern for Waldo is:

```text
invoke quickly
  -> see or detach the proposed context
  -> discuss, clarify, or define a bounded goal
  -> review scope and authority
  -> run in a separate inspectable Activity surface
  -> interrupt, redirect, approve, or recover
  -> inspect evidence and return path
  -> verify or leave the outcome explicitly unknown
```

The benchmark supports the approved `Conversation | Activity` rail and the proposed dedicated Home Chat. It does **not** support making provider personas or specialist agents peer identities with Waldo. Linear provides the clearest responsibility precedent: an issue can be delegated to an agent while the human remains its owner, and agent activity stays visible on the same underlying work object ([Linear delegation](https://linear.app/docs/assigning-issues); [Linear Agent Interaction Guidelines](https://linear.app/developers/aig)).

For the requested Dayflow-like Insights, the transferable mechanism is a bounded, correctable, source-backed account of a day. The dangerous translation is to turn captured activity into intention, personality, responsibility, or proof of an outcome. Dayflow itself describes a screen-derived timeline and daily/weekly synthesis, while its current source permits timeline-card editing/deletion and its privacy modes disclose where screenshots are processed ([Dayflow](https://www.dayflow.so/); [official repository](https://github.com/JerryZLiu/Dayflow/tree/86f5288d7267c9c8da1ab47e9c25ad1cd0b80046); [privacy policy](https://www.dayflow.so/privacy/)).

Premium macOS quality is not a coat of visual polish. It requires window resizing that preserves task hierarchy, one clear scroll owner per axis, standard pointer and context-menu behavior, keyboard and mouse parity, undo/recovery, system shortcuts, visible focus, Full Keyboard Access, minimum hit regions, and reduced-motion behavior. Apple explicitly recommends resizable/configurable windows, familiar menu and keyboard commands, automatically collapsing sidebars as space shrinks, standard pointer/gesture behavior, and alternatives to gesture-only actions ([Designing for macOS](https://developer.apple.com/design/human-interface-guidelines/designing-for-macos/); [Sidebars](https://developer.apple.com/design/human-interface-guidelines/sidebars); [Pointing devices](https://developer.apple.com/design/human-interface-guidelines/pointing-devices); [Accessibility](https://developer.apple.com/design/human-interface-guidelines/accessibility)).

## Evidence discipline

Source classes used:

1. official first-party product documentation and help centers;
2. official product announcements;
3. official open-source repositories pinned to a revision;
4. Apple Human Interface Guidelines.

Labels:

- **Observed** — directly documented or implemented in a cited first-party source.
- **Inference** — a bounded interpretation of observed behavior.
- **Unknown** — not established by the inspected first-party evidence.
- **Adopt / Adapt / Reject** — a Waldo product recommendation, not shipped behavior.

Official marketing establishes the operator's claim, not independent reliability. Open-source code establishes implementation at a revision, not deployment or production quality. Screenshots establish a visible state, not accessibility, responsive behavior, or durable semantics.

The originally ambiguous names are now resolved for this local comparison by the installed packages the user named: `/Applications/Grok Bot.app` (`com.anysphere.sand`, version `0.24.0`) and the running Hermes desktop app (`com.nousresearch.hermes`; the installed setup wrapper is `com.nousresearch.hermes.setup`, version `0.0.1`). This identifies the inspected local builds; it is not proof of feature parity with a public product or a newer release.

## Local installed-application audit

**Method:** Read-only public UI and macOS accessibility interaction on 2026-08-23. No application-support database, credential store, private API, DevTools, hidden state, provider token, or network payload was inspected. Grok Bot opened on an existing saved conversation; user-specific transcript content was excluded from the evidence below and is not reproduced. No message was sent, profile/bot/channel was created, plugin was added, credential was entered, setting was changed, or paid flow was accepted.

### Grok Bot 0.24.0 — primary specialist benchmark

**Observed:**

- The default shell is a resizable left `Bots` sidebar plus one conversation surface. The sidebar exposes Search, a `New Bot or Channel` menu with `New Bot` and `New Channel`, compact bot rows, Plugins, account access, and a draggable sidebar splitter.
- Conversation chrome exposes the selected bot, `View conversation details`, `Grok Bot's Computer`, a saved/offline freshness banner, a transcript, file attachment, voice input, and plan availability at the composer boundary.
- Transcript items expose native per-message reaction, reply, and more-action controls. Questions and consequential connection steps can appear as structured cards rather than plain transcript text.
- Credential collection is presented in a dedicated secure field with a separate save action and explicit custody copy instead of asking for a token in ordinary chat. The field was not used.
- Plugins open as a dedicated overlay with Close, `Your plugins`, search, category filters, featured/team/category sections, plugin detail entry points, and explicit `Add` versus `Added` state.

**Inference / limit:** The bot list makes named specialists highly discoverable, but the visible shell gives each bot peer identity and does not expose a Waldo-compatible purpose/scope/authority/budget/evidence/return contract in the list itself. The inspected saved state showed conversational narration around connector work, but no separate Activity timeline, verification lineage, or recovery boundary was visible. Creating a bot/channel, running work, approvals, secondary-click behavior, minimum-width recomposition, VoiceOver order, and Reduce Motion were not exercised because doing so would create state, require a live entitlement, or exceed the read-only audit boundary.

**Waldo decision:** **Adopt** compact specialist discovery, structured in-conversation cards, item actions, a dedicated capability library, and credential entry outside ordinary chat. **Adapt** `New Bot` into explicit preview-only specialist creation with purpose, sources/tools, authority ceiling, budget, pause/revoke, evidence, and return path under Waldo Activity. **Reject** a Home roster of peer bot identities, connector success as evidence, or a secure-looking renderer field before Keychain-backed custody exists.

### Hermes desktop — secondary specialist benchmark

**Observed:**

- The desktop shell exposes collapsible/swappable sidebars, layout editing, HUD mode, settings, an optional right sidebar, sessions and projects, `Capabilities`, `Messaging`, `Artifacts`, and `Kanban`, plus a main draft/terminal tab group.
- The draft composer visibly separates context attachment, model selection, text entry, dictation, spoken replies, wake word, and voice conversation. The empty draft explicitly says nothing has been sent.
- `Capabilities` separates Skills, Tools, MCP, and Hub discovery. Individual capabilities have category labels and enabled switches, a detail pane, and copy stating changes apply to new sessions.
- Profiles are explicit independent Hermes environments with separate configuration, skills, and `SOUL.md`. Creation asks for a name, clone source, and optional persona/system prompt; import and management are separate actions.
- Safety settings expose approval mode, approval timeout, MCP-reload confirmation, command allowlist, secret redaction, private-URL policy, and file checkpoints.
- Memory/context settings expose persistent memory, a compact user profile, budgets, provider choice, compression strategy, thresholds, and protected recent messages.
- The shortcut browser makes queue/steer/cancel, profile switching, session navigation, command palette, settings, review pane, terminals, find, layout, and other commands discoverable and rebindable.

**Inference / limit:** Hermes provides the richer configuration grammar for specialist environments and operator control, while Grok Bot provides the cleaner conversational/card presentation. Hermes' visible profile contract is still identity/persona- and configuration-led; purpose, completion condition, authority ceiling, budget, evidence, and native return destination are not one coherent profile summary. Persistent memory and user-profile behavior are configurable but are not evidence of Waldo's required admission/correction/provenance lifecycle. Live runs, approval cards, artifacts, messaging, Kanban recovery, pointer context menus, minimum-width behavior, VoiceOver order, and Reduce Motion remain unverified.

**Waldo decision:** **Adopt** searchable capabilities, explicit profile creation, session-scoped capability changes, safety policy, queue/steer/cancel grammar, and shortcut discoverability. **Adapt** profiles into scoped workers coordinated by Waldo and require governance fields before creation/pinning. **Reject** automatic durable memory, persona as the primary specialist contract, provider/model selection as Home IA, and profile switching as switching the user's personal agent.

### Combined product consequence

Grok Bot should lead Waldo's compact specialist discovery and conversational card behavior; Hermes should lead capability/profile configuration and operator-control coverage. Neither establishes Waldo's responsibility, evidence, verification, acceptance, or memory-admission semantics. The resulting Waldo surface therefore keeps specialists subordinate under Activity, keeps Home Chat the same Waldo relationship as the rail, and requires an inspectable profile/run contract before showing a created or pinned specialist.

## Product benchmark

| Product | Observed high-value utility | Limit or unknown | Waldo decision |
| --- | --- | --- | --- |
| **OpenAI Codex / ChatGPT desktop** | Codex organizes parallel agents as project threads, isolates code work with worktrees, and puts diff review and comments inside the thread. Automations return completed work to a review queue. The newer ChatGPT desktop docs add goal progress with pause/resume/edit/clear, side chat, an attention-oriented Activity queue, explicit permission modes, review panes, and subordinate subagent threads ([Codex app announcement](https://openai.com/index/introducing-the-codex-app/); [long-running work](https://learn.chatgpt.com/docs/long-running-work); [notifications](https://learn.chatgpt.com/docs/notifications); [permissions](https://learn.chatgpt.com/docs/permission-modes); [code review](https://learn.chatgpt.com/docs/code-review); [subagents](https://learn.chatgpt.com/docs/agent-configuration/subagents)). | Review artifacts and provider completion do not establish a durable verified real-world outcome. Public docs do not specify responsive breakpoints, complete VoiceOver order, or reduced-motion acceptance. | **Adopt** bounded threads, review queue, pause/resume, inspectable subagent return, and evidence panes. **Adapt** projects into Waldo topics/outcomes rather than repo-only containers. **Reject** a roster as the Home identity or completion as acceptance. |
| **ChatGPT macOS quick surface** | `Option-Space` opens a draggable Chat Bar; it can start/refocus/reopen conversation and attach files, photos, or screenshots. Work-with-Apps shows a banner naming the active app/context before send. The app documents stop-streaming, restored drafts, background completion notifications, horizontal overflow for wide content, and focus/accessibility fixes ([Chat Bar](https://help.openai.com/en/articles/9295241-accessing-the-launcher-chatgpt-macos-app); [Work with Apps](https://help.openai.com/en/articles/10119604-work-with-apps-on-macos); [macOS release notes](https://help.openai.com/en/articles/9703738-macos-app-release-notes)). | Active-app access relies on system permissions; the docs do not prove that every captured context path is user-understood or minimally scoped. | **Adopt** quick invocation, recoverable focus, visible context banner, and draft restoration. **Adapt** into one Waldo launcher/rail plus detachable message context. **Reject** silent active-window attachment and an always-on-top window without a user-controlled placement policy. |
| **Claude Desktop / Claude Code Desktop** | Code sessions bind folder, environment, model, and permission mode before work. Desktop exposes draggable/resizable panes, diff, preview, terminal, side chat, shortcut browser, task list, and `Normal / Verbose / Summary` trace density. It supports stop, queued steering, file/context attachment, inline diff comments, local preview/DOM inspection, and OS completion notifications ([Claude Code Desktop](https://code.claude.com/docs/en/desktop)). | Anthropic explicitly says terminal shortcuts do not all carry into Desktop. Coordinated agent teams are not available in Desktop. Public docs do not establish exact narrow-window rules or complete accessibility conformance. | **Adopt** short-first trace density, edge-resizable panes, side chat, contextual attachments, and clear session configuration. **Adapt** Tasks into bounded Waldo Activity runs. **Reject** exposing model/runtime selection as the primary product job. |
| **Claude Code authority and recovery** | Permissions distinguish read, edit, shell, and modes such as default, accept edits, plan, auto, and deny/bypass variants. Checkpoints are created from user prompts and can rewind file edits; external effects cannot be rewound. Parallel-work docs distinguish subagents, agent view, agent teams, worktrees, and batch by coordination and isolation needs ([permissions](https://code.claude.com/docs/en/permissions); [checkpointing](https://code.claude.com/docs/en/checkpointing); [how Claude Code works](https://code.claude.com/docs/en/how-claude-code-works); [parallel agents](https://code.claude.com/docs/en/agents)). | Checkpoints cover Claude-managed file edits, not arbitrary remote actions. Permission bypass remains dangerous even when available. | **Adopt** explicit authority modes, interrupt, local checkpoint, and recovery-limit disclosure. **Adapt** into purpose/source/action-specific Waldo authority rather than one global autonomy toggle. **Reject** “full access” as a casual default or undo language for irreversible effects. |
| **Cursor** | Plan Mode researches and asks questions before producing an editable plan; Agent/Ask/Manual modes expose different capabilities. Context can be attached through `@` references, drag-and-drop, terminals, chats, git diffs, browser, images, and voice. Agent edits have a file-by-file/fine-grained diff review, and checkpoints restore agent-made file states. Background agents appear in a searchable sidebar and can be followed up or taken over ([Plan Mode](https://prod.cursor.com/docs/agent/plan-mode); [modes](https://docs.cursor.com/agent); [prompting/context](https://prod.cursor.com/docs/agent/prompting); [diff review](https://docs.cursor.com/en/agent/review); [checkpoints](https://docs.cursor.com/en/agent/chat/checkpoints); [background agents](https://docs.cursor.com/background-agent)). | Cursor warns that remote background agents auto-run commands in internet-connected environments and create different retention/risk conditions. Checkpoints are not version control. | **Adopt** explicit plan-before-build, source chips, review-by-artifact, and take-over. **Adapt** background runs to Kennel-owned capability boundaries. **Reject** opaque context condensation and remote execution without visible custody/retention. |
| **Raycast** | Quick AI opens from Root Search, stays in the same compact window, and can hand the complete conversation, model, and attachments to full AI Chat. `@`/Add Context scopes selected text, focused window, screen area, or screen to a message. Long messages collapse. The system uses consistent list navigation and an `⌘K` Action Panel; commands can receive aliases or global hotkeys ([Quick AI](https://manual.raycast.com/ai/quick-ai); [AI Chat](https://manual.raycast.com/ai/chat); [Search Bar](https://manual.raycast.com/search-bar); [keyboard shortcuts](https://manual.raycast.com/keyboard-shortcuts); [aliases and hotkeys](https://manual.raycast.com/command-aliases-and-hotkeys)). | A universal launcher is excellent for invocation but too hidden to carry Waldo's responsibility, evidence, and approval semantics by itself. | **Adopt** quick-to-full handoff, short-first detail, global invoke, consistent action panel, and discoverable shortcuts. **Adapt** Quick Capture and Waldo Chat without merging their jobs. **Reject** a command palette as the only route to important state or authority. |
| **Linear** | Mouse, keyboard, command menu, and context menu operate on the same selected object. `Escape` clears selection, `⌘K` opens actions, right-click exposes contextual actions, Shift-click supports range selection, and `⌘Z` restores destructive or bulk changes. Crucially, delegation keeps a human as the assignee/owner while an agent is a delegate; agent sessions expose working, waiting, error, and finished states, semantic activities, and stop signals ([selection](https://linear.app/docs/select-issues); [context menus](https://linear.app/now/invisible-details); [undo](https://linear.app/changelog/undo-actions); [delegation](https://linear.app/docs/assigning-issues); [agent interaction](https://linear.app/developers/agent-interaction); [agent signals](https://linear.app/developers/agent-signals)). | Linear's issue ontology is work-team-specific and does not prove personal memory or whole-life context. Detailed reasoning visibility can also become noisy if always expanded. | **Adopt** human ownership after delegation, semantic run states, direct stop, pointer/keyboard parity, selection-preserving undo, and contextual menus. **Adapt** issue activity to Waldo's bounded runs and evidence lineage. **Reject** treating the agent like a human assignee or turning every activity into a notification. |
| **Windsurf** | Cascade combines Code/Chat modes, selected-editor/terminal context, named checkpoints, voice, tools, and linter awareness. Terminal authority has Disabled, Allowlist, Auto, and Turbo levels with deny precedence; a dedicated agent terminal separates its commands from the user's terminal ([Cascade](https://docs.windsurf.com/windsurf/cascade/cascade); [terminal](https://docs.windsurf.com/windsurf/terminal); [memories and rules](https://docs.windsurf.com/windsurf/cascade/memories)). | Turbo-style execution and automatically generated memories are incompatible with Waldo's default authority and memory-admission boundaries. | **Adopt** visible selected context, dedicated execution surface, checkpoints, and deny precedence. **Adapt** reusable workflows into explicit skills. **Reject** Turbo as a consumer default and automatic memory as personal truth. |
| **Dayflow** | Low-frequency screen capture becomes correctable chronological cards, daily standup material, weekly review, and grounded chat. Current `main` is pinned at `86f5288d…`. Cards can be edited/deleted; failures can retain source material for retry; capture supports timed/indefinite pause and per-app exclusions. Processing mode discloses local, BYO-provider, or opt-in Dayflow Cloud routing ([product](https://www.dayflow.so/); [repository](https://github.com/JerryZLiu/Dayflow/tree/86f5288d7267c9c8da1ab47e9c25ad1cd0b80046); [card storage/editing](https://github.com/JerryZLiu/Dayflow/blob/86f5288d7267c9c8da1ab47e9c25ad1cd0b80046/Dayflow/Dayflow/Core/Recording/StorageManager%2BTimelineCards.swift#L138-L227); [pause manager](https://github.com/JerryZLiu/Dayflow/blob/86f5288d7267c9c8da1ab47e9c25ad1cd0b80046/Dayflow/Dayflow/App/PauseManager.swift#L13-L157); [app exclusions](https://github.com/JerryZLiu/Dayflow/blob/86f5288d7267c9c8da1ab47e9c25ad1cd0b80046/Dayflow/Dayflow/Core/Recording/RecordingPrivacyPreferences.swift#L19-L153); [privacy](https://www.dayflow.so/privacy/)). | Screen evidence cannot reveal intention, off-screen events, responsibility, or whether a real-world result occurred. The privacy policy says cloud Activity Data may later be refined, so derived cards are projections, not immutable evidence. | **Adopt** temporal re-entry, source windows, pause/exclusion, correction, gaps, and provider disclosure. **Adapt** into Insight candidates below Today, not a surveillance-first Home. **Reject** time/productivity scoring, intent inference, and activity as proof of outcome. |

## Cross-product interaction lessons

### 1. Conversation and work need different reading densities

**Observed:** OpenAI uses conversation plus goals/Activity/review panes; Claude Code Desktop offers Summary, Normal, and Verbose views; Raycast collapses long messages; Linear distinguishes the compact issue view from inspectable agent activity ([OpenAI long-running work](https://learn.chatgpt.com/docs/long-running-work); [Claude Code Desktop](https://code.claude.com/docs/en/desktop); [Raycast Quick AI](https://manual.raycast.com/ai/quick-ai); [Linear agent interaction](https://linear.app/developers/agent-interaction)).

**Waldo adaptation:** Conversation should show the shortest useful human-readable explanation. Activity owns plans, tool/run detail, approvals, evidence, and failure. “Show details” expands within a semantic card or opens the owning Work object. Raw logs must not dominate the relationship surface.

### 2. Context must be visible before send

**Observed:** ChatGPT Work with Apps shows which app/context will be included; Cursor accepts explicit `@` attachments and drag/drop; Raycast supports message-scoped focused-window, selected-text, and screen context; Claude Desktop supports file/context attachment from picker, drag/drop, paste, and context menus ([ChatGPT Work with Apps](https://help.openai.com/en/articles/10119604-work-with-apps-on-macos); [Cursor prompting](https://prod.cursor.com/docs/agent/prompting); [Raycast Quick AI](https://manual.raycast.com/ai/quick-ai); [Claude Code Desktop](https://code.claude.com/docs/en/desktop)).

**Waldo adaptation:** Every message gets a detachable context chip stating surface, object, source range, and freshness. Hover or secondary click reveals what will be sent. Detaching context cannot silently detach an already-approved run; that requires a new proposal or cancellation.

### 3. Progress must be semantic, interruptible, and honest

**Observed:** OpenAI goal controls include pause/resume/edit/clear; Linear agent sessions distinguish working, waiting, error, and finished and support a stop signal; Apple says determinate progress is preferable only when duration is actually known and stalled processes need actionable feedback ([OpenAI long-running work](https://learn.chatgpt.com/docs/long-running-work); [Linear agent interaction](https://linear.app/developers/agent-interaction); [Linear stop signal](https://linear.app/developers/agent-signals); [Apple progress indicators](https://developer.apple.com/design/human-interface-guidelines/progress-indicators)).

**Waldo adaptation:** Use plan-step completion or exact state, not invented percentages. Required run states are `queued`, `working`, `waiting for approval`, `waiting for user`, `paused`, `interrupted`, `failed`, `unknown outcome`, and `result ready`. A visible stop/pause action remains available wherever it is safe.

### 4. Approval is a scoped contract, not a chat sentiment

**Observed:** OpenAI and Claude separate sandbox/permission modes from the user's goal; Cursor distinguishes run modes and safety review; Windsurf uses allow/deny precedence ([OpenAI agent approvals](https://learn.chatgpt.com/docs/agent-approvals-security); [Claude permissions](https://code.claude.com/docs/en/permissions); [Cursor run modes](https://prod.cursor.com/docs/agent/security/run-modes); [Windsurf terminal](https://docs.windsurf.com/windsurf/terminal)).

**Waldo adaptation:** An approval card must name the action, target, data/tools, authority scope, one-time/session duration, external consequence, reversibility, and revoke route. `Return` approves and `Escape` declines only when the approval card owns focus; ordinary composer focus must never turn those keys into authority.

### 5. Results require evidence and a return path

**Observed:** Codex and Cursor review code as diffs; Claude exposes diffs, preview, DOM inspection, logs, and files; Linear agent activities stay attached to the issue and can include response/error/elicitation ([Codex app](https://openai.com/index/introducing-the-codex-app/); [Cursor diff review](https://docs.cursor.com/en/agent/review); [Claude Code Desktop](https://code.claude.com/docs/en/desktop); [Linear agent interaction](https://linear.app/developers/agent-interaction)).

**Waldo adaptation:** A result card needs artifacts, source/evidence links, verification attempted, what remains unknown, and the exact Home/Work object it returns to. Agent or provider completion can create `Result ready`; only the accepted verification lineage can create a verified outcome.

### 6. Multi-agent work stays subordinate to ownership

**Observed:** OpenAI exposes subagent threads and their returned summaries; Claude differentiates subagents, agent view, teams, and worktrees; Linear keeps the human assignee responsible while agents are delegates ([OpenAI subagents](https://learn.chatgpt.com/docs/agent-configuration/subagents); [Claude parallel agents](https://code.claude.com/docs/en/agents); [Linear delegation](https://linear.app/docs/assigning-issues)).

**Waldo adaptation:** The Home sidebar contains `Chat`, not a peer-agent roster. Explicitly created specialists belong under Work/Settings and appear in Activity as bounded delegated runs. Each shows purpose, permitted sources/tools, authority ceiling, budget, state, evidence, and return destination. Waldo coordinates; the user owns the responsibility.

### 7. Recovery must say what it can and cannot restore

**Observed:** Cursor checkpoints track agent changes but are not Git; Claude checkpoints restore tracked local edits but not external side effects; Linear supports undo/redo and returns the user to the affected selection; ChatGPT desktop restores drafts and documents troubleshooting for missing chats and environment mismatch ([Cursor checkpoints](https://docs.cursor.com/en/agent/chat/checkpoints); [Claude checkpointing](https://code.claude.com/docs/en/checkpointing); [Claude recovery boundary](https://code.claude.com/docs/en/how-claude-code-works); [Linear undo](https://linear.app/changelog/undo-actions); [ChatGPT troubleshooting](https://learn.chatgpt.com/docs/reference/troubleshooting)).

**Waldo adaptation:** Distinguish `undo UI/local proposal`, `restore local checkpoint`, `cancel pending effect`, and `compensate external effect`. Never label all four “Undo.” Restore route, selection, scroll, draft, and focus when a rail, modal, or detail layer closes.

## Dayflow-like Insights for Waldo

### Observed Dayflow mechanism

Dayflow converts periodic screen evidence into bounded timeline cards rather than merely reporting active-app duration. Its current product presents Timeline, Daily summary, Weekly summary, and grounded Chat; the official repository describes local-first storage and user-selected local or provider-backed analysis ([product](https://www.dayflow.so/); [README](https://github.com/JerryZLiu/Dayflow/blob/86f5288d7267c9c8da1ab47e9c25ad1cd0b80046/README.md)).

At the pinned revision, card generation can revise a recent connected time range rather than treating the first inference as immutable. Users can edit card titles/categories or delete cards ([generation path](https://github.com/JerryZLiu/Dayflow/blob/86f5288d7267c9c8da1ab47e9c25ad1cd0b80046/Dayflow/Dayflow/Core/AI/LLMService.swift#L834-L959); [card editing](https://github.com/JerryZLiu/Dayflow/blob/86f5288d7267c9c8da1ab47e9c25ad1cd0b80046/Dayflow/Dayflow/Core/Recording/StorageManager%2BTimelineCards.swift#L200-L227)). Processing failures produce distinct retryable/failure states rather than fabricated continuity ([failure handling](https://github.com/JerryZLiu/Dayflow/blob/86f5288d7267c9c8da1ab47e9c25ad1cd0b80046/Dayflow/Dayflow/Core/AI/LLMService.swift#L995-L1079)).

Dayflow's privacy policy distinguishes local processing, direct BYO-provider processing, and opt-in Dayflow Cloud. It says screenshots remain local in local mode, go directly to the configured provider in BYO mode, and are removed after cloud processing while derived Activity Data is retained for sync in cloud mode. It also offers entry deletion, account deletion, export, mode switching, and analytics opt-out ([privacy policy](https://www.dayflow.so/privacy/)).

### Waldo Insight object and reading path

**Inference / proposed adaptation:**

```text
Noticed observation
  -> source window and capture coverage
  -> bounded interpretation candidate
  -> why it may matter now
  -> Confirm / Correct / Dismiss / Why this?
  -> optional Open Loop, Memory proposal, or bounded experiment
```

An Insight card should show:

- exact observed period and source types;
- source freshness and any capture/processing gaps;
- direct observation separately from Waldo's interpretation;
- provider/route used for synthesis when external processing occurred;
- `Noticed`, `Candidate`, `Confirmed`, `Corrected`, or `Dismissed` state;
- the user correction or counter-evidence when applicable;
- source drill-down without forcing raw screenshots into the default Home view;
- promotion destination and consequence before creating an Open Loop, Memory proposal, or Work item.

### Reject for Insights

- no personality, mood, health, motivation, or commitment inference from screen time;
- no productivity score, streak, shame framing, or opaque confidence ordering;
- no statement that an unobserved interval means nothing happened;
- no silent fallback to a capture/model provider that lacks source-specific authorization;
- no captured activity promoted directly into responsibility, admitted memory, evidence acceptance, or outcome verification;
- no “private/local” shorthand without separately disclosing raw storage, model processing, derived-data retention, and deletion coverage.

## Premium macOS interaction contract for Kennel

This is the minimum interaction behavior to test, not a visual style prescription.

### Window and responsive layout

Apple recommends letting people resize, hide, show, and move windows, using large displays to reduce unnecessary modality, and allowing sidebars to collapse as a window narrows ([Designing for macOS](https://developer.apple.com/design/human-interface-guidelines/designing-for-macos/); [Sidebars](https://developer.apple.com/design/human-interface-guidelines/sidebars)).

Waldo acceptance:

- wide: sidebar + primary content + one contextual inspector/rail; never two competing right rails;
- medium: collapse optional detail before shrinking primary content below a readable measure;
- narrow: one content layer at a time with a labelled Back action; do not squeeze three columns;
- resizing is live and stable: no clipped headers, horizontal page overflow, jumping composer, or detached focus ring;
- each pane owns its vertical scroll; same-axis nested scroll regions are avoided because Apple warns they are unpredictable ([Scroll views](https://developer.apple.com/design/human-interface-guidelines/scroll-views));
- toolbar actions overflow or collapse predictably as width shrinks; Apple describes system-managed toolbar overflow for constrained space ([Toolbars](https://developer.apple.com/design/human-interface-guidelines/toolbars)).

### Mouse and trackpad

Apple documents primary click for selection/activation, secondary click for contextual menus, two-axis scrolling, smart zoom where appropriate, and swipe-between-pages for page navigation. It advises against redefining systemwide gestures and recommends consistent behavior across pointer, keyboard, and assistive input ([Pointing devices](https://developer.apple.com/design/human-interface-guidelines/pointing-devices)).

Waldo acceptance:

- primary click selects or activates; selection and activation are not ambiguously combined on dense rows;
- secondary click on a topic, card, run, evidence item, or context chip opens item-specific actions near the pointer;
- Back/Forward mouse buttons and `⌘[` / `⌘]` traverse genuine navigation history, never destructive state transitions;
- horizontal trackpad scrolling works only where horizontal content is intentional; it must not drag the entire app sideways;
- hover may reveal secondary controls, but critical state/action is never hover-only;
- resize dividers use the standard resize cursor and generous invisible hit area;
- drag-and-drop supports explicit attachments and reordering where meaningful, with a visible drop target and an accessible menu/keyboard alternative. Apple recommends drag-and-drop alternatives and undo where feasible ([Drag and drop](https://developer.apple.com/design/human-interface-guidelines/drag-and-drop)).

### Menus, keyboard, focus, and selection

Apple says macOS apps should expose commands through the menu bar, respect standard shortcuts, and support keyboard-only work. Context menus should contain short item-specific actions while the main menu retains the app's complete command set ([Designing for macOS](https://developer.apple.com/design/human-interface-guidelines/designing-for-macos/); [Keyboards](https://developer.apple.com/design/human-interface-guidelines/keyboards); [Context menus](https://developer.apple.com/design/human-interface-guidelines/context-menus)). Linear and Raycast show that a discoverable command/action menu can teach shortcuts without removing mouse access ([Linear contextual menus](https://linear.app/now/invisible-details); [Raycast Action Panel](https://manual.raycast.com/search-bar)).

Waldo acceptance:

- `⌘K` opens a searchable command palette; commands remain available in standard app menus;
- `⌘,` opens Settings, `⌘F` searches the current collection, `⌘N` starts the appropriate new episode/window, and `Escape` dismisses the topmost dismissible layer;
- all custom shortcuts appear in one shortcut browser and avoid system/app collisions;
- selected, focused, active, running, and hovered are visually distinct states;
- opening and closing Waldo returns focus to the launcher or prior valid element;
- multi-selection, if introduced, supports Shift-click plus keyboard selection and a visible bulk-action state;
- destructive actions support `⌘Z` where locally reversible and otherwise name the confirmation/compensation path.

### Accessibility and motion

Apple's current accessibility guidance lists 28×28 pt as the default macOS control size and 20×20 pt as the minimum; it also requires adequate spacing, gesture alternatives, meaningful labels, VoiceOver/Full Keyboard Access testing, and keyboard-only navigation. Color or animation alone cannot carry state ([Accessibility](https://developer.apple.com/design/human-interface-guidelines/accessibility)).

Apple recommends purposeful, brief, interruptible motion and responding to Reduce Motion by reducing zoom, scale, peripheral, depth, and blur transitions ([Motion](https://developer.apple.com/design/human-interface-guidelines/motion); [Accessibility motion guidance](https://developer.apple.com/design/human-interface-guidelines/accessibility)).

Waldo acceptance:

- every icon-only control has an accessible name and help tooltip;
- custom controls meet at least the 20×20 pt macOS minimum hit region, with 28×28 pt preferred;
- the complete core flow works with Full Keyboard Access and VoiceOver;
- focus order follows visual/task order after responsive recomposition;
- semantic status includes text/icon, not color alone;
- Reduce Motion replaces rail/overlay translation and scale with a short fade or immediate state change;
- animations never block interruption, approval, Back, or closing;
- dynamic text/type scaling does not clip cards, tabs, badges, or approval actions.

### Loading, progress, failure, and recovery

Apple recommends showing content promptly, keeping unrelated work usable while loading, preferring accurate determinate progress when duration is known, keeping indeterminate indicators moving, and offering cancellation where safe ([Loading](https://developer.apple.com/design/human-interface-guidelines/loading); [Progress indicators](https://developer.apple.com/design/human-interface-guidelines/progress-indicators)).

Waldo acceptance:

- no blank surface while a route, conversation, or run loads;
- skeleton/placeholder content never resembles a plausible fact, message, run, or Insight;
- stalled work becomes a named blocked/error/unknown state with Retry, Inspect, Change connection, or Stop as applicable;
- retry does not duplicate external effects;
- offline/unconfigured/provider-mismatch states preserve local Home and responsibility views;
- drafts, route, selection, scroll, and focus survive non-destructive error recovery.

## Adopt / Adapt / Reject synthesis

### Adopt

- one quick global invocation plus a full durable conversation surface;
- explicit, detachable context chips and drag/drop attachment;
- short-first Conversation with inspectable Activity detail;
- pause, resume, stop, queued redirect, and attention-needed states;
- approval cards with target, scope, consequence, duration, and revoke;
- artifact/evidence review beside conversation, with source return paths;
- subordinate specialist runs under one Waldo relationship;
- human responsibility retained after delegation;
- checkpoint/undo with explicit external-effect limits;
- command palette, standard app menus, right-click context menus, Back/Forward, keyboard parity, focus restoration, and pane resizing;
- Dayflow-like temporal synthesis with source coverage, correction, pause/exclusion, and provider disclosure.

### Adapt

- ChatGPT/Codex projects and Claude sessions -> Waldo topic episodes plus native Home/Work object links;
- agent Attention queues -> only `Needs you`, `Running`, `Result ready`, `Blocked`, or `Unknown`, not every event;
- diff/artifact review -> any inspectable artifact, evidence item, or verified-result check;
- Quick AI/Chat handoff -> Quick Capture for recording a thought and Home Chat for reasoning, without duplicate stores;
- Cursor/Claude permission modes -> source/action-specific policy and bounded approvals, not one opaque autonomy slider;
- Linear agents -> explicit specialist profiles and bounded task delegation while the person remains responsible;
- Dayflow Daily/Weekly -> Insights and source-backed re-entry, not surveillance analytics or a performance score.

### Reject

- an unbounded transcript as the only source of progress or evidence;
- a Home sidebar of peer provider/agent personas competing with Waldo;
- model/provider selection as the main information architecture;
- silent context attachment, silent provider fallback, silent action authority, or silent memory admission;
- invented progress percentages, generic “working” spinners, and completion-as-outcome;
- gesture-only, hover-only, color-only, or pointer-only actions;
- fixed desktop widths that clip or squeeze at narrow sizes;
- nested same-direction scroll traps and custom behavior that overrides macOS navigation gestures;
- activity/time data presented as intention, personality, commitment, health, or verified accomplishment;
- plausible populated content without an explicit preview seam or native evidence.

## Recommended product-surface completion sequence

### P0 — macOS interaction and responsive acceptance harness

Before adding more destinations, define and automate a resize/interaction matrix for Today, Chat, Open Loops, Insights, Work, Waldo Conversation, Waldo Activity, approvals, and run detail. Test pointer, trackpad, keyboard, Full Keyboard Access, focus restoration, internal scroll, text scaling, and Reduce Motion in the actual Electron app.

### P1 — navigation and interaction grammar

Add the shared command palette/menu commands, Back/Forward history, secondary-click menus, pane resizing/collapse, stable scroll ownership, hover/focus/selection states, shortcut browser, and reliable draft/focus restoration. These utilities should be shared shell behaviors, not rewritten per route.

### P2 — dedicated Home Chat and Waldo handoff

Make Home Chat a full reading destination for the same Waldo topics used by the rail. Support quick invoke, detachable context, topic creation/re-entry, short-first messages, queued redirects, and side-by-side artifact/detail presentation without creating a second Waldo store.

### P3 — truthful preview Insights

Replace History with Insights using synthetic, explicitly preview-only observation/source/correction states. Preserve continuity records beneath each candidate. Prove gaps, unknowns, provider disclosure, correction, dismissal, and no-capture empty states before adding real collection.

### P4 — bounded specialists and Activity

Add explicit specialist profile preview/configuration in Work or Settings, with purpose, sources/tools, authority, budget, pause/revoke, and return path. Activity must show subordinate runs and attention states while Home remains Waldo-first.

### P5 — backend and harness contracts only after surface acceptance

Define daemon-owned conversation, context, Insight, run, approval, evidence, and verification contracts. Add secure provider/runtime connection and Keychain custody. Do not wire a token directly to the renderer or treat provider completion as acceptance.

## Acceptance matrix for the actual Electron app

| Dimension | Required scenarios |
| --- | --- |
| Width | minimum supported window; narrow single-pane; medium collapsed detail; wide three-region; ultrawide readable measure |
| Height | short laptop window; standard; tall; composer and primary actions always reachable |
| Density | empty, one item, many items, long title, long code/source path, expanded run, approval modal |
| Pointer | primary click, double click only where conventional, secondary click, hover reveal, divider drag, drag/drop, mouse Back/Forward |
| Trackpad | momentum vertical scroll, intentional horizontal scroll, page Back/Forward, pinch only where zoom is meaningful |
| Keyboard | Tab/Shift-Tab, arrows, Return, Space, Escape, command palette, app menus, standard shortcuts, approval focus boundary |
| Accessibility | VoiceOver labels/order, Full Keyboard Access, non-color status, 20 pt minimum hit regions, type scaling |
| Motion | standard and Reduce Motion; transitions interruptible; no information conveyed only by animation |
| Recovery | close/reopen rail, route Back, retry, offline, provider unavailable, daemon mismatch, interrupted run, recover draft/checkpoint |
| Truth | canonical unconfigured state; preview-only populated state; gaps, unknown outcome, no silent responsibility/outcome creation |

## Material unknowns

- Public product docs do not establish complete VoiceOver reading order, Full Keyboard Access coverage, contrast conformance, reduced-motion behavior, minimum supported widths, exact pane-collapse rules, scroll chaining, or hit-target compliance for the benchmark products. Waldo must specify and test these itself.
- No inspected product establishes Waldo's required semantic lineage from observation to responsibility to execution to evidence to verification to acceptance.
- Dayflow's real-world synthesis accuracy and deletion behavior across every external provider are not proven by its UI or repository.
- The exact installed Grok Bot and Hermes packages are now resolved and locally inspected, but their live-run, complete accessibility, minimum-width, reduced-motion, and recovery behavior remains unverified under the read-only audit boundary.
- SOTA here means a current first-party pattern benchmark, not a completeness ranking, endorsement, or evidence that every documented feature ships reliably to every account.
