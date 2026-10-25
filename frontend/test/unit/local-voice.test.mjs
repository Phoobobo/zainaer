// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Phoobobo
import { test } from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import vm from "node:vm";
import ts from "typescript";
const code = ts.transpileModule(readFileSync(new URL("../../lib/local-voice.ts", import.meta.url), "utf8"), {
  compilerOptions: { target: ts.ScriptTarget.ES2022, module: ts.ModuleKind.CommonJS },
}).outputText;
function harness({ available = async () => "available", supported = true, local = true } = {}) {
  const events = [], instances = [], timers = new Map();
  let timerId = 0;
  class Recognition {
    constructor() { if (local) this.processLocally = false; instances.push(this); }
    static available = available;
    start() { this.started = true; this.onstart?.(); }
    stop() { this.stopped = true; }
    abort() { this.aborted = true; }
  }
  const scope = { exports: {}, setTimeout: fn => { timers.set(++timerId, fn); return timerId; }, clearTimeout: id => timers.delete(id) };
  vm.runInNewContext(code, scope);
  const voice = scope.exports.createLocalVoice({
    recognition: () => supported ? Recognition : undefined,
    state: state => events.push(["state", state]), error: error => events.push(["error", error]), text: text => events.push(["text", text]),
  });
  return { voice, events, instances, timers, Recognition };
}
const result = text => ({ results: [{ isFinal: true, 0: { transcript: text } }] });
test("local recognition stops and transcribes once without automatic submission", async () => {
  const h = harness(); await h.voice.start("zh-CN"); const r = h.instances[0];
  assert.equal(r.processLocally, true); assert.equal(r.lang, "zh-CN"); assert.equal(r.continuous, false);
  await h.voice.start("en-US"); assert.equal(h.instances.length, 1);
  h.voice.stop(); assert.equal(r.stopped, true);
  r.onresult(result("合成钥匙")); r.onend();
  assert.deepEqual(h.events.filter(e => e[0] === "text"), [["text", "合成钥匙"]]); assert.equal(h.timers.size, 0);
});
test("unsupported browsers and absent language packs never start the microphone", async () => {
  for (const options of [{ supported: false }, { local: false }, { available: async () => "downloadable" }]) {
    const h = harness(options); await h.voice.start("zh-CN"); assert.ok(h.instances.every(r => !r.started));
    assert.ok(h.events.some(e => e[0] === "error" && e[1])); assert.equal(h.timers.size, 0);
  }
});
test("permission denial, interruption and empty speech permit an explicit retry", async () => {
  for (const error of ["not-allowed", "audio-capture", "aborted", "network"]) {
    const h = harness(); await h.voice.start("en-US"); h.instances[0].onerror({ error });
    assert.equal(h.instances[0].aborted, true); await h.voice.start("en-US"); assert.equal(h.instances.length, 2);
    h.instances[1].onend(); assert.ok(h.events.some(e => e[1] === "no-speech"));
  }
});
test("cancellation discards late results and pending checks", async () => {
  let resolve; const h = harness({ available: () => new Promise(r => { resolve = r; }) });
  const pending = h.voice.start("zh-CN"); h.voice.cancel(); resolve("available"); await pending; assert.ok(!h.instances[0].started);
  h.Recognition.available = async () => "available";
  await h.voice.start("zh-CN"); const r = h.instances[1], late = r.onresult;
  r.onresult(result("discard")); h.voice.cancel(); late(result("late"));
  assert.equal(r.aborted, true); assert.ok(!h.events.some(e => e[0] === "text"));
});
test("bounded sessions and stop failures clean up the microphone", async () => {
  const h = harness(); await h.voice.start("en-US"); [...h.timers.values()][0](); assert.equal(h.instances[0].aborted, true);
  await h.voice.start("en-US"); h.instances[1].stop = () => { throw Error(); }; h.voice.stop();
  assert.equal(h.instances[1].aborted, true); assert.equal(h.timers.size, 0);
});
