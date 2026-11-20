# SPDX-License-Identifier: MIT; Copyright (c) 2026 Phoobobo
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest


class SenderTests(unittest.TestCase):
    def test_header_only_requires_explicit_preloaded_mode(self):
        for mode in ('false', 'true', 'invalid'):
            with self.subTest(mode=mode), tempfile.TemporaryDirectory() as temp:
                root = Path(temp)
                dist = root / 'dist'
                dist.mkdir()
                manifest = {'version': '0.1.30', 'commit': 'a' * 40}
                (dist / 'release.json').write_text(json.dumps(manifest))
                (dist / 'BINARY_SHA256').write_text('b' * 64)
                if mode == 'false':
                    (dist / 'zainaer-linux-amd64.gz').write_bytes(b'compressed release')
                result = subprocess.run([sys.executable, str(Path(__file__).with_name('send.py').resolve())],
                                        cwd=root, env=dict(os.environ, ZAINAER_PRELOADED_ARTIFACT=mode), capture_output=True)
                if mode == 'invalid':
                    self.assertNotEqual(result.returncode, 0)
                    self.assertEqual(result.stdout, b'')
                    continue
                self.assertEqual(result.returncode, 0, result.stderr)
                header, payload = result.stdout.split(b'\n', 1)
                expected = dict(manifest, sha256='b' * 64)
                if mode == 'true':
                    expected['artifact_source'] = 'preloaded'
                self.assertEqual(json.loads(header), expected)
                self.assertEqual(payload, b'' if mode == 'true' else b'compressed release')
