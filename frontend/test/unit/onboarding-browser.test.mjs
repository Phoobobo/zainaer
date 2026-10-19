// SPDX-License-Identifier: MIT; Copyright (c) 2026 Phoobobo
// Run after building: node test/unit/onboarding-browser.test.mjs
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
  if (!fs.existsSync(file) || fs.statSync(file).isDirectory())
    file = path.join(root, "index.html");
  res.setHeader(
    "Content-Type",
    file.endsWith(".js")
      ? "text/javascript"
      : file.endsWith(".css")
        ? "text/css"
        : file.endsWith(".json")
          ? "application/json"
          : "text/html",
  );
  res.end(fs.readFileSync(file));
});
(async () => {
  await new Promise((r) => server.listen(0, "127.0.0.1", r));
  const origin = `http://127.0.0.1:${server.address().port}`;
  const browser = await chromium.launch({
    headless: true,
    executablePath: process.env.PLAYWRIGHT_EXECUTABLE_PATH,
  });
  try {
    for (const registration of [false, true]) {
      const context = await browser.newContext();
      const page = await context.newPage();
      const token = "AYQ4W4K5MT4CZOPB2PZRCZ4PTY";
      let registered = false,
        joined = false;
      await page.route("**/api/v1/**", async (route) => {
        const p = new URL(route.request().url()).pathname;
        let data = [];
        let headers = {};
        if (p.endsWith("/status"))
          data = {
            health: true,
            build: { version: "test", commit: "12345678" },
            allowRegistration: true,
            oidc: { enabled: false, allowLocal: true },
          };
        else if (p.endsWith("/users/register")) {
          registered = true;
          data = {};
          expect(route.request().postDataJSON().token).toBe(token);
        } else if (p.endsWith("/users/login")) {
          data = { expiresAt: "2030-01-01T00:00:00Z", attachmentToken: "test" };
          headers = {
            "set-cookie": "hb.auth.session=true; Path=/; SameSite=Lax",
          };
        } else if (p.endsWith("/users/self"))
          data = {
            item: {
              id: "user",
              name: "Tester",
              email: "test@example.com",
              defaultGroupId: registered ? "invited" : "personal",
            },
          };
        else if (p.includes("/groups/invitations/")) {
          joined = true;
          data = { id: "invited" };
        } else if (p.endsWith("/groups/all"))
          data = joined
            ? [
                { id: "personal", name: "我的家" },
                { id: "invited", name: "受邀家庭" },
              ]
            : [
                {
                  id: registered ? "invited" : "personal",
                  name: registered ? "受邀家庭" : "我的家",
                },
              ];
        else if (p.endsWith("/settings")) data = { item: {} };
        else if (p.endsWith("/items"))
          data = { items: [], total: 0, page: 1, pageSize: 20 };
        await route.fulfill({ json: data, headers });
      });
      await page.goto(origin + "/?token=" + token);
      if (registration) {
        await page.locator("#register-email").fill("test@example.com");
        await page.locator("#register-name").fill("Tester");
        await page.locator("#register-password").fill("StrongPass123!");
        await page.getByTestId("confirm-register-button").click();
        await expect(page).not.toHaveURL(/token=/);
      } else await page.getByTestId("register-button").click();
      await page.locator("#login-username").fill("test@example.com");
      await page.locator("#login-password").fill("StrongPass123!");
      await page.locator("#login-form button[type=submit]").click();
      await expect(page).toHaveURL(/\/items/);
      if (!registration) {
        await expect(page.getByRole("dialog")).toBeVisible();
        await expect(page.getByRole("dialog").locator("input")).toHaveValue(
          token,
        );
        await page.getByRole("dialog").locator("button[type=submit]").click();
        await expect(
          page.getByRole("combobox").filter({ hasText: "受邀家庭" }).first(),
        ).toBeVisible();
        await expect(page.getByRole("dialog")).toHaveCount(0);
      } else {
        await expect(
          page.getByRole("combobox").filter({ hasText: "受邀家庭" }).first(),
        ).toBeVisible();
        await expect(page.getByRole("dialog")).toHaveCount(0);
      }
      console.log(
        registration
          ? "PASS invitation registration does not prompt to join again"
          : "PASS existing user retains invitation through login and switches collection after joining",
      );
      await context.close();
    }
    for (const customName of [null, "小明和小红的家"]) {
      const context = await browser.newContext();
      await context.addCookies([
        { name: "hb.auth.session", value: "true", url: origin },
      ]);
      const page = await context.newPage();
      let pending = true;
      let collectionName = "小明的家";
      let rejectNext = customName !== null;
      await page.route("**/api/v1/**", async (route) => {
        const p = new URL(route.request().url()).pathname;
        let data = [];
        if (p.endsWith("/status"))
          data = {
            health: true,
            build: { version: "test", commit: "12345678" },
            oidc: { enabled: false },
          };
        else if (p.endsWith("/users/self"))
          data = {
            item: {
              id: "owner",
              name: "小明",
              defaultGroupId: "personal",
              collectionNamingId: pending ? "personal" : null,
            },
          };
        else if (p.endsWith("/groups/all"))
          data = [{ id: "personal", name: collectionName }];
        else if (p.endsWith("/users/self/collection-name")) {
          if (rejectNext) {
            rejectNext = false;
            await route.fulfill({
              status: 500,
              json: { error: "temporary failure" },
            });
            return;
          }
          collectionName = route.request().postDataJSON().name;
          pending = false;
          await route.fulfill({ status: 204 });
          return;
        } else if (p.endsWith("/settings")) data = { item: {} };
        await route.fulfill({ json: data });
      });
      await page.goto(origin + "/items");
      const form = page.getByTestId("collection-naming-form");
      await expect(form).toBeVisible();
      await expect(form.locator("input")).toHaveValue("小明的家");
      if (customName) {
        await form.locator("input").fill("   ");
        await expect(form.locator("button[type=submit]")).toBeDisabled();
        await form.locator("input").fill(customName);
        await form.locator("button[type=submit]").click();
        await expect(form.getByRole("alert")).toBeVisible();
        await expect(form).toBeVisible();
      }
      await form.locator("button[type=submit]").click();
      await expect(form).toHaveCount(0);
      await expect(
        page
          .getByRole("combobox")
          .filter({ hasText: customName ?? "小明的家" })
          .first(),
      ).toBeVisible();
      await page.reload();
      await expect(
        page
          .getByRole("combobox")
          .filter({ hasText: customName ?? "小明的家" })
          .first(),
      ).toBeVisible();
      await expect(page.getByTestId("collection-naming-form")).toHaveCount(0);
      console.log(
        customName
          ? "PASS custom name, failed-save retry, and completion persists"
          : "PASS default name confirmation and completion persists",
      );
      await context.close();
    }
  } finally {
    await browser.close();
    server.close();
  }
})().catch((e) => {
  console.error(e);
  server.close();
  process.exitCode = 1;
});
