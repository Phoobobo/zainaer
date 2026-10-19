#!/usr/bin/env python3
# SPDX-License-Identifier: MIT
# Copyright (c) 2026 Phoobobo
"""Online SQLite and attachment backups; encrypt before leaving the host."""
import base64
import contextlib
import datetime as dt
import fcntl
import hashlib
import json
import os
from pathlib import Path
import re
import shutil
import sqlite3
import subprocess
import tarfile
import tempfile
import time
import urllib.request
import uuid

UTC = dt.timezone.utc


def digest(path):
    h = hashlib.sha256()
    with path.open('rb') as stream:
        for block in iter(lambda: stream.read(1024 * 1024), b''):
            h.update(block)
    return h.hexdigest()


def copy_files(source, target, db_relative):
    excluded = {db_relative.as_posix() + suffix for suffix in ('', '-wal', '-shm', '-journal')}
    for current, dirs, names in os.walk(source, followlinks=False):
        current = Path(current)
        for name in dirs + names:
            if (current / name).is_symlink():
                raise ValueError('Symlinks in application storage are not supported')
        relative = current.relative_to(source)
        (target / relative).mkdir(parents=True, exist_ok=True)
        for name in names:
            rel = relative / name
            if rel.as_posix() in excluded or name.endswith('.tmp'):
                continue
            try:
                shutil.copy2(current / name, target / rel, follow_symlinks=False)
            except FileNotFoundError:
                # A deleted attachment can disappear during the live copy.
                # The database snapshot's references are checked afterwards.
                continue


def validate_snapshot(data, db_relative):
    db = data / db_relative
    counts = {}
    with sqlite3.connect(db.as_uri() + '?mode=ro', uri=True) as connection:
        if connection.execute('PRAGMA integrity_check').fetchall() != [('ok',)]:
            raise ValueError('SQLite integrity check failed')
        if connection.execute('PRAGMA foreign_key_check').fetchone():
            raise ValueError('SQLite foreign key check failed')
        tables = {row[0] for row in connection.execute("SELECT name FROM sqlite_master WHERE type='table'")}
        for table in ('entities', 'attachments', 'groups', 'users', 'tags'):
            if table in tables:
                counts[table] = connection.execute('SELECT count(*) FROM "' + table + '"').fetchone()[0]
        if 'attachments' in tables:
            for (value,) in connection.execute('SELECT path FROM attachments'):
                rel = Path(value.replace('\\', '/'))
                if rel.is_absolute() or '..' in rel.parts or not (db.parent / rel).is_file():
                    raise ValueError('Snapshot is missing a referenced attachment')
    ai = db.parent / 'ai'
    for path in ai.glob('*.json'):
        value = json.loads(path.read_text())
        encrypted = value.get('encryptedKey')
        if encrypted:
            from cryptography.hazmat.primitives.ciphers.aead import AESGCM
            key = (ai / 'master.key').read_bytes()
            raw = base64.b64decode(encrypted, validate=True)
            if len(key) != 32 or not AESGCM(key).decrypt(raw[:12], raw[12:], b'zainaer-ai-v1'):
                raise ValueError('Model configuration cannot be decrypted')
    return counts


def snapshot(source, target, db_relative):
    """Copy attachments on both sides of SQLite's consistent online snapshot.

    Preserve earlier copies, so a deletion after the DB snapshot cannot remove
    an attachment already captured. Retry if concurrent creation/deletion leaves
    a referenced file missing. Never publish an incomplete snapshot.
    """
    last = None
    for attempt in range(3):
        try:
            copy_files(source, target, db_relative)
            dest = target / db_relative
            dest.parent.mkdir(parents=True, exist_ok=True)
            with sqlite3.connect((source / db_relative).as_uri() + '?mode=ro', uri=True, timeout=20) as src:
                with sqlite3.connect(dest) as dst:
                    src.backup(dst, pages=256, sleep=0.05)
                    dst.execute('PRAGMA journal_mode=DELETE')
            copy_files(source, target, db_relative)
            return validate_snapshot(target, db_relative)
        except (FileNotFoundError, ValueError, sqlite3.Error) as error:
            last = error
            if attempt != 2:
                time.sleep(1)
    raise RuntimeError('Cannot obtain a complete online snapshot') from last


@contextlib.contextmanager
def locked(path, seconds=600):
    with path.open('a') as stream:
        deadline = time.monotonic() + seconds
        while True:
            try:
                fcntl.flock(stream, fcntl.LOCK_EX | fcntl.LOCK_NB)
                break
            except BlockingIOError:
                if time.monotonic() > deadline:
                    raise TimeoutError('Another deployment or backup still holds the lock')
                time.sleep(1)
        yield


def retention(files, now):
    """Keep 48 recent hourly points, daily points for 30 days, monthly for a year."""
    values = []
    for path in files:
        match = re.fullmatch(r'zainaer-(\d{8}T\d{6}Z)-[0-9a-f]{8}\.tar\.gz\.age', path.name)
        if match and path.is_file() and not path.is_symlink():
            stamp = dt.datetime.strptime(match[1], '%Y%m%dT%H%M%SZ').replace(tzinfo=UTC)
            values.append((stamp, path))
    values.sort(reverse=True)
    keep = {path for _, path in values[:48]}
    days, months = set(), set()
    for stamp, path in values:
        if stamp > now - dt.timedelta(days=30) and stamp.date() not in days:
            days.add(stamp.date()); keep.add(path)
        if stamp > now - dt.timedelta(days=366) and (stamp.year, stamp.month) not in months:
            months.add((stamp.year, stamp.month)); keep.add(path)
    return [path for _, path in values if path not in keep]


def build_info():
    # Back up recoverable data even when the application itself is unavailable.
    try:
        with urllib.request.urlopen('http://127.0.0.1:7745/api/v1/status', timeout=5) as response:
            return json.load(response)['build']
    except Exception:
        return {'version': 'unknown', 'commit': 'unknown', 'statusUnavailable': True}


def run(config):
    os.umask(0o077)
    root = Path(config.get('root', '/opt/zainaer'))
    source = root / 'data'
    relative = Path(config.get('db_relative', '.data/homebox.db'))
    if relative.is_absolute() or '..' in relative.parts:
        raise ValueError('Invalid database path')
    output = Path(config.get('destination', '/var/backups/zainaer'))
    output.mkdir(parents=True, exist_ok=True, mode=0o700)
    now = dt.datetime.now(UTC)
    name = 'zainaer-' + now.strftime('%Y%m%dT%H%M%SZ') + '-' + uuid.uuid4().hex[:8] + '.tar.gz.age'
    archive = output / name
    with locked(output / '.backup.lock'), tempfile.TemporaryDirectory(prefix='.snapshot-', dir=output) as temporary:
        stage = Path(temporary)
        with locked(root / '.deploy.lock'):
            size = sum(p.stat().st_size for p in source.rglob('*') if p.is_file() and not p.is_symlink())
            if shutil.disk_usage(output).free < size * 4 + 256 * 1024 * 1024:
                raise RuntimeError('Insufficient space for a safe backup')
            counts = snapshot(source, stage / 'data', relative)
            shutil.copy2(root / 'service.env', stage / 'service.env')
            build = build_info()
        files = {str(p.relative_to(stage)): digest(p) for p in stage.rglob('*') if p.is_file()}
        manifest = {'format': 1, 'createdAt': now.isoformat(), 'build': build,
                    'dbRelative': str(relative), 'rowCounts': counts, 'files': files}
        (stage / 'manifest.json').write_text(json.dumps(manifest, ensure_ascii=False, indent=2))
        raw = stage / 'snapshot.tar.gz'
        with tarfile.open(raw, 'w:gz') as tar:
            for entry in ('data', 'service.env', 'manifest.json'):
                tar.add(stage / entry, arcname=entry, recursive=True)
        partial = output / (name + '.partial')
        try:
            subprocess.run(['age', '-r', config['recipient'], '-o', str(partial), str(raw)], check=True, timeout=600)
            os.replace(partial, archive)
        finally:
            partial.unlink(missing_ok=True)
        checksum = digest(archive)
        receipt = {'file': name, 'sha256': checksum, 'bytes': archive.stat().st_size,
                   'createdAt': now.isoformat(), 'build': build, 'rowCounts': counts,
                   'offsiteVerified': False}
        receipt_path = output / (name + '.json')
        receipt_path.write_text(json.dumps(receipt, indent=2))
        # An optional uploader is installed/configured separately after storage approval.
        if config.get('upload_command'):
            subprocess.run(config['upload_command'] + [str(archive), str(receipt_path)], check=True, timeout=1200)
            receipt = json.loads(receipt_path.read_text())
            if not receipt.get('offsiteVerified'):
                raise RuntimeError('Uploader did not verify the offsite object')
        for old in retention(output.glob('zainaer-*.tar.gz.age'), now):
            old.unlink()
            Path(str(old) + '.json').unlink(missing_ok=True)
        (output / 'latest.json').write_text(json.dumps(receipt, indent=2))
        print(json.dumps({'ok': True, 'file': name, 'bytes': receipt['bytes'],
                          'offsiteVerified': receipt['offsiteVerified'], 'rowCounts': counts}))
        return archive


if __name__ == '__main__':
    import sys
    run(json.loads(Path(sys.argv[1] if len(sys.argv) > 1 else '/etc/zainaer/backup.json').read_text()))
