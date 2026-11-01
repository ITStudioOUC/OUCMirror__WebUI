import copy
import importlib.util
import json
import os
import tempfile
import unittest
from unittest.mock import patch

spec = importlib.util.spec_from_file_location('helper', os.path.join(os.path.dirname(__file__), 'config-helper.py'))
helper = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)

class ValidationTests(unittest.TestCase):
    def test_preserves_custom_commands(self):
        before = '[[mirrors]]\nname="debian"\nprovider="rsync"\nupstream="rsync://old/debian"\ncommand="/usr/local/bin/parallel-rsync"\n'
        after = before.replace('old/debian', 'new/debian') + 'interval=60\nretry=2\ntimeout=120\n'
        helper.validate(helper.INCLUDES + '/debian.conf', before, after)
        with self.assertRaises(AssertionError):
            helper.validate(helper.INCLUDES + '/debian.conf', before, after.replace('parallel-rsync', 'other-command'))

    def test_blocks_other_global_fields_and_paths(self):
        before = '[global]\nname="worker"\nconcurrent=12\ninterval=60\nretry=2\ntimeout=120\n'
        helper.validate(helper.MAIN, before, before.replace('concurrent=12', 'concurrent=8'))
        for path, after in [('/etc/shadow', before), (helper.MAIN, before.replace('worker', 'other')), (helper.MAIN, before.replace('concurrent=12', 'concurrent=0'))]:
            with self.assertRaises(AssertionError):
                helper.validate(path, before, after)

    def test_new_mirrors_and_injection(self):
        value = '[[mirrors]]\nname="test"\nprovider="rsync"\nupstream="rsync://mirror/test"\ninterval=60\nretry=0\ntimeout=120\n'
        helper.validate_new(helper.tomllib.loads(value), helper.INCLUDES + '/mirrorouc-test.conf')
        for changed in [value + 'command="evil"\n', value.replace('rsync://mirror/test', 'rsync://mirror/`cmd`'), value.replace('retry=0', 'retry=-1')]:
            with self.assertRaises(AssertionError):
                helper.validate_new(helper.tomllib.loads(changed), helper.INCLUDES + '/mirrorouc-test.conf')

    def test_reload_failure_restores_files(self):
        with tempfile.TemporaryDirectory() as root:
            path = root + '/worker.conf'
            before = '[global]\nname="worker"\nconcurrent=12\ninterval=60\nretry=2\ntimeout=120\n'
            helper.atomic(path, before)
            changes = {path: {'before': before, 'after': before.replace('concurrent=12', 'concurrent=8')}}
            with patch.object(helper, 'MAIN', path), patch.object(helper, 'JOURNAL', root + '/pending.json'), patch.object(helper, 'signature', return_value='base'), patch.object(helper, 'reload_worker', side_effect=[OSError('unavailable'), None]):
                self.assertEqual(helper.apply({'files': changes, 'baseHash': 'base'}), 'CONFIG_APPLY_FAILED')
                self.assertEqual(helper.read(path), before)
                self.assertFalse(os.path.exists(root + '/pending.json'))

if __name__ == '__main__':
    unittest.main()
