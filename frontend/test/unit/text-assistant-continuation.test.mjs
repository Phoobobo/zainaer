// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Phoobobo
import { test } from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import vm from "node:vm";
import ts from "typescript";

// Execute the actual component's setup logic with UI primitives and HTTP mocked.
const component = readFileSync(new URL("../../components/App/TextAssistant.vue", import.meta.url), "utf8");
const setup = component.match(/<script setup lang="ts">([\s\S]*?)<\/script>/)[1].replace(/^\s*import .*;$/gm, "");
const code = ts.transpileModule(setup + "\nglobalThis.subject = { messages, draft, confirm, busy, saving, error, configured };", {
  compilerOptions: { target: ts.ScriptTarget.ES2022, module: ts.ModuleKind.None },
}).outputText;

function harness(responses) {
  const requests = [];
  const scope = {
    ref: value => ({ value }),
    computed: fn => ({ get value() { return fn(); } }),
    useI18n: () => ({ t: key => key, locale: { value: "zh" } }),
    useViewPreferences: () => ({ value: { collectionId: "collection" } }),
    watch: () => {}, onBeforeUnmount: () => {}, nextTick: async () => {},
    AbortController,
    fetch: async (url, options) => {
      requests.push({ url, body: JSON.parse(options.body) });
      const response = responses.shift();
      assert.ok(response, "unexpected extra request");
      return { ok: response.ok !== false, json: async () => response.body };
    },
  };
  vm.runInNewContext(code, scope);
  const subject = scope.subject;
  subject.configured.value = true;
  const locationDraft = { id: "location-draft", proposal: { kind: "location", name: "B", description: "" }, parentPath: "", tags: [] };
  subject.messages.value.push({ role: "user", content: "把物品A存到位置B" }, { role: "assistant", content: "确认位置", drafts: [locationDraft] });
  return { subject, locationDraft, requests };
}

test("saving missing location continues original request and item still requires confirmation", async () => {
  const item = { id: "item-draft", proposal: { kind: "item", name: "A", description: "" }, parentPath: "B", tags: [] };
  const h = harness([
    { body: { id: "location-id", kind: "location", name: "B", path: "B" } },
    { body: { reply: "确认物品", drafts: [item], items: [] } },
    { body: { id: "item-id", kind: "item", name: "A", path: "B / A" } },
  ]);
  h.subject.draft.value = "尚未发送的输入";
  await h.subject.confirm(h.locationDraft);
  assert.equal(h.requests.length, 2);
  assert.equal(h.requests[1].url, "/api/v1/ai/chat");
  assert.ok(h.requests[1].body.messages.some(m => m.content === "把物品A存到位置B"));
  assert.ok(h.requests[1].body.messages.some(m => m.content.includes("已保存") && m.content.includes("B")));
  assert.equal(h.locationDraft.state, "saved");
  assert.equal(item.state, undefined);
  assert.equal(h.subject.draft.value, "尚未发送的输入");
  await h.subject.confirm(item);
  assert.equal(h.requests.length, 3);
  assert.equal(item.state, "saved");
  assert.equal(h.subject.busy.value, false);
});

test("cancelled or failed prerequisite never starts continuation", async () => {
  for (const cancel of [true, false]) {
    const h = harness([cancel ? { body: { cancelled: true } } : { ok: false, body: { error: "save failed" } }]);
    await h.subject.confirm(h.locationDraft, cancel);
    assert.equal(h.requests.length, 1);
    assert.equal(h.locationDraft.state, cancel ? "cancelled" : undefined);
  }
});

test("continuation failure preserves successfully saved location and original input", async () => {
  const h = harness([
    { body: { id: "location-id", kind: "location", name: "B", path: "B" } },
    { ok: false, body: { error: "model unavailable" } },
  ]);
  h.subject.draft.value = "尚未发送的输入";
  await h.subject.confirm(h.locationDraft);
  assert.equal(h.locationDraft.state, "saved");
  assert.equal(h.subject.draft.value, "尚未发送的输入");
  assert.equal(h.subject.error.value, "model unavailable");
  assert.equal(h.subject.busy.value, false);
  assert.equal(h.subject.saving.value, "");
});
