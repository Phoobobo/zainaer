// SPDX-License-Identifier: MIT; Copyright (c) 2026 Phoobobo
// Run against an isolated empty instance only: PASSKEY_TEST_ORIGIN=http://localhost:17745 node test/passkeys.e2e.cjs
const { chromium, request, expect } = require("@playwright/test");
const assert = require("node:assert/strict");
(async () => {
  const origin = process.env.PASSKEY_TEST_ORIGIN;
  assert(
    origin && new URL(origin).hostname === "localhost",
    "This test creates a user; only isolated localhost instances are allowed",
  );
  const email = `passkey-${Date.now()}@example.test`;
  const password = "Test-only-password-456";
  const api = await request.newContext({ baseURL: origin });
  const register = await api.post("/api/v1/users/register", {
    data: { name: "Passkey test", email, password, token: "" },
  });
  assert(register.ok(), `register ${register.status()}`);
  await api.dispose();
  const browser = await chromium.launch({ headless: true });
  try {
    const context = await browser.newContext({ locale: "en-US" });
    const page = await context.newPage();
    page.setDefaultTimeout(12000);
    page.setDefaultNavigationTimeout(15000);
    page.on("pageerror", (error) => console.error("page error", error.message));
    const cdp = await context.newCDPSession(page);
    await cdp.send("WebAuthn.enable", { enableUI: false });
    await cdp.send("WebAuthn.addVirtualAuthenticator", {
      options: {
        protocol: "ctap2",
        ctap2Version: "ctap2_1",
        transport: "usb",
        hasResidentKey: true,
        hasUserVerification: true,
        isUserVerified: true,
        automaticPresenceSimulation: true,
      },
    });
    await page.goto(origin);
    await page.locator("#login-username").fill(email);
    await page.locator("#login-password").fill(password);
    await page.locator("#login-form button[type=submit]").click();
    await page.waitForURL("**/home");
    await page
      .getByRole("button", { name: "Save and continue", exact: true })
      .click();
    await page.goto(`${origin}/profile`);
    await page.locator("input[name=passkey-name]").fill("Browser test key");
    await page.locator("input[name=passkey-password]").fill(password);
    const [added] = await Promise.all([
      page.waitForResponse((r) =>
        r.url().endsWith("/users/self/passkeys/finish"),
      ),
      page.getByRole("button", { name: "Add passkey", exact: true }).click(),
    ]);
    assert.equal(added.status(), 201);
    await expect(
      page.getByText("Browser test key", { exact: true }),
    ).toBeVisible();
    await page.screenshot({
      path: "/tmp/zainaer-passkey-profile.png",
      fullPage: true,
    });
    await page.evaluate(() =>
      fetch("/api/v1/users/logout", { method: "POST" }),
    );
    await page.goto(origin);
    const [loginResponse] = await Promise.all([
      page.waitForResponse((r) =>
        r.url().endsWith("/users/login/passkey/finish"),
      ),
      page
        .getByRole("button", { name: "Sign in with a passkey", exact: true })
        .click(),
    ]);
    assert.equal(loginResponse.status(), 200);
    await page.waitForURL("**/home");
    await page.goto(`${origin}/profile`);
    await page
      .getByRole("button", { name: "Delete Browser test key", exact: true })
      .click();
    const [deleted] = await Promise.all([
      page.waitForResponse(
        (r) =>
          r.request().method() === "DELETE" &&
          r.url().includes("/users/self/passkeys/"),
      ),
      page.getByRole("button", { name: "Confirm", exact: true }).click(),
    ]);
    assert.equal(deleted.status(), 204);
    await expect(
      page.getByText("No passkeys added yet.", { exact: true }),
    ).toBeVisible();
    await page.evaluate(() =>
      fetch("/api/v1/users/logout", { method: "POST" }),
    );
    await page.goto(origin);
    const [rejected] = await Promise.all([
      page.waitForResponse((r) =>
        r.url().endsWith("/users/login/passkey/finish"),
      ),
      page
        .getByRole("button", { name: "Sign in with a passkey", exact: true })
        .click(),
    ]);
    assert.equal(rejected.status(), 401);
    await page.locator("#login-username").fill(email);
    await page.locator("#login-password").fill(password);
    await page.locator("#login-form button[type=submit]").click();
    await page.waitForURL("**/home");
    console.log(
      "PASS: password login, passkey registration, discoverable login, removal, revoked key rejection, password fallback",
    );
    await page.evaluate(() =>
      fetch("/api/v1/users/logout", { method: "POST" }),
    );
    await page.goto(origin);
    await page.setViewportSize({ width: 390, height: 844 });
    await page.screenshot({
      path: "/tmp/zainaer-passkey-login-mobile.png",
      fullPage: true,
    });
    // Feature detection must retain a usable password form in unsupported browsers.
    await page.addInitScript(() => {
      Object.defineProperty(window, "PublicKeyCredential", {
        value: undefined,
      });
    });
    await page.reload();
    await expect(
      page.getByRole("button", { name: "Sign in with a passkey", exact: true }),
    ).toBeDisabled();
    await expect(page.locator("#login-form button[type=submit]")).toBeEnabled();
    console.log("PASS: unsupported-browser password fallback");
  } finally {
    await browser.close();
  }
})().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
