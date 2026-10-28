// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Phoobobo
// Run after the frontend build. All API calls use synthetic in-memory fixtures.
const { createRequire } = require("module");
const path = require("path"),
  fs = require("fs"),
  http = require("http"),
  assert = require("assert/strict");
const root = path.resolve(__dirname, "..");
const req = createRequire(root + "/package.json");
const { chromium } = req("@playwright/test");
(async () => {
  const server = http.createServer((r, s) => {
    let p = path.join(root, ".output/public", decodeURIComponent(r.url.split("?")[0]));
    if (!path.extname(p)) p = path.join(p, "index.html");
    if (!fs.existsSync(p)) p = root + "/.output/public/index.html";
    s.setHeader(
      "Content-Type",
      p.endsWith(".js") ? "application/javascript" : p.endsWith(".css") ? "text/css" : "text/html"
    );
    s.end(fs.readFileSync(p));
  });
  await new Promise(resolve => server.listen(0, "127.0.0.1", resolve));
  const origin = "http://127.0.0.1:" + server.address().port;
  let browser;
  try {
    browser = await chromium.launch({
      headless: true,
      ...(process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE
        ? { executablePath: process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE }
        : {}),
    });
    for (const viewport of [
      { width: 1440, height: 900 },
      { width: 390, height: 844 },
    ]) {
      const ctx = await browser.newContext({ viewport });
      await ctx.addCookies([{ name: "hb.auth.session", value: "true", url: origin }]);
      const page = await ctx.newPage();
      let records = {},
        chats = 0,
        delayChat = false,
        confirms = 0,
        failHistory = false;
      const group = "11111111-1111-4111-8111-111111111111";
      await page.route("**/api/v1/**", async route => {
        const r = route.request(),
          url = new URL(r.url()),
          p = url.pathname;
        let data = [];

        if (p.endsWith("/status"))
          data = p.includes("/ai/")
            ? { configured: true, model: "mock" }
            : { version: "test", allowRegistration: false, otlpEnabled: false };
        else if (p.endsWith("/users/self"))
          data = {
            item: {
              id: "22222222-2222-4222-8222-222222222222",
              name: "Test",
              email: "synthetic@example.invalid",
              defaultGroupId: group,
              group: { id: group, name: "Test collection" },
              isOwner: true,
            },
          };
        else if (p.endsWith("/groups/all")) data = [{ id: group, name: "Test collection" }];
        else if (p.endsWith("/groups")) data = { id: group, name: "Test collection" };
        else if (p.endsWith("/settings")) data = { item: {} };
        else if (p.endsWith("/ai/history")) {
          if (failHistory) {
            await route.fulfill({
              status: 500,
              json: { error: "Synthetic history unavailable" },
            });
            return;
          }
          if (r.method() === "POST") {
            data = r.postDataJSON();
            data.revision++;
            data.title = data.messages[0]?.content || "";
            data.updatedAt = new Date().toISOString();
            records[data.id] = data;
          } else
            data = url.searchParams.has("id")
              ? records[url.searchParams.get("id")]
              : Object.values(records)
                  .sort((a, b) => b.updatedAt.localeCompare(a.updatedAt))
                  .map(({ messages, summary, ...rest }) => rest);
        } else if (p.endsWith("/ai/confirm")) {
          confirms++;
          data = {};
        } else if (p.endsWith("/ai/chat")) {
          chats++;
          if (delayChat) await new Promise(resolve => setTimeout(resolve, 400));
          data = {
            reply: "Synthetic reply " + chats,
            usedModel: "mock",
            items: [],
            drafts: [],
          };
        } else if (p.endsWith("/entities")) data = { items: [], total: 0, page: 1, pageSize: 20 };
        await route.fulfill({ json: data });
      });
      page.on("pageerror", e => console.log("PAGEERROR", e.message));
      await page.goto(origin + "/items");
      await page.getByTestId("text-assistant-button").click({ timeout: 15000 });
      const input = page.locator("textarea");
      await input.fill("find synthetic keys");
      await page.getByRole("button", { name: /Send|发送/ }).click();
      await page.getByText("Synthetic reply 1", { exact: true }).waitFor();
      await page.getByRole("button", { name: /New chat|新建会话/ }).click();
      await input.fill("find synthetic bag");
      await page.getByRole("button", { name: /Send|发送/ }).click();
      await page.getByText("Synthetic reply 2", { exact: true }).waitFor();
      await page.getByRole("button", { name: /History|历史会话/, exact: true }).click();
      await page.getByRole("button", { name: /find synthetic keys/ }).click();
      await page.getByText("Synthetic reply 1", { exact: true }).waitFor();
      await input.fill("continue");
      await page.getByRole("button", { name: /Send|发送/ }).click();
      await page.getByText("Synthetic reply 3", { exact: true }).waitFor();
      await page.reload();
      await page.getByTestId("text-assistant-button").click();
      await page.getByText("Synthetic reply 3", { exact: true }).waitFor();
      assert.equal(chats, 3);
      assert.equal(Object.keys(records).length, 2);
      assert(await page.locator("[role=dialog]").evaluate(el => el.getBoundingClientRect().width <= innerWidth));
      // Historical action cards are display-only, including pending cards from a crash.
      const active = Object.values(records).sort((a, b) => b.updatedAt.localeCompare(a.updatedAt))[0];
      delete active.summary;
      delete active.historyStart;
      active.messages.push({
        role: "assistant",
        content: "Historical actions",
        drafts: [
          {
            id: "old-saved",
            proposal: {
              kind: "item",
              name: "Synthetic saved item",
              description: "",
            },
            parentPath: "",
            tags: [],
            state: "saved",
          },
          {
            id: "old-pending",
            proposal: {
              kind: "item",
              name: "Synthetic pending item",
              description: "",
            },
            parentPath: "",
            tags: [],
          },
        ],
      });
      await page.reload();
      await page.getByTestId("text-assistant-button").click();
      await page.getByText("Historical actions", { exact: true }).waitFor();
      assert.equal(await page.getByRole("button", { name: /Confirm|Cancel|确认|取消/ }).count(), 0);
      assert.equal(confirms, 0);
      assert.equal(chats, 3);
      // Switching during inference aborts the old request and discards its late reply.
      delayChat = true;
      await input.fill("delayed synthetic request");
      const started = page.waitForRequest("**/ai/chat");
      await page.getByRole("button", { name: /Send|发送/ }).click();
      await started;
      await input.dispatchEvent("keydown", { key: "Enter", ctrlKey: true });
      await page.getByRole("button", { name: /New chat|新建会话/ }).click();
      await page.waitForTimeout(600);
      assert.equal(await page.getByText("Synthetic reply 4", { exact: true }).count(), 0);
      assert.equal(chats, 4);
      failHistory = true;
      await page.reload();
      await page.getByTestId("text-assistant-button").click();
      await page.getByText("Synthetic history unavailable", { exact: true }).waitFor();
      assert.equal(await page.getByRole("button", { name: /^Send$|^发送$/ }).isDisabled(), true);
      failHistory = false;
      await page.getByRole("button", { name: /^History$|^历史会话$/ }).click();
      await page.getByText("Historical actions", { exact: true }).waitFor();
      await page.screenshot({
        path: "/tmp/zainaer-history-" + viewport.width + ".png",
      });
      console.log("PASS", viewport.width, "new/list/switch/continue/refresh/cancel; historical actions never replay");
      await ctx.close();
    }
  } finally {
    await browser?.close();
    server.close();
  }
})().catch(e => {
  console.error(e);
  process.exitCode = 1;
});
