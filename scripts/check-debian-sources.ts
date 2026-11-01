import assert from 'node:assert/strict'
import { marked } from 'marked'
import { debianReleases, debianSources, type DebianSourceOptions } from '../src/debian-sources.ts'
import { helpBlocks } from '../src/help-content.ts'

const origin = 'https://mirrors.example.edu'
const base: DebianSourceOptions = { origin, release: 'trixie', format: 'list', sources: false, nonFree: true, securityMirror: false }
const defaultList = debianSources(base)
assert.ok(defaultList.includes('deb https://mirrors.example.edu/mirrors/debian/ trixie main contrib non-free non-free-firmware\n'))
assert.ok(defaultList.includes('deb https://security.debian.org/debian-security trixie-security'))
assert.ok(defaultList.includes('# deb-src https://mirrors.example.edu/mirrors/debian/ trixie '))

let cases = 0
for (const release of debianReleases) for (const format of ['list', 'deb822'] as const) for (const sources of [false, true]) for (const nonFree of [false, true]) for (const securityMirror of [false, true]) {
  const value = debianSources({ origin, release: release.id, format, sources, nonFree, securityMirror })
  assert.ok(!/undefined|NaN|trusted=yes|allow-insecure|testing-security|sid-security/.test(value))
  assert.equal(value.includes('non-free-firmware'), nonFree && release.firmware)
  assert.equal(value.includes('contrib non-free'), nonFree)
  assert.equal(value.includes(`${release.id}-updates`), release.stable)
  assert.equal(value.includes(`${release.id}-backports`), release.backports)
  assert.equal(value.includes(`${release.id}-security`), release.stable)
  if (release.stable) {
    assert.equal(value.includes('https://mirrors.example.edu/mirrors/debian-security/'), securityMirror)
    assert.equal(value.includes('https://security.debian.org/debian-security'), !securityMirror)
  }
  if (format === 'list') {
    assert.equal(value.split('\n').some(line => line.startsWith('deb-src ')), sources)
    assert.ok(value.split('\n').every(line => !line || /^(?:# )?deb(?:-src)? https:\/\/\S+ \S+ main(?: contrib non-free(?: non-free-firmware)?)?$/.test(line)))
  } else {
    for (const stanza of value.trim().split('\n\n')) {
      const fields = Object.fromEntries(stanza.split('\n').map(line => line.split(/: (.*)/s).slice(0, 2)))
      assert.equal(fields.Types, sources ? 'deb deb-src' : 'deb')
      assert.equal(fields['Signed-By'], '/usr/share/keyrings/debian-archive-keyring.gpg')
      assert.ok(fields.URIs && fields.Suites && fields.Components)
    }
  }
  cases++
}
assert.ok(debianSources({ ...base, origin: 'http://100.69.2.2:8080/path?ignored=true' }).includes('http://100.69.2.2:8080/mirrors/debian/'))
assert.throws(() => debianSources({ ...base, origin: 'javascript:alert(1)' }))
assert.throws(() => debianSources({ ...base, origin: 'https://secret:password@example.edu' }))
assert.throws(() => debianSources({ ...base, release: 'unknown' as DebianSourceOptions['release'] }))

const fence = (data: string) => '```mirror-config\n' + data + '\n```\n'
const source = '## 配置\n\n[Debian][official]\n\n' + fence('{"type":"debian","format":"list"}') + '\n## 新格式\n\n' + fence('{"type":"debian","format":"deb822"}') + '\n[official]: https://www.debian.org/\n'
const blocks = helpBlocks(source)
assert.deepEqual(blocks.filter(block => block.kind === 'debian'), [{ kind: 'debian', format: 'list' }, { kind: 'debian', format: 'deb822' }])
assert.ok(blocks.filter(block => block.kind === 'markdown').map(block => marked.parser(block.tokens)).join('').includes('href="https://www.debian.org/"'), 'Reference-style Markdown links must survive widget insertion')
for (const data of ['not JSON', '{"type":"shell","format":"list"}', '{"type":"debian","format":"other"}', '{"type":"debian","format":"list","script":"alert(1)"}', 'null']) {
  assert.ok(helpBlocks(fence(data)).every(block => block.kind === 'markdown'), 'Invalid or unsupported blocks must remain ordinary Markdown code')
}
assert.ok(helpBlocks('````markdown\n' + fence('{"type":"debian","format":"list"}') + '````\n').every(block => block.kind === 'markdown'), 'A quoted code example must not instantiate a live widget')
console.log(`Debian help check passed: ${cases} option combinations, release-specific suites, official security defaults, DEB822 fields, widget parsing and Markdown links.`)
