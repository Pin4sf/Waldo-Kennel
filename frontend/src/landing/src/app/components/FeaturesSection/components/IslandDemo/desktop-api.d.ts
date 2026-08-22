import type {
  KennelApprovalResolution,
  KennelConversationRequest,
  KennelConversationState,
  KennelDesktopSnapshot,
  KennelInputResolution,
  KennelInterruptRequest,
  KennelOpenRequest,
  KennelSteerRequest,
} from "./island/kennel-contract";

declare global {
  interface KennelStageGeometry {
    stageWidth: number;
    stageHeight: number;
    hasNotch: boolean;
    notchWidth: number;
    notchHeight: number;
    menuBarHeight: number;
    scaleFactor: number;
  }

  interface KennelMediaTrack {
    title: string;
    artist: string;
    /**
     * Album art as a `data:` URI, when the player hands it over.
     *
     * A data URI and not a path or a URL: the renderer runs under a strict CSP
     * that allows `img-src 'self' data:` and nothing else, and a canvas reading
     * a data URI is never tainted, which is what lets the accent colour be
     * sampled from it.
     */
    artwork?: string;
  }

  interface KennelMediaActivity {
    playing: boolean;
    /** Null whenever the source will not name itself, which is every browser. */
    track: KennelMediaTrack | null;
  }

  /** Transport the island can ask the host to send to the current player. */
  type KennelMediaCommand = "next" | "previous" | "play-pause";

  /** Force Touch patterns the host will perform, softest first. */
  type KennelHapticPattern = "alignment" | "generic" | "level";

  interface KennelNotchSettings {
    /** Points added to each side of the derived notch width. */
    widthOffset: number;
    /** Points added to the derived notch height. */
    heightOffset: number;
    /** Points between the housing and a header cluster. */
    contentPadding: number;
  }

  interface KennelHoverSettings {
    /** The dormant notch swells under the pointer before it opens. */
    peek: boolean;
    peekWidth: number;
    peekHeight: number;
    /** Pointer dwell before the peek commits, in milliseconds. */
    peekDelayMs: number;
    /** Skip the peek and open the full panel on hover alone. */
    openOnHover: boolean;
    /** Keep an open panel up after the pointer leaves. */
    holdOnMouseLeave: boolean;
    haptics: boolean;
  }

  interface KennelGestureSettings {
    enabled: boolean;
    verticalOpenClose: boolean;
    horizontalMedia: boolean;
    invertMedia: boolean;
  }

  interface KennelAppearanceSettings {
    /** Draw the calibration outline over the housing. */
    calibrating: boolean;
    /** Keep the island awake so a screen recording can catch it. */
    demoMode: boolean;
  }

  interface KennelMediaSettings {
    /** Fetch album art from the player's CDN. The only outbound request made. */
    artwork: boolean;
    waveform: boolean;
  }

  interface KennelSettings {
    notch: KennelNotchSettings;
    hover: KennelHoverSettings;
    gestures: KennelGestureSettings;
    media: KennelMediaSettings;
    appearance: KennelAppearanceSettings;
  }

  /** A patch names only the fields it changes. */
  type KennelSettingsPatch = {
    [Section in keyof KennelSettings]?: Partial<KennelSettings[Section]>;
  };

  interface KennelDesktopAPI {
    setInteractive: (interactive: boolean) => Promise<{ interactive: boolean }>;
    getStageGeometry: () => Promise<KennelStageGeometry | null>;
    onStageGeometry: (listener: (geometry: KennelStageGeometry) => void) => () => void;
    getMediaActivity: () => Promise<KennelMediaActivity>;
    onMediaActivity: (listener: (activity: KennelMediaActivity) => void) => () => void;
    sendMediaCommand: (command: KennelMediaCommand) => Promise<{ sent: boolean }>;
    recenter: () => void | Promise<void>;
    getKennelSnapshot: () => Promise<KennelDesktopSnapshot>;
    getKennelConversation: (input: KennelConversationRequest) => Promise<KennelConversationState>;
    resolveApproval: (input: KennelApprovalResolution) => Promise<unknown>;
    resolveInput: (input: KennelInputResolution) => Promise<unknown>;
    steer: (input: KennelSteerRequest) => Promise<unknown>;
    interrupt: (input: KennelInterruptRequest) => Promise<unknown>;
    openKennel: (input?: KennelOpenRequest) => Promise<unknown>;
    getSettings: () => Promise<KennelSettings>;
    updateSettings: (patch: KennelSettingsPatch) => Promise<KennelSettings>;
    resetSettings: () => Promise<KennelSettings>;
    onSettings: (listener: (settings: KennelSettings) => void) => () => void;
    openSettings: () => Promise<{ open: boolean }>;
    closeSettings: () => Promise<{ open: boolean }>;
    performHaptic: (pattern: KennelHapticPattern) => Promise<{ performed: boolean }>;
  }

  interface Window {
    kennelDesktop?: KennelDesktopAPI;
  }
}

export {};
