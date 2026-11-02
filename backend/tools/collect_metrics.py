#!/usr/bin/python3
"""Read Linux/NFS counters. A separate passive nftables table supplies protocol bytes."""
import json
import os
from pathlib import Path
import subprocess
import tempfile
import time

ROOT = Path(os.environ.get('MIRROROUC_METRICS_ROOT', '/var/lib/mirrorouc/metrics'))
ROOT.mkdir(parents=True, exist_ok=True)
path = ROOT / 'history.json'
try:
    history = json.loads(path.read_text())
except (OSError, ValueError):
    history = {'samples': []}

now = time.time()
cpu = list(map(int, Path('/proc/stat').read_text().splitlines()[0].split()[1:9]))
mem = {line.split(':')[0]: int(line.split()[1]) * 1024 for line in Path('/proc/meminfo').read_text().splitlines() if len(line.split()) > 1}
nfs = {}
for block in Path('/proc/self/mountstats').read_text().split('device '):
    if ' mounted on /data ' not in block:
        continue
    for line in block.splitlines():
        values = line.strip().split()
        if values and values[0] in ('READ:', 'WRITE:'):
            nfs[values[0][:-1]] = list(map(int, values[1:]))

counters = {}
try:
    result = subprocess.run(['/usr/sbin/nft', '-j', 'list', 'counters', 'table', 'inet', 'mirrorouc_monitor'], capture_output=True, timeout=4, check=True)
    for item in json.loads(result.stdout).get('nftables', []):
        counter = item.get('counter', {})
        if counter.get('name') in ('http4', 'http6', 'rsync4', 'rsync6'):
            counters[counter['name']] = counter['bytes']
except (OSError, ValueError, subprocess.SubprocessError):
    pass

previous = history.get('previous')
sample = {'time': round(now * 1000)}
if previous:
    elapsed = max(.1, now - previous['time'])
    delta = [max(0, a - b) for a, b in zip(cpu, previous['cpu'])]
    total = sum(delta)
    if total:
        sample.update(user=(delta[0] + delta[1]) / total * 100, system=(delta[2] + delta[5] + delta[6] + delta[7]) / total * 100, wait=delta[4] / total * 100, idle=delta[3] / total * 100)
    for key, value in counters.items():
        old = previous.get('counters', {}).get(key)
        if old is not None and value >= old:
            sample[key] = (value - old) * 8 / elapsed / 1e9
    for operation, prefix in [('READ', 'read'), ('WRITE', 'write')]:
        a, b = nfs.get(operation), previous.get('nfs', {}).get(operation)
        if a and b and a[0] >= b[0]:
            sample[prefix + 'Ops'] = (a[0] - b[0]) / elapsed / 1000
            byte_index = 4 if operation == 'READ' else 3
            sample[prefix + 'Bytes'] = max(0, a[byte_index] - b[byte_index]) / elapsed / 1e9
    cached = max(0, mem.get('Cached', 0) + mem.get('SReclaimable', 0) - mem.get('Shmem', 0))
    free, buffers, total_mem = mem['MemFree'], mem.get('Buffers', 0), mem['MemTotal']
    used = max(0, total_mem - free - cached - buffers)
    sample.update(used=used / 2**30, cached=cached / 2**30, buffers=buffers / 2**30, free=free / 2**30)
    if 'user' in sample:
        history['samples'].append(sample)

disks = []
for mount in ['/data', '/']:
    try:
        disk = os.statvfs(mount)
        disks.append({'name': mount, 'total': disk.f_blocks * disk.f_frsize / 1e12, 'used': (disk.f_blocks - disk.f_bfree) * disk.f_frsize / 1e12})
    except OSError:
        pass
history.update(previous={'time': now, 'cpu': cpu, 'nfs': nfs, 'counters': counters}, updatedAt=round(now * 1000), memoryTotal=mem['MemTotal'] / 2**30, disks=disks)
history['samples'] = [point for point in history['samples'] if point['time'] >= (now - 86400) * 1000][-8640:]
fd, temporary = tempfile.mkstemp(dir=ROOT, prefix='.metrics-')
try:
    with os.fdopen(fd, 'w') as output:
        json.dump(history, output, allow_nan=False, separators=(',', ':'))
        output.flush()
        os.fsync(output.fileno())
    os.chmod(temporary, 0o640)
    os.replace(temporary, path)
finally:
    if os.path.exists(temporary):
        os.unlink(temporary)
