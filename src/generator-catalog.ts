import defaults from '../backend/generator-defaults.json' with { type: 'json' }

export const versionedGenerators = ['debian', 'ubuntu', 'kubernetes', 'tailscale', 'proxmox'] as const
export type VersionedGenerator = typeof versionedGenerators[number]
export interface GeneratorRelease {
  id: string
  label: string
  stable?: boolean
  firmware?: boolean
  backports?: boolean
  keyring?: string
}
export interface GeneratorVersions { defaultRelease: string; releases: GeneratorRelease[] }
export type GeneratorCatalog = Record<VersionedGenerator, GeneratorVersions>
export const defaultGeneratorCatalog: GeneratorCatalog = defaults

export function validGeneratorVersions(type: VersionedGenerator, value: unknown): value is GeneratorVersions {
  if (!value || typeof value !== 'object' || Array.isArray(value)) return false
  const data = value as GeneratorVersions
  if (Object.keys(data).some(key => !['defaultRelease', 'releases'].includes(key)) || typeof data.defaultRelease !== 'string' || !Array.isArray(data.releases) || !data.releases.length || data.releases.length > 100) return false
  const seen = new Set<string>()
  for (const release of data.releases) {
    if (!release || typeof release !== 'object' || Array.isArray(release) || Object.keys(release).some(key => !['id', 'label', 'stable', 'firmware', 'backports', 'keyring'].includes(key))) return false
    const idPattern = type === 'kubernetes' ? /^v[1-9][0-9]{0,2}\.(0|[1-9][0-9]{0,2})$/ : type === 'tailscale' ? /^(debian|ubuntu)\/[a-z][a-z0-9-]{0,31}$/ : /^[a-z][a-z0-9-]{0,31}$/
    if (typeof release.id !== 'string' || !idPattern.test(release.id) || seen.has(release.id) || typeof release.label !== 'string' || !release.label.trim() || [...release.label].length > 160 || /[\p{Cc}]/u.test(release.label)) return false
    seen.add(release.id)
    if (type === 'debian') {
      if (['stable', 'firmware', 'backports'].some(key => typeof release[key as keyof GeneratorRelease] !== 'boolean') || release.backports && !release.stable) return false
    } else if (['stable', 'firmware', 'backports'].some(key => key in release)) return false
    if (type === 'proxmox') {
      if (typeof release.keyring !== 'string' || !/^\/(usr\/share\/keyrings|etc\/apt\/trusted\.gpg\.d)\/[a-zA-Z0-9][a-zA-Z0-9_-]{0,95}\.gpg$/.test(release.keyring)) return false
    } else if ('keyring' in release) return false
  }
  return seen.has(data.defaultRelease) && (type !== 'debian' || data.releases.some(release => release.stable))
}

export function parseGeneratorCatalog(value: unknown): GeneratorCatalog | null {
  if (!value || typeof value !== 'object' || Array.isArray(value)) return null
  const data = value as GeneratorCatalog
  if (Object.keys(data).length !== versionedGenerators.length || versionedGenerators.some(type => !Object.hasOwn(data, type) || !validGeneratorVersions(type, data[type]))) return null
  return data
}
