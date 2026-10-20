#!/bin/sh
# SPDX-License-Identifier: MIT; Copyright (c) 2026 Phoobobo
set -eu
cd "$(dirname "$0")/.."
node miniprogram/scripts/build.cjs
node --test miniprogram/tests/*.test.cjs
node frontend/node_modules/typescript/bin/tsc --strict --noEmit --skipLibCheck \
  --target ES2020 --moduleResolution bundler --module esnext shared/transport.typecheck.ts
# Build callers may cross-compile the application. Run tests on the build host.
cd backend
GOOS="$(go env GOHOSTOS)" GOARCH="$(go env GOHOSTARCH)" \
  go test ./internal/sys/wechat ./internal/core/services ./app/api -count=1
