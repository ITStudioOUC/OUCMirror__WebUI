import { aptSources, mirrorOrigin, type AptFormat } from './apt-sources.ts'
import { defaultGeneratorCatalog, type GeneratorRelease } from './generator-catalog.ts'

export const debianReleases = defaultGeneratorCatalog.debian.releases
export type DebianRelease = string
export type DebianFormat = AptFormat
export interface DebianSourceOptions {
  origin: string
  release: DebianRelease
  format: DebianFormat
  sources: boolean
  nonFree: boolean
  securityMirror: boolean
}

export function debianSources(options: Omit<DebianSourceOptions, 'release'> & { release: string }, releases: readonly GeneratorRelease[] = defaultGeneratorCatalog.debian.releases) {
  const release = releases.find(entry => entry.id === options.release)
  if (!release || !['list', 'deb822'].includes(options.format)) throw new Error('Invalid Debian configuration')
  const origin = mirrorOrigin(options.origin)
  const mirror = `${origin}/mirrors/debian/`
  const security = options.securityMirror ? `${origin}/mirrors/debian-security/` : 'https://security.debian.org/debian-security'
  const components = ['main', ...(options.nonFree ? ['contrib', 'non-free', ...(release.firmware ? ['non-free-firmware'] : [])] : [])].join(' ')
  const suites = [release.id, ...(release.stable ? [`${release.id}-updates`] : []), ...(release.backports ? [`${release.id}-backports`] : [])]
  const groups = [{ uri: mirror, suites }, ...(release.stable ? [{ uri: security, suites: [`${release.id}-security`] }] : [])]
  return aptSources(groups.map(group => ({ ...group, components, keyring: '/usr/share/keyrings/debian-archive-keyring.gpg' })), options.format, options.sources)
}
