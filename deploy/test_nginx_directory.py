"""Exercise the shipped Nginx routes using disposable files and an isolated port."""
from html.parser import HTMLParser
from pathlib import Path
import shutil
import socket
import subprocess
import tempfile
import time
import unittest
import urllib.error
import urllib.parse
import urllib.request


class NginxDirectoryTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        nginx = shutil.which('nginx')
        if not nginx:
            raise unittest.SkipTest('Nginx is required for the integration fixture')
        cls.temp = tempfile.TemporaryDirectory(prefix='mirrorouc-nginx-test-')
        cls.root = Path(cls.temp.name)
        cls.mirrors = cls.root / 'mirrors'
        cls.dist = cls.root / 'dist'
        (cls.mirrors / 'repo' / 'sub').mkdir(parents=True)
        (cls.mirrors / 'repo' / 'empty').mkdir()
        build = Path(__file__).resolve().parent.parent / 'dist'
        if not (build / 'fancy-index' / 'header.html').is_file():
            raise RuntimeError('Run npm run build before the Nginx integration fixture')
        shutil.copytree(build, cls.dist)
        (cls.dist / 'assets' / 'test.js').write_text('window.fixture = true', encoding='utf-8')
        (cls.mirrors / 'repo' / 'index.html').write_text('repository index must not replace listing', encoding='utf-8')
        (cls.mirrors / 'repo' / '中文 #+%.txt').write_bytes(b'0123456789abcdef')
        (cls.mirrors / 'repo' / 'zero').write_bytes(b'')
        (cls.mirrors / 'repo' / '.hidden').write_text('hidden', encoding='utf-8')
        (cls.mirrors / 'repo' / '\"><img src=x onerror=alert(1)>.txt').write_text('escaped name', encoding='utf-8')
        (cls.root / 'private-secret').write_text('outside-secret', encoding='utf-8')
        with socket.socket() as available:
            available.bind(('127.0.0.1', 0))
            port = available.getsockname()[1]
        cls.base = f'http://127.0.0.1:{port}'
        server = Path(__file__).with_name('nginx-preview.conf').read_text(encoding='utf-8')
        server = server.replace('listen 100.69.2.2:8080;', f'listen 127.0.0.1:{port};')
        server = server.replace('/data/mirrors/', cls.mirrors.as_posix() + '/')
        server = server.replace('/opt/mirrorouc/current/dist', cls.dist.as_posix())
        # The Go upstream is deliberately unavailable throughout these tests.
        server = server.replace('http://127.0.0.1:8081', 'http://127.0.0.1:9')
        config = cls.root / 'nginx.conf'
        config.write_text(f'load_module /usr/lib/nginx/modules/ngx_http_fancyindex_module.so;\nworker_processes 1;\npid {cls.root}/nginx.pid;\nerror_log {cls.root}/error.log;\nevents {{ worker_connections 32; }}\nhttp {{ access_log off; types {{ text/html html; text/plain txt; application/javascript js; }}\n{server}\n}}\n', encoding='utf-8')
        validation = subprocess.run([nginx, '-p', str(cls.root), '-c', str(config), '-t'], capture_output=True, text=True)
        if validation.returncode:
            raise RuntimeError(validation.stderr)
        cls.process = subprocess.Popen([nginx, '-p', str(cls.root), '-c', str(config), '-g', 'daemon off; master_process off;'], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
        for _ in range(100):
            if cls.process.poll() is not None:
                raise RuntimeError((cls.root / 'error.log').read_text())
            try:
                with urllib.request.urlopen(cls.base + '/assets/test.js', timeout=1):
                    return
            except (OSError, urllib.error.URLError):
                time.sleep(0.03)
        cls.process.terminate()
        raise RuntimeError('Isolated Nginx did not become ready')

    @classmethod
    def tearDownClass(cls):
        cls.process.terminate()
        cls.process.wait(timeout=5)
        cls.temp.cleanup()

    def request(self, path, method='GET', headers=None):
        request = urllib.request.Request(self.base + path, method=method, headers=headers or {})
        try:
            response = urllib.request.build_opener(NoRedirect).open(request, timeout=5)
        except urllib.error.HTTPError as response:
            return response.code, response.headers, response.read()
        with response:
            return response.status, response.headers, response.read()

    def test_complete_server_html_without_go_or_javascript(self):
        code, headers, body = self.request('/mirrors/repo/')
        self.assertEqual(code, 200)
        self.assertIn('text/html', headers['Content-Type'])
        html = body.decode('utf-8')
        self.assertIn('<table id="list">', html)
        self.assertIn('href="sub/"', html)
        self.assertIn('href="index.html"', html)
        self.assertIn('中文 #+%.txt', html)
        self.assertIn('data-i18n="directory.name">文件名</span>', html)
        self.assertIn('data-i18n="directory.parent">返回上级</span>', html)
        self.assertIn('UTC', html)
        self.assertIn('</footer>', html)
        self.assertNotIn('{{', html)
        self.assertNotIn('repository index must not replace listing', html)
        self.assertNotIn('.hidden', html)
        parsed = Tags(); parsed.feed(html)
        self.assertFalse(any(tag == 'img' and attrs.get('onerror') for tag, attrs in parsed.tags))
        self.assertNotIn('sandbox', headers['Content-Security-Policy'])
        self.assertEqual(self.request('/assets/test.js')[0], 200)
        self.assertEqual(self.request('/')[0], 502)

    def test_files_still_support_range_and_head(self):
        url = '/mirrors/repo/' + urllib.parse.quote('中文 #+%.txt', safe='')
        code, headers, body = self.request(url, headers={'Range': 'bytes=1-4'})
        self.assertEqual((code, body), (206, b'1234'))
        self.assertEqual(headers['Content-Range'], 'bytes 1-4/16')
        self.assertIn('sandbox', headers['Content-Security-Policy'])
        self.assertEqual(headers['X-Content-Type-Options'], 'nosniff')
        code, headers, body = self.request(url, method='HEAD')
        self.assertEqual((code, int(headers['Content-Length']), body), (200, 16, b''))

    def test_empty_missing_internal_and_parent(self):
        code, _, body = self.request('/mirrors/repo/empty/')
        self.assertEqual(code, 200)
        self.assertIn(b'<table id="list">', body)
        self.assertEqual(self.request('/mirrors/missing/')[0], 404)
        self.assertEqual(self.request('/mirrors/repo/', method='POST')[0], 403)
        self.assertEqual(self.request('/_directory/header.html')[0], 404)
        self.assertEqual(self.request('/_directory/footer.html')[0], 404)
        code, headers, _ = self.request('/mirrors/')
        self.assertEqual(code, 302)
        self.assertTrue(headers['Location'].endswith('/'))
        code, _, body = self.request('/mirrors/%2e%2e/private-secret/')
        self.assertNotEqual(code, 200)
        self.assertNotIn(b'outside-secret', body)

    def test_native_sort_links_and_head(self):
        code, _, body = self.request('/mirrors/repo/?C=N&O=D')
        self.assertEqual(code, 200)
        html = body.decode('utf-8')
        self.assertLess(html.index('href="sub/'), html.index('href="empty/'))
        code, _, body = self.request('/mirrors/repo/', method='HEAD')
        self.assertEqual((code, body), (200, b''))


class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        return None


class Tags(HTMLParser):
    def __init__(self):
        super().__init__()
        self.tags = []

    def handle_starttag(self, tag, attrs):
        self.tags.append((tag, dict(attrs)))


if __name__ == '__main__':
    unittest.main()
