import assert from 'node:assert/strict'
import { marked } from 'marked'
import { mirrorOrigin } from '../src/apt-sources.ts'
import { ubuntuReleases, ubuntuSources } from '../src/ubuntu-sources.ts'
import { configDefaults, configFields, generateMirrorConfig, mirrorConfigTypes, mirrorGenerators, parseMirrorConfig, type ConfigValues, type MirrorConfig } from '../src/mirror-config.ts'
import { helpBlocks } from '../src/help-content.ts'
import { defaultGeneratorCatalog, parseGeneratorCatalog, validGeneratorVersions, type GeneratorCatalog } from '../src/generator-catalog.ts'

const origin = 'https://mirrors.example.edu'
let ubuntuCases = 0
for (const release of ubuntuReleases) for (const format of ['list', 'deb822'] as const) for (const sources of [true, false]) for (const proposed of [true, false]) for (const securityMirror of [true, false]) {
  const text = ubuntuSources({ origin, release: release.id, format, sources, proposed, securityMirror })
  assert.equal(text.includes(`${release.id}-proposed`), proposed)
  assert.ok(text.includes(`${release.id}-security`))
  assert.ok(text.includes(`${release.id}-updates`))
  assert.ok(text.includes(`${release.id}-backports`))
  assert.equal(text.includes('http://security.ubuntu.com/ubuntu/'), !securityMirror)
  assert.ok(text.includes('main restricted universe multiverse'))
  assert.ok(!/trusted=yes|allow-insecure|undefined|NaN|ubuntu-ports/.test(text))
  if (format === 'deb822') {
    assert.equal(text.includes('Types: deb deb-src'), sources)
    assert.ok(text.includes('Signed-By: /usr/share/keyrings/ubuntu-archive-keyring.gpg'))
  } else assert.equal(text.split('\n').some(line => line.startsWith('deb-src ')), sources)
  ubuntuCases++
}

const fixtures: { name: string; file: string; format: string; body: string }[] = []
let profileCases = 0
for (const type of mirrorConfigTypes) {
  const generator = mirrorGenerators[type]
  for (const selected of generator.formats) {
    const config: MirrorConfig = { type, format: selected.value }
    const initial = configDefaults(config, origin)
    const states: ConfigValues[] = [initial]
    for (const field of configFields(type)) {
      for (const value of field.kind === 'switch' ? [true, false] : field.choices!.map(choice => choice.value)) states.push({ ...initial, [field.key]: value })
    }
    for (const values of states) {
      const files = generateMirrorConfig(config, origin, values)
      assert.ok(files.length > 0)
      for (const file of files) {
        assert.ok(file.path || file.titleKey)
        assert.ok(file.code.trim())
        assert.ok(!/undefined|NaN|trusted=yes|SigLevel\s*=\s*Never|gpgcheck=0|curl\s+-k/.test(file.code))
        assert.ok(file.code.includes('mirrors.example.edu') || file.titleKey === 'mirrors.generator.signingKey')
        if (file.path?.endsWith('.sources') || file.path?.endsWith('.list') && file.code.startsWith('deb ')) fixtures.push({ name: type, file: file.path, format: file.path.endsWith('.sources') ? 'sources' : 'list', body: file.code })
      }
      profileCases++
    }
  }
}

assert.ok(generateMirrorConfig({ type: 'ubuntu' }, origin).at(-1)!.code.includes('noble-updates'))
assert.ok(!generateMirrorConfig({ type: 'ubuntu' }, origin).at(-1)!.code.includes('noble-proposed'))
assert.ok(generateMirrorConfig({ type: 'archlinux' }, origin)[0]!.code.includes('/$repo/os/$arch'))
assert.ok(generateMirrorConfig({ type: 'archlinuxarm' }, origin)[0]!.code.includes('/$arch/$repo'))
assert.ok(generateMirrorConfig({ type: 'archlinuxcn' }, origin)[0]!.code.includes('[archlinuxcn]'))
assert.ok(generateMirrorConfig({ type: 'openwrt' }, origin)[0]!.code.includes('https\\?://downloads.openwrt.org'))
assert.ok(generateMirrorConfig({ type: 'kubernetes', options: { release: 'v1.36' } }, origin)[0]!.code.includes('v1.36/deb/Release.key'))
const tailscale = generateMirrorConfig({ type: 'tailscale', options: { release: 'ubuntu/noble' } }, origin)
assert.ok(tailscale[0]!.code.includes('/stable/ubuntu/noble.noarmor.gpg'))
assert.ok(tailscale[1]!.code.includes('/mirrors/tailscale/ubuntu noble main'))
assert.ok(!tailscale[1]!.code.includes('/mirrors/tailscale/stable/'))
assert.ok(generateMirrorConfig({ type: 'proxmox', format: 'deb822', options: { release: 'bookworm' } }, origin)[0]!.code.includes('Signed-By: /etc/apt/trusted.gpg.d/proxmox-release-bookworm.gpg'))
assert.ok(generateMirrorConfig({ type: 'proxmox', format: 'deb822', options: { release: 'trixie' } }, origin)[0]!.code.includes('Signed-By: /usr/share/keyrings/proxmox-archive-keyring.gpg'))
assert.throws(() => generateMirrorConfig({ type: 'ubuntu', options: { release: 'sid' } }, origin))
assert.throws(() => generateMirrorConfig({ type: 'anaconda' }, "https://bad'host.example"))
assert.throws(() => mirrorOrigin('javascript:alert(1)'))
assert.throws(() => mirrorOrigin('https://user:password@example.edu'))
assert.ok(generateMirrorConfig({ type: 'ubuntu', options: { protocol: 'http' } }, origin)[0]!.code.includes('http://mirrors.example.edu/mirrors/ubuntu/'))

const fence = (body: string) => '```mirror-config\n' + body + '\n```\n'
for (const type of mirrorConfigTypes) {
  assert.deepEqual(helpBlocks(fence(JSON.stringify({ type }))), [{ kind: 'config', config: { type } }])
}
for (const value of [null, [], {}, { type: 'constructor' }, { type: '__proto__' }, { type: 'ubuntu', format: 'shell' }, { type: 'ubuntu', options: { release: 'sid' } }, { type: 'ubuntu', options: { sources: 'true' } }, { type: 'ubuntu', options: { __invalid: true } }, { type: 'ubuntu', options: [] }, { type: 'ubuntu', script: 'alert(1)' }, { type: 'ubuntu', options: { protocol: 'javascript' } }]) {
  assert.equal(parseMirrorConfig(value), null)
  assert.ok(helpBlocks(fence(JSON.stringify(value))).every(block => block.kind === 'markdown'))
}
assert.ok(helpBlocks(fence('x'.repeat(2049))).every(block => block.kind === 'markdown'))
assert.ok(helpBlocks('````markdown\n' + fence('{"type":"ubuntu"}') + '````').every(block => block.kind === 'markdown'))
const blocks = helpBlocks('[Ubuntu][official]\n\n' + fence('{"type":"ubuntu"}') + '\n[official]: https://ubuntu.com/\n')
assert.ok(blocks.filter(block => block.kind === 'markdown').map(block => marked.parser(block.tokens)).join('').includes('href="https://ubuntu.com/"'))

// Use deliberately fictional release names to prove data-only additions work
// without asserting the codenames of unreleased distributions.
const future: GeneratorCatalog = structuredClone(defaultGeneratorCatalog)
future.ubuntu.releases.unshift({ id: 'future-test', label: 'Ubuntu future (test fixture)' })
future.ubuntu.defaultRelease = 'future-test'
future.debian.releases.unshift({ id: 'future-stable', label: 'Debian future (test fixture)', stable: true, firmware: true, backports: true })
future.debian.releases.push({ id: 'future-rolling', label: 'Rolling (test fixture)', stable: false, firmware: true, backports: false })
future.debian.defaultRelease = 'future-stable'
future.kubernetes.releases.unshift({ id: 'v2.99', label: 'Kubernetes future (test fixture)' })
future.kubernetes.defaultRelease = 'v2.99'
future.tailscale.releases.unshift({ id: 'ubuntu/future-test', label: 'Ubuntu future (test fixture)' })
future.tailscale.defaultRelease = 'ubuntu/future-test'
future.proxmox.releases.unshift({ id: 'future-test', label: 'PVE future (test fixture)', keyring: '/usr/share/keyrings/proxmox-future-test.gpg' })
future.proxmox.defaultRelease = 'future-test'
assert.ok(parseGeneratorCatalog(future))
const futureUbuntu = generateMirrorConfig({ type: 'ubuntu' }, origin, {}, future)[0]!.code
assert.ok(futureUbuntu.includes('Suites: future-test future-test-updates future-test-backports'))
assert.ok(futureUbuntu.includes('Suites: future-test-security'))
assert.ok(generateMirrorConfig({ type: 'debian' }, origin, {}, future)[0]!.code.includes('future-stable-backports'))
assert.ok(configFields('debian-security', future)[0]!.choices!.some(choice => choice.value === 'future-stable'))
assert.ok(!configFields('debian-security', future)[0]!.choices!.some(choice => choice.value === 'future-rolling'))
assert.ok(!generateMirrorConfig({ type: 'debian', options: { release: 'future-rolling', securityMirror: true } }, origin, {}, future)[0]!.code.includes('future-rolling-security'))
assert.ok(generateMirrorConfig({ type: 'kubernetes' }, origin, {}, future)[0]!.code.includes('/v2.99/deb/Release.key'))
assert.ok(generateMirrorConfig({ type: 'tailscale' }, origin, {}, future)[0]!.code.includes('/ubuntu/future-test.noarmor.gpg'))
assert.ok(generateMirrorConfig({ type: 'proxmox' }, origin, {}, future)[0]!.code.includes('Signed-By: /usr/share/keyrings/proxmox-future-test.gpg'))
assert.equal(helpBlocks(fence('{"type":"ubuntu","options":{"release":"future-test"}}'), future)[0]!.kind, 'config')
assert.equal(helpBlocks(fence('{"type":"ubuntu","options":{"release":"future-test"}}'))[0]!.kind, 'markdown')
for (const bad of ['foo\ntrusted=yes', 'foo;id', 'foo bar', '$(id)', '../../etc/passwd', '__proto__']) {
  const invalid = structuredClone(future)
  invalid.ubuntu.releases[0]!.id = bad
  invalid.ubuntu.defaultRelease = bad
  assert.equal(parseGeneratorCatalog(invalid), null)
}
assert.equal(validGeneratorVersions('proxmox', { defaultRelease: 'foo', releases: [{ id: 'foo', label: 'PVE', keyring: '/tmp/evil.gpg' }] }), false)
assert.equal(validGeneratorVersions('ubuntu', { defaultRelease: 'foo', releases: [{ id: 'foo', label: 'A' }, { id: 'foo', label: 'B' }] }), false)
assert.equal(validGeneratorVersions('debian', { defaultRelease: 'foo', releases: [{ id: 'foo', label: 'Rolling', stable: false, firmware: true, backports: false }] }), false)
assert.equal(parseGeneratorCatalog({ ...future, script: 'id' }), null)

if (process.argv.includes('--fixtures')) console.log(JSON.stringify(fixtures))
else console.log(`Generator checks passed: ${mirrorConfigTypes.length} profiles, ${profileCases} profile states, ${ubuntuCases} Ubuntu combinations, data-only future releases for 5 families, safe Markdown parsing, version-coupled keys and repository paths.`)
