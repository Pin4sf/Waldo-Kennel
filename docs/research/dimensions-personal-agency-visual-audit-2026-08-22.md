# Dimension personal-agency visual audit

**Date:** 2026-08-22
**Source class:** public web, official primary source
**Scope:** every public route in Dimension's official marketing sitemap, public same-origin navigation, rendered UI media, and the first-party privacy statement where it constrains the marketed personal-assistant model
**Purpose:** extract reusable product jobs and screen mechanics for Waldo Personal Home without copying Dimension's brand, treating marketing as shipped truth, or weakening Waldo's provenance, confirmation, privacy, and Home-to-Work authority boundaries

## Executive conclusion

Dimension's strongest idea is not its blue split-pane visual treatment. It is a finite daily service rhythm:

1. orient the user with a Morning Briefing;
2. let them process a bounded Catch Up queue;
3. surface candidate actions rather than requiring manual extraction;
4. let the user hand selected work to an agent;
5. manage messages and draft replies;
6. restore context before meetings; and
7. reconcile the day in a Daily Recap / Evening Briefing.

Those seven jobs are consistently marketed on the [homepage](https://dimension.dev/) and then retold for agencies, engineers, founders, PMs, sales, and VCs. The role pages change the evidence and vocabulary, not the underlying interaction architecture. That is the most useful finding for Waldo: **build one adaptive personal-agency system whose projection changes with the user's current responsibilities and moment, rather than separate static dashboards for each persona or time of day.**

Dimension is also a cautionary reference. Its current public site states that it was winding down on 20 May 2026 ([notice](https://dimension.dev/winding-down)); its visuals are marketing demonstrations, not independently verified runtime behavior. Several screens present model summaries, urgency labels, personality-like claims, inferred completion, and outward actions without visible provenance or authority contracts. Waldo should adopt the rhythm, adapt the review mechanics, and reject those epistemic and privacy shortcuts.

## Method and evidence boundary

- Read [`robots.txt`](https://dimension.dev/robots.txt) first. It allows the public site but disallows authenticated/product routes including `/chat`, `/workflows`, `/skills`, `/integrations`, `/marketplace`, `/settings`, and `/sso`. Those routes were not crawled.
- Read the official [`sitemap.xml`](https://dimension.dev/sitemap.xml) and its [`sitemap-0.xml`](https://dimension.dev/sitemap-0.xml). It enumerates 25 URLs.
- Rendered all 25 URLs at a 1440×1000 desktop viewport because their server HTML is an empty Next.js application shell.
- Captured one full-page PNG for every sitemap URL.
- Inventoried rendered `<img>`, `<video>`, `<source>`, computed-background, and loaded first-party resources. The rendered public pages exposed 103 unique media assets. Their loaded marketing bundles and metadata referenced 40 additional source-only media-like paths: 38 accessible files and two dead paths.
- Downloaded the complete accessible union: 133 original first-party images and eight original first-party MP4s (141 files).
- Extracted contact sheets from all eight videos to inspect state changes rather than judging a single poster frame.
- Compared internal same-origin links against the sitemap. The only additional linked route was `/sso`, which `robots.txt` disallows, so it was not accessed.
- Distinguished visible product UI from decorative marketing backgrounds and treated repeated audience variants as a family while preserving every original file.

All behavior below is either **Observed in official marketing** or explicitly marked **Inference**. It is not evidence that the discontinued product performed the depicted action reliably.

## Page coverage

All 25 sitemap URLs returned HTTP 200 and rendered.

| Route | Page purpose | Coverage result |
|---|---|---|
| [`/`](https://dimension.dev/) | Product overview and seven-part daily loop | Unique; primary source for videos |
| [`/agency`](https://dimension.dev/agency) | Agency role narrative | Unique role variant |
| [`/agent`](https://dimension.dev/agent) | Cross-app action-taking agent | Exact rendered-content duplicate of `/feature/agent` |
| [`/docs`](https://dimension.dev/docs) | Document artifact generation | Unique and more complete than `/documents` |
| [`/documents`](https://dimension.dev/documents) | Older/incomplete document page | Exact rendered-content duplicate of `/feature/documents`; contains placeholder copy and unrelated briefing visuals |
| [`/email`](https://dimension.dev/email) | Managed inbox, drafting, Catch Up | Exact rendered-content duplicate of `/feature/email` |
| [`/engineers`](https://dimension.dev/engineers) | Engineering role narrative | Unique role variant |
| [`/feature/agent`](https://dimension.dev/feature/agent) | Alias of agent page | Duplicate |
| [`/feature/documents`](https://dimension.dev/feature/documents) | Alias of incomplete documents page | Duplicate |
| [`/feature/email`](https://dimension.dev/feature/email) | Alias of email page | Duplicate |
| [`/feature/morning-briefing`](https://dimension.dev/feature/morning-briefing) | Alias of Morning Briefing page | Duplicate |
| [`/feature/search`](https://dimension.dev/feature/search) | Alias of search page | Duplicate |
| [`/founder`](https://dimension.dev/founder) | Founder role narrative | Unique role variant |
| [`/morning-briefing`](https://dimension.dev/morning-briefing) | Overnight orientation, Catch Up, suggestions, day overview | Exact rendered-content duplicate of feature alias |
| [`/pm`](https://dimension.dev/pm) | Product-management role narrative | Unique role variant |
| [`/pricing`](https://dimension.dev/pricing) | Plans and capability packaging | Unique; repeats morning/evening brief, inbox, meeting, agent claims |
| [`/privacy-policy`](https://dimension.dev/privacy-policy) | Data, inference, context graph, integration retention | Unique legal/policy evidence |
| [`/sales`](https://dimension.dev/sales) | Sales role narrative | Unique role variant |
| [`/search`](https://dimension.dev/search) | Cross-integration search and file mentions | Exact rendered-content duplicate of feature alias |
| [`/sheets`](https://dimension.dev/sheets) | Spreadsheet artifact generation | Unique |
| [`/slides`](https://dimension.dev/slides) | Presentation artifact generation | Unique |
| [`/terms-of-service`](https://dimension.dev/terms-of-service) | Legal terms | Covered; no marketed UI media |
| [`/todo`](https://dimension.dev/todo) | Suggested actions and agent delegation | Unique |
| [`/vc`](https://dimension.dev/vc) | Venture-capital role narrative | Unique role variant |
| [`/winding-down`](https://dimension.dev/winding-down) | Product closure notice | Unique; no UI media |

### Coverage conflicts and gaps

- The current site is an archive-like marketing surface for a product that says it has wound down. Current availability, reliability, persistence semantics, and external-effect behavior are **Unknown**.
- `/documents` and `/feature/documents` advertise documents but render “Lorem ipsum” plus Morning Briefing, Catch Up, Suggestions, and Overview images. Use `/docs` as the stronger artifact-generation reference; treat the older pages as a publishing defect, not product evidence.
- `/agent`, `/documents`, `/email`, `/morning-briefing`, and `/search` each have an exact `/feature/*` duplicate.
- No public page markets a distinct **Afternoon Brief**. Adding one to Waldo is a product synthesis, not a Dimension behavior.
- Videos have no declared poster URLs. The saved contact sheets are local analysis derivatives, not first-party assets.
- Eight responsive product images and nine persona artifact examples were present only in loaded page source/bundles at the audited desktop viewport. They were downloaded and are inventoried separately below; their intended job is inferred from their filenames and the owning page's marketed copy, not from visible desktop placement.
- Two source references were unusable: `/seo/favicon/site.webm` and `/videos/new-landing/managed_inbox.mp4` returned HTTP 404. The live homepage uses the accessible hyphenated `/videos/new-landing/managed-inbox.mp4`.
- Authenticated routes were neither accessed nor inferred from. No defense, auth wall, or disallow rule was bypassed.

## The seven daily jobs

### 1. Morning Briefing — orient, do not browse

**Observed, high confidence.** The homepage says Dimension collates overnight updates from connected integrations into one briefing. The dedicated [Morning Briefing page](https://dimension.dev/morning-briefing) markets overnight email summaries, surfaced action items, and a mapped day on one screen. The primary screenshot combines:

- a greeting and schedule load;
- a short narrative across people, accounts, messages, and project status;
- confirmed-looking to-dos;
- separate suggestions; and
- a right-hand Catch Up review pane.

The first homepage video shows a narrow status capsule moving from “Catching up on email...” to “Reviewing Updates...”. The second progressively reveals the briefing rather than flashing a dense dashboard.

**Exact marketed job:** wake up with an interpreted starting point across apps, not open each app and manually reconstruct the day.

**Waldo:** **Adapt.** Preserve a short, calm orientation narrative and separate candidate actions from confirmed responsibilities. Add freshness, capture gaps, “Why this is here,” and source inspection. Do not equate message or meeting counts with importance.

### 2. Catch Up — finish a bounded review

**Observed, high confidence.** The homepage and [email page](https://dimension.dev/email) market one queue across Gmail and Slack, summarized messages, draft replies “in your tone,” review/tweak/send, explicit item counts, previous/next navigation, a source-app link, and an “all done” terminal state.

The video shows a source toggle, one item at a time, conversation summary above the original message, an editable draft, a send affordance, and completion after two examples. Role variants add Google Meet as a source for meeting preparation.

**Exact marketed job:** regain control after absence and clear only the communications that matter without visiting each inbox.

**Waldo:** **Adapt.** The finite queue and source-preserving card are strong. Replace “inbox zero” with “review the meaningful changes since your checkpoint.” A reply remains a proposal until edited/confirmed; outward send requires explicit authority. Always retain the original source and capture time beside the summary.

### 3. Action Plan / Todo — detect candidate responsibility

**Observed, high confidence.** The homepage says Dimension detects action items buried across apps and suggests them as tasks. The [Todo page](https://dimension.dev/todo) says it checks email, calendar, tickets, and threads; separates tasks from Suggestions; exposes suggestion detail on hover; and lets a user add, dismiss, write, check, or delegate items.

**Exact marketed job:** convert scattered action language into a reviewable list, then choose whether the person or agent should perform it.

**Waldo:** **Adapt heavily.** Preserve the candidate-versus-confirmed separation. Call the pre-confirmation object an `OpenLoopCandidate` or `AttentionItem`, not a task. Show the user statement and evidence. Confirmation creates responsibility; rejection/correction must train the local interpretation. Checking a box or detecting activity must not establish outcome truth.

### 4. Deep Work / Agent — move from intent to execution

**Observed, high confidence.** The [Agent page](https://dimension.dev/agent) markets emails, tickets, documents, scheduled/triggered workflows, cross-device continuity, and completing assigned todos. Its visuals show:

- the same conversation continuing on desktop, Slack, and iMessage;
- a workflow represented as `When` plus `Plan` across Gmail, Linear, and Slack;
- a task list with completed, in-progress, and agent-owned rows; and
- an execution trace that searches GitHub, Linear, and Slack before acting.

**Exact marketed job:** give an assistant a real work request once, let it gather context and operate across tools, and receive the result where the user already is.

**Waldo:** **Adapt.** Keep one Waldo across Home and Work, with explicit `ResponsibilityLink` and inspectable execution trace. Home preserves user intent and authority; Work owns agent execution. Require confirmation for externally consequential actions and never let provider completion silently mark the user's Outcome complete.

### 5. Inbox — classify and draft before the user arrives

**Observed, high confidence.** The [email page](https://dimension.dev/email) markets auto-labeling, auto-drafting, and Catch Up. Images and video depict numeric Gmail labels such as urgent, action needed, follow-up, awaiting reply, meeting, FYI, done, payment, newsletter, and marketing; a response drafted in the composer; attachments gathered; and the user pressing Send.

**Exact marketed job:** pre-sort incoming communication and prepare safe next actions so the user reviews instead of starting from zero.

**Waldo:** **Adapt cautiously.** Drafting and batching are useful. Reject opaque urgency and “done” classification as truth. Surface why an item was classified, let the user correct the category, and learn from explicit correction. Default to drafts; sending is a separately authorized effect.

### 6. Meeting Prep — enter with reconstructed context

**Observed, high confidence.** The homepage says Dimension briefs attendees, talking points, and past context. Role pages describe recent threads, open tasks/status, prior conversations, and overlooked context. Visuals show a meeting list, join button, short summary, expandable attendee card, and sections such as “what's on her mind,” “worth bringing up,” “heads up,” open questions, contract history, and prior support issues.

**Exact marketed job:** restore the relevant relationship and responsibility context immediately before a conversation.

**Waldo:** **Adapt with an epistemic redesign.** Facts, direct statements, and Waldo proposals need visibly different treatment. Reject personality narration such as “she's not the type to complain” and prescriptive claims such as “don't leave this call without committing” unless explicitly labeled as uncertain advice. The best Waldo prep answers: what changed, what was promised, what remains open, and which source supports each claim.

### 7. Daily Recap / Evening Briefing — reconcile and carry forward

**Observed, high confidence.** The homepage markets a summary of what got done, what remains open, and tomorrow's horizon. Every role page places an end-of-day recap at 6 PM. The homepage video shows a to-do list updating during the day and an evening email summarizing a closed deal, finished tasks, unresolved work, a dinner booking, and next timing. The founder image says “Three calls, two things closed, one still open,” names the open promise, and ends with a forward-looking note.

**Exact marketed job:** stop mentally reconstructing the day, preserve what matters, and prevent open commitments from disappearing overnight.

**Waldo:** **Adopt the job, redesign the authority.** Use `Evening Brief → Closure → Daily Close`:

1. Evening Brief proposes what changed and what may remain open.
2. Closure lets the user confirm, correct, release, or carry each candidate.
3. Daily Close is the resulting trusted artifact, including evidence, explicit dispositions, memory candidates, and tomorrow's re-entry point.

Dimension shows an automated recap; Waldo should not call an inferred state “closed” without evidence and confirmation.

## Supporting capabilities and their screen implications

| Capability | Official marketed behavior | Waldo decision | Screen implication |
|---|---|---|---|
| Cross-app search | [Search](https://dimension.dev/search) asks natural-language questions across documents, email, messages, and files; source tabs and `@` file selection are shown | **Adapt** | Global Waldo search with source/freshness, permission scope, and “not searched” gaps |
| Documents | [`/docs`](https://dimension.dev/docs) markets prompt → generate → edit/refine → export to Docs, DOCX, PDF, or Dropbox | **Adapt** | An approved Home proposal can continue into Work as a document-producing AgentSession |
| Sheets | [Sheets](https://dimension.dev/sheets) markets formulas, formatting, connected data, chat refinement, and export | **Adapt** | Work artifact capability; Home only proposes it when tied to a responsibility |
| Slides | [Slides](https://dimension.dev/slides) markets research, structure, design, chat revision, and export/present | **Adapt** | Same Home-to-Work handoff; never present synthetic metrics as observed facts |
| Scheduled workflows | Agent and role pages show event/clock triggers followed by multi-app plans | **Adapt cautiously** | A governed Automation screen with explicit trigger, plan, authority, dry run, audit trail, pause, and revoke |
| Everywhere access | Agent visual markets desktop, Slack, iMessage, mobile, and shared context | **Adopt structurally** | One Waldo identity and canonical responsibility; each presence is permissioned and host-constrained |
| Integration graph | Search and homepage visuals market 30+ connected apps | **Reject as a goal by itself** | Show only sources contributing to a current job; connection count is not user value |

## Waldo's adaptive Home derived from the audit

Dimension uses fixed named moments. Waldo should use those moments as projections over one continuity model.

| Waldo chapter | Trigger | Primary Home content | Judgment desk | Dimension source | Distinct Waldo addition |
|---|---|---|---|---|---|
| Morning Brief | First meaningful open / configured morning window | What matters, schedule shape, overnight changes, confirmed open loops | One high-value candidate requiring confirmation | Morning Briefing | provenance, freshness, gaps, explicit candidate state |
| Resume | Return to an active responsibility or device | Exact last trusted state and next reversible move | Relevant evidence or correction | Cross-device agent continuity | responsibility-level checkpoint, not generic chat history |
| Catch Up | Return after absence or meaningful source changes | Changes since acknowledged checkpoint | Finite one-at-a-time review | Catch Up | absence boundary, source coverage, no inbox-zero pressure |
| Before the Next Thing | Imminent meeting/commitment | Attendees, promises, changes, open questions | Inspectable context card | Meeting Prep | statement/fact/inference separation |
| Afternoon Brief | Midday transition or plan drift, not clock alone | What became true, what changed, realistic remaining priorities | Replan proposal | **No direct Dimension equivalent**; synthesis from brief + action plan | capacity-aware recalibration and explicit defer/release |
| Evening Brief | Winding down signal | Proposed account of the day and unresolved responsibilities | Candidate dispositions | Daily Recap / Evening Briefing | no automatic closure |
| Closure | User enters reconciliation | Confirm, correct, release, carry forward, or remember | One decision at a time | Dimension terminal recap is only a partial reference | explicit control surface |
| Daily Close | Closure accepted or intentionally deferred | Trusted day artifact and tomorrow re-entry | Evidence/history inspection | Daily Recap | provenance-preserving, editable, revocable record |
| Quiet Focus / Plans Changed | Focus or disruption override | Current outcome or changed reality | Silent unless intervention threshold is met | Deep Work / proactive agent | attention budget and interruption policy |

### Recommended Waldo screen set

1. **Personal Home / Adaptive Brief** — current split chassis; left is the narrative of now, right is one `Needs You` item.
2. **Catch Up** — finite changes since an explicit checkpoint, with source tabs as filters rather than separate inboxes.
3. **Before the Next Thing** — meeting/person/responsibility context with claim-level provenance.
4. **Open Loops** — confirmed responsibilities plus a separate candidate inbox; no generic to-do dump.
5. **Closure** — guided dispositions: completed with evidence, still open, defer, release, correct, remember.
6. **Daily Close / History** — immutable event evidence plus editable interpretations and a clear tomorrow re-entry.
7. **Memory Review** — “Should Waldo remember this?” candidates with origin, validity interval, correction, expiry, and deletion.
8. **Automations / Personal Agent** — trigger, plan, permitted tools, external-effect checkpoints, run history, pause/revoke.
9. **Work handoff** — “Continue in Work” creates an accepted execution contract while preserving the originating statement and Home lifecycle.

Quick Capture remains a consistent bottom anchor on the Adaptive Brief. Its expanded copy changes with context (“Capture a thought”, “Note what changed”, “Leave this for tomorrow”), but it must not compete with the single primary judgment card.

## Adopt / Adapt / Reject summary

### Adopt

- A finite daily arc rather than a widget dashboard.
- Narrative synthesis before lists.
- One-at-a-time Catch Up review with an end state.
- Separate suggestions from user-owned tasks.
- Pre-meeting re-entry and cross-app context gathering.
- One assistant across surfaces.
- An evening reconciliation that carries context forward.

### Adapt

- `Morning Briefing` → evidence-bounded Morning Brief.
- `Catch Up` → meaningful changes since an acknowledged checkpoint.
- `Suggestions` → candidates with source, confidence language, expiry, and correction.
- `Assign to Dimension` → explicit Home-to-Work responsibility contract.
- `Runs on autopilot` → governed automation with dry-run, scope, pause, revoke, and audit.
- `Daily Recap` → Evening Brief → user Closure → trusted Daily Close.
- Search and artifacts → Work capabilities invoked from a confirmed personal responsibility.

### Reject

- Copying Dimension's brand, blue gradient, exact layout, or route labels as Waldo's identity.
- Inbox zero and generic productivity scoring as the success measure.
- Opaque numeric urgency/action/done labels.
- Silent task creation, sending, scheduling, or cross-app mutation.
- Personality inference presented as fact.
- Model or provider completion treated as Outcome truth.
- Automatic durable memory from summaries.
- Broad “connect everything” capture without job-specific permission and revocation.

## Privacy and memory boundary

Dimension's [privacy policy](https://dimension.dev/privacy-policy) says its service may build a cloud “Context Graph” from interactions, connected integrations, public web retrieval, facts, summaries, and inferences; may sync emails, calendar events, files, messages, tasks, issues, projects, comments, and attachments; and uses vector/knowledge-graph stores for search, summaries, drafting, preparation, suggestions, and personalization. It also states that profile data can be reviewed/edited/deleted, integration access can be revoked, and associated synced content/indexes are removed after disconnection.

This is policy evidence, not proof of UI implementation. It sharpens Waldo's divergence:

- local-first capture and canonical state;
- explicit source-level permission and visible capture gaps;
- raw observation separated from episode, candidate, confirmed responsibility, and durable memory;
- no public-web personal profile enrichment by default;
- claim-level correction, expiry, revoke, and deletion;
- no model training on user data;
- minimum necessary retrieval into each agent run;
- outward effects governed independently from read permission.

## Visual and interaction findings

- **Strong:** large narrative region plus focused review pane creates an understandable `orient → judge → finish` flow.
- **Strong:** item count and end state make Catch Up finite.
- **Strong:** source application remains visible and original content sits beneath the summary.
- **Strong:** suggestions use a plus action rather than appearing silently as tasks.
- **Strong:** meeting cards reveal detail on demand instead of placing every person fact in the main brief.
- **Weak:** light/dark and audience variants are often separate marketing renders rather than evidence of a coherent adaptive system.
- **Weak:** urgency and “done” labels lack visible derivation.
- **Weak:** the morning screen puts quantitative inbox load in the greeting, which can create pressure rather than agency.
- **Weak:** meeting-prep copy sometimes crosses from evidence into personality judgment.
- **Weak:** evening recap claims completion without a visible verification or correction step.
- **Weak:** the main site demonstrates time-of-day bookends but not a midday recalibration, disruption recovery, or durable memory review.

## Asset manifest

### Local directory

- `docs/research/assets/dimension-2026-08-22/pages/` — 25 full-page rendered captures, one per sitemap route.
- `docs/research/assets/dimension-2026-08-22/media/` — 141 original first-party files: 133 images and eight MP4s. This includes 103 rendered assets and 38 accessible source-only assets.
- `docs/research/assets/dimension-2026-08-22/video-frames/` — eight derived contact sheets used to inspect video state transitions.

Original media filenames flatten `/` to `_`; for example `/images/features/morning-briefing-full.png` becomes `images_features_morning-briefing-full.png`.

### Homepage product videos

| Local file | Original asset | Surrounding marketed use case | Observed sequence | Waldo disposition |
|---|---|---|---|---|
| `media/videos_new-landing_morning-briefing-start.mp4` | [`morning-briefing-start.mp4`](https://dimension.dev/videos/new-landing/morning-briefing-start.mp4) | Morning Briefing | Status capsule changes from email catch-up to reviewing updates | Adapt: transparent preparation state, but name actual permitted sources |
| `media/videos_new-landing_morning-briefing-results.mp4` | [`morning-briefing-results.mp4`](https://dimension.dev/videos/new-landing/morning-briefing-results.mp4) | Morning Briefing | Narrative and schedule context progressively appear | Adopt progressive reveal; add provenance/freshness |
| `media/videos_new-landing_catch-up.mp4` | [`catch-up.mp4`](https://dimension.dev/videos/new-landing/catch-up.mp4) | Catch Up | Switch source, review summary/original/draft, send, reach all-done state | Adapt with explicit authority and correction |
| `media/videos_new-landing_todo.mp4` | [`todo.mp4`](https://dimension.dev/videos/new-landing/todo.mp4) | Action Plan | Hover suggestion to reveal source-specific details; add/remove candidate | Adapt as OpenLoopCandidate review |
| `media/videos_new-landing_todo-assign-ai.mp4` | [`todo-assign-ai.mp4`](https://dimension.dev/videos/new-landing/todo-assign-ai.mp4) | Deep Work | Select a to-do for agent assignment | Adapt as Home-to-Work contract |
| `media/videos_new-landing_managed-inbox.mp4` | [`managed-inbox.mp4`](https://dimension.dev/videos/new-landing/managed-inbox.mp4) | Inbox | Auto-label inbox, show drafted response and gathered attachments, user sends | Adapt; reject opaque labels and implicit attachment authority |
| `media/videos_new-landing_meeting.mp4` | [`meeting.mp4`](https://dimension.dev/videos/new-landing/meeting.mp4) | Meeting Prep | Open meeting summary then attendee context and talking points | Adapt with epistemic separation |
| `media/videos_new-landing_evening-briefing.mp4` | [`evening-briefing.mp4`](https://dimension.dev/videos/new-landing/evening-briefing.mp4) | Daily Recap | Task state changes followed by an emailed day summary | Adapt into Evening Brief → Closure → Daily Close |

### Canonical feature-image families

| Family and local files | First-party source page(s) | Exact marketed use case | Observed UI |
|---|---|---|---|
| `morning-briefing-full`, `morning-briefing`, `morning-briefing-borderless` | [Morning Briefing](https://dimension.dev/morning-briefing), older [Documents](https://dimension.dev/documents) page | Overnight activity, suggested actions, and full-day overview | Narrative brief, to-do/suggestion split, Catch Up pane, finish action |
| `catch-up-mini`, `catch-up` | [Morning Briefing](https://dimension.dev/morning-briefing), [Email](https://dimension.dev/email) | Summarize new Gmail/Slack messages and prepare replies | Source toggle, item counter, summary, original content, draft/send |
| `suggestions` | [Morning Briefing](https://dimension.dev/morning-briefing) | Pull action items from calendar, Linear, GitHub, Slack, and Gmail | Candidate list with explicit add-to-list action |
| `auto-drafting`, `managed-inbox` | [Email](https://dimension.dev/email) | Label email and prepare response drafts before the user opens it | Gmail labels, draft composer, original thread, user Send action |
| `todo-list`, `todo-suggestions`, `todo-dimension`, `todo-auto` | [Todo](https://dimension.dev/todo) | Capture cross-app actions, add suggestions, delegate, show execution/result | Confirmed list separated from suggestions; assignment affordance; agent work trace |
| `agent/devices`, `agent/workflow`, `agent/task` | [Agent](https://dimension.dev/agent) | Same assistant across surfaces; triggered workflows; agent-completed tasks | Desktop/iMessage/Slack continuity, when-plan builder, task statuses |
| `rag-search`, `dimension-integrations`, `mentions-picker`, `security-certified` | [Search](https://dimension.dev/search) | Search connected apps, ask about a file, communicate security posture | Source tabs/results, integration constellation, `@` picker, certification badges |
| `artifact-docs-hero`, `artifact-docs-1..3` | [Docs](https://dimension.dev/docs) | Prompt/generate, edit/refine, export a document | Generated doc gallery/editor and export workflow |
| `artifact-sheets-hero`, `artifact-sheets-1..3` | [Sheets](https://dimension.dev/sheets) | Prompt/generate, edit formulas/data, export a spreadsheet | Spreadsheet editor and artifact views |
| `artifact-slides-hero`, `artifact-slides-1..3` | [Slides](https://dimension.dev/slides) | Prompt/generate, chat-refine, present/export a deck | Slide editor, deck gallery, export/presentation views |
| `usecase/pdfs/engineer-draft-prd`, `usecase/pdfs/vc-ic-memo` | [Engineers](https://dimension.dev/engineers), [VC](https://dimension.dev/vc) | Produce a PRD or IC memo from scattered connected context | PDF result preview |
| `new-landing/features/feature-otg`, `feature-integrations` | [Homepage](https://dimension.dev/) | Be available in Slack/iMessage and connect across the work stack | Conceptual product composites, not runtime screens |

Rendered-resource completeness files not named above are also preserved:

- six decorative flute variants under `images_fluteds_original_*`;
- six responsive audience-inbox images under `images_features_inbox_inbox-*-mobile.png`;
- `images_logo_dimension-desktop.png`; and
- `images_new-landing_cta-mobile-bg.png`.

All files above are in `docs/research/assets/dimension-2026-08-22/media/` using flattened names. Their original URLs are `https://dimension.dev/` followed by the slash-delimited path represented in the filename.

### Audience-specific daily-flow variants

Each of the six role pages publishes seven unique UI images. All 42 originals are preserved. The file families are:

- `images_features_morning-briefing_morning-briefing-{audience}.png`
- `images_features_catchup_catchup-{audience}.png`
- `images_features_prep_prep-{audience}.png`
- `images_features_evening_evening-{audience}.png`
- `images_features_inbox_inbox-{audience}.png`
- `images_features_document_doc-{audience}.png`
- `images_features_todo-list_todo-{audience}.png`

The engineer to-do file uses singular `todo-engineer.png`; all other slugs match the page audience.

| Audience / source page | Morning Brief | Catch Up | Meeting Prep | Evening Brief | Inbox | Artifact | Todo / agent |
|---|---|---|---|---|---|---|---|
| [Agency](https://dimension.dev/agency) | Overnight activity across clients | Slack/Gmail replies | Client thread, task, and project status | Accomplished/open across clients | Client communication triage | Client-ready proposal/case-study document | Follow-up, invoice, onboarding work |
| [Engineers](https://dimension.dev/engineers) | Sprint, PR, and deployment status | Engineering email/Slack queue | Recent threads, open tasks, project state | Shipped work and remaining blockers | Engineering inbox classification | PRD/deployment/postmortem document | Bug triage, deploy checks, docs |
| [Founder](https://dimension.dev/founder) | Calls, warm intro, product/customer changes | Investor/customer/team replies | Prior customer context, open questions | Closed and open company responsibilities | Founder inbox classification/drafting | Board/investor/customer artifact | Customer triage, research, document prep |
| [PM](https://dimension.dev/pm) | Sprint, bugs, meetings | Product Gmail/Slack queue | Issues, account context, forgotten thread history | Shipped, triaged, discussed, still open | Product inbox classification | PRD/sprint/feedback report | Bug triage, synthesis, status update |
| [Sales](https://dimension.dev/sales) | Pipeline, replies, demos | Deal communication queue | Deal stage, prior conversations | Calls, deals, follow-ups | Sales inbox classification/drafting | Proposal/one-pager/case study | CRM update, follow-up, research |
| [VC](https://dimension.dev/vc) | Portfolio, deal flow, schedule | Founder/LP communication | Deal status and prior conversation | Meetings and follow-ups | Deal/portfolio inbox classification | IC deck, LP update, portfolio review | Intro research, tracking, memo prep |

**Observed pattern:** audience customization is a content/provenance substitution over the same seven-screen grammar. **Inference for Waldo:** personalize modules from the user's active responsibility graph; do not ask them to pick a permanent persona or maintain six navigation systems.

### Decorative and non-product marketing images

These were downloaded for completeness but should not be treated as UI evidence:

- `images_new-landing_hero_shadow-bg.png`
- `images_new-landing_hero_sun-halo.png`
- `images_new-landing_cta-desktop-bg.png`
- `images_new-landing_cta-mobile-bg.png`
- `images_new-landing_pricing_pricing-bg.jpg`
- `images_fluteds_original_original-flute-{black,blue,green,orange,purple,yellow}.png`
- `images_logo_dimension-desktop.png`

The responsive mobile CTA had zero rendered size at the audited desktop viewport but was downloaded from its first-party resource URL.

### Accessible source-only assets

These 38 files were referenced by the official public pages' loaded JS/CSS or metadata but were not observed as visible body media at the audited 1440px viewport. They are downloaded into `media/` and are marked separately so source presence is not misreported as observed desktop behavior.

#### Responsive product screens

| Local/source family | Marketed job inferred from official filename and homepage section | Confidence |
|---|---|---|
| `images_new-landing_hero_mobile_agent.png` | Agent execution / Deep Work | Medium-high |
| `images_new-landing_hero_mobile_catchup.png` | Finite Catch Up review | Medium-high |
| `images_new-landing_hero_mobile_email.png` | Managed inbox and drafting | Medium-high |
| `images_new-landing_hero_mobile_evening.png` | Evening Brief / Daily Recap | Medium-high |
| `images_new-landing_hero_mobile_meeting.png` | Meeting Prep | Medium-high |
| `images_new-landing_hero_mobile_morning-briefing.png` | Morning Briefing | Medium-high |
| `images_new-landing_hero_mobile_todo.png` | Action Plan / Todo | Medium-high |
| `images_new-landing_hero_morning-briefing.png` | Desktop hero representation of Morning Briefing | Medium-high |

Their original URL is `https://dimension.dev/` followed by the filename converted back from `_` separators to its path; the exact paths are `/images/new-landing/hero/mobile/{agent,catchup,email,evening,meeting,morning-briefing,todo}.png` and `/images/new-landing/hero/morning-briefing.png`.

#### Persona artifact examples

| Local file | Owning marketed audience/job | Confidence |
|---|---|---|
| `images_usecase_docs_agency-pitch-prep.png` | Agency: research and prepare a client pitch | Medium-high |
| `images_usecase_docs_vc-founder-meeting-prep.png` | VC: prepare for a founder meeting | Medium-high |
| `images_usecase_pdfs_engineer-customer-feedback-synthesis.png` | Engineer: synthesize customer feedback into an artifact | Medium-high |
| `images_usecase_pdfs_engineer-sprint-review-prep.png` | Engineer: prepare a sprint review from connected context | Medium-high |
| `images_usecase_pdfs_founder-fundraise-prep.png` | Founder: prepare fundraising material | Medium-high |
| `images_usecase_pdfs_founder-prep-investor-meeting.png` | Founder: prepare for an investor meeting | Medium-high |
| `images_usecase_pdfs_sales-prep-call.png` | Sales: prepare for a customer/prospect call | Medium-high |
| `images_usecase_pdfs_senior-engineer.png` | Engineering persona/context artifact; exact placement unknown | Medium |
| `images_usecase_pdfs_vc-board-meeting-prep.png` | VC: prepare for a portfolio board meeting | Medium-high |

#### Download, native-auth, and metadata imagery

- Product/download imagery: `images_logo_apple-store.png`, `images_macbook_mac-folder-icon.png`, `images_macbook_macbook-background.jpg`, and `images_native-auth_hero.png`.
- Favicons: `seo_favicon_apple-touch-icon.png`, `seo_favicon_favicon-96x96.png`, `seo_favicon_favicon.ico`, and `seo_favicon_favicon.svg`.
- Feature social cards: `seo_og_feature-{agent,docs,email,morning-briefing,search,todo}.png`.
- Persona social cards: `seo_og_persona-{agency,engineers,founder,pm,sales,vc}.png`.
- General social card: `seo_opengraph-2.png`.

The source-only download set is complete. The two 404 paths are intentionally not represented as local files.

### Full-page capture manifest

The 25 files in `pages/` correspond one-to-one with sitemap routes:

`home.png`, `agency.png`, `agent.png`, `docs.png`, `documents.png`, `email.png`, `engineers.png`, `feature__agent.png`, `feature__documents.png`, `feature__email.png`, `feature__morning-briefing.png`, `feature__search.png`, `founder.png`, `morning-briefing.png`, `pm.png`, `pricing.png`, `privacy-policy.png`, `sales.png`, `search.png`, `sheets.png`, `slides.png`, `terms-of-service.png`, `todo.png`, `vc.png`, and `winding-down.png`.

## Product-development recommendation

Use the present Dimension-inspired screen as a temporary visual chassis, then implement Waldo's difference in this order:

1. Define the `HomeProjection` chapter and evidence contract before adding more page-specific copy.
2. Implement Morning Brief, Catch Up, and Evening Brief as projections over fixtures that contain source, observed-at, freshness, confidence language, and confirmation state.
3. Add Closure and produce Daily Close only from explicit dispositions.
4. Add the midday `Afternoon Brief` as a recalibration triggered by plan drift, completed evidence, or transition—not merely 12:00 PM.
5. Preserve one Quick Capture anchor and one primary `Needs You` card.
6. Add explicit `Continue in Work` responsibility handoff; keep execution state separate from Home truth.
7. Prove the model with synthetic day replays and shadow-mode capture before enabling external effects or durable automatic memory.

The visual reference answers **how to make personal agency legible**. Waldo's architecture must answer the harder question Dimension's marketing leaves unresolved: **why should the user trust that this interpretation is true, current, correctable, private, and acting within authority?**
