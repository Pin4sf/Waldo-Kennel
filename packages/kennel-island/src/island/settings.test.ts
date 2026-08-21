import assert from "node:assert/strict";
import test from "node:test";
import { defaultSettings, SETTINGS_SCHEMA } from "../../desktop/settings.mjs";
import {
  controlDisabled,
  controlPatch,
  controlValue,
  defaultKennelSettings,
  settingsTabs,
  type SettingsControl,
} from "./settings.ts";

function allControls(): SettingsControl[] {
  return settingsTabs.flatMap((tab) => tab.groups.flatMap((group) => [...group.controls]));
}

test("the renderer's defaults are the main process's defaults", () => {
  // The two sides cannot import each other, so this is the seam that keeps the
  // duplicated document honest.
  const { version, ...sections } = defaultSettings();
  assert.equal(typeof version, "number");
  assert.deepEqual(sections, defaultKennelSettings);
});

test("every control names a field the schema actually has", () => {
  for (const control of allControls()) {
    const section = SETTINGS_SCHEMA[control.section];
    assert.ok(section, `unknown section: ${control.section}`);
    assert.ok(
      Object.hasOwn(section, control.field as string),
      `unknown field: ${control.section}.${String(control.field)}`,
    );
  }
});

test("every control's kind matches the type the schema stores", () => {
  for (const control of allControls()) {
    const field = SETTINGS_SCHEMA[control.section][control.field as string];
    assert.equal(
      field.kind,
      control.kind === "toggle" ? "boolean" : "integer",
      `${control.section}.${String(control.field)} is a ${field.kind}`,
    );
  }
});

test("every slider stays inside the range the schema will clamp to", () => {
  for (const control of allControls()) {
    if (control.kind !== "range") continue;
    const field = SETTINGS_SCHEMA[control.section][control.field as string];
    assert.equal(field.kind, "integer", `${String(control.field)} is not a number`);
    if (field.kind !== "integer") continue;

    assert.ok(control.min >= field.min, `${String(control.field)} slider starts below the schema`);
    assert.ok(control.max <= field.max, `${String(control.field)} slider ends above the schema`);
  }
});

test("no field is offered by two controls", () => {
  const seen = allControls().map((control) => `${control.section}.${String(control.field)}`);
  assert.deepEqual(seen, [...new Set(seen)]);
});

test("a control reads and writes the field it names", () => {
  const control = allControls().find((candidate) => candidate.field === "widthOffset");
  assert.ok(control);

  assert.equal(controlValue(defaultKennelSettings, control), 0);
  assert.deepEqual(controlPatch(control, 6), { notch: { widthOffset: 6 } });
});

test("opening on hover disables the peek controls it replaces", () => {
  const settings: KennelSettings = {
    ...defaultKennelSettings,
    hover: { ...defaultKennelSettings.hover, openOnHover: true },
  };

  for (const control of allControls()) {
    if (control.section !== "hover") continue;
    const expected = ["peek", "peekWidth", "peekHeight", "peekDelayMs"].includes(
      control.field as string,
    );
    assert.equal(controlDisabled(settings, control), expected, String(control.field));
  }
});

test("turning the peek off disables its sliders but not its own switch", () => {
  const settings: KennelSettings = {
    ...defaultKennelSettings,
    hover: { ...defaultKennelSettings.hover, peek: false },
  };

  assert.equal(controlDisabled(settings, { kind: "toggle", section: "hover", field: "peek", label: "" }), false);
  assert.equal(
    controlDisabled(settings, {
      kind: "range",
      section: "hover",
      field: "peekWidth",
      label: "",
      min: 0,
      max: 48,
      step: 1,
    }),
    true,
  );
});

test("turning gestures off disables the gestures they contain", () => {
  const settings: KennelSettings = {
    ...defaultKennelSettings,
    gestures: { ...defaultKennelSettings.gestures, enabled: false },
  };

  for (const control of allControls()) {
    if (control.section !== "gestures") continue;
    assert.equal(controlDisabled(settings, control), control.field !== "enabled", String(control.field));
  }
});

test("nothing is disabled on a default install", () => {
  for (const control of allControls()) {
    assert.equal(controlDisabled(defaultKennelSettings, control), false, String(control.field));
  }
});
