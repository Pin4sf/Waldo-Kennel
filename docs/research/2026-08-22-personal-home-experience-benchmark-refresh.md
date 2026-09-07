# Personal Home experience benchmark refresh

**Date:** 2026-08-22
**Status:** product and interaction research only; no implementation authority
**Scope:** Dimension, Omi, Dayflow, Granola, ChatGPT mode switching, and Project Minimi as references for Waldo Kennel Personal Home
**Question:** What should Waldo borrow to help a person orient, regain context, capture deliberately, inspect or correct evidence, carry a responsibility into Work, and consciously close or re-enter it?

## Executive finding

The benchmark does not support a generic card dashboard, an ambient transcript catalogue, or six equally weighted Home destinations. The coherent shared pattern is:

```text
orient -> review a bounded change -> inspect source -> decide/correct
       -> continue personally or hand off to Work -> consciously close -> re-enter later
```

- **Dimension** is the strongest full-flow command-center reference: Morning Briefing, finite Catch Up, deliberate suggested actions, and Evening Recap. It is historical/deprecated and work-tool-centric, so it is a structure reference rather than a product to copy.
- **Omi** now provides two different lessons: its mobile product gathers capture, conversations, tasks, memories, and chat; its current macOS source makes Home a chat/search surface with flat top pills and persistent capture controls. The latter is useful evidence for capture placement and provenance routing, but not a reason to make Waldo Home a transcript or universal search box.
- **Dayflow** is the strongest inspectable temporal-evidence reference. Its timeline is useful for re-entry and source drill-down, not as Home's canonical object.
- **Granola** shows calm, calendar-led orientation and unusually legible source-linked answers. Its meeting scope is deliberately narrower than Waldo's responsibility scope.
- **ChatGPT's Chat / Work control** is the closest first-party reference for a compact horizontal mode switch. It changes the job and tool/permission envelope while keeping Chat and Work in one recent-history system. Codex remains separate. Waldo should adapt the compactness, not inherit ChatGPT's custody boundaries or names.
- **Project Minimi (`minimi`)** is a product-mechanism reference, not a visual Home reference: quiet capture, selective resurfacing, contextual recall, and open-loop help inside another LLM. Its automatic closure and “never misses” claims are not a safe authority model.

**Waldo synthesis:** use one persistent `Home | Work` mode pill in global shell chrome; let the surrounding navigation become contextual to the selected mode. Keep one global capture-status/shortcut affordance, but place the expanded Quick Capture input contextually on Home's orientation surface and open it on demand elsewhere. Home should lead with a finite brief and the smallest genuine `Needs You` queue, then reveal provenance, continuity, timeline, memory correction, and history through drill-down rather than stable peer dashboards.

## Evidence discipline and source routing

Source classes used here:

1. **Local repo research:** the source-pinned [Omi implementation reference](2026-08-21-omi-implementation-reference.md), [Dayflow versus Omi capture reference](2026-08-21-dayflow-vs-omi-capture-reference.md), and [Project Minimi dossier](2026-08-21-project-minimi-reference.md).
2. **Official source code:** pinned public repositories for Omi and Dayflow.
3. **First-party product/docs:** official product sites, help centers, and OpenAI product documentation.
4. **User-provided visual evidence:** the earlier Dimension screenshot described in the prior benchmark; it supports that historical screen only, not current runtime behavior.

Labels:

- **Observed** — directly present in source code, official documentation, or the previously supplied screen.
- **Reported** — claimed by the product operator but not independently reproduced here.
- **Inference** — a bounded interpretation of observed or reported evidence.
- **Unknown** — not established by inspected first-party evidence.
- **Waldo recommendation** — a proposed transfer, not shipped Waldo behavior.

Public source code establishes implementation at a revision, not deployment, reliability, or production completeness. Official marketing establishes the operator's claim, not accuracy. No private artifacts, account data, personal captures, or privacy-invasive retrieval were used.

## Identity resolution and currency

### Dimension, not “Dimensions”

**Observed — high confidence:** the intended product is [Dimension](https://dimension.dev/) (singular). The exact labels in the prior user-supplied screen — `Morning Briefing`, `Catch Up`, and the finite briefing flow — match Dimension's [first-party proactive-feature documentation](https://docs.dimension.dev/proactive-features). The docs call it an AI personal assistant and say Morning Briefing appears on Home.

**Conflict:** the official homepage says Dimension was winding down on 20 May 2026, while its [status page](https://status.dimension.dev/) currently reports fully operational and its docs still describe web availability. A status page is not proof that a product remains generally available.

**Conclusion:** use Dimension as a **historical/deprecated design reference**. Current access, current deployed layout, and post-wind-down support are **Unknown**.

### Project Minimi, not a route called `/minimi`

**Observed — high confidence:** the intended product is [Project Minimi](https://www.projectminimi.com/), styled `minimi`, operated by Shram Intelligence, Inc. Its [About page](https://www.projectminimi.com/about) says it began as an ambient-memory project named `mini-me`. The slash in the prompt is treated as punctuation, not part of the product identity.

**Observed:** no official public source repository is linked from the first-party pages. Its exact Home/tray UI is therefore not source-verifiable in the same way as Omi or Dayflow.

**Conclusion:** use Minimi for ambient-continuity and open-loop product mechanisms, not as visual evidence for a dashboard layout.

## Benchmark matrix

| Product | Dominant user job and full flow | Information architecture and mode switching | Capture placement | Correction, provenance, continuity | Confidence and limit | Waldo decision |
| --- | --- | --- | --- | --- | --- | --- |
| **Dimension** | Morning Briefing -> Catch Up across email/Slack/meetings -> add/dismiss/assign suggested todo -> Evening Recap. [Official docs](https://docs.dimension.dev/proactive-features) say the brief includes schedule and attention, Catch Up provides editable drafts, and recap covers happened/done/left. | Historical supplied screen used a narrow app rail and a two-column brief + one-at-a-time review. Background agents are individually toggled. | Input is subordinate to briefing/review; the historical screen did not make a repeated capture composer the identity of every area. | Original messages appeared under summaries in the supplied screen. Suggestions require add/dismiss/assign rather than silent promotion. Durable claim-level correction and the effect of `Finish Morning Briefing` are Unknown. | **High** for named flow and historical screenshot; **low** for current deployment. Work-tool-centric, not whole-life. | **Adapt** finite orientation -> review -> action -> recap. **Reject** inbox-zero authority, brand/visual copying, and treating suggested work as confirmed responsibility. |
| **Omi mobile** | Capture -> Conversation -> transcript/summary/key points/tasks/memories -> search/chat. Official help describes Home as the start point for conversations, recording, and AI chat, with Tasks, Memories, Apps, and Settings as main areas ([usage guide](https://help.omi.me/en/articles/13153540-using-the-omi-app); [conversation/memory guide](https://help.omi.me/en/articles/13153612-conversations-memories-and-chats)). | Mobile top-level areas gather capture-derived objects by type. The source-pinned Home additionally composes Today tasks, recaps, recent conversations, and memory/mind-map material; see the [pinned local dossier](2026-08-21-omi-implementation-reference.md). | New recording is available from Home; mobile capture is prominent because capture is the product's intake substrate. | Official docs report editing/deleting items and searching history. Current source has candidate/review and authority mechanisms, but not every client path is proven converged. | **High** for documented mobile flow; client parity and production invariants remain Unknown. | **Adapt** visible capture entry, Today/recap, candidate review, and source-aware chat. **Reject** object-type Home, inferred-memory graph dominance, and checkbox completion as conscious closure. |
| **Omi macOS at `36eec2a19b26`** | Current source makes Home a search/chat surface: search narrows one chronological spine; a question uses the same chat transcript/composer; citations route to conversations, memories, tasks, goals, screenshots, or web sources. See [`QueryShellHome.swift`](https://github.com/BasedHardware/omi/blob/36eec2a19b269836de4e389d07b24d2c194fb297/desktop/macos/Desktop/Sources/MainWindow/QueryShell/QueryShellHome.swift#L1-L29), its [single surface/composer](https://github.com/BasedHardware/omi/blob/36eec2a19b269836de4e389d07b24d2c194fb297/desktop/macos/Desktop/Sources/MainWindow/QueryShell/QueryShellHome.swift#L60-L168), and [typed citation routing](https://github.com/BasedHardware/omi/blob/36eec2a19b269836de4e389d07b24d2c194fb297/desktop/macos/Desktop/Sources/MainWindow/QueryShell/QueryShellHome.swift#L449-L533). | A flat top bar exposes `Home`, `Library`, `Tasks`, `Rewind`, and `Apps`, then audio, screen capture, and Settings; narrow layouts collapse only navigation into a menu. See [`DesktopTopBar.swift`](https://github.com/BasedHardware/omi/blob/36eec2a19b269836de4e389d07b24d2c194fb297/desktop/macos/Desktop/Sources/MainWindow/DesktopTopBar.swift#L4-L24) and [layout](https://github.com/BasedHardware/omi/blob/36eec2a19b269836de4e389d07b24d2c194fb297/desktop/macos/Desktop/Sources/MainWindow/DesktopTopBar.swift#L66-L116). A cohort-gated chat-first shell reuses canonical owners rather than creating second stores ([source](https://github.com/BasedHardware/omi/blob/36eec2a19b269836de4e389d07b24d2c194fb297/desktop/macos/Desktop/Sources/MainWindow/ChatFirst/ChatFirstShell.swift#L5-L67)). | Persistent wordless audio/screen controls live in global top chrome, with state dots, off marks, tooltips, permission/repair actions, and a settings path. The full input lives in the Home search/chat surface, not in every destination. | Search results merge conversations, memories, tasks, and screen evidence chronologically; answer citations retain typed identity and open the owning detail surface. | **High** for source at current `main`; **medium** for shipped cohort because source includes a legacy preference and capability gate. Current `main` is 73 commits ahead of the earlier Omi dossier pin; `home_content.dart` was not among the changed files, so the prior mobile composition remains the best pinned mobile evidence. | **Adopt** persistent capture-state placement and typed citation handoff. **Adapt** one-composer/single-owner discipline. **Reject** five flat peer pills and chat/search as the whole identity of Waldo Home. |
| **Dayflow** | Automatic screen capture -> bounded activity cards -> Timeline -> Daily/Weekly review -> chat/export. The current official repository still points `main` to `86f5288d...`; its README describes timeline, daily standup, weekly review, and grounded chat ([repository](https://github.com/JerryZLiu/Dayflow/tree/86f5288d7267c9c8da1ab47e9c25ad1cd0b80046)). | Initial macOS destination is Timeline with a fixed left rail for Timeline, Daily, Weekly, Chat, and tools ([MainView](https://github.com/JerryZLiu/Dayflow/blob/86f5288d7267c9c8da1ab47e9c25ad1cd0b80046/Dayflow/Dayflow/Views/UI/MainView/MainView.swift); [Sidebar](https://github.com/JerryZLiu/Dayflow/blob/86f5288d7267c9c8da1ab47e9c25ad1cd0b80046/Dayflow/Dayflow/Views/UI/MainView/SidebarView.swift)). | Recording can be paused from menu bar/app; source has timed/indefinite pause and app exclusions. Capture state is persistent but does not consume the full content surface. | Cards are model-derived, editable/deletable temporal projections. Raw capture, gaps, batch failure, and provider route can be inspected; activity is not responsibility truth. See [local pinned capture dossier](2026-08-21-dayflow-vs-omi-capture-reference.md). | **High** for open-source shell and capture mechanics. Real-world accuracy and full deletion cascade remain Unknown. | **Adopt** quiet coverage/gap state and evidence timeline. **Adapt** timeline into source drill-down and re-entry. **Reject** timeline-first Home, productivity scoring, and activity-as-outcome. |
| **Granola** | Upcoming meeting -> user notes/capture -> AI-enhanced note -> source-linked chat -> follow-up/retrieval. The [setup guide](https://docs.granola.ai/help-center/getting-started/setting-up-granola-for-the-first-time) says the post-setup Home shows upcoming meetings and a demo note. | Conventional desktop Home and meeting corpus; folders/workspaces deepen only as the team context grows. It does not claim a whole-life Home. | Capture is meeting/audio-contextual. Quick note/meeting entry is available on Home, not repeated across every knowledge view. | Human notes steer the artifact. Granola Chat exposes inline citations and jump-to-source, and tells the user which notes it used ([official Chat update](https://www.granola.ai/blog/granola-chat-just-got-smarter)); shared-folder answers cite exact transcript lines ([Granola 2.0](https://www.granola.ai/blog/two-dot-zero)). | **High** for official desktop workflow and provenance. Narrow meeting scope limits transfer. | **Adopt** calm density, human steering, inline source affordances. **Adapt** calendar as supporting context. **Reject** calendar as the ontology of Personal Home and inferred personality as truth. |
| **ChatGPT Chat / Work / Codex** | `Chat` handles fast conversational help; `Work` handles longer multi-step work and finished deliverables; `Codex` handles software development. The desktop app uses a top-of-page `Chat | Work` toggle inside ChatGPT, while `ChatGPT | Codex` remains in the top-left menu ([official guide](https://help.openai.com/en/articles/20001275-chatgpt-work-and-codex)). | Chat and Work share Recents and Projects; users can filter them and begin either mode within project context. Codex remains a separate view with separate history. Personal/Business workspace switching is a different profile-menu control with isolated data ([workspace guide](https://help.openai.com/en/articles/8542216-managing-members-seat-types-and-roles-in-chatgpt-business)). | Voice is a contextual control for Work/Codex; local file/app access follows selected experience and permission. It is not a global text composer cloned into every screen. | The selected experience changes tool, execution, and permission expectations. Official docs do not establish a cross-mode responsibility object equivalent to Waldo's `Outcome`/`OpenLoop`. | **High** for current official documentation, updated 2026-08-21. Exact visual dimensions/animation are not specified in text docs. | **Adopt** compact horizontal mode clarity and mode-specific capability envelope. **Adapt** into one Waldo identity with linked Home/Work state. **Reject** using account/workspace isolation as the model and hiding a custody change behind the pill. |
| **Project Minimi (`minimi`)** | Quiet Mac presence -> ambient context -> detected commitment/open loop -> contextual retrieval in a chosen LLM over MCP -> claimed automatic resolution. See [homepage](https://www.projectminimi.com/) and [MCP page](https://www.projectminimi.com/mcp). | First-party evidence establishes tray-like presence and an external-LLM connector, not a rich Home IA. | Reported screen/browser/app and microphone/system-audio capture; persistent capture UI, source exclusions, gaps, and exact tray interaction remain Unknown. | The [privacy policy](https://www.projectminimi.com/privacy-policy) reports local durable memory but transient Gemini/OpenAI/Deepgram processing and disclosure of retrieved slices to the connected model. Public evidence does not establish citations, admission/correction lineage, or reversible closure. | **High** for identity and operator claims; **low** for exact UI and governance. No official public source repository was found. | **Adopt** selective present-tense help and quiet presence. **Adapt** open-loop detection into provenance-bearing candidates and `Ready to Close`. **Reject** automatic truth/closure, universal-recall claims, and “nothing uploaded” shorthand that hides transient processing. |

## End-to-end job comparison

| Waldo job | Strongest reference | Transferable mechanism | Boundary Waldo must add |
| --- | --- | --- | --- |
| Start the day and know what matters | Dimension; Granola | A short narrative brief plus schedule/context, not a grid of widgets | Only confirmed facts and explicitly marked projections; every derived claim has source/freshness |
| Regain context after absence | Dimension Catch Up; Dayflow timeline; Minimi recall | Finite changes since a checkpoint, with a temporal route into raw evidence | Acknowledged checkpoint, capture gaps, uncertainty, and no claim that missing capture means nothing happened |
| Decide what needs attention | Dimension suggested todos; Omi candidate queue | A bounded review queue with add/dismiss/assign/defer choices | Separate `AttentionItem` from confirmed `OpenLoop`; no opaque urgency score |
| Inspect or correct why something is here | Granola citations; Omi typed citation routing; Dayflow cards | Source jump, timestamp, original content, and correction entry point | Canonical provenance, supersession/correction history, current user statement outranking inference |
| Capture deliberately | Omi capture; Granola contextual note; Minimi quiet presence | One obvious entry plus always-understandable capture state | Capture creates intake/candidate context, never automatic memory, Open Loop, Outcome, evidence, or acceptance |
| Carry personal responsibility into execution | ChatGPT Chat/Work; Dimension assign-to-agent | Switch execution mode without rewriting the user's intent; preserve a shared context anchor | Explicit `ResponsibilityLink`, accepted contract/authority, and independent Home/Work lifecycle state |
| Close or release | Dimension Evening Recap; Minimi's open-loop premise | A finite end-of-day reconciliation and candidate `Ready to Close` | Only explicit `LoopDisposition`; disappearance, activity, model confidence, task check, or agent completion cannot close |
| Re-enter later | Dayflow Daily/Weekly; Omi/Granola source history | Date/source-backed history and query | History is a derived re-entry projection, not an immutable behavioral judgment or surveillance feed |

## Explicit decision: `Home | Work` horizontal mode pill

### Adopt

Use one compact, persistent two-option control in the app's global top chrome:

```text
             [ Home | Work ]
global shell ---------------- capture state / search / settings
context rail ---------------- selected mode's hierarchy only
content --------------------- current job and drill-down
```

The ChatGPT reference is exact at the interaction-model level: the top toggle changes the job from conversational help to multi-step deliverable work while preserving shared Recents and Projects. For Waldo:

- `Home` means personal orientation, continuity, capture review, correction, and conscious closure.
- `Work` means accepted Outcomes, bounded work, agent sessions, evidence, verification, and acceptance.
- The pill changes the **responsibility-space lens and capability envelope**, not user identity, owner, account, or canonical store.
- Each side preserves its last meaningful local context so switching back feels like re-entry, not a reset.
- Home-to-Work is an explicit action (`Continue in Work`, `Create draft Outcome`, or `Link to Outcome`) with a visible link; flipping the pill alone does not move or duplicate an item.
- The selected state needs label, shape, focus ring, keyboard access, and a reduced-motion-safe transition. Color alone is insufficient.

### Adapt

Unlike ChatGPT, Work needs a deeper project/outcome/session hierarchy. The horizontal pill should therefore control the global mode while the existing sidebar becomes contextual:

- In Home, the rail can remain minimal and show re-entry/history/settings only when useful.
- In Work, the rail may reveal Outcomes, projects, sessions, or tools.
- The rail must not repeat `Home` and `Work` as generic rows when the pill already owns that choice.

At narrow width, preserve a single compact labelled mode control or menu in the same top-chrome location; do not move the mode switch to a floating bottom dock.

### Reject

- Do not use the pill as an account/workspace switch; ChatGPT's workspace switch is intentionally elsewhere because it changes custody and isolation.
- Do not make the pill a visual-only filter over one undifferentiated list.
- Do not create two Waldos, two histories, or two intake engines.
- Do not silently transfer a personal item, memory, or responsibility when the user changes mode.
- Do not add Dimension/Omi-style peer route pills around it until the jobs prove they need stable top-level destinations.

## Explicit decision: contextual Quick Capture

The benchmark supports separating **capture availability** from **an expanded capture field**.

### Persistent global layer

Always keep these understandable in shell chrome or Kennel's compact presence:

- capture source state (`off`, `active`, `partial`, `blocked`, `paused`), with a label/tooltip and repair/revoke route;
- one shortcut/button to open Quick Capture;
- any live recording stop/pause control whose consequence cannot be safely hidden.

Omi's current macOS top bar is the strongest direct reference: audio and screen status controls remain visible while the expanded query/composer stays inside Home. Dayflow reinforces persistent pause/coverage legibility.

### Contextual expanded layer

Show the expanded Quick Capture input:

1. once on Home's default orientation/Today surface, after or beside the brief where the user naturally externalizes a thought;
2. in a global popover/Island/command interaction when invoked from any other Home or Work drill-down;
3. inline only when a detail has a meaningful target, such as `Correct this memory`, `Add context to this Open Loop`, or `Capture evidence for this Outcome`.

The contextual version must say where the input will go before submission and confirm what it became afterward: note/intake, candidate memory, candidate Open Loop, or draft Work Outcome. It must never silently choose among these based only on inferred content.

### Reject

- No identical Quick Capture composer repeated at the top of Today, Catch Up, Open Loops, Memory, Daily Close, and History.
- No always-open bottom composer that makes every Home screen feel like chat.
- No capture field inside evidence drill-down unless it is explicitly scoped to correction/context.
- No hidden recording state merely because a global capture shortcut exists.
- No automatic promotion from text/audio/screen capture to durable memory, responsibility, execution, evidence, or closure.

## Adopt / Adapt / Reject synthesis for Waldo

### Adopt

- Dimension's finite `brief -> catch up -> decision -> recap` rhythm.
- Granola's inline citations, source jump, and human steering.
- Dayflow's visible capture coverage, honest gaps, and temporal evidence drill-down.
- Omi's source-typed citation routing, single-owner/single-composer discipline, persistent capture controls, bounded candidate queues, and anti-resurrection mechanisms.
- ChatGPT's compact top-level mode switch between lightweight/personal help and longer execution work.
- Minimi's product principle that memory earns attention by helping with present or future responsibility.
- Honest empty states: “nothing needs you” is a successful Home state.

### Adapt

- `Morning Briefing` -> a short evidence-bounded Waldo orientation, with freshness and `Why this is here`.
- `Catch Up` -> changes since the user's last acknowledged checkpoint, not an inbox-zero demand.
- Suggested tasks/action items -> expiring `AttentionItem` or candidate Open Loop, never canonical responsibility by extraction.
- Timeline/recaps -> History and source detail for re-entry, not a behavioral scorecard.
- Chat/search -> a supporting recall and question surface, not the Home ontology.
- Home/Work mode switching -> one identity and canonical daemon, contextual navigation, explicit responsibility links, separate lifecycle semantics.
- Automatic closure -> provenance-bearing `Ready to Close`, requiring the user's explicit disposition.
- Local-first claims -> separately show durable custody, transient processor disclosure, and agent/model retrieval disclosure.

### Reject

- A widget/card dashboard as the default composition.
- Six peer Home routes merely because Today, Catch Up, Open Loops, Memory, Daily Close, History, and Activity all exist as concepts.
- A transcript feed, memory graph, calendar, task list, or productivity timeline as Home's dominant object.
- Duplicate capture composers across routes.
- Flat peer navigation pills that compete with the `Home | Work` mode decision.
- Provider/app logos as primary navigation or separate assistant identities.
- Opaque ranking, streaks, scores, personality judgments, or novelty-driven insights.
- Inferred capture, task completion, provider completion, commits, checks, or missing activity as proof of completion/acceptance/closure.
- Any UI-owned second store, second intake engine, second Waldo identity, or second lifecycle truth.

## Design implications to carry into concept work

Every full-flow concept should prove the same scenario:

1. Open Home after an absence and read a three-to-five-sentence evidence-bounded brief.
2. Review one Catch Up change with a partial-capture/gap state visible.
3. Open its source and distinguish user-stated fact from Waldo inference.
4. Correct the inference or confirm it as an Open Loop candidate.
5. Choose `Continue in Work`; see the proposed Outcome/authority boundary before accepting it.
6. Return Home using the persistent pill without losing the source/loop link.
7. Use Daily Close to consciously defer, transfer, release, or close; do not auto-close.
8. Re-enter the item from History with its source, correction, Work link, and disposition intact.

Concepts may differ spatially — command center, guided daily narrative, or focus/review workspace — but none should evade this complete loop, duplicate Quick Capture, or make the mode pill a decorative filter.

## Material unknowns

- The current deployed Dimension product and the persistence semantics of `Finish Morning Briefing` are Unknown.
- Omi's current macOS source is cohort/feature-gated; source presence is not proof every user sees the chat-first shell.
- The precision, correction burden, and closure behavior of Minimi's open-loop detection are not publicly reproducible.
- Granola's current Home screenshot establishes meeting orientation, not a general personal-responsibility model.
- ChatGPT's official text documents the toggle and custody model but not exact animation, sizing, or narrow-window macOS behavior.
- No benchmark establishes that an ambient agent can safely create or close Waldo responsibility without explicit confirmation.

These unknowns do not block concept design. They do block claims of parity, automatic truth, or automatic closure.
