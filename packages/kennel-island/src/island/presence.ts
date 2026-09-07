import type { IslandPresence, IslandPresenceCard } from "./types";

/**
 * Descending urgency. A session holding a gate outranks one that merely wants
 * your attention, which outranks one that is getting on with its work.
 */
export const PRESENCE_ORDER: readonly IslandPresence[] = ["blocked", "paused", "running"];

/**
 * How long each card holds the chip before the next one takes it.
 *
 * Long enough to read a count and a label without feeling like a slideshow;
 * short enough that a running session is not hidden behind four approvals for
 * an uncomfortable stretch.
 */
export const PRESENCE_ROTATE_MS = 3_500;

export function presenceRank(presence: IslandPresence) {
  const rank = PRESENCE_ORDER.indexOf(presence);
  return rank === -1 ? PRESENCE_ORDER.length : rank;
}

/** Most urgent first, and never two cards for the same presence. */
export function orderPresenceCards(cards: readonly IslandPresenceCard[]): IslandPresenceCard[] {
  const byPresence = new Map<IslandPresence, IslandPresenceCard>();
  for (const card of cards) {
    if (card.count <= 0) continue;
    const existing = byPresence.get(card.presence);
    byPresence.set(
      card.presence,
      existing ? { ...existing, count: existing.count + card.count } : card,
    );
  }
  return [...byPresence.values()].sort((left, right) =>
    presenceRank(left.presence) - presenceRank(right.presence));
}

/**
 * Identity of a rotation cycle.
 *
 * The rotation restarts from the most urgent card whenever the *set* of live
 * presences changes, so a new approval takes the chip immediately instead of
 * waiting its turn. Counts are deliberately excluded: a fifth session joining
 * a running group should not yank the display back to the top.
 */
export function presenceRotationKey(cards: readonly IslandPresenceCard[]) {
  return cards.map((card) => card.presence).join("|");
}

export function nextPresenceIndex(index: number, length: number) {
  if (length <= 0) return 0;
  return (index + 1) % length;
}

/** Clamps a remembered index onto a list that may have shrunk under it. */
export function safePresenceIndex(index: number, length: number) {
  if (length <= 0) return 0;
  return Math.min(Math.max(index, 0), length - 1);
}
