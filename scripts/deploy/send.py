#!/usr/bin/env python3
# SPDX-License-Identifier: MIT
# Copyright (c) 2026 Phoobobo
import json
import os
from pathlib import Path
import shutil
import sys

manifest = json.loads(Path('dist/release.json').read_text())
manifest['sha256'] = Path('dist/BINARY_SHA256').read_text().strip()
preloaded = os.environ.get('ZAINAER_PRELOADED_ARTIFACT', 'false')
if preloaded not in ('true', 'false'):
    raise ValueError('Invalid preloaded artifact option')
if preloaded == 'true':
    manifest['artifact_source'] = 'preloaded'
sys.stdout.buffer.write(json.dumps(manifest).encode() + b'\n')
if preloaded == 'true':
    sys.exit(0)
with Path('dist/zainaer-linux-amd64.gz').open('rb') as stream:
    shutil.copyfileobj(stream, sys.stdout.buffer)
