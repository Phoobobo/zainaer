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
  } finally {
    await browser.close();
    server.close();
  }
})().catch((e) => {
  console.error(e);
  server.close();
  process.exitCode = 1;
});
