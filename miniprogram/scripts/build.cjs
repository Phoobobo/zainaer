// SPDX-License-Identifier: MIT; Copyright (c) 2026 Phoobobo
const fs = require("node:fs");
const path = require("node:path");
const ts = require("../../frontend/node_modules/typescript");
const root = path.resolve(__dirname, "..");
const out = path.join(root, "dist");
fs.rmSync(out, { recursive: true, force: true });
fs.cpSync(path.join(root, "src"), out, { recursive: true });
const local = path.join(root, "src/config.local.js");
const config = fs.existsSync(local) ? require(local) : require("../src/config");
if (config.apiOrigin && !/^https:\/\/[^/?#]+$/.test(config.apiOrigin))
  throw new Error("apiOrigin must be an HTTPS origin without a trailing slash");
fs.writeFileSync(
  path.join(out, "config.js"),
  "module.exports = " +
    JSON.stringify({ apiOrigin: config.apiOrigin || "" }) +
    ";\n",
);
fs.rmSync(path.join(out, "config.local.js"), { force: true });
fs.mkdirSync(path.join(out, "shared"), { recursive: true });
fs.writeFileSync(
  path.join(out, "shared/household.js"),
  ts.transpileModule(
    fs.readFileSync(path.join(root, "../shared/household.ts"), "utf8"),
    {
      compilerOptions: {
        module: ts.ModuleKind.CommonJS,
        target: ts.ScriptTarget.ES2018,
      },
    },
  ).outputText,
);
console.log(
  "Ready: import miniprogram/ into WeChat DevTools. AppID and API origin are configured separately.",
);
