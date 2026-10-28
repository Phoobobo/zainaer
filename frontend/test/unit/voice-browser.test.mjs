// SPDX-License-Identifier: MIT; Copyright (c) 2026 Phoobobo
// Run after building: node test/unit/voice-browser.test.mjs
// Set PLAYWRIGHT_EXECUTABLE_PATH to use an installed Chromium browser.
import { chromium, expect } from "@playwright/test";
import { createRequire } from "node:module";
import { fileURLToPath } from "node:url";
const require = createRequire(import.meta.url);
const http = require("http"),
  fs = require("fs"),
  path = require("path");
const root = fileURLToPath(new URL("../../.output/public", import.meta.url));
const server = http.createServer((req, res) => {
  let file = path.join(root, new URL(req.url, "http://localhost").pathname);
  if (!fs.existsSync(file) || fs.statSync(file).isDirectory()) file = path.join(root, "index.html");
  res.setHeader(
    "Content-Type",
    file.endsWith(".js")
      ? "text/javascript"
      : file.endsWith(".css")
        ? "text/css"
        : file.endsWith(".json")
          ? "application/json"
          : "text/html"
  );
  res.end(fs.readFileSync(file));
});
// Synthetic microphone events only; never captures real audio.
await new Promise(r => server.listen(0, "127.0.0.1", r));
let browser;
try {
  browser = await chromium.launch({ headless: true, executablePath: process.env.PLAYWRIGHT_EXECUTABLE_PATH });
  const origin = `http://127.0.0.1:${server.address().port}`;
  for (const width of [390, 1280]) {
    const context = await browser.newContext({ viewport: { width, height: 844 } });
    await context.addCookies([{ name: "hb.auth.session", value: "true", url: origin }]);
    await context.addInitScript(() => {
      window.SpeechRecognition = class {
        processLocally = false;
        static async available() {
          return "available";
        }
        start() {
          window.fixtureVoice = this;
          this.onstart();
        }
        stop() {
          this.onresult({ results: [{ isFinal: true, 0: { transcript: "synthetic keys" } }] });
          this.onend();
        }
        abort() {
          this.aborted = true;
        }
      };
    });
    const page = await context.newPage();
    const chats = [];
    const records = {};
    await page.route("**/api/v1/**", async route => {
      const p = new URL(route.request().url()).pathname;
      let data = [];
      if (p === "/api/v1/ai/history") {
        if (route.request().method() === "POST") {
          data = route.request().postDataJSON();
          data.revision++;
          data.title = data.messages[0]?.content || "";
          data.updatedAt = new Date().toISOString();
          records[data.id] = data;
        } else {
          const id = new URL(route.request().url()).searchParams.get("id");
          data = id ? records[id] : Object.values(records);
        }
      } else if (p === "/api/v1/ai/status") data = { configured: true, model: "fixture" };
      else if (p === "/api/v1/ai/chat") {
        chats.push(route.request().postDataJSON());
        data = { reply: "Synthetic reply" };
      } else if (p.endsWith("/status"))
        data = { health: true, build: { version: "test", commit: "fixture" }, oidc: { enabled: false } };
      else if (p.endsWith("/users/self"))
        data = { item: { id: "user", name: "Tester", email: "test@example.com", defaultGroupId: "personal" } };
      else if (p.endsWith("/groups/all")) data = [{ id: "personal", name: "Fixture home" }];
      else if (p.endsWith("/settings")) data = { item: {} };
      else if (p.endsWith("/items")) data = { items: [], total: 0, page: 1, pageSize: 20 };
      await route.fulfill({ json: data });
    });
    await page.goto(origin + "/items");
    await page.getByTestId("text-assistant-button").first().click();
    const dialog = page.getByRole("dialog");
    const voice = () => dialog.getByRole("button", { name: /语音输入|Voice input|重新录音|Record again/ });
    await voice().click();
    await expect(dialog.getByRole("status")).toHaveText(/正在录音|Recording/);
    await dialog.getByRole("button", { name: /停止并转写|Stop and transcribe/ }).click();
    await expect(page.locator("#assistant-draft")).toHaveValue("synthetic keys");
    expect(chats.length).toBe(0);
    await page.locator("#assistant-draft").fill("edited fixture");
    await dialog.getByRole("button", { name: /^(发送|Send)$/ }).click();
    await expect.poll(() => chats.length).toBe(1);
    expect(chats[0].messages.at(-1).content).toBe("edited fixture");
    await voice().click();
    await dialog.getByRole("button", { name: /取消录音|Cancel recording/ }).click();
    expect(await page.evaluate(() => window.fixtureVoice.aborted)).toBe(true);
    await voice().click();
    await page.evaluate(() => window.fixtureVoice.onerror({ error: "not-allowed" }));
    await expect(dialog.getByRole("alert")).toHaveText(/权限被拒绝|permission denied/);
    await voice().click();
    await dialog.getByRole("button", { name: /新建会话|New chat/ }).click();
    expect(await page.evaluate(() => window.fixtureVoice.aborted)).toBe(true);
    await page.evaluate(() =>
      window.fixtureVoice.onresult?.({ results: [{ isFinal: true, 0: { transcript: "late discarded fixture" } }] })
    );
    await expect(page.locator("#assistant-draft")).toHaveValue("");
    await voice().click();
    await page.keyboard.press("Escape");
    expect(await page.evaluate(() => window.fixtureVoice.aborted)).toBe(true);
    await page.getByTestId("text-assistant-button").first().click();
    await page.evaluate(() => {
      delete window.SpeechRecognition;
    });
    await voice().click();
    await expect(dialog.getByRole("alert")).toHaveText(/不支持|unavailable/);
    const box = await dialog.boundingBox();
    expect(box.x).toBeGreaterThanOrEqual(0);
    expect(box.x + box.width).toBeLessThanOrEqual(width + 1);
    console.log(`PASS voice flow and layout ${width}px`);
    await context.close();
  }
} finally {
  await browser?.close();
  await new Promise(r => server.close(r));
}
