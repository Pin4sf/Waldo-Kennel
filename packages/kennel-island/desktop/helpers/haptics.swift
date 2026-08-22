// Force Touch feedback for the island.
//
// AppKit performs haptics through NSHapticFeedbackManager, which Electron does
// not bridge. Rather than a native addon — which would put a compiled binary in
// the dependency tree and a rebuild in every install — the island talks to this
// helper over a pipe.
//
// The helper is resident, not per-tap. Spawning a process costs tens of
// milliseconds, and feedback that arrives tens of milliseconds after the shape
// moved is feedback the hand reads as a second, unrelated event. Staying
// resident makes a tap a single line write.
//
// Protocol: one pattern name per line on stdin, one tap out. Unknown names fall
// back to `alignment`, which is the softest of the three and the right default
// for an edge snapping into place.

import AppKit

setbuf(stdout, nil)

let performer = NSHapticFeedbackManager.defaultPerformer

func pattern(named name: String) -> NSHapticFeedbackManager.FeedbackPattern {
    switch name {
    case "generic": return .generic
    case "level": return .levelChange
    default: return .alignment
    }
}

while let line = readLine(strippingNewline: true) {
    let name = line.trimmingCharacters(in: .whitespaces)
    // `.now` and not `.drawCompleted`: the caller writes the line as the
    // animation starts, so deferring to the next draw would land the tap a
    // frame into a shape that is already moving.
    performer.perform(pattern(named: name), performanceTime: .now)
}
