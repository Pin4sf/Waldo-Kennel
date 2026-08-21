import {
  useCallback,
  useEffect,
  useLayoutEffect,
  useRef,
  useState,
  type RefObject,
} from "react";
import { dominantColor, SAMPLE_SIZE, toCssColor } from "./artwork";
import { defaultStageGeometry } from "./layout";
import {
  otherSubject,
  peekSubjectFor,
  peekZonesFor,
  pointerHasMoved,
  PEEK_ROTATE_IDLE_MS,
  PEEK_ROTATE_INTERVAL_MS,
  zoneAt,
  ZONE_OVERSHOOT_BOTTOM,
  ZONE_OVERSHOOT_TOP,
  type PeekSubject,
  type PeekZone,
} from "./peek-layout";
import {
  nextPresenceIndex,
  presenceRotationKey,
  PRESENCE_ROTATE_MS,
  safePresenceIndex,
} from "./presence";
import { createGestureRecognizer, scrollCanConsume, type IslandGesture } from "./gestures";
import { defaultKennelSettings } from "./settings";
import { isPointerInside } from "./stage-rules";
import type { IslandPresenceCard } from "./types";

/**
 * Live stage geometry from the desktop host.
 *
 * The host owns the window and the display; the renderer only mirrors what it
 * reports, so a resolution change or an external display arriving reshapes the
 * island without a restart. Outside Electron this stays on the default, which
 * is what the browser state lab wants.
 */
export function useStageGeometry(): KennelStageGeometry {
  const [geometry, setGeometry] = useState<KennelStageGeometry>(defaultStageGeometry);

  useEffect(() => {
    const desktop = window.kennelDesktop;
    if (typeof desktop?.getStageGeometry !== "function") return;

    let active = true;
    void desktop.getStageGeometry().then((next) => {
      if (active && next) setGeometry(next);
    }).catch(() => {
      // A geometry read failing is not worth degrading the island for; the
      // default describes a notched Mac and the host will push a correction.
    });

    const unsubscribe = desktop.onStageGeometry?.((next) => setGeometry(next));
    return () => {
      active = false;
      unsubscribe?.();
    };
  }, []);

  return geometry;
}

/**
 * Keeps the stage click-through except while the pointer is over the island.
 *
 * The window ignores the mouse by default so the menu bar and everything under
 * the stage stay usable. Electron keeps forwarding mousemove while ignoring,
 * which is the only signal available for deciding when to take the pointer
 * back. Focus is honoured too: a caret inside the island must not lose its
 * clicks because the pointer drifted off.
 */
export function useStageInteractivity(islandRef: RefObject<HTMLElement | null>): boolean | null {
  // `null` means the host is not tracking the pointer for us — the browser
  // state lab, or a renderer whose bridge failed to attach. Callers must not
  // read that as "the pointer is elsewhere", or every hover-held surface would
  // close the instant it opened.
  const [hovered, setHovered] = useState<boolean | null>(null);
  const interactiveRef = useRef(false);

  useEffect(() => {
    const desktop = window.kennelDesktop;
    if (typeof desktop?.setInteractive !== "function") return;

    setHovered(false);

    const apply = (interactive: boolean) => {
      if (interactive === interactiveRef.current) return;
      interactiveRef.current = interactive;
      setHovered(interactive);
      void desktop.setInteractive(interactive).catch(() => {
        // Losing a toggle would strand the stage in the wrong mode, so fall
        // back to the click-through default rather than trusting the cache.
        interactiveRef.current = false;
      });
    };

    const holdsFocus = () => {
      const active = document.activeElement;
      return active !== null && active !== document.body && Boolean(islandRef.current?.contains(active));
    };

    const evaluate = (x: number, y: number) => {
      const element = islandRef.current;
      if (!element) return;
      apply(isPointerInside(element.getBoundingClientRect(), x, y) || holdsFocus());
    };

    let pointerX = Number.NEGATIVE_INFINITY;
    let pointerY = Number.NEGATIVE_INFINITY;
    let recheck: number | undefined;

    const handleMove = (event: MouseEvent) => {
      pointerX = event.clientX;
      pointerY = event.clientY;
      evaluate(pointerX, pointerY);
    };
    const handleLeave = () => {
      pointerX = Number.NEGATIVE_INFINITY;
      pointerY = Number.NEGATIVE_INFINITY;
      apply(holdsFocus());
    };
    const handleFocusIn = () => {
      if (holdsFocus()) apply(true);
    };
    // focusout runs before the incoming element is focused, so re-read after
    // the move settles rather than dropping the pointer between two fields.
    const handleFocusOut = () => {
      window.clearTimeout(recheck);
      recheck = window.setTimeout(() => evaluate(pointerX, pointerY), 0);
    };

    window.addEventListener("mousemove", handleMove);
    document.addEventListener("mouseleave", handleLeave);
    document.addEventListener("focusin", handleFocusIn);
    document.addEventListener("focusout", handleFocusOut);

    return () => {
      window.clearTimeout(recheck);
      window.removeEventListener("mousemove", handleMove);
      document.removeEventListener("mouseleave", handleLeave);
      document.removeEventListener("focusin", handleFocusIn);
      document.removeEventListener("focusout", handleFocusOut);
      if (interactiveRef.current) void desktop.setInteractive(false).catch(() => {});
    };
  }, [islandRef]);

  return hovered;
}

/**
 * Live settings from the desktop host.
 *
 * Both windows use this: the island reads preferences, and the settings pane
 * both reads and writes them. Whoever changes a value, the host is the single
 * copy and it pushes the result to everyone — so two open surfaces never
 * disagree about what the notch is currently set to.
 *
 * Outside Electron this stays on the defaults, which is what the browser state
 * lab wants.
 */
export function useKennelSettings(): KennelSettings {
  const [settings, setSettings] = useState<KennelSettings>(defaultKennelSettings);

  useEffect(() => {
    const desktop = window.kennelDesktop;
    if (typeof desktop?.getSettings !== "function") return;

    let active = true;
    void desktop.getSettings().then((next) => {
      if (active && next) setSettings(next);
    }).catch(() => {
      // Settings that cannot be read are settings at their defaults, which is
      // a working island rather than a blank one.
    });

    const unsubscribe = desktop.onSettings?.((next) => setSettings(next));
    return () => {
      active = false;
      unsubscribe?.();
    };
  }, []);

  return settings;
}

/** Writes a settings patch. Errors are swallowed: the host pushes the truth. */
export function useSettingsWriter() {
  return useCallback((patch: KennelSettingsPatch) => {
    const desktop = window.kennelDesktop;
    if (typeof desktop?.updateSettings !== "function") return;
    void desktop.updateSettings(patch).catch(() => {});
  }, []);
}

/**
 * Force Touch feedback.
 *
 * Returns a stable function that never throws and never reports anything: the
 * caller is a hover handler, and there is nothing useful it could do with the
 * news that a trackpad did not buzz.
 */
export function useHaptics() {
  return useCallback((pattern: KennelHapticPattern = "alignment") => {
    const desktop = window.kennelDesktop;
    if (typeof desktop?.performHaptic !== "function") return;
    void desktop.performHaptic(pattern).catch(() => {});
  }, []);
}

/**
 * Whether the pointer has been on the island long enough to mean it.
 *
 * The dwell is the whole reason this is not just `hovered`. The notch sits at
 * the top edge of the screen, which is where the pointer travels to reach the
 * menu bar and every window control below it — so a shape that answered the
 * instant a cursor crossed it would answer dozens of times an hour to somebody
 * who was going somewhere else.
 */
export function usePointerDwell(hovered: boolean, delayMs: number): boolean {
  const [settled, setSettled] = useState(false);

  useEffect(() => {
    if (!hovered) {
      // Leaving is immediate. A shape that lingered after the pointer had gone
      // would read as the island failing to notice rather than as a grace.
      setSettled(false);
      return;
    }
    if (delayMs <= 0) {
      setSettled(true);
      return;
    }

    const timer = window.setTimeout(() => setSettled(true), delayMs);
    return () => window.clearTimeout(timer);
  }, [delayMs, hovered]);

  return settled;
}

/**
 * What the peek should be talking about, from where the pointer is.
 *
 * The zones are measured from the strip's own chips rather than declared, so a
 * chip appearing or leaving redraws the boundaries without anything here
 * knowing what the chips are. Each chip marks itself with `data-peek`; this
 * only reads them.
 *
 * Measurement happens when the layout changes, not on every pointer move: the
 * strip re-renders on every rotation tick, and reading geometry from a
 * mousemove handler would put a layout flush in the middle of one.
 */
export function usePeekSubject({
  headerRef,
  hovered,
  hasMedia,
  hasSession,
}: {
  headerRef: RefObject<HTMLElement | null>;
  hovered: boolean;
  hasMedia: boolean;
  hasSession: boolean;
}): { subject: PeekSubject | null; zone: PeekSubject | null } {
  const [zone, setZone] = useState<PeekSubject | null>(null);
  const [rotated, setRotated] = useState<PeekSubject | null>(null);
  const zonesRef = useRef<PeekZone[]>([]);
  const boundsRef = useRef<DOMRect | null>(null);

  const measure = useCallback(() => {
    const header = headerRef.current;
    if (!header) return;

    const bounds = header.getBoundingClientRect();
    boundsRef.current = bounds;

    const items = [...header.querySelectorAll<HTMLElement>("[data-peek]")]
      .map((element) => {
        const kind = element.dataset.peek;
        if (kind !== "media" && kind !== "session") return null;
        const box = element.getBoundingClientRect();
        return { kind, left: box.left, right: box.right };
      })
      .filter((item): item is { kind: PeekSubject; left: number; right: number } => item !== null);

    zonesRef.current = peekZonesFor(items, { left: bounds.left, right: bounds.right });
  }, [headerRef]);

  useLayoutEffect(() => {
    const header = headerRef.current;
    if (!header) return;

    measure();
    const observer = new ResizeObserver(measure);
    observer.observe(header);
    for (const item of header.querySelectorAll("[data-peek]")) observer.observe(item);
    return () => observer.disconnect();
  }, [headerRef, measure, hasMedia, hasSession]);

  useEffect(() => {
    if (!hovered) {
      setZone(null);
      setRotated(null);
      return;
    }

    let parkedAt: { x: number; y: number } | null = null;
    let idleTimer: number | undefined;
    let rotateTimer: number | undefined;
    // The zone the pointer parked in, so the rotation knows what to rotate away
    // from. It is a ref rather than state because the countdown reads it once,
    // at the moment it fires, and re-running the effect on every zone change
    // would restart the countdown the rotation is waiting on.
    const parkedZone = { current: null as PeekSubject | null };

    const stopRotating = () => {
      window.clearTimeout(idleTimer);
      window.clearInterval(rotateTimer);
      rotateTimer = undefined;
    };

    // A pointer that has stopped moving has stopped choosing, so after a long
    // enough pause the peek offers the other subject rather than holding the
    // one the cursor happens to be resting over.
    const startIdleCountdown = () => {
      stopRotating();
      if (!hasMedia || !hasSession) return;

      idleTimer = window.setTimeout(() => {
        setRotated((current) => otherSubject(current ?? parkedZone.current ?? "session"));
        rotateTimer = window.setInterval(
          () => setRotated((current) => otherSubject(current ?? "session")),
          PEEK_ROTATE_INTERVAL_MS,
        );
      }, PEEK_ROTATE_IDLE_MS);
    };

    const handleMove = (event: MouseEvent) => {
      const point = { x: event.clientX, y: event.clientY };
      const bounds = boundsRef.current;
      const inside =
        bounds !== null &&
        point.y >= bounds.top - ZONE_OVERSHOOT_TOP &&
        point.y <= bounds.bottom + ZONE_OVERSHOOT_BOTTOM;

      const next = inside ? zoneAt(zonesRef.current, point.x)?.subject ?? null : null;
      parkedZone.current = next;
      setZone(next);

      if (!pointerHasMoved(parkedAt, point)) return;
      parkedAt = point;
      // Any real movement is a fresh choice, so the rotation stands down.
      setRotated(null);
      startIdleCountdown();
    };

    window.addEventListener("mousemove", handleMove);
    startIdleCountdown();
    return () => {
      stopRotating();
      window.removeEventListener("mousemove", handleMove);
    };
  }, [hasMedia, hasSession, hovered]);

  return { subject: peekSubjectFor({ zone, hasMedia, hasSession, rotated }), zone };
}

/**
 * The accent colour of the current artwork, or null when there is none to take.
 *
 * The artwork is a data URI, so the canvas it is drawn on is never tainted and
 * the pixels can be read without a cross-origin dance. A cover with no usable
 * colour — anything black and white — resolves to null rather than to a grey,
 * and the island keeps its default.
 */
export function useArtworkAccent(artwork: string | undefined): string | null {
  const [accent, setAccent] = useState<string | null>(null);

  useEffect(() => {
    if (!artwork) {
      setAccent(null);
      return;
    }

    let active = true;
    const image = new Image();
    image.decoding = "async";

    image.onload = () => {
      if (!active) return;

      try {
        const canvas = document.createElement("canvas");
        canvas.width = SAMPLE_SIZE;
        canvas.height = SAMPLE_SIZE;
        const context = canvas.getContext("2d", { willReadFrequently: true });
        if (!context) return;

        context.drawImage(image, 0, 0, SAMPLE_SIZE, SAMPLE_SIZE);
        const pixels = context.getImageData(0, 0, SAMPLE_SIZE, SAMPLE_SIZE).data;
        const dominant = dominantColor(pixels);
        setAccent(dominant ? toCssColor(dominant) : null);
      } catch {
        // A canvas that will not give up its pixels is a wash we do not get.
        setAccent(null);
      }
    };
    image.onerror = () => {
      if (active) setAccent(null);
    };
    image.src = artwork;

    return () => {
      active = false;
    };
  }, [artwork]);

  return accent;
}

const SILENT_MEDIA: KennelMediaActivity = { playing: false, track: null };

/**
 * Media activity from the desktop host.
 *
 * Host state, not Kennel state, which is why it arrives beside the island model
 * rather than inside it: the snapshot stays entirely server-derived.
 */
export function useMediaActivity(): KennelMediaActivity {
  const [media, setMedia] = useState<KennelMediaActivity>(SILENT_MEDIA);

  useEffect(() => {
    const desktop = window.kennelDesktop;
    if (typeof desktop?.getMediaActivity !== "function") return;

    let active = true;
    void desktop.getMediaActivity().then((next) => {
      if (active && next) setMedia(next);
    }).catch(() => {
      // Not knowing what is playing renders as silence, never as a stuck
      // waveform over a quiet machine.
    });

    const unsubscribe = desktop.onMediaActivity?.((next) => setMedia(next));
    return () => {
      active = false;
      unsubscribe?.();
    };
  }, []);

  return media;
}

/**
 * Rotates the resting cards, most urgent first.
 *
 * Rotation holds while `paused` — the pointer is on the island, and a card
 * changing under an aiming cursor is the same bug as a menu that moves. The
 * cycle restarts whenever the set of live presences changes, so a new approval
 * takes the chip at once rather than waiting behind a running session.
 */
export function usePresenceRotation(
  cards: readonly IslandPresenceCard[],
  paused: boolean,
): IslandPresenceCard | null {
  const [index, setIndex] = useState(0);
  const key = presenceRotationKey(cards);

  useEffect(() => setIndex(0), [key]);

  useEffect(() => {
    if (paused || cards.length <= 1) return;

    const timer = window.setInterval(
      () => setIndex((current) => nextPresenceIndex(current, cards.length)),
      PRESENCE_ROTATE_MS,
    );
    return () => window.clearInterval(timer);
  }, [cards.length, key, paused]);

  return cards[safePresenceIndex(index, cards.length)] ?? null;
}

/**
 * Trackpad gestures over the island.
 *
 * The listener sits on the island itself rather than the window, so a swipe
 * anywhere else on the stage is somebody else's. Nothing arrives until the
 * stage has taken the pointer back, which is exactly the condition the
 * gestures are supposed to have: the cursor is on the island.
 */
export function useIslandGestures(
  islandRef: RefObject<HTMLElement | null>,
  onGesture: (gesture: IslandGesture) => void,
) {
  const latest = useRef(onGesture);

  useEffect(() => {
    latest.current = onGesture;
  }, [onGesture]);

  useEffect(() => {
    const element = islandRef.current;
    if (!element) return;

    const recognizer = createGestureRecognizer();

    const handleWheel = (event: WheelEvent) => {
      // A scrollable panel under the pointer answers first. Closing the island
      // instead of scrolling its list would put half the queue out of reach.
      const target = event.target;
      if (target instanceof Element) {
        for (let node: Element | null = target; node && element.contains(node); node = node.parentElement) {
          const metrics = {
            overflowY: window.getComputedStyle(node).overflowY,
            scrollTop: node.scrollTop,
            scrollHeight: node.scrollHeight,
            clientHeight: node.clientHeight,
          };
          if (scrollCanConsume(metrics, event.deltaY)) {
            recognizer.reset();
            return;
          }
        }
      }

      // The island is an overlay. A swipe that reached it was meant for it and
      // must not also scroll whatever happens to sit underneath.
      event.preventDefault();

      const gesture = recognizer.read({
        deltaX: event.deltaX,
        deltaY: event.deltaY,
        // Only Chromium reports this, and only on macOS — which is the only
        // place this app runs. Absent, an unflipped trackpad is assumed.
        inverted:
          (event as WheelEvent & { webkitDirectionInvertedFromDevice?: boolean })
            .webkitDirectionInvertedFromDevice === true,
        timeStamp: event.timeStamp,
      });
      if (gesture) latest.current(gesture);
    };

    element.addEventListener("wheel", handleWheel, { passive: false });
    return () => element.removeEventListener("wheel", handleWheel);
  }, [islandRef]);
}

/**
 * Track transport for the horizontal swipe.
 *
 * Only Music and Spotify answer, for the same reason only they name a track:
 * everything else plays through an API Apple no longer opens to third parties.
 * A swipe over a browser therefore does nothing rather than something wrong.
 */
export function useMediaTransport() {
  return useCallback((command: KennelMediaCommand) => {
    const desktop = window.kennelDesktop;
    if (typeof desktop?.sendMediaCommand !== "function") return;
    void desktop.sendMediaCommand(command).catch(() => {
      // A player that refused the command leaves the island as it was. There
      // is nothing to report: the next activity poll tells the truth anyway.
    });
  }, []);
}
