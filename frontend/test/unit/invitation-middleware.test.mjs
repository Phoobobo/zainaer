// SPDX-License-Identifier: MIT; Copyright (c) 2026 Phoobobo
import { readFileSync } from "node:fs";
import vm from "node:vm";
import assert from "node:assert/strict";
import ts from "typescript";

// Exercise the actual page guard with a stale current route, as happens during
// cold authenticated navigation. Browser mocks can hide this middleware timing.
const source = readFileSync(
  new URL("../../pages/index.vue", import.meta.url),
  "utf8",
);
const script = source.match(/<script setup lang="ts">([\s\S]*?)<\/script>/)[1];
const ast = ts.createSourceFile(
  "index.ts",
  script,
  ts.ScriptTarget.Latest,
  true,
);
let guard;
function visit(node) {
  if (
    ts.isCallExpression(node) &&
    node.expression.getText(ast) === "definePageMeta"
  ) {
    const middleware = node.arguments[0].properties.find(
      (p) => p.name?.getText(ast) === "middleware",
    );
    guard = middleware.initializer.elements[0].getText(ast);
  }
  ts.forEachChild(node, visit);
}
visit(ast);
assert.ok(guard, "the login page must define its redirect guard");
const js = ts.transpileModule(`const guard = (${guard}); guard;`, {
  compilerOptions: {
    target: ts.ScriptTarget.ES2022,
    module: ts.ModuleKind.None,
  },
}).outputText;
const redirect = vm.runInNewContext(js, {
  useAuthContext: () => ({ isAuthorized: () => true }),
  useRoute: () => ({ path: "/items", query: { token: "STALE" } }),
});
const token = "AYQ4W4K5MT4CZOPB2PZRCZ4PTY";
assert.equal(redirect({ path: "/", query: { token } }), `/join?token=${token}`);
assert.equal(
  redirect({ path: "/", query: {} }),
  "/home",
  "a stale token must never replace the target query",
);
assert.equal(
  redirect({ path: "/", query: { token: [token, "other"] } }),
  "/home",
);
console.log(
  "PASS invitation guard uses target route and ignores stale or ambiguous tokens",
);
