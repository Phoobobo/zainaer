#!/bin/sh
# SPDX-License-Identifier: MIT; Copyright (c) 2026 Phoobobo
set -eu
cd "$(dirname "$0")/.."
node frontend/node_modules/typescript/bin/tsc --strict --noEmit --skipLibCheck \
 --target ES2020 --moduleResolution bundler --module esnext frontend/lib/passkeys.ts shared/transport.typecheck.ts
cd backend
GOOS="$(go env GOHOSTOS)" GOARCH="$(go env GOHOSTARCH)" \
 go test ./internal/core/services ./app/api/handlers/v1 ./app/api -count=1
