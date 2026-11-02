import copy
import importlib.util
from pathlib import Path
import tempfile
import unittest

spec = importlib.util.spec_from_file_location('protocol_counters', Path(__file__).with_name('protocol_counters.py'))
protocol = importlib.util.module_from_spec(spec)
spec.loader.exec_module(protocol)


class ProtocolTests(unittest.TestCase):
    def test_http_contexts_and_included_custom_listeners(self):
        dump = '''# configuration file /etc/nginx/nginx.conf:
events {}
http {
    include /etc/nginx/sites-enabled/*;
    server { listen 80; listen [::]:80; }
}
stream { include /etc/nginx/stream.d/*.conf; }
# configuration file /etc/nginx/sites-enabled/mirrors:
server {
    listen "100.69.2.2:18080";
    listen [::]:443 ssl;
    listen unix:/tmp/http.sock;
    listen 9443 quic;
    # listen 9999;
    location ~ "^/test{2}#" { return 404; }
    include ssl/cert.conf;
}
# configuration file /etc/nginx/ssl/cert.conf:
ssl_certificate /tmp/cert;
# configuration file /etc/nginx/stream.d/database.conf:
server { listen 5432; proxy_pass database; }
'''
        self.assertEqual(protocol.nginx_http_ports(dump), {80, 443, 18080})

    def test_upstream_schemes_and_custom_ports(self):
        with tempfile.TemporaryDirectory() as name:
            root = Path(name)
            (root / 'worker.conf').write_text('[include]\ninclude_mirrors = "' + (root / '*.mirror').as_posix() + '"\n')
            (root / 'repos.mirror').write_text('''[[mirrors]]
name = "http"
upstream = "http://example.invalid:19090/repo/"
[[mirrors]]
name = "https"
upstream = "https://example.invalid:8443/repo/"
[[mirrors]]
name = "rsync"
upstream = "rsync://example.invalid:1873/repo/"
''')
            self.assertEqual(protocol.http_upstream_ports(root / 'worker.conf'), {80, 443, 19090, 8443})

    def test_migration_keeps_existing_rsync_and_other_tables(self):
        current = {'nftables': [{'table': {'name': protocol.TABLE}}, *({'counter': {'name': name, 'bytes': 12345}} for name in ('https4', 'https6', 'rsync4', 'rsync6')), *({'chain': {'name': name}} for name in ('input', 'output', 'account'))]}
        commands = protocol.transaction(current, {80, 18080}, {80, 443, 19090})
        self.assertNotIn('flush ruleset', commands)
        self.assertNotIn('flush table', commands)
        self.assertNotIn('reset counter', commands)
        for name in ('rsync4', 'rsync6'):
            self.assertNotIn('add counter inet mirrorouc_monitor ' + name, commands)
            self.assertNotIn('delete counter inet mirrorouc_monitor ' + name, commands)
        self.assertIn('delete chain inet mirrorouc_monitor account', commands)
        self.assertIn('meta nfproto ipv6 tcp sport @http_listeners counter name http6 return', commands)
        self.assertIn('meta nfproto ipv4 tcp sport @http_upstreams counter name http4 return', commands)
        self.assertIn('meta nfproto ipv4 tcp dport @http_upstreams counter name http4 return', commands)

    def test_counter_growth_does_not_reapply_rules(self):
        before = {'nftables': [{'counter': {'name': 'http4', 'handle': 1, 'bytes': 100, 'packets': 1}}, {'rule': {'expr': [{'counter': 'http4'}]}}]}
        after = copy.deepcopy(before)
        after['nftables'][0]['counter'].update(bytes=200, packets=2, handle=9)
        self.assertEqual(protocol.shape(before), protocol.shape(after))
        after['nftables'][1]['rule']['expr'][0]['counter'] = 'rsync4'
        self.assertNotEqual(protocol.shape(before), protocol.shape(after))


if __name__ == '__main__':
    unittest.main()
