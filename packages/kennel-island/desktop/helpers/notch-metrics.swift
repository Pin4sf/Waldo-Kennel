// Measures the built-in display's camera housing.
//
// macOS knows exactly where the notch is. `NSScreen.auxiliaryTopLeftArea` and
// `auxiliaryTopRightArea` are the two strips of menu bar either side of it, so
// the housing is the gap between them — an exact figure in points, not an
// estimate. Electron bridges neither property, which is why the island has been
// deriving a width from the menu bar's height instead.
//
// This prints the real numbers so it does not have to. One JSON object on
// stdout, then exit; anything the caller cannot parse means "no measurement",
// and the island falls back to the derivation it already had.

import AppKit

func measurement(for screen: NSScreen) -> [String: Any] {
    var result: [String: Any] = [
        "menuBarHeight": Int((screen.frame.height - screen.visibleFrame.height).rounded()),
        "scaleFactor": screen.backingScaleFactor,
    ]

    guard
        let left = screen.auxiliaryTopLeftArea,
        let right = screen.auxiliaryTopRightArea
    else {
        // A flat bezel has no auxiliary areas: the whole menu bar is one strip.
        result["hasNotch"] = false
        return result
    }

    result["hasNotch"] = true
    result["notchWidth"] = Int((right.minX - left.maxX).rounded())
    // The housing is as tall as the strips beside it.
    result["notchHeight"] = Int(left.height.rounded())
    return result
}

// The island lives on the built-in display, so that is the one measured — not
// whichever screen happens to be main when this runs.
let screen = NSScreen.screens.first { $0.safeAreaInsets.top > 0 } ?? NSScreen.main

guard
    let screen,
    let data = try? JSONSerialization.data(withJSONObject: measurement(for: screen)),
    let json = String(data: data, encoding: .utf8)
else {
    exit(1)
}

print(json)
