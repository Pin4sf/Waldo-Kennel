import { act, fireEvent, render, screen, waitFor, cleanup } from "@testing-library/react";
import { I18nextProvider } from "react-i18next";
import { afterEach, describe, expect, it, vi } from "vitest";
import { APP_LOCALES, createAppI18n, catalogFor } from "../../i18n";
import { WaldoConnectionSection, WaldoConnectionView } from "./WaldoConnectionSection";
import type { WaldoConnectionState } from "../../lib/waldo-connection";

const code = "A".repeat(43); // Synthetic zero bytes, never a real pairing code.
function view(node: React.ReactNode) { return render(<I18nextProvider i18n={createAppI18n()}>{node}</I18nextProvider>); }
afterEach(() => { cleanup(); vi.restoreAllMocks(); vi.unstubAllGlobals(); });
describe("inactive Waldo settings", () => {
 it("always shows unavailable with disabled secret entry, no preview controls or network", () => {
  const fetch = vi.fn(); const socket = vi.fn(); vi.stubGlobal("fetch",fetch); vi.stubGlobal("WebSocket",socket);
  view(<WaldoConnectionSection />);
  expect(screen.getByText("Pairing is not enabled on this build")).toBeInTheDocument();
  expect(screen.getByLabelText("Waldo-issued code")).toBeDisabled();
  expect(screen.getByLabelText("Device label")).toBeDisabled();
  expect(screen.getByRole("button", {name:"Connect"})).toBeDisabled();
  expect(screen.queryByTestId("waldo-preview-label")).not.toBeInTheDocument();
  expect(fetch).not.toHaveBeenCalled(); expect(socket).not.toHaveBeenCalled();
 });
 it.each<WaldoConnectionState>([{kind:"unpaired"},{kind:"pairing"},{kind:"recovery_required"},{kind:"paired",transport:"offline",deviceId:"synthetic-device",label:"Synthetic Mac"},{kind:"paired",transport:"online"},{kind:"error",reason:"failed"},{kind:"error",reason:"unavailable"}])("keeps state %j distinct and offers no recovery retry", state => {
  view(<WaldoConnectionView state={state} previewAction={vi.fn()} />);
  expect(screen.getByTestId("waldo-preview-label")).toBeInTheDocument();
  if(state.kind === "paired") { expect(screen.getByText(`Paired · ${state.transport === "offline" ? "Offline" : "Online"}`)).toBeInTheDocument(); expect(screen.queryByText("Not paired")).not.toBeInTheDocument(); }
  expect(screen.queryByRole("button", {name:/retry|reset|pair again/i})).not.toBeInTheDocument();
  expect(screen.getByLabelText("Waldo-issued code")).toHaveProperty("disabled",state.kind !== "unpaired");
 });
 it("blocks double submit, never persists/logs a code, and freshly clears the code after a local action", async () => {
  let finish!: (state: WaldoConnectionState) => void;
  const action = vi.fn(() => new Promise<WaldoConnectionState>(resolve => {finish=resolve;}));
  const store = vi.spyOn(Storage.prototype,"setItem"); const log = vi.spyOn(console,"log");
  const fetch=vi.fn(); const socket=vi.fn(); vi.stubGlobal("fetch",fetch);vi.stubGlobal("WebSocket",socket);
  view(<WaldoConnectionView state={{kind:"unpaired"}} previewAction={action} />);
  fireEvent.change(screen.getByLabelText("Waldo-issued code"),{target:{value:code}});
  fireEvent.change(screen.getByLabelText("Device label"),{target:{value:"機".repeat(40)}});
  const form=screen.getByRole("button",{name:"Connect"}).closest("form")!;
  fireEvent.submit(form);fireEvent.submit(form);
  expect(action).toHaveBeenCalledTimes(1); expect(screen.getByText("Pairing…")).toBeInTheDocument();
  await act(async () => finish({kind:"paired",transport:"offline"}));
  expect(screen.getByLabelText("Waldo-issued code")).toHaveValue("");
  expect(store).not.toHaveBeenCalled();expect(log).not.toHaveBeenCalled();expect(fetch).not.toHaveBeenCalled();expect(socket).not.toHaveBeenCalled();
 });
 it("rejects malformed code/label without normalization or a submit action", () => {
  const action=vi.fn();view(<WaldoConnectionView state={{kind:"unpaired"}} previewAction={action}/>);
  fireEvent.change(screen.getByLabelText("Waldo-issued code"),{target:{value:code+"="}});
  fireEvent.change(screen.getByLabelText("Device label"),{target:{value:"é".repeat(61)}});
  fireEvent.submit(screen.getByRole("button",{name:"Connect"}).closest("form")!);
  expect(screen.getAllByRole("alert")).toHaveLength(2);expect(action).not.toHaveBeenCalled();
  expect(screen.getByLabelText("Waldo-issued code")).toHaveValue(code+"=");
 });
 it("clears on close and ignores an action that settles after closing", async () => {
  let finish!: (state: WaldoConnectionState) => void;
  const state: WaldoConnectionState={kind:"unpaired"};const action=vi.fn(() => new Promise<WaldoConnectionState>(resolve=>{finish=resolve;}));
  const wrap=(open:boolean)=><I18nextProvider i18n={createAppI18n()}><WaldoConnectionView state={state} open={open} previewAction={action}/></I18nextProvider>;
  const rendered=render(wrap(true));fireEvent.change(screen.getByLabelText("Waldo-issued code"),{target:{value:code}});fireEvent.change(screen.getByLabelText("Device label"),{target:{value:"Synthetic Mac"}});
  fireEvent.submit(screen.getByRole("button",{name:"Connect"}).closest("form")!);rendered.rerender(wrap(false));
  await act(async()=>finish({kind:"paired",transport:"online"}));rendered.rerender(wrap(true));
  expect(screen.getByLabelText("Waldo-issued code")).toHaveValue("");expect(screen.getByText("Not paired")).toBeInTheDocument();
 });
 it("shows only a safe error when an injected action throws",async()=>{
  view(<WaldoConnectionView state={{kind:"unpaired"}} previewAction={async()=>{throw new Error(code);}}/>);
  fireEvent.change(screen.getByLabelText("Waldo-issued code"),{target:{value:code}});fireEvent.change(screen.getByLabelText("Device label"),{target:{value:"Synthetic Mac"}});
  fireEvent.submit(screen.getByRole("button",{name:"Connect"}).closest("form")!);
  await waitFor(()=>expect(screen.getByText(/Connection status is unknown/)).toBeInTheDocument());expect(screen.getByLabelText("Waldo-issued code")).toHaveValue("");expect(document.body.textContent).not.toContain(code);
 });
 it.each(APP_LOCALES)("has all new strings and localized unavailable copy in %s",locale=>{
  const catalog=catalogFor(locale);const keys=Object.keys(catalogFor("en")).filter(k=>k.startsWith("settings.waldo."));
  for(const key of keys)expect(catalog[key]).toBeTruthy();
  render(<I18nextProvider i18n={createAppI18n(locale)}><WaldoConnectionSection/></I18nextProvider>);
  expect(screen.getByText(catalog["settings.waldo.status.unavailable"])).toBeInTheDocument();
 });
});
