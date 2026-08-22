import { useCallback, useEffect, useState } from "react";
import {
  controlDisabled,
  controlPatch,
  controlValue,
  settingsTabs,
  type SettingsControl,
  type SettingsGroup,
  type SettingsTab,
} from "../island/settings";
import { useKennelSettings, useSettingsWriter } from "../island/useIslandStage";

/* --------------------------------------------------------------------------
   Settings window
   --------------------------------------------------------------------------
   A form, and only a form. Nothing here is notch-shaped, translucent, or
   animated: the island is the thing that performs, and a preferences pane that
   also performs is a preferences pane nobody can read while dragging a slider
   against their own hardware.

   Every control is rendered from the descriptions in `island/settings.ts`,
   which is what keeps adding a preference to two rows of data rather than a new
   component. Writes go straight to the host and come back as the pushed
   document, so this window never holds an opinion about state the host owns.
   -------------------------------------------------------------------------- */

export function SettingsApp() {
  const settings = useKennelSettings();
  const write = useSettingsWriter();
  const [activeTab, setActiveTab] = useState(settingsTabs[0].id);

  useEffect(() => {
    document.documentElement.classList.add("is-settings-window");
    return () => document.documentElement.classList.remove("is-settings-window");
  }, []);

  const close = useCallback(() => {
    void window.kennelDesktop?.closeSettings?.().catch(() => {});
  }, []);

  // An accessory app carries no menu bar, so the two shortcuts every macOS
  // window is expected to answer have to be answered here.
  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape" || (event.metaKey && event.key.toLowerCase() === "w")) {
        event.preventDefault();
        close();
      }
    };
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [close]);

  const tab = settingsTabs.find((candidate) => candidate.id === activeTab) ?? settingsTabs[0];

  return (
    <div className="settings">
      <header className="settings__chrome">
        <nav aria-label="Settings sections" className="settings__tabs" role="tablist">
          {settingsTabs.map((candidate) => (
            <SettingsTabButton
              active={candidate.id === tab.id}
              key={candidate.id}
              tab={candidate}
              onSelect={setActiveTab}
            />
          ))}
        </nav>
      </header>

      <main className="settings__body" id={`settings-panel-${tab.id}`} role="tabpanel">
        {tab.groups.map((group) => (
          <SettingsGroupSection
            group={group}
            key={group.id}
            settings={settings}
            onChange={write}
          />
        ))}
      </main>

      <footer className="settings__footer">
        <p className="settings__footer-note">
          Changes apply as you make them. Settings live in the app's own support folder.
        </p>
        <button
          className="settings__reset"
          onClick={() => void window.kennelDesktop?.resetSettings?.().catch(() => {})}
          type="button"
        >
          Reset all
        </button>
      </footer>
    </div>
  );
}

function SettingsTabButton({
  tab,
  active,
  onSelect,
}: {
  tab: SettingsTab;
  active: boolean;
  onSelect: (id: string) => void;
}) {
  return (
    <button
      aria-controls={`settings-panel-${tab.id}`}
      aria-selected={active}
      className={active ? "settings__tab is-active" : "settings__tab"}
      onClick={() => onSelect(tab.id)}
      role="tab"
      type="button"
    >
      {tab.label}
    </button>
  );
}

function SettingsGroupSection({
  group,
  settings,
  onChange,
}: {
  group: SettingsGroup;
  settings: KennelSettings;
  onChange: (patch: KennelSettingsPatch) => void;
}) {
  return (
    <section aria-labelledby={`settings-group-${group.id}`} className="settings-group">
      <h2 className="settings-group__title" id={`settings-group-${group.id}`}>
        {group.label}
      </h2>
      {group.hint ? <p className="settings-group__hint">{group.hint}</p> : null}
      <div className="settings-group__controls">
        {group.controls.map((control) => (
          <SettingsField
            control={control}
            disabled={controlDisabled(settings, control)}
            key={`${control.section}.${String(control.field)}`}
            settings={settings}
            onChange={onChange}
          />
        ))}
      </div>
    </section>
  );
}

function SettingsField({
  control,
  settings,
  disabled,
  onChange,
}: {
  control: SettingsControl;
  settings: KennelSettings;
  disabled: boolean;
  onChange: (patch: KennelSettingsPatch) => void;
}) {
  const id = `setting-${control.section}-${String(control.field)}`;
  const hintId = control.hint ? `${id}-hint` : undefined;
  const value = controlValue(settings, control);

  if (control.kind === "toggle") {
    return (
      <div className="settings-field settings-field--toggle" data-disabled={disabled}>
        <input
          aria-describedby={hintId}
          checked={value === true}
          className="settings-field__checkbox"
          disabled={disabled}
          id={id}
          onChange={(event) => onChange(controlPatch(control, event.target.checked))}
          type="checkbox"
        />
        <label className="settings-field__label" htmlFor={id}>
          {control.label}
        </label>
        {control.hint ? (
          <p className="settings-field__hint" id={hintId}>
            {control.hint}
          </p>
        ) : null}
      </div>
    );
  }

  const numeric = typeof value === "number" ? value : control.min;

  return (
    <div className="settings-field settings-field--range" data-disabled={disabled}>
      <label className="settings-field__label" htmlFor={id}>
        {control.label}
      </label>
      <input
        aria-describedby={hintId}
        className="settings-field__range"
        disabled={disabled}
        id={id}
        max={control.max}
        min={control.min}
        // A range input reports a string; the host coerces it, and sending the
        // number here keeps the bridge's numeric check from dropping it.
        onChange={(event) => onChange(controlPatch(control, Number(event.target.value)))}
        step={control.step}
        type="range"
        value={numeric}
      />
      <output className="settings-field__value" htmlFor={id}>
        {control.format ? control.format(numeric) : numeric}
      </output>
      {control.hint ? (
        <p className="settings-field__hint" id={hintId}>
          {control.hint}
        </p>
      ) : null}
    </div>
  );
}
