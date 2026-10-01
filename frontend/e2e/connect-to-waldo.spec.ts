import { mkdirSync, writeFileSync } from "node:fs";
import { join } from "node:path";
import { tmpdir } from "node:os";
import { execFileSync } from "node:child_process";
import { catalogFor, type AppLocale } from "../src/renderer/i18n";
import { test, expect } from "@playwright/test";
import { installFakeBridge } from "./support/fake-bridge";

// No real code is used. All pairing states are labelled local fixtures.
test("production Settings stays unavailable; keyboard opens/closes and no bridge traffic", async ({page}) => {
 await installFakeBridge(page);
 const traffic:string[]=[];
 page.on("request",r=>{if(/\/devices\/(redeem|connect)|\/pair(?:$|\?)/.test(r.url()))traffic.push(r.url());});
 page.on("websocket",s=>{if(/devices\/connect/.test(s.url()) || !s.url().includes("127.0.0.1"))traffic.push(s.url());});
 await page.goto("/work");
 await page.getByRole("button",{name:"Settings",exact:true}).click();
 const destination=page.getByRole("button",{name:"Connect to Waldo",exact:true});
 await destination.focus();await page.keyboard.press("Enter");
 await expect(page.getByText("Pairing is not enabled on this build")).toBeVisible();
 await expect(page.getByLabel("Waldo-issued code")).toBeDisabled();
 await expect(page.getByRole("button",{name:"Connect",exact:true})).toBeDisabled();
 await expect(page.getByTestId("waldo-preview-label")).toHaveCount(0);
 await expect(page.getByLabel("Preview state")).toHaveCount(0);
 await page.keyboard.press("Tab");await expect(page.getByRole("dialog")).toContainText("Approvals remain native");
 await page.keyboard.press("Escape");await expect(page.getByRole("dialog")).toHaveCount(0);
 expect(traffic).toEqual([]);
});

test("synthetic preview retains state distinctions, clears on Escape and uses no transport", async ({page})=>{
 const traffic:string[]=[];page.on("request",r=>{if(/devices\/(redeem|connect)|\/pair(?:$|\?)/.test(r.url()))traffic.push(r.url());});page.on("websocket",s=>{if(/devices\/connect/.test(s.url()) || !s.url().includes("127.0.0.1"))traffic.push(s.url());});
 await page.goto("/__preview/connect-waldo");
 for(const state of ["unavailable","unpaired","pairing","recovery_required","offline","online","failed"]){
  await page.getByRole("button",{name:"Close settings"}).click();
  await page.getByLabel("Preview state",{exact:true}).selectOption(state);
  await page.getByRole("button",{name:"Open Settings",exact:true}).click();
  await expect(page.getByTestId("waldo-preview-label")).toBeVisible();
  await expect(page.getByRole("button",{name:/retry|reset|pair again/i})).toHaveCount(0);
  if(state === "offline" || state === "online")await expect(page.getByRole("dialog").getByText(`Paired · ${state === "offline" ? "Offline" : "Online"}`,{exact:true})).toBeVisible();
 }
 await page.getByRole("button",{name:"Close settings"}).click();await page.getByLabel("Preview state",{exact:true}).selectOption("unpaired");await page.getByRole("button",{name:"Open Settings",exact:true}).click();
 const synthetic="A".repeat(43);
 await page.getByLabel("Waldo-issued code").fill(synthetic);await page.getByLabel("Device label").fill("Synthetic Mac");
 await page.keyboard.press("Escape");await page.getByRole("button",{name:"Open Settings",exact:true}).click();await expect(page.getByLabel("Waldo-issued code")).toHaveValue("");
 const storage=await page.evaluate(()=>JSON.stringify({...localStorage,...sessionStorage}));expect(storage).not.toContain(synthetic);
 expect(traffic).toEqual([]);
});


test("visual fixture matrix: every state, themes, sizes and shipped translations",async({page})=>{
 test.setTimeout(120000);
 const out=process.env.WALDO_EVIDENCE_DIR ?? join(tmpdir(),"kennel-waldo-ui-visual");mkdirSync(out,{recursive:true});
 const manifest:Array<Record<string,unknown>>=[];
 const sha=execFileSync("git",["rev-parse","HEAD"],{encoding:"utf8"}).trim();
 const states=["unavailable","unpaired","pairing","recovery_required","offline","online","failed"];
 await page.goto("/__preview/connect-waldo");
 await expect(page.getByRole("dialog")).toBeVisible();
 const locales:AppLocale[]=["en","de","es","fr","ja","ko","pt-BR","zh-CN"];
 for(const locale of locales){
  for(const theme of ["light","dark"]){
   for(const size of [{width:1280,height:900},{width:390,height:844}]){
    if(locale!=="en" && ((theme==="light" && size.width===390)||(theme==="dark" && size.width===1280)))continue;
    await page.setViewportSize(size);
    await page.evaluate(theme=>{localStorage.setItem("kennel.theme",theme);},theme);
    await page.goto("/__preview/connect-waldo");
    await expect(page.getByRole("dialog")).toBeVisible();
    await expect(page.locator("html")).toHaveAttribute("data-theme",theme);
    const catalog=catalogFor(locale);
    for(const state of locale==="en"?states:["unavailable"]){
     await page.keyboard.press("Escape");await expect(page.getByRole("dialog")).toHaveCount(0);
     await page.locator("select").nth(1).selectOption(locale);
     await page.locator("select").nth(0).selectOption(state);
     await page.getByRole("button",{name:catalog["settings.waldo.previewOpen"],exact:true}).click();
     const dialog=page.getByRole("dialog");await expect(dialog).toBeVisible();
     await expect(page.getByTestId("waldo-preview-label")).toBeVisible();
     await expect(page.getByLabel(catalog["settings.waldo.code"])).toHaveValue("");
     const bounds=await dialog.boundingBox();expect(bounds).not.toBeNull();expect(bounds!.x).toBeGreaterThanOrEqual(0);expect(bounds!.x+bounds!.width).toBeLessThanOrEqual(size.width+1);
     const overflow=await page.locator('[data-section="waldo"]').evaluate(e=>e.scrollWidth>e.clientWidth+1);expect(overflow).toBe(false);
     await dialog.getByRole("button",{name:catalog["settings.waldo.connect"],exact:true}).scrollIntoViewIfNeeded();
     await expect(dialog.getByRole("button",{name:catalog["settings.waldo.connect"],exact:true})).toBeVisible();
     await page.getByTestId("waldo-preview-label").scrollIntoViewIfNeeded();
     const file=`preview-${state}-${theme}-${size.width}-${locale}.png`;
     await page.screenshot({path:join(out,file)});
     if(size.width===390){
      await dialog.getByRole("button",{name:catalog["settings.waldo.connect"],exact:true}).scrollIntoViewIfNeeded();
      await page.screenshot({path:join(out,file.replace(".png","-fields.png"))});
     }
     manifest.push({file,kind:"browser fixture preview — synthetic data, not this Mac",state,theme,...size,locale,sha,emptyCode:true,noHorizontalOverflow:true});
    }
   }
  }
 }
 writeFileSync(join(out,"manifest.json"),JSON.stringify(manifest,null,2));
});
