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
// It is also a real application, not a bare command-line tool. NSHapticFeedbackManager
// resolves its performer through the AppKit application object; a process that
// never finished NSApplication startup gets a performer that accepts every call
// and taps for none of them, which is a silent failure and was one. `.accessory`
// keeps it out of the Dock and the app switcher while still being an app.
//
// Protocol: one pattern name per line on stdin, one tap out. Unknown names fall
// back to `alignment`, which is the softest of the three and the right default
// for an edge snapping into place.

import AppKit

setbuf(stdout, nil)

let application = NSApplication.shared
application.setActivationPolicy(.accessory)
application.finishLaunching()

func pattern(named name: String) -> NSHapticFeedbackManager.FeedbackPattern {
    switch name {
    case "generic": return .generic
    case "level": return .levelChange
    default: return .alignment
    }
}

// stdin is read off the main thread so the main thread can stay on the run loop.
// AppKit performs haptics from the main thread and nowhere else: a tap
// dispatched from the reader thread lands on a performer that is not the
// application's, which is the same silent no-op as never having launched.
let reader = Thread {
    while let line = readLine(strippingNewline: true) {
        let name = line.trimmingCharacters(in: .whitespaces)
        DispatchQueue.main.async {
            // `.now` and not `.drawCompleted`: the caller writes the line as the
            // animation starts, so deferring to the next draw would land the tap a
            // frame into a shape that is already moving.
            NSHapticFeedbackManager.defaultPerformer.perform(pattern(named: name), performanceTime: .now)
        }
    }
    // The island closed the pipe, or exited. Nothing left to feel.
    DispatchQueue.main.async { application.terminate(nil) }
}
reader.start()

application.run()
