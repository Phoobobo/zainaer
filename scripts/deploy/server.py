#!/usr/bin/env python3
# SPDX-License-Identifier: MIT
# Copyright (c) 2026 Phoobobo
"""Root-owned forced SSH command. Accept one JSON header and a gzipped executable.

No shell arguments, archive extraction, arbitrary paths or remote commands are accepted.
Install this file on ECS; never run a copy supplied in a release payload.
"""
import contextlib
import fcntl
import gzip
import hashlib
import json
import os
from pathlib import Path
import re
import shutil
import signal
import subprocess
import sys
import tempfile
import time
import urllib.error
import urllib.request

ROOT = Path('/opt/zainaer')
SERVICE = 'zainaer'
BASE_URL = 'http://127.0.0.1:7745'
MAX_SIZE = 512 * 1024 * 1024


def metadata(stream):
    line = stream.readline(1025)
    if len(line) > 1024 or not line.endswith(b'\n'):
        raise ValueError('Invalid deployment header')
    value = json.loads(line)
    if set(value) != {'version', 'commit', 'sha256'}:
        raise ValueError('Unexpected deployment header fields')
    for key, pattern in [('version', r'0\.[0-9]+\.[0-9]+|[1-9][0-9]*\.[0-9]+\.[0-9]+'),
                         ('commit', r'[0-9a-f]{40}'), ('sha256', r'[0-9a-f]{64}')]:
        if not isinstance(value[key], str) or not re.fullmatch(pattern, value[key]):
            raise ValueError('Invalid ' + key)
    return value


def unpack(stream, target, expected):
    digest = hashlib.sha256()
    size = 0
    with gzip.GzipFile(fileobj=stream) as src, target.open('xb') as dst:
        while True:
            block = src.read(1024 * 1024)
            if not block:
                break
            size += len(block)
            if size > MAX_SIZE:
                raise ValueError('Executable exceeds size limit')
            digest.update(block)
            dst.write(block)
        dst.flush()
        os.fsync(dst.fileno())
    if digest.hexdigest() != expected:
        raise ValueError('Executable checksum mismatch')
    with target.open('rb') as src:
        header = src.read(20)
    # ELF64, little endian, AMD64. Reject accidental macOS/ARM artifacts.
    if header[:6] != b'\x7fELF\x02\x01' or header[18:20] != b'\x3e\x00':
        raise ValueError('Expected Linux amd64 executable')
    target.chmod(0o755)


def service(action):
    subprocess.run(['systemctl', action, SERVICE], check=True, timeout=45)


def copy_tree(src, dst):
    # Preserve ownership, permissions, symlinks and timestamps for all household data.
    subprocess.run(['cp', '-a', '--', str(src), str(dst)], check=True, timeout=600)


def current_status():
    with urllib.request.urlopen(BASE_URL + '/api/v1/status', timeout=4) as response:
        result = json.load(response)
    if not result.get('health'):
        raise RuntimeError('Health endpoint unhealthy')
    return result


def healthy(expected):
    last = None
    for _ in range(30):
        try:
            status = current_status()
            build = status['build']
            if build['version'] != expected['version'] or build['commit'] != expected['commit']:
                raise RuntimeError('Unexpected deployed version or commit')
            with urllib.request.urlopen(BASE_URL + '/home', timeout=4) as response:
                if response.status != 200:
                    raise RuntimeError('Frontend health check failed')
            for path in ['/api/v1/ai/settings', '/api/v1/tags']:
                try:
                    urllib.request.urlopen(BASE_URL + path, timeout=4).close()
                except urllib.error.HTTPError as error:
                    if error.code == 401:
                        continue
                raise RuntimeError('Protected endpoint did not return 401')
            return
        except Exception as error:
            last = error
            time.sleep(1)
    raise RuntimeError('Deployment health checks failed') from last


def deploy(candidate, expected):
    previous = current_status()['build']
    backup = ROOT / 'backups' / (time.strftime('%Y%m%dT%H%M%SZ', time.gmtime()) + '-' + expected['version'])
    backup.mkdir(mode=0o700, parents=True, exist_ok=False)
    copy_tree(ROOT / 'zainaer', backup / 'zainaer')
    copy_tree(ROOT / 'service.env', backup / 'service.env')
    (backup / 'metadata.json').write_text(json.dumps({'previous': previous, 'target': expected}))
    old_umask = os.umask(0o077)
    stopped = False
    snapshot = False
    switched = False
    try:
        stopped = True  # Even a timed-out stop must attempt recovery.
        service('stop')
        copy_tree(ROOT / 'data', backup / 'data')
        snapshot = True
        os.replace(candidate, ROOT / 'zainaer')
        switched = True
        service('start')
        healthy(expected)
        print('DEPLOYED version=' + expected['version'] + ' backup=' + str(backup), flush=True)
    except BaseException:
        # Ignore further termination while rolling back; SIGKILL/host failure still need operator recovery.
        for signum in (signal.SIGTERM, signal.SIGINT, signal.SIGHUP):
            signal.signal(signum, signal.SIG_IGN)
        print('Deployment failed; recovering previous service', file=sys.stderr, flush=True)
        if stopped:
            service('stop')
            if switched and snapshot:
                # Preserve failed state for diagnosis; never merge migrated data into old data.
                os.rename(ROOT / 'data', backup / 'failed-data')
                copy_tree(backup / 'data', ROOT / 'data')
                copy_tree(backup / 'zainaer', ROOT / '.rollback-executable')
                os.replace(ROOT / '.rollback-executable', ROOT / 'zainaer')
            service('start')
            healthy({'version': previous['version'], 'commit': previous['commit']})
            print('ROLLED_BACK version=' + previous['version'], file=sys.stderr, flush=True)
        raise
    finally:
        os.umask(old_umask)


def main():
    os.umask(0o077)
    if os.geteuid() != 0:
        raise PermissionError('Deployment helper requires root')
    # JSON deployment metadata is on stdin. SSH_ORIGINAL_COMMAND is deliberately ignored.
    with (ROOT / '.deploy.lock').open('a') as lock:
        fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
        expected = metadata(sys.stdin.buffer)
        # Enough room for full snapshots plus staging; never prune old backups automatically.
        data_bytes = sum(p.stat().st_size for p in (ROOT / 'data').rglob('*') if p.is_file() and not p.is_symlink())
        if shutil.disk_usage(ROOT).free < data_bytes + 3 * MAX_SIZE:
            raise RuntimeError('Insufficient disk space for safe deployment')
        with tempfile.TemporaryDirectory(prefix='.deploy-', dir=ROOT) as temp:
            candidate = Path(temp) / 'zainaer'
            unpack(sys.stdin.buffer, candidate, expected['sha256'])
            # Keep deployment independent of SSH disconnects once the complete artifact is received.
            signal.signal(signal.SIGHUP, signal.SIG_IGN)
            def terminate(signum, frame):
                raise RuntimeError('Deployment interrupted by signal ' + str(signum))
            signal.signal(signal.SIGTERM, terminate)
            signal.signal(signal.SIGINT, terminate)
            deploy(candidate, expected)


if __name__ == '__main__':
    main()
