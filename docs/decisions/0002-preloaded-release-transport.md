<!-- SPDX-License-Identifier: MIT; Copyright (c) 2026 Phoobobo -->
# Preloaded Release transport

## Context

Some GitHub-hosted runner routes upload production artifacts too slowly for a reliable deployment. Retrying has sometimes selected a fast route, but repeated retries are not a dependable delivery mechanism.

## Decision

Keep the ordinary streamed transport as the default. Add an explicit `preloaded_artifact` deployment input for a trusted operator to preload the **published GitHub Release** compressed asset through Workbench. No locally built executable is eligible.

The operator verifies the Release against its tag, source commit and checksum manifest, then stages its compressed asset under `/opt/zainaer/release-cache/<uncompressed-sha256>.gz`. The directory and file must be administrator-owned and not writable by other users. Installation or replacement of the deployment helper remains a separate trusted administrator operation; CI cannot install a helper.

Actions still downloads and validates the selected Release. In preloaded mode it sends only the same metadata plus a fixed source marker. The installed helper can select only the fixed cache path derived from the validated hash. It rejects symlinks, nonregular files, unsafe ownership/permissions and excessive size, then runs the existing decompression, checksum and ELF validation. Backup, atomic replacement, exact version/commit health checks and full rollback remain unchanged.

## Consequences

This avoids uploading the executable over a poor runner SSH route. A missing or invalid cache entry fails before the service stops. The input accepts no path, URL or shell command, and does not grant the deployment account write access to the cache. Cached assets are retained for operator-managed cleanup; deployment snapshots and application data are never pruned by this mode.
