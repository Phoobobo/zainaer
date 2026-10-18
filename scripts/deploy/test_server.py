# SPDX-License-Identifier: MIT
# Copyright (c) 2026 Phoobobo
import gzip
import hashlib
import io
import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

import server


class DeploymentTests(unittest.TestCase):
    def test_rejects_invalid_or_oversized_header(self):
        for value in [b'x' * 1025, b'{}\n', b'{"version":"../../x","commit":"a","sha256":"b"}\n']:
            with self.assertRaises((ValueError, json.JSONDecodeError)):
                server.metadata(io.BytesIO(value))

    def test_checksum_failure_never_stops_service(self):
        with tempfile.TemporaryDirectory() as temp, patch.object(server, 'service') as service:
            with self.assertRaisesRegex(ValueError, 'checksum'):
                server.unpack(io.BytesIO(gzip.compress(b'bad')), Path(temp) / 'new', '0' * 64)
            service.assert_not_called()

    def test_oversized_artifact_rejected(self):
        with tempfile.TemporaryDirectory() as temp, patch.object(server, 'MAX_SIZE', 4):
            with self.assertRaisesRegex(ValueError, 'size limit'):
                server.unpack(io.BytesIO(gzip.compress(b'too large')), Path(temp) / 'new', '0' * 64)

    def test_wrong_platform_rejected(self):
        data = b'not a linux executable'
        with tempfile.TemporaryDirectory() as temp:
            with self.assertRaisesRegex(ValueError, 'Linux amd64'):
                server.unpack(io.BytesIO(gzip.compress(data)), Path(temp) / 'new', hashlib.sha256(data).hexdigest())

    def scenario(self, root):
        (root / 'data/.data/ai').mkdir(parents=True)
        (root / 'data/.data/homebox.db').write_bytes(b'old-database')
        (root / 'data/.data/ai/master.key').write_bytes(b'original-key')
        (root / 'data/.data/ai/config.json').write_bytes(b'encrypted-provider-key')
        (root / 'data/photo.jpg').write_bytes(b'photo')
        (root / 'zainaer').write_bytes(b'old-program')
        (root / 'service.env').write_text('existing-private-env')
        candidate = root / 'new-program'
        candidate.write_bytes(b'new-program')
        return candidate, {'version': '0.1.5', 'commit': 'b' * 40, 'sha256': 'a' * 64}

    def test_success_keeps_full_snapshot_and_preserves_live_data(self):
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            candidate, expected = self.scenario(root)
            previous = {'build': {'version': '0.1.4', 'commit': 'c' * 40}}
            with patch.object(server, 'ROOT', root), patch.object(server, 'service') as service, \
                    patch.object(server, 'current_status', return_value=previous), patch.object(server, 'healthy'):
                server.deploy(candidate, expected)
            backup = next((root / 'backups').iterdir())
            self.assertEqual((root / 'zainaer').read_bytes(), b'new-program')
            self.assertEqual((backup / 'zainaer').read_bytes(), b'old-program')
            self.assertEqual((backup / 'data/.data/ai/master.key').read_bytes(), b'original-key')
            self.assertEqual((backup / 'data/photo.jpg').read_bytes(), b'photo')
            self.assertEqual([c.args[0] for c in service.call_args_list], ['stop', 'start'])

    def test_failed_health_restores_program_database_and_keys(self):
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            candidate, expected = self.scenario(root)
            previous = {'build': {'version': '0.1.4', 'commit': 'c' * 40}}
            calls = []
            def fake_service(action):
                calls.append(action)
                if calls == ['stop', 'start']:
                    # Simulate schema migration or mutation by the failing new service.
                    (root / 'data/.data/homebox.db').write_bytes(b'migrated-db')
                    (root / 'data/.data/ai/master.key').write_bytes(b'changed-key')
            with patch.object(server, 'ROOT', root), patch.object(server, 'service', side_effect=fake_service), \
                    patch.object(server, 'current_status', return_value=previous), \
                    patch.object(server, 'healthy', side_effect=[RuntimeError('unhealthy'), None]), \
                    patch.object(server.signal, 'signal'):
                with self.assertRaisesRegex(RuntimeError, 'unhealthy'):
                    server.deploy(candidate, expected)
            backup = next((root / 'backups').iterdir())
            self.assertEqual((root / 'zainaer').read_bytes(), b'old-program')
            self.assertEqual((root / 'data/.data/homebox.db').read_bytes(), b'old-database')
            self.assertEqual((root / 'data/.data/ai/master.key').read_bytes(), b'original-key')
            self.assertEqual((backup / 'failed-data/.data/homebox.db').read_bytes(), b'migrated-db')
            self.assertEqual(calls, ['stop', 'start', 'stop', 'start'])

    def test_failed_snapshot_restarts_old_service_without_replacing_data(self):
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            candidate, expected = self.scenario(root)
            previous = {'build': {'version': '0.1.4', 'commit': 'c' * 40}}
            real_copy = server.copy_tree
            def fail_snapshot(src, dst):
                if src == root / 'data':
                    raise OSError('snapshot failed')
                real_copy(src, dst)
            with patch.object(server, 'ROOT', root), patch.object(server, 'copy_tree', side_effect=fail_snapshot), \
                    patch.object(server, 'service') as service, patch.object(server, 'healthy'), \
                    patch.object(server, 'current_status', return_value=previous), patch.object(server.signal, 'signal'):
                with self.assertRaisesRegex(OSError, 'snapshot failed'):
                    server.deploy(candidate, expected)
            self.assertEqual((root / 'zainaer').read_bytes(), b'old-program')
            self.assertEqual((root / 'data/.data/homebox.db').read_bytes(), b'old-database')
            self.assertEqual([c.args[0] for c in service.call_args_list], ['stop', 'stop', 'start'])


if __name__ == '__main__':
    unittest.main()
