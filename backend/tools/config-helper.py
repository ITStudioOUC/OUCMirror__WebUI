#!/usr/bin/python3
"""Root-only configuration writer, reachable only through a group-restricted Unix socket.

Independently verifies allowed TOML changes; never executes submitted commands.
"""
import copy
import glob
import grp
import hashlib
import json
import os
import re
import socketserver
import tempfile
import tomllib
import urllib.parse
import urllib.request

ROOT = '/etc/tunasync'
MAIN = ROOT + '/worker.conf'
INCLUDES = ROOT + '/mirrors.d'
SOCKET = '/run/mirrorouc-config/control.sock'
JOURNAL = '/var/lib/mirrorouc-config/pending.json'
IDENTIFIER = re.compile(r'^[a-zA-Z0-9][a-zA-Z0-9_.+-]{0,127}$')

def number(value, lower, upper):
    return type(value) is int and lower <= value <= upper

def upstream(value):
    if not isinstance(value, str) or len(value) >= 2048 or re.search(r'[\s`;$|<>\\\"\']', value):
        return False
    parsed = urllib.parse.urlsplit(value)
    return parsed.scheme in ('rsync', 'https', 'http') and bool(parsed.hostname) and parsed.username is None

def sync_numbers(table):
    return all(key not in table or number(table[key], low, high) for key, low, high in [('interval', 1, 525600), ('retry', 0, 20), ('timeout', 30, 1209600)])

def stripped_mirrors(doc):
    result = copy.deepcopy(doc)
    mirrors = result.get('mirrors', [])
    assert isinstance(mirrors, list)
    for mirror in mirrors:
        for field in ('upstream', 'interval', 'retry', 'timeout'):
            mirror.pop(field, None)
    return result

def validate_new(doc, path):
    assert set(doc) == {'mirrors'} and len(doc['mirrors']) == 1
    mirror = doc['mirrors'][0]
    assert set(mirror) <= {'name', 'provider', 'upstream', 'interval', 'retry', 'timeout', 'stage1_profile'}
    assert IDENTIFIER.fullmatch(mirror['name']) and mirror['name'] not in ('.', '..')
    assert path == INCLUDES + '/mirrorouc-' + mirror['name'] + '.conf'
    assert mirror['provider'] in ('rsync', 'two-stage-rsync')
    assert upstream(mirror['upstream']) and sync_numbers(mirror)
    assert 'stage1_profile' not in mirror or mirror['stage1_profile'] == 'debian'

def validate(path, before, after):
    assert path == MAIN or (os.path.dirname(path) == INCLUDES and os.path.basename(path).endswith('.conf'))
    assert not os.path.islink(path)
    old = tomllib.loads(before) if before is not None else None
    new = tomllib.loads(after) if after is not None else None
    assert old is not None or new is not None
    if path == MAIN:
        assert old is not None and new is not None
        assert number(new['global'].get('concurrent'), 1, 64) and sync_numbers(new['global'])
        for doc in (old, new):
            for field in ('concurrent', 'interval', 'retry', 'timeout'):
                doc['global'].pop(field, None)
        assert old == new
    elif old is None or new is None:
        validate_new(new if new is not None else old, path)
    else:
        assert stripped_mirrors(old) == stripped_mirrors(new)
        for previous, current in zip(old.get('mirrors', []), new.get('mirrors', [])):
            if previous != current:
                assert upstream(current['upstream']) and sync_numbers(current)

def read(path):
    try:
        with open(path, encoding='utf-8', newline='') as stream:
            return stream.read()
    except FileNotFoundError:
        return None

def atomic(path, value):
    if value is None:
        os.unlink(path)
        return
    mode = os.stat(path).st_mode & 0o777 if os.path.exists(path) else 0o644
    fd, temporary = tempfile.mkstemp(prefix='.mirrorouc-', dir=os.path.dirname(path))
    try:
        with os.fdopen(fd, 'w', encoding='utf-8', newline='') as stream:
            stream.write(value)
            stream.flush()
            os.fsync(stream.fileno())
        os.chmod(temporary, mode)
        os.replace(temporary, path)
    finally:
        if os.path.exists(temporary):
            os.unlink(temporary)

def reload_worker():
    worker = tomllib.loads(read(MAIN))['global']['name']
    body = json.dumps({'worker_id': worker, 'mirror_id': '', 'cmd': 'reload', 'args': [], 'options': {}}).encode()
    request = urllib.request.Request('http://127.0.0.1:12345/cmd', data=body, headers={'Content-Type': 'application/json'})
    with urllib.request.urlopen(request, timeout=10) as response:
        assert response.status == 200

def signature():
    pattern = tomllib.loads(read(MAIN)).get('include', {}).get('include_mirrors', '')
    assert pattern == INCLUDES + '/*.conf'
    names = sorted(set([MAIN] + glob.glob(pattern)))
    digest = hashlib.sha256()
    for path in names:
        assert not os.path.islink(path)
        raw = read(path).encode()
        digest.update(str(len(path.encode())).encode() + b':' + path.encode() + str(len(raw)).encode() + b':' + raw)
    return digest.hexdigest()

def restore(changes):
    for path, change in changes.items():
        if change['before'] is None and not os.path.exists(path):
            continue
        atomic(path, change['before'])
    reload_worker()

def apply(request):
    changes = request['files']
    assert isinstance(changes, dict) and 0 < len(changes) <= 20
    for path, change in changes.items():
        validate(path, change['before'], change['after'])
        if read(path) != change['before']:
            return 'CONFLICT'
    if signature() != request['baseHash']:
        return 'CONFLICT'
    # Write-ahead journal survives a helper/process restart during a multi-file update.
    atomic(JOURNAL, json.dumps(changes))
    try:
        for path, change in changes.items():
            atomic(path, change['after'])
        reload_worker()
    except Exception:
        try:
            restore(changes)
        except Exception:
            return 'CONFIG_ROLLBACK_FAILED'
        os.unlink(JOURNAL)
        return 'CONFIG_APPLY_FAILED'
    os.unlink(JOURNAL)
    return ''

class Handler(socketserver.StreamRequestHandler):
    def handle(self):
        self.connection.settimeout(5)
        try:
            raw = self.rfile.readline(2 * 1024 * 1024 + 1)
            assert len(raw) <= 2 * 1024 * 1024 and raw.endswith(b'\n')
            if os.path.exists(JOURNAL):
                code = 'CONFIG_ROLLBACK_FAILED'
            else:
                code = apply(json.loads(raw))
        except (AssertionError, KeyError, TypeError, ValueError):
            code = 'INVALID_INPUT'
        except Exception:
            code = 'CONFIG_APPLY_FAILED'
        self.wfile.write(json.dumps({'code': code}).encode() + b'\n')

if __name__ == '__main__':
    if os.path.exists(JOURNAL):
        with open(JOURNAL) as stream:
            restore(json.load(stream))
        os.unlink(JOURNAL)
    if os.path.exists(SOCKET):
        os.unlink(SOCKET)
    with socketserver.UnixStreamServer(SOCKET, Handler) as server:
        os.chown(SOCKET, 0, grp.getgrnam('mirrorouc').gr_gid)
        os.chmod(SOCKET, 0o660)
        server.serve_forever()
