import { aptSources, mirrorOrigin, type AptFormat } from './apt-sources.ts'
import { defaultGeneratorCatalog, type GeneratorRelease } from './generator-catalog.ts'

export const ubuntuReleases = defaultGeneratorCatalog.ubuntu.releases

export interface UbuntuSourceOptions {
  origin: string
  release: string
  format: AptFormat
  sources: boolean
  proposed: boolean
  securityMirror: boolean
}

export function ubuntuSources(options: UbuntuSourceOptions, releases: readonly GeneratorRelease[] = defaultGeneratorCatalog.ubuntu.releases) {
  if (!releases.some(release => release.id === options.release)) throw new Error('Invalid Ubuntu release')
  const origin = mirrorOrigin(options.origin)
  const uri = `${origin}/mirrors/ubuntu/`
  const suites = [options.release, `${options.release}-updates`, `${options.release}-backports`]
  if (options.proposed) suites.push(`${options.release}-proposed`)
  return aptSources([
    { uri, suites, components: 'main restricted universe multiverse', keyring: '/usr/share/keyrings/ubuntu-archive-keyring.gpg' },
    { uri: options.securityMirror ? uri : 'http://security.ubuntu.com/ubuntu/', suites: [`${options.release}-security`], components: 'main restricted universe multiverse', keyring: '/usr/share/keyrings/ubuntu-archive-keyring.gpg' },
  ], options.format, options.sources)
}
