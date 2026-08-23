# Dynamic Island and Live Activities reference for Kennel

- Status: source-backed research plus a local reference-fixture gallery in the canonical Island; no vendor integration, daemon contract, or production live-data authorization
- Research date: 2026-08-22
- Shipping baseline: iOS 26
- Announced baseline: iOS 27 beta is isolated in a separate watchlist
- External scope: representative third-party iOS apps, not a complete App Store inventory
- Kennel baseline: the canonical unified implementation in [packages/kennel-island](../../packages/kennel-island)
- Source access date unless otherwise stated: 2026-08-22

> **Research boundary:** “Imitate” means adapting behavior, information hierarchy, lifecycle, and attention rules. The companion gallery uses fixed local fixtures, generic labels, and code-native symbols; it does not copy Apple or vendor visuals, copy, code, trademarks, or private behavior. Connecting any pattern to production data still requires a separately reviewed implementation contract.

## What people actually see in the Dynamic Island

The Dynamic Island is the small black area around the front camera on supported iPhones. Most of the time it is quiet. When something is happening, it can show a compact fact such as a timer, recording waveform, delivery stage, ride ETA, flight gate, score, workout metric, or charging state. Touching and holding it reveals more detail. Tapping normally opens the relevant place in the app.

The familiar experiences fall into two practical groups:

### Apple and system experiences

| Familiar activity | What remains visible | Typical expanded value | Evidence posture |
| --- | --- | --- | --- |
| **Voice recording** | Recording is active; a waveform or elapsed state makes that unmistakable. | Recording identity and controls. | **Apple documented** for Voice Memos in the iOS 26 iPhone guide. Exact public fields are not documented. |
| **Phone or FaceTime call** | An active call and its duration/status. | Call controls and participant context. | System-owned experience; the current public sources do not establish it as ActivityKit. |
| **Music / Now Playing** | Artwork or playback state on one side and an animated audio cue on the other. | Track context and playback controls. | **Older reference** from Apple's 2022 Dynamic Island launch; interaction reference only. |
| **Timer** | Remaining time in a glanceable countdown. | Timer context and controls. | **Older reference** from Apple's 2022 launch and current HIG examples. |
| **Maps directions** | Next direction or route progress while another app is open. | More route context and the next instruction. | **Apple documented** in the iOS 26 iPhone guide; system-owned. |
| **AirDrop** | A bounded send/receive connection or transfer state. | Transfer detail and completion/failure context. | **Apple documented** in the iOS 26 iPhone guide; system-owned. |
| **Face ID and privacy/mode feedback** | A brief confirmation or indicator, not a persistent activity. | Usually no ongoing expanded state. | Face ID is an **Older reference**; privacy/mode examples are classified as **Inference**. |

These system experiences are useful references for timing, hierarchy, and restraint. They do **not** prove that third-party apps receive the same controls, priority, persistence, or private system integration.

### Third-party Live Activities

| Familiar activity | Representative apps | What the Island tracks |
| --- | --- | --- |
| **Food delivery and order tracking** | Domino's | Order stage, driver movement, and useful ready/delivery timing. |
| **Rides** | Uber | Pickup or destination ETA, trip phase, driver/vehicle context, and progress. |
| **Transit navigation** | Transit GO | Leave-now prompts, stops remaining, next stop, and get-off warnings. |
| **Flights** | Flighty | Departure timing, gate changes, taxi time, flight progress, and arrival timing. |
| **Live sports** | MLB, FotMob | Score, inning or match context, and important game changes. |
| **Focus and time tracking** | Structured, Timery | Remaining focus time or elapsed tracked time, project/task identity, and Stop. |
| **Workouts** | SmartGym | Active workout context, changing metrics, rest/work state, and the next useful action. |
| **EV charging** | Tesla | The active Supercharging session and available charging information. |
| **Camera recording** | Insta360 | Connection/recording state, Start/Stop, and battery, heat, or storage errors. |
| **Weather** | CARROT Weather | Imminent rain/snow timing and a short precipitation chart. |
| **Smart-home automation** | Home Assistant Labs | A user-defined phase, progress bar, countdown/count-up, status, or critical text. |

This is the central research inventory. The detailed catalog below records exactly what each source proves, what remains unknown, and which claims are vendor- or store-supplied.

## The common interaction shape

Across these activities, the Island usually follows the same sequence:

1. A real event starts: recording begins, an order is placed, a ride is accepted, a timer starts, or navigation begins.
2. The compact Island keeps one changing fact visible: elapsed time, ETA, score, stage, instruction, or metric.
3. The expanded Island adds context and, sometimes, one immediate action.
4. A delay, failure, warning, or required instruction temporarily takes priority.
5. The activity ends with a clear completion state and then leaves the Island.

The reusable idea is not a particular app's visual design. It is the information hierarchy: **identity → current state → changing fact → next action or exception → completion**.

## Evidence labels

| Label | Meaning in this dossier |
| --- | --- |
| **Apple documented** | Current first-party Apple documentation or an Apple platform guide supports the claim. |
| **Vendor documented** | A first-party vendor page, help article, release note, manual, or engineering post supports the claim. |
| **Store claimed** | A current App Store listing or version note supplied by the developer supports the claim. Apple has not independently verified the product or privacy statement. |
| **Older reference** | A dated first-party source establishes a historical design or behavior but is not, by itself, proof of current compatibility. |
| **Beta** | The behavior is announced for prerelease software or is distributed through a beta/TestFlight surface. |
| **Inference** | A bounded interpretation of documented evidence, not a directly stated behavior. |
| **Proposed** | A Kennel-specific recommendation, not an Apple, vendor, or shipped Kennel capability. |

An undated living page is recorded as “undated; accessed 2026-08-22.” That access date establishes the research snapshot, not the page's publication date. App Store statements remain **Store claimed**, including their privacy declarations.

## Scope and terminology

### Three mechanisms share the Dynamic Island

| Mechanism | Owner and lifecycle | Representative experiences | Relevance to Kennel |
| --- | --- | --- | --- |
| **1. App-defined ActivityKit Live Activity** | A third-party app defines static attributes, dynamic content, presentations, updates, actions, freshness, and ending through ActivityKit, WidgetKit, SwiftUI, the host app, and optional ActivityKit push notifications. | Rides, orders, navigation, flights, scores, focus sessions, workouts, charging, recording, precipitation, and beta home automations. | Primary behavioral reference. Kennel can adapt phase progression, glanceable facts, bounded actions, stale state, and completion handling. |
| **2. System-owned ongoing experience** | iOS owns the experience and may use private system integration rather than the public ActivityKit contract. | Maps, calls, Timer, Now Playing/Music, Voice Memos, and AirDrop. | Interaction reference only. Do not infer that Kennel or a third-party iOS app can reproduce system-only controls, priority, or persistence. |
| **3. Transient system feedback** | iOS briefly expands or marks the surface to confirm a system event, permission, sensor, or mode change. It is not a long-running ActivityKit event. | Face ID, camera/microphone privacy indicators, and mode-change feedback. | Reference for short confirmation and restraint, not for persistent tracking. |

**Apple documented; 2026 snapshot.** The [iOS 26 iPhone guide](https://support.apple.com/guide/iphone/view-live-activities-in-the-dynamic-island-iph28f50d10d/26/ios/26) explicitly names Voice Memos, AirDrop, and Maps as current Dynamic Island experiences. Apple's [iPhone 14 Pro launch reference](https://www.apple.com/ie/newsroom/2022/09/apple-debuts-iphone-14-pro-and-iphone-14-pro-max/) documents Maps, Music, Timer, alerts, notifications, and Face ID as surface examples. The latter is an **Older reference** from 2022 and is used only to classify interaction types.

ActivityKit also has a feature named a **transient Live Activity**. Despite the name, it belongs to mechanism 1: it is an app-defined expanded presentation that ends when the person locks the device, collapses the Island, leaves the app, or interacts elsewhere. Its public API is stable from iOS 18; it is not the same as mechanism 3 and is not an iOS 27 beta feature.

### What counts as a reusable Live Activity

A reusable activity has a stable identity, a defined start, changing factual state, a defined end, and a reason to remain glanceable between those points. A notification is better for a single event. A widget is better for ambient information without a short event lifecycle. A full app is better when the user must read sensitive detail, compare many items, enter unbounded text, or make a consequential decision.

Apple Watch, Mac, StandBy, and CarPlay appear here only where they clarify the same Live Activity model. They are not separate product benchmarks.

## Shipping iOS 26: Apple platform model

The current model below is based on the [Live Activities HIG](https://developer.apple.com/design/human-interface-guidelines/live-activities), last changed by Apple on 2025-12-16, the [ActivityKit overview](https://developer.apple.com/documentation/activitykit), the [displaying-live-data guide](https://developer.apple.com/documentation/activitykit/displaying-live-data-with-live-activities), and the version-pinned [iOS 26 iPhone guide](https://support.apple.com/guide/iphone/view-live-activities-in-the-dynamic-island-iph28f50d10d/26/ios/26).

### Presentations and concurrency

| Presentation | Shipping behavior | Reusable design lesson |
| --- | --- | --- |
| **Compact** | With one visible Live Activity, the system composes separate leading and trailing views around the TrueDepth camera into one coherent activity. Tapping opens the related app location. | The resting state should carry one identity/context cue and one changing fact. |
| **Minimal** | When activities from multiple apps compete, the Island can show two minimal presentations, one attached and one detached. Two is the visible presentation count, not a documented maximum number of running activities. | Each activity still needs a recognizable icon or metric when very little space is available. Related sub-events should be aggregated. |
| **Expanded** | Touching and holding a compact or minimal activity opens a larger view around the sensors. Essential alerts can also present the expanded state. | Use expansion for the next useful details and one safe action; do not turn it into a miniature dashboard. |
| **Lock Screen** | Every activity needs a Lock Screen presentation. It is also used as a banner on devices without the Dynamic Island when an alerting update arrives. | The detailed state must remain coherent without relying on compact geometry. |
| **StandBy** | The minimal form appears in StandBy; tapping expands the Lock Screen presentation to fill the display. | A minimal state must still explain identity and current phase. |
| **Other devices** | The same iPhone/iPad activity can project into Apple Watch Smart Stack, Mac menu bar, and CarPlay Dashboard. CarPlay does not execute Live Activity buttons or toggles. | Define the fact hierarchy before surface-specific layout. Do not assume actions are portable everywhere. |

**Apple documented.** An app may start several activities and a device may run activities from several apps, but the simultaneous active/scheduled limit is device-dependent and unspecified. If one app has several activities, ActivityKit's relevance score determines its display order and which one appears in the Island; equal scores favor the activity started first.

**Apple documented design guidance, not a requirement.** The HIG recommends one activity with a dynamic layout that efficiently rotates through related events instead of many separate activities. The important lesson is to aggregate related work rather than issuing an activity for every sub-event.

### Interaction model

| Interaction | Shipping behavior | Reusable boundary |
| --- | --- | --- |
| Tap | Opens the app at the activity's related scene; expanded and Lock Screen layouts can use context-specific links. | Deep-link to the exact item represented by the activity. |
| Touch and hold | Expands compact or minimal content. | Reveal the next useful details rather than all logs. |
| Swipe outward/inward | The iOS 26 guide documents swipe outward to expand and inward to collapse. | Existing Kennel expansion/collapse maps well to the gesture hierarchy. |
| Side swipe | Switches between the two visible activities. | Arbitration between concurrent items is a system responsibility; an imitation may use a queue or rotation instead of copying the gesture. |
| Button or toggle | App Intents can perform quick activity-related actions without launching the app. The HIG recommends direct, essential actions and preferably one interactive element. | Keep only fresh, one-step, reversible or clearly understood actions inline. Open the full app for sensitive or incomplete context. |
| Dismissal | A person can remove the activity. Removing it does not cancel the ride, order, workout, or other underlying event. | Hiding the presentation must not cancel the underlying task. Cancellation remains an explicit action. |

### Lifecycle, updates, stale state, and dismissal

| Concern | Apple behavior | Reusable lesson |
| --- | --- | --- |
| Start | Standard activities normally start while the app is foregrounded, through an App Intent, or remotely with an ActivityKit push-to-start notification. Scheduled activities and the pending state are stable in iOS 26 and require an alert configuration. | Start only when a trackable event genuinely begins. A notification is better for an isolated event with no continuing state. |
| Update | The host app or ActivityKit APNs notifications update dynamic content. The Live Activity extension itself has no network or location access. Static plus dynamic data is capped at 4 KB. | Keep a bounded projection of authoritative facts; do not treat the presentation as the source of truth. |
| Material alert | An alert can light the screen, play the default notification sound, and show expanded content. Apple says to alert only for essential updates and not duplicate the same event with a push notification. | Alert for a meaningful exception, urgent instruction, or completion—not routine updates or unchanged polling. |
| Freshness | Activity content can carry a stale date. At that time ActivityKit changes the state to stale and the view should say that the information is outdated. | Make stale, offline, or unavailable information explicit rather than leaving an old value looking current. |
| Standard lifetime | A Live Activity can remain active for up to 8 hours. The system then removes it from Dynamic Island immediately; it may remain on the Lock Screen for up to 4 more hours, for a maximum 12-hour Lock Screen presence. | Long-running activity still needs defined phases, a freshness signal, and an end condition. |
| End | End immediately when the task or event ends and provide final content. Dynamic Island and CarPlay remove it immediately. Lock Screen, Mac, and Watch may retain it for up to 4 hours. | Do not leave a finished event pretending to be active. End it and provide a concise final state where the surface permits. |
| Receipt | The HIG says 15–30 minutes is adequate in most cases for a post-end summary on surfaces that retain it; this is guidance, not a hard limit. | A short completion receipt is useful, but it is not evidence that ended content remains in the Dynamic Island itself. |

The HIG's expanded and Lock Screen presentations are 84–160 points high in current iOS specifications. Content above the maximum can be truncated. Image assets must fit their target presentation; Apple's minimal example must not exceed 45 × 36.67 points. These are platform constraints, not target pixel measurements for Kennel's macOS notch surface.

### Privacy and attention rules

**Apple documented.**

- Do not use Live Activities for advertising or promotion.
- Avoid sensitive content because the surface is visible on Lock Screen and Always-On displays. Prefer an innocuous summary, redaction, and a tap into the authenticated app.
- Update only when the underlying content or status changes.
- Alert only for essential changes that a person should not miss.
- Make the compact view legible, concise, and recognizable.
- Deep-link directly to the related detail instead of making the person search.
- End the activity when its underlying event ends.
- Let people turn Live Activities off, and handle disabled or exhausted activity capacity gracefully.

These rules map directly to Kennel: raw prompts, commands, paths, private file names, provider payloads, and unrestricted approval context do not belong on the resting surface.

### Apple-owned reference experiences

| Experience | Mechanism and tracked fact | Evidence and limitation |
| --- | --- | --- |
| Maps directions | System-owned ongoing experience; navigation remains visible while another app is active. | **Apple documented** in the iOS 26 guide. Apple does not state that Maps uses public ActivityKit, so no private fields or controls are inferred. |
| Voice Memos | System-owned recording-in-progress state. | **Apple documented** in the iOS 26 guide. Useful reference for persistent recording identity, not proof of third-party recording privileges. |
| AirDrop | System-owned connection/progress experience. | **Apple documented** in the iOS 26 guide. Useful reference for a bounded transfer and exception lifecycle. |
| Music / Now Playing | System-owned playback and controls. | **Older reference** in Apple's 2022 iPhone 14 Pro launch; current HIG also uses media playback as a simple-action example. |
| Timer | System-owned countdown that keeps the remaining time recognizable in minimal form. | **Older reference** in Apple's 2022 launch; current HIG uses Timer as the minimal-presentation example. |
| Calls | System-owned communication alert/ongoing control surface. | **Older reference** to Apple's description of alerts, notifications, and activities on the Island. It is not a public ActivityKit capability claim. |
| Face ID | Short system feedback that expands only to confirm authentication. | **Older reference** explicitly documented in Apple's 2022 launch. |
| Privacy indicators and mode feedback | Short OS-owned indication adjacent to or within the same physical surface. | **Inference** used only to distinguish transient system feedback from a Live Activity; no third-party control is implied. |

### Apple first-party tracked-event references

These examples establish additional event shapes in Apple's own ecosystem. They do not all establish an exact Dynamic Island presentation. Where Apple documents only Lock Screen or Watch behavior, Island placement is not promoted from generic ActivityKit capability to a product-specific claim.

| Experience | Tracked event and fields | Evidence and limitation |
| --- | --- | --- |
| Apple Sports | Live score, plays, and game clock. Activities can be scheduled before a game and enabled automatically for followed teams, with an update-rate preference. | [Apple Sports guide](https://support.apple.com/en-lamr/guide/apple-sports-app/apdc0cb7ad64/web) and [support article](https://support.apple.com/en-us/116979), published 2025-11-17 — **Apple documented**. Apple explicitly documents Lock Screen and Watch; product-specific Island placement remains **Inference** from the platform model. |
| Wallet flight tracking | Flight status, departure/arrival times, gates, delays, and boarding information. A person can share the Live Activity through Messages so another iPhone follows the flight. | [Wallet flight support](https://support.apple.com/en-us/123179), published 2025-10-29, [semantic boarding passes](https://developer.apple.com/documentation/walletpasses/creating-an-airline-boarding-pass-using-semantic-tags), and [WWDC25 Wallet](https://developer.apple.com/videos/play/wwdc2025/202/) — **Apple documented**. |
| Wallet event tickets | A semantic poster ticket can start a Live Activity at the relevant time and show primary seating/entry information. | [WWDC24 Wallet session](https://developer.apple.com/videos/play/wwdc2024/10108/), June 2024 — **Apple documented; Older reference**. Explicit examples are Lock Screen and Watch, not a separately documented Island layout. |
| Cycling workout | Starting a cycling workout on Apple Watch automatically creates an iPhone Live Activity; tapping opens a full-screen workout view with metrics. | [watchOS 10 availability](https://www.apple.com/newsroom/2023/09/watchos-10-is-available-today/), 2023-09-18, and [current Watch guide](https://support.apple.com/en-ie/guide/watch/apd4cbc876c7/watchos) — **Apple documented; Older reference** for the launch date. |
| Urgent reminder | In iOS 26.2, an urgent reminder alarm that is not completed supports snooze and a Live Activity. | [iOS 26 update history](https://support.apple.com/en-us/123075) — **Apple documented**. It is a useful reference for an activity that escalates when action becomes overdue. |
| Messages Check In | A time-based Check In uses a Lock Screen Live Activity and response confirmation. | [iOS 17 feature guide](https://www.apple.com/mideast/ios/ios-17/a/pdf/iOS_All_New_Features.pdf), September 2023 — **Apple documented; Older reference**. A product-specific Dynamic Island layout is not established. |

## Reusable pattern taxonomy

| Pattern | What the surface answers | Representative examples | What makes the pattern useful |
| --- | --- | --- | --- |
| **Countdown or ETA** | When is the next meaningful change? | Uber pickup/destination ETA, Transit leave/get-off countdowns, Flighty departure/taxi/arrival time, CARROT precipitation timing. | It turns time into the primary glanceable fact. It only works when the source provides a real clock or ETA. |
| **Phase progression** | Which discrete stage is active, and what comes next? | Domino's order stages, Structured focus, Home Assistant automation states, Uber ride phases. | It explains progress without requiring a percentage: placed → prepared → out for delivery, or ready → recording → stopped. |
| **Continuous progress or metric** | How much factual work or measurement has accumulated? | Timery duration, SmartGym workout data, Tesla charging session, Flighty arrival progress. | It suits measured quantities such as elapsed duration, distance, charge, or a source-provided progress value. |
| **Event stream or score** | What changed in a fast-moving event? | MLB score/inning/key updates and FotMob live score. | It keeps the latest important result visible while allowing significant changes to take temporary priority. |
| **Next instruction/action** | What should the person do now? | Transit leave/get-off prompts, SmartGym Predictive Action, Insta360 start/stop recording. | It combines a short instruction with one safe, immediate control or a direct app link. |
| **Exception requiring attention** | What failed or is unsafe to ignore? | Flighty gate change, Transit hurry/get-off, Insta360 heat/battery/SD-card errors, Home Assistant critical text. | It briefly replaces ordinary progress with the specific delay, warning, or corrective action. |
| **Completion receipt** | What finished, and is there anything left to do? | Order/ride/workout end states plus Apple's retained Lock Screen summary guidance. | It confirms the end state without leaving a finished activity permanently active. |
| **Multiple concurrent activities** | Which event deserves the resting surface, and where are the others? | ActivityKit minimal presentations, relevance score, and HIG single-activity aggregation guidance. | It forces prioritization and recognizable minimal forms instead of showing every sub-event at once. |

## External evidence catalog by primary interaction pattern

Each case records the same comparison fields. “Not documented” is intentional; it prevents a product screenshot, in-app feature, or reasonable guess from becoming a false Live Activity claim.

### Countdown or ETA

#### Uber Rider

| Field | Recorded behavior |
| --- | --- |
| Identity/context | Active Rider trip with driver, vehicle, pickup, and destination context. |
| Phases | Waiting for pickup and on-trip are directly discussed; completed is the intended terminal trip state. |
| Time field | Pickup ETA and destination ETA. |
| Progress representation | Overall trip progress. |
| Live metric | Current ETA plus license plate, vehicle image, driver name, and driver image. |
| Exception states | No user-facing Live Activity exception taxonomy is documented. Uber's engineering post discusses stale/end reliability and debounced backend updates. |
| Immediate action | Current first-party news says a rider can tip from the Lock Screen before the trip ends. |
| Start trigger | Tied to an active Rider trip; the exact creation trigger is not documented. |
| End condition | Intended to end when the trip completes; exact dismissal timing is not documented. |
| Deep link | Not documented in the cited current sources. |
| Privacy risk | Driver identity, vehicle identity, plate, and trip timing are intentionally visible outside the app; no Live Activity redaction mode is documented. |
| Evidence / limit | [Uber engineering case study](https://www.uber.com/gb/en/blog/live-activity-on-ios/), 2024-07-25 — **Vendor documented; Older reference**. [Holiday/tipping update](https://www.uber.com/us/en/newsroom/help-for-the-holidays/), 2025-12-09 — **Vendor documented**. The engineering article proves shipped architecture at that date, not 2026 compatibility. |

#### Transit GO

| Field | Recorded behavior |
| --- | --- |
| Identity/context | One actively navigated transit trip started with GO. |
| Phases | Leave soon, hurry/leave now, riding with stops remaining, get off soon, destination. |
| Time field | Countdown until departure or action. |
| Progress representation | Entire-trip progress and stops remaining. |
| Live metric | Next stop, destination stop, and the number of stops remaining. |
| Exception states | Hurry, leave now, and get-off states enlarge the Island because action is imminent. |
| Immediate action | No inline button is documented; tapping opens the full GO trip. |
| Start trigger | The rider explicitly taps GO. |
| End condition | GO and its widget dismiss automatically at the destination. |
| Deep link | Tap opens the corresponding trip in Transit. |
| Privacy risk | Transit says exact position is not shared with other riders; vehicle position is shared only while GO is active and the user is aboard. Precise trip-history collection is separately opt-in. |
| Evidence / limit | [How to use GO](https://help.transitapp.com/article/549-how-to-use-go), updated 2026-08-20 — **Vendor documented**. Trips must be within the next 60 minutes; multimodal trips are unsupported; background location affects battery. |

#### Flighty

| Field | Recorded behavior |
| --- | --- |
| Identity/context | One active flight with route, departure, gate, taxi, and arrival context. |
| Phases | **Inference:** pre-departure, gate/taxi, in flight, arrival. Current help lists the changing facts but not a formal phase enum. |
| Time field | Departure countdown, taxi time, and arrival timing. |
| Progress representation | Arrival progress; a 2025 redesign note also describes a visual flight path. |
| Live metric | Gate changes, taxi time, and arrival progress. |
| Exception states | Gate changes are the explicitly documented attention event. |
| Immediate action | None documented. |
| Start trigger | Help says the activity updates in the background once a flight is active; the exact activation control is not documented. |
| End condition | Not documented. |
| Deep link | Not documented. |
| Privacy risk | Flight identity and itinerary are visible. Flighty says an account is not required and its optional location feature is not sold/shared and is deleted outside an airport; App Store privacy still lists data that may be linked when relevant features are used. |
| Evidence / limit | [Flighty Live Activities help](https://flighty.com/help/live-activities-widgets), undated, accessed 2026-08-22 — **Vendor documented**. [App Store version 4.10.1](https://apps.apple.com/us/app/flighty-live-flight-tracker/id1358823008), 2026-07-14 — **Store claimed**. Live Activities are a Pro feature, free for the first flight; in-flight updates require available Wi-Fi. |

#### CARROT Weather

| Field | Recorded behavior |
| --- | --- |
| Identity/context | One imminent precipitation event at the user's weather location. |
| Phases | No precipitation, approximately 15 minutes before precipitation, raining/snowing, stopped. |
| Time field | When rain or snow will start or stop. |
| Progress representation | A precipitation chart. |
| Live metric | Current precipitation timing and chart updates. |
| Exception states | Coverage or forecast availability can prevent the experience; no separate user-facing error state is documented. |
| Immediate action | None documented. |
| Start trigger | User opts in; a notification starts the activity about 15 minutes before precipitation. |
| End condition | Removes itself after precipitation stops. |
| Deep link | Not documented. |
| Privacy risk | Precise location may be collected. The developer says location is not sold; Apple has not verified the App Store privacy declaration. |
| Evidence / limit | [CARROT Weather App Store](https://apps.apple.com/us/app/carrot-weather-alerts-radar/id961390574), version 6.7 in August 2026; relevant Live Activity release 6.2 on 2025-01-16 — **Store claimed**. Automatic precipitation activities require Premium Ultra and supported precipitation data. |

### Phase progression

#### Domino's

| Field | Recorded behavior |
| --- | --- |
| Identity/context | One US delivery or pickup order in Domino's Tracker. |
| Phases | The Live Activity explicitly supports key order stages. Tracker's broader stages are placed, make, deliver/pick up, and “mmm.” Do not assume every Tracker substage is shown in the Live Activity. |
| Time field | Tracker documents precise ready time, oven time, and driver-departure time; the source does not prove that every field appears in the Live Activity. |
| Progress representation | Key stage changes; the broader Tracker also has a car-progress bar. |
| Live metric | Live driver-location updates are explicitly confirmed for the Live Activity. |
| Exception states | Not documented for the Live Activity. |
| Immediate action | No inline action documented. |
| Start trigger | Live Activities must be enabled; exact post-order creation timing is not documented. |
| End condition | Order completion is the natural end; exact Live Activity dismissal is not documented. |
| Deep link | The release describes a single tap into richer Tracker detail, but does not unambiguously say that the tap originates from the Live Activity. |
| Privacy risk | Live driver location can appear on the Lock Screen; no Live Activity redaction statement is documented. |
| Evidence / limit | [Domino's Tracker update](https://ir.dominos.com/news-releases/news-release-details/dominosr-updates-its-iconic-industry-first-tracker-even-better), 2026-03-24 — **Vendor documented**. [Current App Store listing](https://apps.apple.com/us/app/dominos-pizza-usa/id436491861) — **Store claimed**. US app only and explicitly excludes Puerto Rico. |

#### Structured

| Field | Recorded behavior |
| --- | --- |
| Identity/context | One currently running, incomplete task in Structured's Focus Timer. |
| Phases | Task selected, Focus Now active, expired/stopped/completed. It does not automatically continue to the next timeline task. |
| Time field | Remaining task/focus time. |
| Progress representation | Timer progress. |
| Live metric | Active focus timer and remaining time. |
| Exception states | No Live Activity exception state documented. Only a running incomplete task qualifies. |
| Immediate action | None documented for the Live Activity. |
| Start trigger | Select an ongoing task and tap Focus Now. |
| End condition | Time expires, the task is marked complete, or the user stops the timer in the app; exact Live Activity dismissal timing is not separately documented. |
| Deep link | Not documented. |
| Privacy risk | Task identity and remaining time may be visible; no Lock Screen redaction behavior is documented. |
| Evidence / limit | [Live Activities help](https://help.structured.app/en/articles/330626), edited 2026-06-02, and [Focus Timer help](https://help.structured.app/en/articles/331010), edited 2026-06-04 — **Vendor documented**. The reworked timer requires Structured 4.5.0; intervals require Pro, but intervals are not confirmed as Live Activity content. |

#### Home Assistant Labs

| Field | Recorded behavior |
| --- | --- |
| Identity/context | A user-authored Home Assistant automation identified by a stable notification tag. |
| Phases | User-defined, such as washer/dishwasher phase, EV charging, package delivery, or alarm state. |
| Time field | Configurable count-up or countdown chronometer. |
| Progress representation | Configurable progress/current and progress-maximum bar. |
| Live metric | Title, status/message, icon, color, progress, and chronometer. |
| Exception states | Configurable critical text; delivery can be throttled or dropped if updates are excessive. |
| Immediate action | No inline App Intent control is documented. |
| Start trigger | An automation sends a push with live-update enabled and a tag. |
| End condition | A clear-notification message with the same tag ends it. |
| Deep link | Tap opens a configured Home Assistant path or URL on the server that started the activity. |
| Privacy risk | The user controls the payload, so home, package, charging, or security state can be exposed on Lock Screen. The documentation provides no automatic redaction. |
| Evidence / limit | [Home Assistant Live Activities](https://companion.home-assistant.io/docs/notifications/live-activities/), undated living documentation accessed 2026-08-22 — **Beta**. Labs/TestFlight only; requires iOS 17.2+, Home Assistant Core 2026.7+, token handshake, and working remote connectivity. |

### Continuous progress or metric

#### Timery

| Field | Recorded behavior |
| --- | --- |
| Identity/context | Current Toggl Track time entry and project. |
| Phases | Stopped, timer running, stopped/completed. |
| Time field | Running duration, optionally including seconds. |
| Progress representation | Elapsed duration and current project duration today; it is not a completion percentage. |
| Live metric | Current time entry, running time, and project duration today. |
| Exception states | Toggl API quotas or connectivity can temporarily prevent actions; no specific Live Activity error design is documented. |
| Immediate action | Interactive Stop button. |
| Start trigger | Timery actions and Shortcuts can start/update/end the activity; an exact universal automatic-start rule is not documented. |
| End condition | Stop action ends the running timer/activity. |
| Deep link | Not documented. |
| Privacy risk | Current time-entry identity is deliberately visible; no redaction option is documented. Toggl and optional iCloud are external data boundaries. |
| Evidence / limit | [Timery product site](https://timeryapp.com/), undated, accessed 2026-08-22 — **Vendor documented**. [App Store version 1.8](https://apps.apple.com/us/app/timery-time-tracker/id1425368544), 2026-06-03 — **Store claimed**. A Toggl Track account and internet are required for actions. |

#### SmartGym

| Field | Recorded behavior |
| --- | --- |
| Identity/context | One active workout session. |
| Phases | Current documentation confirms persistent workout presence and a changing next recommended step. Workout/rest/set phase details come from Apple's 2022 case study and are **Older reference**, not re-confirmed current fields. |
| Time field | Workout duration and rest time are historical fields; current pages do not enumerate a current Live Activity time field. |
| Progress representation | The changing Predictive Action represents the next step; no percentage is documented. |
| Live metric | Unspecified real-time workout data received from external devices. Do not infer heart rate or calories as Live Activity fields. |
| Exception states | Not documented. |
| Immediate action | Predictive Action performs the next recommended step and can complete the workout without opening the app. |
| Start trigger | The activity persists throughout a workout; exact creation timing is not documented. |
| End condition | Predictive Action can complete the workout; normal dismissal timing is not documented. |
| Deep link | Not documented. |
| Privacy risk | Workout identity and metrics can be sensitive; no Live Activity privacy/redaction statement is documented. |
| Evidence / limit | [SmartGym current features](https://smartgymapp.com/features), undated, accessed 2026-08-22, and [SmartGym 7.7](https://smartgymapp.com/releases/smartgym-7-7), 2025-09-03 — **Vendor documented**. Historical fields are deliberately not promoted to current claims. |

#### Tesla Supercharging

| Field | Recorded behavior |
| --- | --- |
| Identity/context | A Tesla vehicle while it is Supercharging. |
| Phases | **Inference:** Supercharging active and stopped. The manual does not enumerate a phase model. |
| Time field | Not documented. |
| Progress representation | “Information about your charging session” is confirmed; an exact percentage, target, or gauge is not. |
| Live metric | No individual metric—battery percentage, ETA, target, cost, or charge rate—is enumerated in the cited manual. |
| Exception states | Not documented. |
| Immediate action | Not documented. |
| Start trigger | Available while the vehicle is Supercharging; exact automation is not documented. |
| End condition | Not documented beyond the charging-session context. |
| Deep link | Not documented. |
| Privacy risk | Vehicle/charging state may be visible; no Live Activity redaction behavior is documented. |
| Evidence / limit | [Current Model Y owner manual](https://www.tesla.com/ownersmanual/modely/en_gb/GUID-F6E2CD5E-F226-4167-AC48-BD021D1FFDAB.html), living manual accessed 2026-08-22 — **Vendor documented**. Requires Tesla app 4.45.0+, iOS 17.2+, connectivity, and may vary by region. The source is Model Y-specific. |

### Event stream or score

#### MLB

| Field | Recorded behavior |
| --- | --- |
| Identity/context | One game selected in MLB Gameday. |
| Phases | Live game with changing innings and key updates; no formal Live Activity phase list is documented. |
| Time field | No MLB-specific time field is confirmed in the cited demonstration. |
| Progress representation | Current inning. |
| Live metric | Score. |
| Exception states | Key game updates are shown; an exception taxonomy is not documented. |
| Immediate action | None documented. |
| Start trigger | Open a game in Gameday and tap Track; it is no longer limited to a favorite team. |
| End condition | Not documented. |
| Deep link | Not documented. |
| Privacy risk | No Live Activity-specific privacy controls are documented. |
| Evidence / limit | [MLB App Store](https://apps.apple.com/us/app/mlb/id493619333), version 26.15.0 in August 2026 — **Store claimed**. [WWDC26 Live Activities essentials](https://developer.apple.com/videos/play/wwdc2026/223/), June 2026 — **Apple documented** demonstration of score, inning, and key updates. Presentation details can change. |

#### FotMob

| Field | Recorded behavior |
| --- | --- |
| Identity/context | Followed team or World Cup tournament/game context. |
| Phases | Not documented for the Live Activity. Scheduled/live/final would be a reasonable sports model but remains **Inference** and is not used as a field claim. |
| Time field | Match time is not explicitly confirmed as Live Activity content in the cited current sources. |
| Progress representation | No progress representation beyond live score is confirmed. |
| Live metric | Live score; current release imagery also supports country/team context. |
| Exception states | Alerts can be configured, but the sources do not establish them as Live Activity exception fields. |
| Immediate action | None documented. |
| Start trigger | For the World Cup: follow the tournament, configure alerts, and enable Live Activities for all games. The current listing also says a user can follow a team. |
| End condition | Not documented. |
| Deep link | Not documented. |
| Privacy risk | No Live Activity-specific privacy/redaction detail is documented. |
| Evidence / limit | [Official World Cup guide](https://www.fotmob.com/en-GB/topnews/28453-how-follow-world-cup-fotmob), June 2026 context with no exact publication date — **Vendor documented**. [App Store version 1239](https://apps.apple.com/us/app/fotmob-soccer-live-scores/id488575683), 2026-07-28 — **Store claimed**. The World Cup flow does not prove identical behavior for every competition. |

### Next instruction and exception

#### Insta360 GO Ultra

| Field | Recorded behavior |
| --- | --- |
| Identity/context | One GO Ultra camera connected to the Insta360 iPhone app. |
| Phases | Connected/ready, shooting/recording, stopped, disconnected, abnormal/recording stopped. |
| Time field | No duration field is documented in the cited manual. |
| Progress representation | Recording state rather than a numeric progress value. |
| Live metric | Camera connection and shooting/recording status. |
| Exception states | Overheating, low battery, no SD card, full SD card, and SD-card error; abnormal states explicitly stop recording. |
| Immediate action | Start shooting before recording and Stop shooting while recording are direct Live Activity controls. |
| Start trigger | Enable Live Activities, open the Insta360 app, background it, and enter Lock Screen. |
| End condition | Stop shooting ends recording; normal activity dismissal is not documented. The FAQ acknowledges that a disconnected activity can remain stuck. |
| Deep link | Not documented. |
| Privacy risk | Recordings started through the Live Activity do not receive GPS information. Recording control on Lock Screen still requires careful accidental-action design. |
| Evidence / limit | [Insta360 Live Activity manual](https://onlinemanual.insta360.com/app/en-us/operation-tutorial/page-introduction/live-activity), undated, accessed 2026-08-22 — **Vendor documented**. GO Ultra only; iPad unsupported; the app states iPhone 14 Pro or later and iOS 16.1+. |

### Cross-cutting: completion receipts

The third-party sources are much better at documenting active tracking than exact post-completion retention. That absence matters. The strongest reusable rule comes from Apple:

- end the active event immediately;
- provide final content;
- remove it from Dynamic Island immediately;
- retain it only on surfaces that support post-end content;
- use a proportionate dismissal time, commonly 15–30 minutes.

For any imitation, the safe reusable behavior is a brief, factual receipt that collapses or disappears after explicit dismissal or a short timeout. Because Kennel's macOS Island is not an ActivityKit surface, its exact retention behavior would be a product decision rather than an iOS rule.

### Cross-cutting: multiple concurrent activities

Apple documents display arbitration and recommends aggregation, but the representative vendors rarely document their multi-activity policy. No reliable vendor evidence supports one visible activity per worker-like subtask.

The reusable behavior is to give each top-level event a recognizable minimal form, aggregate related sub-events, and let an exception temporarily outrank ordinary progress. An imitation should keep the complete list in its expanded or full-app view instead of cycling every low-level event through the resting surface.

## Patterns Kennel can borrow

This section comes after the ecosystem catalog deliberately. The references above are real Apple and third-party activities; the ideas below are **Proposed** adaptations for Kennel. They are not integrations with Uber, Domino's, Transit, Flighty, or any other vendor, and they are not permission to copy those products' visual design or language.

### Reference scenarios worth prototyping

| Reference shape | Familiar example | A Kennel prototype could explore | Important boundary |
| --- | --- | --- | --- |
| **Recording** | Voice Memos or Insta360 | A running task with elapsed time, a clear active indicator, Pause/Stop-style controls, and a visible failure state. | Do not expose prompts, commands, file paths, or private content at rest. |
| **Food delivery** | Domino's | A small set of factual stages such as queued → working → checking → ready for review, with the current stage dominant. | These are phases, not a fabricated completion percentage or ETA. |
| **Ride ETA** | Uber | Elapsed time, time in the current phase, or a source-provided deadline beside the current task identity. | Do not invent delivery estimates from worker activity. |
| **Navigation** | Transit GO | The current step, one next instruction, and an exception that temporarily takes priority. | The instruction must come from authoritative task state, not model narration. |
| **Flight tracking** | Flighty | A long-running activity with milestones, change alerts, freshness, and a direct route to detail. | Show “last updated” or stale state when the source is unavailable. |
| **Sports score** | MLB or FotMob | Factual check counts or the latest verification result presented like a score/event update. | Never turn agent activity into an opaque quality or productivity score. |
| **Focus timer** | Structured or Timery | Elapsed execution time, current phase, and one bounded action such as Stop or Open. | Time is context, not evidence of progress or quality. |
| **Workout** | SmartGym | A measured work session with the next safe action and a compact rollup of related activity. | Do not create one visible activity for every worker. |
| **Charging** | Tesla | A factual capacity or usage window when the provider supplies it, plus reset timing. | Provider limits are capacity context, not completion progress. |
| **Camera exception** | Insta360 | A prominent failure state with the exact cause and one corrective action. | Sensitive or multi-step remediation belongs in the full app. |
| **Weather alert** | CARROT Weather | A future material change that can legitimately wake the surface. | Alert only when the change matters; ordinary state updates should remain quiet. |
| **Smart-home automation** | Home Assistant Labs | A configurable demo state with phase, progress, chronometer, icon, and critical text. | Label it as a prototype fixture rather than a shipped live-data contract. |
| **Concurrent activities** | iOS compact/minimal arbitration | Two recognizable activities with a count and a compact list rather than two full dashboards. | Aggregate related work; do not create one visible activity for every worker. |

The implemented design gallery demonstrates the reference side of these shapes as clearly labeled local fixtures. Its purpose is to compare countdown, phase, metric, score, instruction, exception, receipt, and concurrency behavior—not to imply that any fixture represents a live Kennel backend capability.

### What the canonical Island already provides

[The unification plan](../plans/island-app-unification.md) identifies [packages/kennel-island](../../packages/kennel-island) as the canonical implementation. It already has:

- compact and expanded presentations;
- a queue for concurrent work;
- blocked, paused, and running attention arbitration;
- bounded choice and permission surfaces;
- steer, interrupt, retry, and open-in-Kennel actions;
- usage and reset-time presentation;
- connecting, offline, degraded, and no-signal states;
- immediate daemon-event invalidation with recovery polling.

These capabilities make a fixture-driven interaction gallery possible without changing application APIs or claiming that every reference scenario is backed by production data.

### Adopt

- One recognizable identity and one changing fact in the compact state.
- Phase-based progression when no honest percentage exists.
- Expanded detail that answers “what happens next?”
- Alerts only for material exceptions or required action.
- Explicit stale, offline, and unavailable states.
- One safe, immediate action and a direct link to complete context.
- A short, factual completion receipt.
- Aggregation of related sub-events.

### Adapt carefully

- Convert delivery stages into factual work stages without copying vendor copy.
- Convert ETA/countdown patterns into elapsed time, a real deadline, or last-update time when no authoritative ETA exists.
- Convert sports scores into source-backed check counts, never a quality or productivity score.
- Convert recording controls into deliberate Stop/Interrupt behavior, with confirmation where interruption is consequential.
- Convert weather or gate-change alerts into genuinely material task exceptions.
- Preserve exact provider decision identifiers behind any approval action.
- Route sensitive, truncated, free-form, or multi-step actions into full Kennel.

### Reject

- Notification floods or alerts for unchanged state.
- One activity per worker, command, tool call, or provider request.
- Promotional content that displaces the current state.
- Raw prompts, commands, paths, or private content on the resting surface.
- Synthetic percentages, ETAs, scores, or health claims.
- Generic approval controls that discard the real decision context.
- Hiding stale data behind a normal-looking progress state.

### Research and implementation boundary

This dossier does not define a public API, daemon endpoint, database table, event payload, or migration. The detailed fields in the vendor catalog are comparison dimensions, not a production schema. The companion browser gallery adds an internal presentation model and demo-only actions for fixed fixtures; connecting a pattern to live Kennel data requires a separately reviewed implementation contract and an authoritative source for every displayed fact.

## iOS 27 beta watchlist — not the shipping baseline

Apple announced one directly verified Dynamic Island change at WWDC26:

- compact and minimal Live Activity presentations remain visible while iPhone is in landscape;
- layouts can inspect the new isDynamicIslandLimitedInWidth environment value and substitute a narrower representation;
- Apple's current API metadata marks the environment value for iOS/iPadOS 27 beta.

Evidence: [WWDC26 Live Activities essentials](https://developer.apple.com/videos/play/wwdc2026/223/), June 2026, and [isDynamicIslandLimitedInWidth](https://developer.apple.com/documentation/swiftui/environmentvalues/isdynamicislandlimitedinwidth) — **Beta**.

Portrait iOS 26 remains the reference baseline. Landscape behavior must not shape a Kennel prototype or be described as shipping until iOS 27 is released and revalidated.

Two features that can look new in Apple's living documentation are **not** part of this beta watchlist:

- scheduled Live Activities and ActivityState pending are stable from iOS 26;
- transient ActivityStyle behavior is available through the stable iOS 18 API.

The living AVFoundation documentation also describes a system-generated Live Activity for non-discretionary offline media downloads. Its OS availability is not explicit enough in the inspected source to classify it as shipping iOS 26 or iOS 27 beta, so this dossier makes no version claim.

## Source register and limitations

### Apple primary sources

| Source | Source date | Supports | Limit |
| --- | --- | --- | --- |
| [Live Activities HIG](https://developer.apple.com/design/human-interface-guidelines/live-activities) | Change log updated 2025-12-16; accessed 2026-08-22 | Presentation, hierarchy, interaction, alerts, aggregation, privacy, ending, dismissal guidance, specifications. | Design guidance is not a private system implementation contract. |
| [ActivityKit](https://developer.apple.com/documentation/activitykit) | Living documentation; accessed 2026-08-22 | Public app-defined framework, surfaces, updates, actions. | Current living pages can include prerelease API; availability must be checked. |
| [Displaying live data](https://developer.apple.com/documentation/activitykit/displaying-live-data-with-live-activities) | Living documentation; accessed 2026-08-22 | Compact/minimal/expanded/Lock Screen, 8-hour lifetime, 4 KB limit, isolation, stale state, relevance, end/dismiss behavior. | Technical documentation does not establish how Apple-owned experiences are implemented. |
| [ActivityKit push notifications](https://developer.apple.com/documentation/activitykit/starting-and-updating-live-activities-with-activitykit-push-notifications) | Living documentation; accessed 2026-08-22 | Remote start/update/end, push tokens/channels, budgeted delivery. | Push delivery remains system-controlled; stale state is still required. |
| [iOS 26 iPhone guide](https://support.apple.com/guide/iphone/view-live-activities-in-the-dynamic-island-iph28f50d10d/26/ios/26) | Version-pinned living guide; accessed 2026-08-22 | Current Voice Memos, AirDrop, Maps, expand/collapse/switch gestures. | Does not identify public ActivityKit use. |
| [Apple Sports support](https://support.apple.com/en-us/116979) | Published 2025-11-17 | Scores, plays, clocks, scheduling, followed-team behavior. | Explicit product surfaces are Lock Screen and Watch; Island placement is not separately shown. |
| [Wallet flight support](https://support.apple.com/en-us/123179) | Published 2025-10-29 | Flight status, timing, gate/delay information, shared Live Activity. | Airline/pass configuration affects available fields. |
| [iOS 26 update history](https://support.apple.com/en-us/123075) | Living release history; accessed 2026-08-22 | Urgent reminder Live Activity in iOS 26.2. | Establishes the feature, not its complete presentation fields. |
| [Turbocharge your app for CarPlay](https://developer.apple.com/videos/play/wwdc2025/216/) | WWDC25, June 2025 | Live Activities in CarPlay for iOS 26 and non-interactive CarPlay behavior. | CarPlay is contextual support, not Kennel's target. |
| [iPhone 14 Pro launch](https://www.apple.com/ie/newsroom/2022/09/apple-debuts-iphone-14-pro-and-iphone-14-pro-max/) | 2022-09-07 | Original system examples: Maps, Music, Timer, Face ID, alerts/notifications. | **Older reference**; not a current third-party capability statement. |
| [WWDC26 Live Activities essentials](https://developer.apple.com/videos/play/wwdc2026/223/) | June 2026 | MLB demonstration and iOS 27 landscape announcement. | Landscape behavior is **Beta**. Stable scheduling/transient examples must be classified by API availability, not the video's date. |

### Third-party first-party sources

| App | Primary source date | Evidence posture and limitation |
| --- | --- | --- |
| Uber | Engineering 2024-07-25; product update 2025-12-09 | **Vendor documented; Older reference** for implementation detail. Current tipping update does not restate the full 2024 field set. |
| Domino's | 2026-03-24 | **Vendor documented** current Tracker/Live Activity update. Only explicitly named Live Activity fields are treated as such. |
| Transit | Updated 2026-08-20 | **Vendor documented** current GO behavior and privacy notes. |
| Flighty | Help undated, accessed 2026-08-22; App Store 2026-07-14 | **Vendor documented** fields plus **Store claimed** current version. Exact end/action behavior remains unknown. |
| MLB | WWDC26 June 2026; App Store August 2026 | **Apple documented** demo plus **Store claimed** start flow/current version. |
| FotMob | June 2026 context; App Store 2026-07-28 | **Vendor documented** World Cup flow plus **Store claimed** broader support. Only live score is confirmed as a field. |
| Structured | Help edited 2026-06-02 and 2026-06-04 | **Vendor documented** current Focus Timer flow. |
| Timery | Site undated, accessed 2026-08-22; App Store 2026-06-03 | **Vendor documented** plus **Store claimed** version notes. |
| SmartGym | Release 2025-09-03; current site accessed 2026-08-22 | **Vendor documented** current interaction. Apple's 2022 field details remain **Older reference** and are not promoted. |
| Tesla | Living Model Y manual accessed 2026-08-22 | **Vendor documented**, but deliberately non-specific about fields and region-dependent. |
| Insta360 | Living manual accessed 2026-08-22 | **Vendor documented** for GO Ultra only; includes known stuck-disconnect limitation. |
| CARROT Weather | App Store current August 2026; relevant release 2025-01-16 | **Store claimed** only. Apple has not verified the developer's behavior or privacy declaration. |
| Home Assistant | Living Labs documentation accessed 2026-08-22 | **Beta** TestFlight surface; not a production App Store claim. |

## Validation and next gate

This source-backed pass intentionally excludes hands-on testing on a physical iPhone. The local fixture gallery now covers compact/expanded presentation, concurrent activities, material exceptions, local actions, stopped states, and completion receipts. It does not yet add reference fixtures for stale/offline recovery or sensitive/truncated routing; those remain production boundaries to validate before any live-data connection.

A later device and live-data study should compare shipping iOS 26 behavior across:

- compact/resting state;
- expansion and collapse;
- multiple concurrent reference activities;
- material alert transitions;
- stale/offline recovery with an authoritative freshness source;
- sensitive and truncated action routing into the full app;
- completion and dismissal;
- portrait shipping iOS 26 references first, then iOS 27 landscape after release.

Acceptance for this pass has two parts: a reader can distinguish the three Dynamic Island mechanisms, name familiar Apple and third-party activities, and compare what they track; and the local gallery can demonstrate the corresponding compact, expanded, exception, action, and completion patterns with deterministic fixtures. Nothing in this dossier or gallery authorizes a production vendor integration.
