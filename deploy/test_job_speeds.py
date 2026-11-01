import importlib.util
import os
from pathlib import Path
import tempfile
import unittest

spec = importlib.util.spec_from_file_location('speeds', os.path.join(os.path.dirname(__file__), 'collect_job_speeds.py'))
speeds = importlib.util.module_from_spec(spec)
spec.loader.exec_module(speeds)

class SpeedTests(unittest.TestCase):
    def test_sockets_deduplicate_shared_pids_and_separate_repos(self):
        data = '0 0 10.0.0.1:123 10.0.0.2:873 users:(("rsync",pid=1,fd=3),("rsync",pid=2,fd=3)) uid:9 ino:20 sk:abc <->\n\t cubic bytes_sent:4 bytes_received:1000\n0 0 10.0.0.1:124 10.0.0.2:873 users:(("rsync",pid=3,fd=3)) uid:9 ino:21 sk:abd <->\n\t cubic bytes_received:2000\n'
        current = speeds.parse_sockets(data, {1: 'debian', 2: 'debian', 3: 'ubuntu'})
        result = speeds.rates(current, {'20:abc': {'bytes': 400}, '21:abd': {'bytes': 1000}}, {'debian', 'ubuntu'}, 2)
        self.assertEqual(result['debian'], {'bytesPerSecond': 300, 'connections': 1})
        self.assertEqual(result['ubuntu']['bytesPerSecond'], 500)

    def test_parallel_and_new_connections_add_without_startup_spike(self):
        current = {'1:a': {'bytes': 1000, 'job': 'debian'}, '2:b': {'bytes': 500, 'job': 'debian'}, '3:c': {'bytes': 9000, 'job': None}}
        self.assertEqual(speeds.rates(current, None, {'debian'}, 2), {})
        result = speeds.rates(current, {'1:a': {'bytes': 800}}, {'debian', 'ubuntu'}, 2)
        self.assertEqual(result['debian']['bytesPerSecond'], 350)
        self.assertEqual(result['ubuntu']['bytesPerSecond'], 0)
        self.assertEqual(speeds.rates(current, {}, {'debian'}, 20), {})

    def test_process_ancestry_and_path_boundaries(self):
        with tempfile.TemporaryDirectory() as root:
            for pid, parent, command in [(1, 0, ['/usr/bin/python', '/worker']), (2, 1, ['rsync', '/data/mirrors/debian/']), (3, 2, ['ssh', 'host']), (4, 0, ['rsync', '/data/mirrors/ubuntu/'])]:
                path = Path(root) / str(pid); path.mkdir()
                (path/'stat').write_text(f'{pid} (process) S {parent} 0 0')
                (path/'cgroup').write_text('0::/system.slice/tunasync-worker.service' if pid != 4 else '0::/system.slice/other.service')
                (path/'cmdline').write_bytes('\0'.join(command).encode())
            jobs = speeds.process_jobs(Path(root))
            self.assertEqual(jobs, {2: 'debian', 3: 'debian'})
            self.assertIsNone(speeds.mirror_name('/data/mirrors-other/secret'))
            self.assertIsNone(speeds.mirror_name('/data/mirrors/../secret'))

if __name__ == '__main__':
    unittest.main()
