#!/usr/bin/env python3
# SPDX-License-Identifier: MIT
# Copyright (c) 2026 Phoobobo
"""Validate only fixed filenames; never feed untrusted SHA256SUMS paths to sha256sum."""
import gzip
import hashlib
import json
from pathlib import Path
import re
import subprocess
import sys


def verify(tag, directory=Path('dist')):
    if not re.fullmatch(r'v(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)', tag):
        raise ValueError('Invalid release tag')
    commit = subprocess.check_output(['git', 'rev-parse', tag + '^{commit}'], text=True).strip()
    subprocess.run(['git', 'merge-base', '--is-ancestor', commit, 'origin/main'], check=True)
    version = subprocess.check_output(['git', 'show', commit + ':VERSION'], text=True).strip()
    manifest = json.loads((directory / 'release.json').read_text())
    if manifest != {'version': tag[1:], 'commit': commit} or version != tag[1:]:
        raise ValueError('Release manifest does not match tag, source and VERSION')
    checksums = {}
    for line in (directory / 'SHA256SUMS').read_text().splitlines():
        match = re.fullmatch(r'([0-9a-f]{64})  ([A-Za-z0-9_.-]+)', line)
        if not match or match[2] in checksums:
            raise ValueError('Invalid checksum manifest')
        checksums[match[2]] = match[1]
    for name in ['zainaer-linux-amd64.gz', 'release.json', 'BINARY_SHA256']:
        if hashlib.sha256((directory / name).read_bytes()).hexdigest() != checksums.get(name):
            raise ValueError('Release asset checksum mismatch: ' + name)
    digest = hashlib.sha256()
    size = 0
    with gzip.open(directory / 'zainaer-linux-amd64.gz', 'rb') as stream:
        for block in iter(lambda: stream.read(1024 * 1024), b''):
            size += len(block)
            if size > 512 * 1024 * 1024:
                raise ValueError('Executable too large')
            digest.update(block)
    if digest.hexdigest() != (directory / 'BINARY_SHA256').read_text().strip():
        raise ValueError('Uncompressed executable checksum mismatch')
    return manifest


if __name__ == '__main__':
    print(json.dumps(verify(sys.argv[1])))
