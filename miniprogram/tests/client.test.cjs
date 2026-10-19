// SPDX-License-Identifier: MIT; Copyright (c) 2026 Phoobobo
const { test } = require("node:test");
const assert = require("node:assert/strict");
const fs = require("node:fs");
const vm = require("node:vm");
const { HouseholdItems, queryString } = require("../dist/shared/household");
test("shared API encodes filters and IDs without DOM globals", async () => {
  let seen;
  const http = { get: async (x) => ((seen = x), { data: { items: [] } }) };
  const api = new HouseholdItems(
    http,
    (p, q) => "/api/v1" + p + queryString(q),
  );
  await api.getAll({
    q: "盒子 & 钥匙",
    tags: ["a", "b"],
    includeArchived: false,
  });
  assert.match(seen.url, /q=%E7%9B%92/);
  assert.match(seen.url, /tags=a&tags=b/);
  assert.match(seen.url, /includeArchived=false/);
  assert.equal(queryString({ q: undefined }), "");
  await api.get("a/b");
  assert.equal(seen.url, "/api/v1/entities/a%2Fb");
});
function client(responder) {
  const store = new Map();
  const requests = [];
  let redirected = false;
  const wx = {
    getStorageSync: (k) => store.get(k),
    setStorageSync: (k, v) => store.set(k, v),
    removeStorageSync: (k) => store.delete(k),
    reLaunch: () => (redirected = true),
    login: (o) => o.success({ code: "fresh-code" }),
    request: (o) => {
      requests.push(o);
      o.success(responder(o));
    },
  };
  const module = { exports: {} };
  vm.runInNewContext(
    fs.readFileSync(require.resolve("../src/lib/api.js"), "utf8"),
    {
      require: (p) =>
        p === "../config"
          ? { apiOrigin: "https://example.com" }
          : require("../dist/shared/household"),
      module,
      wx,
    },
  );
  return { api: module.exports, requests, store, redirected: () => redirected };
}
test("wx login sends fresh code and stores only application session", async () => {
  const c = client(() => ({
    statusCode: 200,
    data: { token: "Bearer app", attachmentToken: "attachment" },
  }));
  await c.api.login();
  assert.deepEqual(JSON.parse(JSON.stringify(c.requests[0].data)), {
    code: "fresh-code",
    register: false,
  });
  assert.equal(c.api.session().token, "Bearer app");
});
test("requests carry collection and session; expired reads clear both without retry", async () => {
  const c = client(() => ({ statusCode: 401, data: {} }));
  c.api.save({ token: "Bearer old" });
  c.store.set("zainaer-tenant", "collection");
  await assert.rejects(c.api.call("GET", "/entities"));
  assert.equal(c.requests.length, 1);
  assert.equal(c.requests[0].header["X-Tenant"], "collection");
  assert.equal(c.requests[0].header.Authorization, "Bearer old");
  assert.equal(c.api.session().token, undefined);
  assert.equal(c.api.tenant(), "");
  assert.equal(c.redirected(), true);
});
test("unbound identity never automatically registers", async () => {
  const c = client(() => ({
    statusCode: 409,
    data: { code: "wechat_unbound" },
  }));
  await assert.rejects(c.api.login(), (e) => e.data.code === "wechat_unbound");
  assert.equal(c.requests.length, 1);
  assert.equal(c.api.session().token, undefined);
});
test("page sources and app configuration are valid", () => {
  const app = JSON.parse(fs.readFileSync(require.resolve("../src/app.json")));
  for (const p of app.pages) {
    new vm.Script(
      fs.readFileSync(require.resolve("../src/" + p + ".js"), "utf8"),
    );
    assert.ok(fs.statSync(require.resolve("../src/" + p + ".wxml")).size > 0);
  }
});
