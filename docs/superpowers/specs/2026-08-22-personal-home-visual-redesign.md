# Waldo Kennel Personal Home visual redesign

- **Status:** Approved visual direction
- **Date:** 2026-08-22
- **Scope:** Frontend-only Personal Home fixture experience
- **Out of scope:** daemon APIs, persistence, capture workers, memory admission, agent execution, harness wiring, push, PR, merge, deploy, and GitHub mutation

## Product decision

Personal Home uses an adaptive brief as its primary surface. The default path is `orient -> review one meaningful change -> inspect or correct its source -> preserve responsibility -> explicitly continue in Work -> consciously close or re-enter`. It does not present Today, Catch Up, Open Loops, Memory, Daily Close, History, and Activity as equal dashboards.

The approved composition combines:

- Concept 1 as the default adaptive brief;
- Concept 2's split pane only for wide Catch Up and provenance review;
- Concept 3's temporal evidence only inside History and Daily Close.

## Global mode and navigation

A persistent horizontal `Home | Work` pill lives in global top chrome. It changes the responsibility-space lens and capability envelope, not account, identity, custody, or canonical storage.

- Home and Work retain their last meaningful local route, focus, selection, and scroll position.
- Changing the pill never moves, links, duplicates, accepts, or closes a responsibility.
- `Continue in Work` is a distinct explicit action and remains an architecture-preview interaction in this visual slice.
- The Work recommendation appears only during first-run guidance, not in the permanent pill.
- The selected state uses text, shape, accessible state, focus treatment, and color; color alone is insufficient.
- Narrow layouts retain a labelled top-chrome control rather than moving it to a floating bottom dock.

When Home is active, the sidebar hierarchy is intentionally reduced:

- `Today` and `Open Loops` are stable primary destinations.
- Catch Up and Daily Close are contextual flows.
- Memory Review, History, Activity, and capture governance are supporting drill-down/control layers.
- Existing routes may remain addressable for visual evaluation without requiring equal sidebar prominence.

When Work is active, the existing Project/session hierarchy remains available. This slice does not redesign Work content.

## Quick Capture

Capture availability and an expanded capture field are separate:

- global chrome exposes truthful preview capture state and one Quick Capture trigger;
- Today renders the expanded text input once, after the orientation brief;
- other Home routes open capture on demand rather than repeating the composer;
- detail contexts use scoped actions such as `Add context`, `Correct this`, or `Add close note`;
- the preview states where input would go and never claims persistence;
- unsent local fixture text survives recoverable Home route changes during the preview session;
- no fixture implies automatic creation of Memory, Open Loop, Outcome, Evidence, authority, Acceptance, or closure.

## Default Home: Adaptive Brief

The first viewport follows one reading path:

1. date and truthful availability/capture state;
2. exact re-entry when one exists;
3. a short evidence-bounded brief;
4. the smallest genuine `Needs You` queue;
5. quiet Waiting and Ready to Close summaries;
6. one expanded Quick Capture input.

Use editorial bands and responsibility rows. Reserve elevation for the current consequential decision or provenance inspector. Do not build a grid of equal cards.

Empty Home is successful and calm. Partial, capture-off, stale, and offline states remain explicit and do not masquerade as empty.

## Catch Up and provenance

Catch Up processes one material candidate at a time. The visual fixture uses the approved scenario: Waldo observed “I'll send Ashish the revised deck tomorrow,” the source has a known capture gap, and the user can correct it to “Prepare the revised deck for Ashish; do not send it.”

The decision surface provides preview-only actions for source inspection, correction, keeping a note, confirming an Open Loop candidate, deferring, dismissing, and continuing in Work.

- At wide widths, content and provenance use a deliberate split pane.
- At narrow widths, provenance temporarily replaces the content pane.
- Closing provenance restores exact focus and scroll to the originating control.
- The source view distinguishes user-stated text, Waldo inference, capture gap, and architecture-preview status.

## Open Loops

Open Loops is the only other primary Home destination in the visual hierarchy. It shows responsibility meaning, owner, trigger/recheck, source strength, and current disposition without representing inferred tasks as confirmed responsibility.

Selecting a row enters responsibility detail. `Continue in Work` previews the explicit lineage boundary; it does not simulate successful daemon creation or Work execution.

## Daily Close and History

Daily Close is a contextual review, not a permanent dashboard and not an inbox-zero ritual. It presents:

- what became true;
- what remains unresolved;
- acknowledged source gaps;
- explicit close, release, defer, or tomorrow re-entry choices.

Completing the visual flow never implies that an Open Loop or Outcome was mutated.

History uses a continuity timeline for user decisions, corrections, links, close receipts, and re-entry. Activity evidence is subordinate drill-down, not the primary story and not a productivity score.

## Visual language

- Reuse existing Kennel type, spacing, color, focus, and surface tokens.
- Prefer editorial hierarchy, dividers, compact rows, and one focused plane over repeated rounded cards.
- Keep body copy specific and short; avoid generic assistant, wellness, dashboard, or “AI insight” language.
- Show provenance and freshness as compact labelled metadata.
- Motion only explains pane/focus changes, remains brief and interruptible, and has an immediate reduced-motion path.
- Validate wide, narrow, empty, partial, capture-off, stale, and offline fixture states.

## Frontend architecture boundary

The renderer remains a fixture-only projection in this slice:

- route components choose visual destinations;
- Home fixture data describes source, availability, attention, re-entry, responsibility, and continuity examples;
- components emit local preview interactions only;
- no component opens SQLite, calls future Home APIs, creates canonical objects, or claims backend capability;
- existing Work project/session navigation remains owned by the current shell.

The later backend/agent/harness phase will separately define daemon services, DTOs, storage, CDC, canonical commands, retrieval packets, and runtime integration.

## Acceptance criteria

1. The permanent shell exposes a keyboard-accessible horizontal `Home | Work` pill and removes duplicate generic Home/Work rows.
2. Home mode shows the reduced primary hierarchy without deleting addressable review routes.
3. Today renders the adaptive brief and exactly one expanded Quick Capture field.
4. Other Home destinations do not repeat the expanded composer.
5. Catch Up and provenance use a wide split pane and narrow replacement view with exact return.
6. The approved source-gap/correction/Continue-in-Work scenario is visually traversable without claiming persistence.
7. Daily Close and History use continuity evidence without activity scoring or implied closure.
8. Empty, partial, capture-off, stale, and offline states remain truthful.
9. Focus states, keyboard paths, narrow layout, and reduced motion remain usable.
10. Existing Work navigation, project/session selection, and non-Home routes continue to pass their focused tests.
