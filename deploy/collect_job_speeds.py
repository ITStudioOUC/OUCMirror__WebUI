#!/usr/bin/python3
"""Sample per-socket TCP receive counters and attribute worker processes to mirrors."""
import json
import os
from pathlib import Path
import re
import subprocess
import tempfile
import time

PROC = Path('/proc')
MIRRORS = '/data/mirrors/'
OUTPUT = Path('/var/lib/mirrorouc/metrics/job-speeds.json')
NAME = re.compile(r'^[a-zA-Z0-9][a-zA-Z0-9_.+-]{0,127}$')

def mirror_name(value):
    if not value.startswith(MIRRORS):
        return None
    name = value[len(MIRRORS):].split('/')[0]
    return name if NAME.fullmatch(name) and name not in ('.', '..') else None

def process_jobs(proc=PROC):
    jobs = {}
    parents = {}
    for entry in proc.iterdir():
        if not entry.name.isdigit():
            continue
        try:
            if '/tunasync-worker.service' not in (entry / 'cgroup').read_text():
                continue
            pid = int(entry.name)
            status = (entry / 'stat').read_text().rsplit(') ', 1)[1].split()
            parents[pid] = int(status[1])
            arguments = (entry / 'cmdline').read_bytes().decode(errors='replace').split('\0')
            candidates = {name for argument in arguments if (name := mirror_name(argument))}
            if not candidates:
                name = mirror_name(os.readlink(entry / 'cwd'))
                if name:
                    candidates.add(name)
            if len(candidates) == 1:
                jobs[pid] = candidates.pop()
        except (OSError, ValueError, IndexError):
            continue
    # git/ssh/curl helpers may inherit a job's parent without its destination argument.
    for pid in parents:
        ancestor = pid
        for _ in range(16):
            if ancestor in jobs:
                jobs[pid] = jobs[ancestor]
                break
            ancestor = parents.get(ancestor, 0)
            if not ancestor:
                break
    return jobs

def parse_sockets(output, jobs):
    sockets = {}
    header = None
    for line in output.splitlines():
        if line and not line[0].isspace():
            header = line
            continue
        if header is None:
            continue
        identity = re.search(r'\bino:(\d+)\s+sk:([a-fA-F0-9]+)', header)
        if not identity:
            continue
        pids = [int(value) for value in re.findall(r'\bpid=(\d+)', header)]
        names = {jobs[pid] for pid in pids if pid in jobs}
        received = re.search(r'\bbytes_received:(\d+)', line)
        # The same inherited socket can list several PIDs: count its inode/cookie once.
        sockets[':'.join(identity.groups())] = {'bytes': int(received.group(1)) if received else 0, 'job': names.pop() if len(names) == 1 else None}
        header = None
    return sockets

def rates(current, previous, active, elapsed):
    result = {name: {'bytesPerSecond': 0.0, 'connections': 0} for name in active}
    if previous is None or elapsed <= 0 or elapsed > 15:
        return {}
    for identity, socket in current.items():
        name = socket['job']
        if name is None:
            continue
        old = previous.get(identity)
        delta = socket['bytes'] - old['bytes'] if old else socket['bytes']
        if delta < 0:  # A reset/reused socket must never produce a negative rate/spike.
            continue
        value = result.setdefault(name, {'bytesPerSecond': 0.0, 'connections': 0})
        value['bytesPerSecond'] += delta / elapsed
        value['connections'] += 1
    return result

def write(value):
    fd, temporary = tempfile.mkstemp(prefix='.speeds-', dir=OUTPUT.parent)
    try:
        with os.fdopen(fd, 'w') as stream:
            json.dump(value, stream, allow_nan=False, separators=(',', ':'))
        os.chmod(temporary, 0o640)
        os.replace(temporary, OUTPUT)
    finally:
        if os.path.exists(temporary):
            os.unlink(temporary)

def run():
    previous = None
    last = time.monotonic()
    while True:
        start = time.monotonic()
        try:
            jobs = process_jobs()
            result = subprocess.run(['/usr/bin/ss', '-tinpeH', 'state', 'established'], capture_output=True, text=True, timeout=4, check=True, env={**os.environ, 'LC_ALL': 'C'})
            current = parse_sockets(result.stdout, jobs)
            sampled = time.monotonic()
            elapsed = sampled - last
            values = rates(current, previous, set(jobs.values()), elapsed)
            write({'updatedAt': round(time.time() * 1000), 'interval': elapsed, 'available': previous is not None and elapsed <= 15, 'jobs': values})
            previous, last = current, sampled
        except (OSError, ValueError, subprocess.SubprocessError):
            previous = None
            write({'updatedAt': round(time.time() * 1000), 'interval': 2, 'available': False, 'jobs': {}})
        time.sleep(max(.1, 2 - (time.monotonic() - start)))

if __name__ == '__main__':
    run()
