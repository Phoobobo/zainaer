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

    def test_preloaded_header_is_strict(self):
        value = {'version': '0.1.30', 'commit': 'a' * 40, 'sha256': 'b' * 64, 'artifact_source': 'preloaded'}
        self.assertEqual(server.metadata(io.BytesIO(json.dumps(value).encode() + b'\n')), value)
        for extra in [{'artifact_source': 'https://example.com/a.gz'}, {'path': '/tmp/a.gz'}, {'artifact_source': True}]:
            invalid = dict(value, **extra)
            with self.assertRaises(ValueError):
                server.metadata(io.BytesIO(json.dumps(invalid).encode() + b'\n'))

    def test_preloaded_artifact_has_identical_validation(self):
        data = b'\x7fELF\x02\x01' + b'\0' * 12 + b'\x3e\0' + b'test program'
        digest = hashlib.sha256(data).hexdigest()
        with tempfile.TemporaryDirectory() as temp, patch.object(server, 'service') as service:
            root = Path(temp)
            cache = root / 'release-cache'
            cache.mkdir(mode=0o700)
            artifact = cache / (digest + '.gz')
            artifact.write_bytes(gzip.compress(data))
            artifact.chmod(0o600)
            stream = io.BytesIO(b'no streamed executable')
            expected = {'sha256': digest, 'artifact_source': 'preloaded'}
            with patch.object(server, 'ROOT', root):
                target = root / 'new'
                server.unpack_artifact(stream, target, expected)
                self.assertEqual(target.read_bytes(), data)
                self.assertEqual(stream.tell(), 0)
                self.assertNotIn('artifact_source', expected)
                artifact.write_bytes(gzip.compress(b'wrong executable'))
                with self.assertRaisesRegex(ValueError, 'checksum'):
                    server.unpack_artifact(stream, root / 'bad', {'sha256': digest, 'artifact_source': 'preloaded'})
            service.assert_not_called()

    def test_preloaded_rejects_untrusted_cache_and_missing_file(self):
        with tempfile.TemporaryDirectory() as temp, patch.object(server, 'service') as service:
            root = Path(temp)
            cache = root / 'release-cache'
            cache.mkdir(mode=0o700)
            digest = 'a' * 64
            artifact = cache / (digest + '.gz')
            def attempt():
                return server.unpack_artifact(io.BytesIO(), root / 'new', {'sha256': digest, 'artifact_source': 'preloaded'})
            with patch.object(server, 'ROOT', root):
                with self.assertRaises(FileNotFoundError):
                    attempt()
                elsewhere = root / 'elsewhere'
                elsewhere.write_bytes(b'not allowed')
                artifact.symlink_to(elsewhere)
                with self.assertRaises(OSError):
                    attempt()
                artifact.unlink()
                artifact.write_bytes(b'bad')
                artifact.chmod(0o666)
                with self.assertRaisesRegex(PermissionError, 'cached Release'):
                    attempt()
                artifact.chmod(0o600)
                cache.chmod(0o777)
                with self.assertRaisesRegex(PermissionError, 'cache directory'):
                    attempt()
                cache.chmod(0o700)
                with patch.object(server, 'MAX_SIZE', 1):
                    with self.assertRaisesRegex(ValueError, 'size limit'):
                        attempt()
            service.assert_not_called()

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
