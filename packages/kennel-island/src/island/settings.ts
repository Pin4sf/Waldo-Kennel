/* --------------------------------------------------------------------------
   Settings, renderer side
   --------------------------------------------------------------------------
   The main process owns the settings file and its schema. This module owns the
   renderer's copy of the defaults and the descriptions the settings pane reads
   to draw itself — labels, ranges, and the sentence explaining what a control
   is for.

   The defaults are duplicated rather than imported because the two sides do not
   share a module system: `desktop/settings.mjs` runs in the main process and
   this runs in the renderer. `settings.test.ts` asserts the two stay equal, so
   the duplication cannot drift silently.
   -------------------------------------------------------------------------- */

export const defaultKennelSettings: KennelSettings = {
  notch: {
    widthOffset: 0,
    heightOffset: 0,
    contentPadding: 12,
  },
  hover: {
    peek: true,
    peekWidth: 14,
    peekHeight: 6,
    peekDelayMs: 90,
    openOnHover: false,
    holdOnMouseLeave: false,
    haptics: true,
  },
  gestures: {
    enabled: true,
    verticalOpenClose: true,
    horizontalMedia: true,
    invertMedia: false,
  },
  media: {
    artwork: true,
    waveform: true,
  },
  appearance: {
    calibrating: false,
    demoMode: false,
  },
};

export interface RangeControl<Section extends keyof KennelSettings> {
  kind: "range";
  section: Section;
  field: keyof KennelSettings[Section];
  label: string;
  min: number;
  max: number;
  step: number;
  /** How the current value reads next to the slider. */
  format?: (value: number) => string;
  hint?: string;
}

export interface ToggleControl<Section extends keyof KennelSettings> {
  kind: "toggle";
  section: Section;
  field: keyof KennelSettings[Section];
  label: string;
  hint?: string;
}

export type SettingsControl =
  | RangeControl<"notch">
  | RangeControl<"hover">
  | ToggleControl<"hover">
  | ToggleControl<"gestures">
  | ToggleControl<"media">
  | ToggleControl<"appearance">;

export interface SettingsGroup {
  id: string;
  label: string;
  hint?: string;
  controls: readonly SettingsControl[];
}

export interface SettingsTab {
  id: string;
  label: string;
  groups: readonly SettingsGroup[];
}

/** Signed offsets read better with their sign shown, including at zero. */
function signed(value: number) {
  return value > 0 ? `+${value}` : `${value}`;
}

function points(value: number) {
  return `${value} pt`;
}

/**
 * The settings pane, described rather than written out.
 *
 * A new preference is a row here plus a row in the main process schema. The
 * pane has no per-control code to add, which is the point: the moment a
 * settings window needs bespoke markup per field, it stops being cheap to
 * extend and preferences start living in menus instead.
 */
export const settingsTabs: readonly SettingsTab[] = [
  {
    id: "notch",
    label: "Notch",
    groups: [
      {
        id: "fine-tune",
        label: "Fine tune",
        hint:
          "The island measures your notch from the menu bar, which is accurate to a few points. "
          + "Nudge these until the shape sits exactly on the housing. Turn on the outline below to see the edges while you drag.",
        controls: [
          {
            kind: "range",
            section: "notch",
            field: "widthOffset",
            label: "Width",
            min: -40,
            max: 40,
            step: 1,
            format: signed,
            hint: "Points added to each side.",
          },
          {
            kind: "range",
            section: "notch",
            field: "heightOffset",
            label: "Height",
            min: -12,
            max: 24,
            step: 1,
            format: signed,
            hint: "Points added below the housing.",
          },
        ],
      },
      {
        id: "notch-layout",
        label: "Layout",
        controls: [
          {
            kind: "range",
            section: "notch",
            field: "contentPadding",
            label: "Content padding",
            min: 0,
            max: 32,
            step: 1,
            format: points,
            hint: "Space between the housing and the icons beside it.",
          },
        ],
      },
      {
        id: "calibration",
        label: "Calibration",
        controls: [
          {
            kind: "toggle",
            section: "appearance",
            field: "calibrating",
            label: "Show notch outline",
            hint: "Draws the measured edges over the housing. Turns itself off when this window closes.",
          },
          {
            kind: "toggle",
            section: "appearance",
            field: "demoMode",
            label: "Demo mode",
            hint: "Keeps the island awake with nothing running, so a screen recording can catch it.",
          },
        ],
      },
    ],
  },
  {
    id: "hover",
    label: "Hover",
    groups: [
      {
        id: "peek",
        label: "Peek",
        hint:
          "A quiet island is exactly the size of the camera housing, so it disappears into the hardware. "
          + "The peek is how it answers the pointer without committing to a panel.",
        controls: [
          {
            kind: "toggle",
            section: "hover",
            field: "peek",
            label: "Swell under the pointer",
          },
          {
            kind: "range",
            section: "hover",
            field: "peekWidth",
            label: "Peek width",
            min: 0,
            max: 48,
            step: 1,
            format: points,
          },
          {
            kind: "range",
            section: "hover",
            field: "peekHeight",
            label: "Peek height",
            min: 0,
            max: 24,
            step: 1,
            format: points,
          },
          {
            kind: "range",
            section: "hover",
            field: "peekDelayMs",
            label: "Delay",
            min: 0,
            max: 600,
            step: 10,
            format: (value) => `${value} ms`,
            hint: "How long the pointer has to rest on the notch before it answers.",
          },
          {
            kind: "toggle",
            section: "hover",
            field: "haptics",
            label: "Haptic feedback",
            hint: "A tap on a Force Touch trackpad as the island answers.",
          },
        ],
      },
      {
        id: "opening",
        label: "Opening",
        controls: [
          {
            kind: "toggle",
            section: "hover",
            field: "openOnHover",
            label: "Open on hover",
            hint: "Skip the peek and open the full panel. Disables the peek above.",
          },
          {
            kind: "toggle",
            section: "hover",
            field: "holdOnMouseLeave",
            label: "Stay open when the pointer leaves",
            hint: "An open panel waits for Escape or a click instead of closing behind you.",
          },
        ],
      },
    ],
  },
  {
    id: "media",
    label: "Media",
    groups: [
      {
        id: "artwork",
        label: "Album art",
        hint:
          "The peek takes its colour from whatever is playing. Music hands the artwork over directly; "
          + "Spotify only names a URL, so its art costs one request to Spotify's own image server per track.",
        controls: [
          {
            kind: "toggle",
            section: "media",
            field: "artwork",
            label: "Fetch album art from Spotify",
            hint: "Off means no request leaves the machine, and Spotify tracks keep the default colour.",
          },
          {
            kind: "toggle",
            section: "media",
            field: "waveform",
            label: "Animate the waveform",
            hint: "The bars are generated, not measured — reading the real output level needs a capture permission.",
          },
        ],
      },
    ],
  },
  {
    id: "gestures",
    label: "Gestures",
    groups: [
      {
        id: "trackpad",
        label: "Trackpad",
        hint: "Two-finger swipes over the island itself. A swipe anywhere else belongs to whatever is underneath.",
        controls: [
          {
            kind: "toggle",
            section: "gestures",
            field: "enabled",
            label: "Allow gestures on the island",
          },
          {
            kind: "toggle",
            section: "gestures",
            field: "verticalOpenClose",
            label: "Swipe down to open, up to close",
          },
          {
            kind: "toggle",
            section: "gestures",
            field: "horizontalMedia",
            label: "Swipe across to change track",
            hint: "Only Music and Spotify answer; everything else plays through an API Apple no longer opens.",
          },
          {
            kind: "toggle",
            section: "gestures",
            field: "invertMedia",
            label: "Invert track direction",
          },
        ],
      },
    ],
  },
];

/** The current value of whatever field a control describes. */
export function controlValue(settings: KennelSettings, control: SettingsControl) {
  // Every settings section is a flat record of booleans and numbers, but the
  // union of the four has no index signature, so the read is widened once here
  // rather than at each of the three call sites.
  const section = settings[control.section] as unknown as Record<string, boolean | number>;
  return section[control.field as string];
}

/** A patch that sets one control, ready for `updateSettings`. */
export function controlPatch(control: SettingsControl, value: boolean | number): KennelSettingsPatch {
  return { [control.section]: { [control.field as string]: value } } as KennelSettingsPatch;
}

/**
 * Whether a control can currently be changed.
 *
 * "Open on hover" makes the peek unreachable, and a slider that still moves
 * while nothing responds to it is a worse explanation than a disabled one.
 */
export function controlDisabled(settings: KennelSettings, control: SettingsControl) {
  if (control.section !== "hover") {
    return control.section === "gestures" && control.field !== "enabled"
      ? !settings.gestures.enabled
      : false;
  }

  const peekControls = new Set(["peek", "peekWidth", "peekHeight", "peekDelayMs"]);
  if (!peekControls.has(control.field as string)) return false;
  if (settings.hover.openOnHover) return true;
  return control.field !== "peek" && !settings.hover.peek;
}
