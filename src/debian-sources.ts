export const debianReleases = [
  { id: 'trixie', version: '13', stable: true, firmware: true, backports: true },
  { id: 'bookworm', version: '12', stable: true, firmware: true, backports: false },
  { id: 'bullseye', version: '11', stable: true, firmware: false, backports: false },
  { id: 'testing', version: '', stable: false, firmware: true, backports: false },
  { id: 'sid', version: '', stable: false, firmware: true, backports: false },
] as const

export type DebianRelease = typeof debianReleases[number]['id']
export type DebianFormat = 'list' | 'deb822'
export interface DebianSourceOptions {
  origin: string
  release: DebianRelease
  format: DebianFormat
  sources: boolean
  nonFree: boolean
  securityMirror: boolean
}

export function debianSources(options: DebianSourceOptions) {
  const release = debianReleases.find(entry => entry.id === options.release)
  if (!release || !['list', 'deb822'].includes(options.format)) throw new Error('Invalid Debian configuration')
  const origin = new URL(options.origin)
  if (!['http:', 'https:'].includes(origin.protocol) || origin.username || origin.password) throw new Error('Invalid mirror origin')
  const mirror = `${origin.origin}/mirrors/debian/`
  const security = options.securityMirror ? `${origin.origin}/mirrors/debian-security/` : 'https://security.debian.org/debian-security'
  const components = ['main', ...(options.nonFree ? ['contrib', 'non-free', ...(release.firmware ? ['non-free-firmware'] : [])] : [])].join(' ')
  const suites = [release.id, ...(release.stable ? [`${release.id}-updates`] : []), ...(release.backports ? [`${release.id}-backports`] : [])]
  const groups = [{ uri: mirror, suites }, ...(release.stable ? [{ uri: security, suites: [`${release.id}-security`] }] : [])]
  if (options.format === 'deb822') {
    return groups.map(group => [
      `Types: ${options.sources ? 'deb deb-src' : 'deb'}`,
      `URIs: ${group.uri}`,
      `Suites: ${group.suites.join(' ')}`,
      `Components: ${components}`,
      'Signed-By: /usr/share/keyrings/debian-archive-keyring.gpg',
    ].join('\n')).join('\n\n') + '\n'
  }
  return groups.flatMap(group => group.suites.map(suite => [
    `deb ${group.uri} ${suite} ${components}`,
    `${options.sources ? '' : '# '}deb-src ${group.uri} ${suite} ${components}`,
  ].join('\n'))).join('\n\n') + '\n'
}
