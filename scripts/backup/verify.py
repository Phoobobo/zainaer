#!/usr/bin/env python3
# SPDX-License-Identifier: MIT
# Copyright (c) 2026 Phoobobo
"""Verify a decrypted archive in a NEW isolated directory. Never restore production."""
import hashlib
import json
from pathlib import Path
import sqlite3
import sys
import tarfile


def verify(archive, destination):
    destination.mkdir(mode=0o700, parents=True, exist_ok=False)
    with tarfile.open(archive, 'r:gz') as tar:
        for member in tar.getmembers():
            path = Path(member.name)
            if path.is_absolute() or '..' in path.parts or not (member.isfile() or member.isdir()):
                raise ValueError('Unsafe archive member')
        tar.extractall(destination)
    manifest = json.loads((destination / 'manifest.json').read_text())
    for name, expected in manifest['files'].items():
        relative = Path(name)
        if relative.is_absolute() or '..' in relative.parts:
            raise ValueError('Unsafe manifest path')
        actual = hashlib.sha256((destination / relative).read_bytes()).hexdigest()
        if actual != expected:
            raise ValueError('Restored file checksum mismatch')
    database = destination / 'data' / manifest['dbRelative']
    if not database.resolve().is_relative_to((destination / 'data').resolve()):
        raise ValueError('Unsafe database path')
    with sqlite3.connect(database.as_uri() + '?mode=ro', uri=True) as connection:
        if connection.execute('PRAGMA integrity_check').fetchall() != [('ok',)]:
            raise ValueError('Restored database integrity check failed')
        if connection.execute('PRAGMA foreign_key_check').fetchone():
            raise ValueError('Restored database foreign key check failed')
        for table, expected in manifest['rowCounts'].items():
            if table not in ('entities', 'attachments', 'groups', 'users', 'tags'):
                raise ValueError('Unexpected table in manifest')
            actual = connection.execute('SELECT count(*) FROM "' + table + '"').fetchone()[0]
            if actual != expected:
                raise ValueError('Restored row count mismatch')
        if 'attachments' in manifest['rowCounts']:
            for (value,) in connection.execute('SELECT path FROM attachments'):
                path = Path(value.replace('\\', '/'))
                if path.is_absolute() or '..' in path.parts or not (database.parent / path).is_file():
                    raise ValueError('Restored photo is missing')
    print(json.dumps({'verified': True, 'build': manifest['build'], 'rowCounts': manifest['rowCounts'],
                      'verifiedFiles': len(manifest['files'])}))
    return manifest


if __name__ == '__main__':
    verify(Path(sys.argv[1]), Path(sys.argv[2]).absolute())
