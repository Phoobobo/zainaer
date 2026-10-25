#!/bin/sh
set -eu
cd "$(dirname "$0")/.."
release_version="$(cat VERSION)"
release_commit="$(git rev-parse HEAD)"
release_time="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
sh scripts/check-passkeys.sh
cd frontend
./node_modules/.bin/nuxt generate
# Remove obsolete hashed chunks and deleted routes from previous builds.
find ../backend/app/api/static/public -mindepth 1 ! -name .gitkeep -delete
cp -R .output/public/. ../backend/app/api/static/public/
cd ../backend
go build -ldflags "-X main.version=$release_version -X main.commit=$release_commit -X main.buildTime=$release_time" -o "${HOUSEHOLD_BUILD_OUTPUT:-homebox}" ./app/api
