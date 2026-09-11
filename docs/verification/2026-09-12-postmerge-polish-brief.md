# Post-merge Work polish brief

This is a design/implementation brief for a later slice. It is deliberately not part of the launch checkpoint's product implementation.

## Interaction direction

- Keep the existing Outcome-first hierarchy: mission graph first, then a session kanban/detail escape hatch. Do not copy the ontology into one suboutcome per session; sessions remain subordinate execution resources.
- Use subtle dark rounded cards, thin curved graph edges, compact node labels, and small status dots. Preserve keyboard/focus behavior, reduced-motion handling, loading/error/recovery states, and accessible names.
- Use concise progressive disclosure: show the next owner decision first, with deeper plan, WorkUnit, Attempt, and provider detail available on demand.
- Make save state explicit: a successful Save reads `Saved` with a check icon, not merely a green color. Keep input, draft, saving, and saved transitions stable under refresh and mutation failure.
- Show processing with a restrained spinner/Waldo mark and a truthful status; never imply provider execution or Outcome completion while planning is still in flight.

## Graph implementation decision

Dependency inspection at this checkpoint found no existing graph/pan-zoom package (`reactflow`, `@xyflow/react`, `dagre`, `elkjs`, `cytoscape`, and `vis-network` were absent). The post-merge spike should evaluate `@xyflow/react` first for pan/zoom, keyboard-accessible nodes, and edge rendering, with a small fixture-driven proof before adding the dependency. A bespoke SVG graph should not become a second interaction/state model.

## Planning updates

Planning updates should continue through the existing native/API boundary and render daemon-owned status, not transcript heuristics. Packet-only context and explicitly authorized repository tools must remain visibly distinct. Any native Codex parity work remains a separate acceptance slice after the beta checkpoint.
