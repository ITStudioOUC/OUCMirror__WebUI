#!/usr/bin/python3
"""Maintain passive HTTP/S and rsync counters without resetting accumulated bytes.

HTTP/S endpoints come from Nginx's HTTP configuration and tunasync HTTP(S)
upstreams. This is endpoint accounting, not packet inspection of arbitrary
encrypted protocols. Local proxy hops are excluded; both receive and transmit
traffic on the mirror host are counted, regardless of standard/custom ports.
"""
import argparse
import fnmatch
import hashlib
import json
import os
from pathlib import Path
import posixpath
import re
import subprocess
import tempfile
import tomllib
from urllib.parse import urlsplit

TABLE = 'mirrorouc_monitor'
VERSION = 1
COUNTERS = ('http4', 'http6', 'rsync4', 'rsync6')
TOKEN = re.compile(r'''"(?:\\.|[^"\\])*"|'(?:\\.|[^'\\])*'|\#[^\n]*|[{};]|[^\s{};\#]+''')


def directives(source, initial=()):
    context, pending = list(initial), []
    for match in TOKEN.finditer(source):
        token = match[0]
        if token.startswith('#'):
            continue
        if token == '{':
            context.append(pending[0] if pending else '')
            pending = []
        elif token == '}':
            if len(context) <= len(initial):
                raise ValueError('Unbalanced Nginx configuration')
            context.pop()
            pending = []
        elif token == ';':
            if pending:
                yield tuple(context), pending
            pending = []
        else:
            pending.append(token.strip('\'"'))


def nginx_http_ports(dump):
    parts = re.split(r'^# configuration file (.+):\s*$', dump, flags=re.M)
    documents = dict(zip(parts[1::2], parts[2::2]))
    if not documents:
        documents = {'/etc/nginx/nginx.conf': dump}
    entry = next(iter(documents))
    prefix = posixpath.dirname(entry)
    todo, seen, ports = [(entry, ())], set(), set()
    while todo:
        file, initial = todo.pop()
        if (file, initial) in seen:
            continue
        seen.add((file, initial))
        for context, args in directives(documents[file], initial):
            if args[0] == 'include' and len(args) == 2:
                pattern = posixpath.normpath(posixpath.join(prefix, args[1]))
                todo.extend((name, context) for name in documents if fnmatch.fnmatchcase(name, pattern))
            elif args[0] == 'listen' and context == ('http', 'server') and len(args) > 1:
                address = args[1]
                if address.startswith('unix:') or 'quic' in args[2:]:
                    continue
                value = address.rsplit(':', 1)[-1]
                port = int(value) if value.isdecimal() else 80
                if not 1 <= port <= 65535:
                    raise ValueError('Invalid Nginx HTTP listener port')
                ports.add(port)
    return ports


def http_upstream_ports(worker_config):
    # Include ordinary HTTP(S) clients as well as explicitly configured custom
    # upstream ports. rsync/SSH/FTP URLs must not contribute ports to this set.
    ports, seen, todo = {80, 443}, set(), [Path(worker_config)]
    while todo:
        path = todo.pop()
        if path in seen or not path.exists():
            continue
        seen.add(path)
        doc = tomllib.loads(path.read_text())
        pattern = doc.get('include', {}).get('include_mirrors')
        if pattern:
            from glob import glob
            todo.extend(Path(name) for name in glob(pattern))
        for mirror in doc.get('mirrors', []):
            upstream = urlsplit(mirror.get('upstream', ''))
            if upstream.scheme in ('http', 'https'):
                ports.add(upstream.port or (443 if upstream.scheme == 'https' else 80))
    return ports


def shape(document):
    def clean(value):
        if isinstance(value, dict):
            return {key: clean(item) for key, item in value.items() if key not in ('handle', 'packets', 'bytes')}
        if isinstance(value, list):
            return [clean(item) for item in value]
        return value
    objects = [clean(item) for item in document.get('nftables', []) if 'metainfo' not in item]
    return hashlib.sha256(json.dumps(objects, sort_keys=True).encode()).hexdigest()


def transaction(current, listeners, upstreams, table=TABLE):
    if not re.fullmatch(r'[a-zA-Z0-9_]+', table):
        raise ValueError('Invalid counter table')
    objects = current.get('nftables', [])
    existing = lambda kind: {item[kind]['name'] for item in objects if kind in item}
    lines = [] if existing('table') else [f'add table inet {table}']
    for name in COUNTERS:
        if name not in existing('counter'):
            lines.append(f'add counter inet {table} {name}')
    for name, ports in (('http_listeners', listeners), ('http_upstreams', upstreams)):
        if name not in existing('set'):
            lines.append(f'add set inet {table} {name} {{ type inet_service; }}')
        lines.append(f'flush set inet {table} {name}')
        if ports:
            if any(not isinstance(port, int) or not 1 <= port <= 65535 for port in ports):
                raise ValueError('Invalid HTTP endpoint port')
            lines.append(f'add element inet {table} {name} {{ ' + ', '.join(map(str, sorted(ports))) + ' }')
    for direction, interface, local, remote in (('input', 'iifname', 'dport', 'sport'), ('output', 'oifname', 'sport', 'dport')):
        if direction not in existing('chain'):
            lines.append(f'add chain inet {table} {direction} {{ type filter hook {direction} priority 300; policy accept; }}')
        lines.append(f'flush chain inet {table} {direction}')
        lines.append(f'add rule inet {table} {direction} {interface} "lo" return')
        for version in (4, 6):
            for field, endpoint in ((local, 'http_listeners'), (remote, 'http_upstreams')):
                # Return after accounting so a packet is counted exactly once.
                lines.append(f'add rule inet {table} {direction} meta nfproto ipv{version} tcp {field} @{endpoint} counter name http{version} return')
            for field in ('sport', 'dport'):
                lines.append(f'add rule inet {table} {direction} meta nfproto ipv{version} tcp {field} 873 counter name rsync{version} return')
    # Migrate the former HTTPS-only rules, preserving both rsync counters.
    if 'account' in existing('chain'):
        lines.extend((f'flush chain inet {table} account', f'delete chain inet {table} account'))
    for name in ('https4', 'https6'):
        if name in existing('counter'):
            lines.append(f'delete counter inet {table} {name}')
    return '\n'.join(lines) + '\n'


def read_table(nft):
    result = subprocess.run([nft, '-j', '-a', 'list', 'table', 'inet', TABLE], capture_output=True, text=True, timeout=5)
    if result.returncode:
        if 'No such file or directory' in result.stderr or 'does not exist' in result.stderr:
            return {}
        raise RuntimeError('Unable to read protocol counters: ' + result.stderr.strip())
    return json.loads(result.stdout)


def ensure(args):
    import fcntl
    args.state.parent.mkdir(parents=True, exist_ok=True)
    with (args.state.parent / '.protocol-counters.lock').open('a') as lock:
        fcntl.flock(lock, fcntl.LOCK_EX)
        dump = subprocess.run([args.nginx, '-T'], capture_output=True, text=True, timeout=10, check=True).stdout
        listeners = nginx_http_ports(dump)
        if not listeners:
            raise RuntimeError('No Nginx TCP HTTP listeners discovered; keeping existing accounting unchanged')
        upstreams = http_upstream_ports(args.worker_config)
        current = read_table(args.nft)
        settings = {'version': VERSION, 'listeners': sorted(listeners), 'upstreams': sorted(upstreams)}
        try:
            previous = json.loads(args.state.read_text())
        except (OSError, ValueError):
            previous = {}
        if all(previous.get(key) == value for key, value in settings.items()) and previous.get('shape') == shape(current):
            return
        commands = transaction(current, listeners, upstreams)
        for flags in (['-c'], []):
            subprocess.run([args.nft, *flags, '-f', '-'], input=commands, capture_output=True, text=True, timeout=5, check=True)
        settings['shape'] = shape(read_table(args.nft))
        fd, name = tempfile.mkstemp(dir=args.state.parent, prefix='.protocol-')
        try:
            with os.fdopen(fd, 'w') as stream:
                json.dump(settings, stream, sort_keys=True)
            os.chmod(name, 0o640)
            os.replace(name, args.state)
        finally:
            if os.path.exists(name):
                os.unlink(name)
        print(json.dumps(settings), flush=True)


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--nginx', default='/usr/sbin/nginx')
    parser.add_argument('--nft', default='/usr/sbin/nft')
    parser.add_argument('--worker-config', default='/etc/tunasync/worker.conf')
    parser.add_argument('--state', type=Path, default=Path('/var/lib/mirrorouc/metrics/protocol-counters.json'))
    ensure(parser.parse_args())
