# SPDX-License-Identifier: MIT
# Copyright (c) 2026 Phoobobo
import datetime as dt
from pathlib import Path
import sqlite3
import tempfile
import unittest
from unittest.mock import patch

import backup


class BackupTests(unittest.TestCase):
    def database(self, root):
        data = root / 'data'
        (data / '.data/photos').mkdir(parents=True)
        db = data / '.data/homebox.db'
        connection = sqlite3.connect(db)
        connection.executescript('PRAGMA journal_mode=WAL; CREATE TABLE entities(id INTEGER); CREATE TABLE attachments(path TEXT); INSERT INTO entities VALUES(1);')
        connection.commit()
        return data, connection

    def test_online_snapshot_captures_wal_and_referenced_photo(self):
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            data, connection = self.database(root)
            (data / '.data/photos/a').write_bytes(b'photo')
            connection.executescript("INSERT INTO entities VALUES(2); INSERT INTO attachments VALUES('photos/a');")
            connection.commit()
            self.assertTrue((data / '.data/homebox.db-wal').exists())
            result = backup.snapshot(data, root / 'snapshot', Path('.data/homebox.db'))
            self.assertEqual(result, {'entities': 2, 'attachments': 1})
            self.assertFalse((root / 'snapshot/.data/homebox.db-wal').exists())
            connection.close()

    def test_missing_photo_rejects_snapshot(self):
        with tempfile.TemporaryDirectory() as temp, patch.object(backup.time, 'sleep'):
            root = Path(temp)
            data, connection = self.database(root)
            connection.execute("INSERT INTO attachments VALUES('photos/missing')"); connection.commit()
            with self.assertRaisesRegex(RuntimeError, 'complete online snapshot'):
                backup.snapshot(data, root / 'snapshot', Path('.data/homebox.db'))
            connection.close()

    def test_second_copy_captures_concurrently_added_photo(self):
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            data, connection = self.database(root)
            original = backup.copy_files
            calls = []
            def copy(*args):
                original(*args)
                calls.append(1)
                if len(calls) == 1:
                    (data / '.data/photos/new').write_bytes(b'new photo')
                    connection.execute("INSERT INTO attachments VALUES('photos/new')"); connection.commit()
            with patch.object(backup, 'copy_files', side_effect=copy):
                counts = backup.snapshot(data, root / 'snapshot', Path('.data/homebox.db'))
            self.assertEqual(counts['attachments'], 1)
            self.assertEqual((root / 'snapshot/.data/photos/new').read_bytes(), b'new photo')
            connection.close()

    def test_model_ciphertext_and_master_key_are_recoverable(self):
        import base64
        import json
        from cryptography.hazmat.primitives.ciphers.aead import AESGCM
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            data, connection = self.database(root)
            ai = data / ".data/ai"; ai.mkdir()
            key = bytes(range(32)); nonce = bytes(range(12))
            (ai / "master.key").write_bytes(key)
            payload = nonce + AESGCM(key).encrypt(nonce, b"synthetic-test-key", b"zainaer-ai-v1")
            (ai / "collection.json").write_text(json.dumps({"encryptedKey": base64.b64encode(payload).decode()}))
            self.assertEqual(backup.validate_snapshot(data, Path(".data/homebox.db"))["entities"], 1)
            (ai / "master.key").write_bytes(bytes(reversed(range(32))))
            with self.assertRaises(Exception):
                backup.validate_snapshot(data, Path(".data/homebox.db"))
            connection.close()

    def test_application_outage_does_not_block_backup_metadata(self):
        with patch.object(backup.urllib.request, 'urlopen', side_effect=OSError('offline')):
            self.assertTrue(backup.build_info()['statusUnavailable'])

    def test_storage_symlink_is_rejected(self):
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp); (root / 'data').mkdir()
            (root / 'data/link').symlink_to('/etc/passwd')
            with self.assertRaisesRegex(ValueError, 'Symlinks'):
                backup.copy_files(root / 'data', root / 'out', Path('.data/homebox.db'))

    def test_retention_keeps_recent_daily_monthly_and_ignores_other_files(self):
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp); now = dt.datetime(2026, 10, 19, 7, tzinfo=backup.UTC)
            paths = []
            for hours in range(80):
                stamp = now - dt.timedelta(hours=hours)
                path = root / ('zainaer-' + stamp.strftime('%Y%m%dT%H%M%SZ') + '-1234abcd.tar.gz.age')
                path.touch(); paths.append(path)
            old = root / 'zainaer-20200101T000000Z-1234abcd.tar.gz.age'; old.touch()
            other = root / 'unrelated.age'; other.touch()
            deleted = backup.retention(paths + [old, other], now)
            self.assertIn(old, deleted)
            self.assertNotIn(other, deleted)
            self.assertFalse(set(paths[:48]) & set(deleted))
            self.assertTrue(set(paths[48:]) & set(deleted))


if __name__ == '__main__':
    unittest.main()
