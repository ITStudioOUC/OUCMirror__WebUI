export type AptFormat = 'list' | 'deb822'
export interface AptRepository { uri: string; suites: string[]; components: string; keyring: string }

export function mirrorOrigin(value: string, protocol?: string) {
  const origin = new URL(value)
  if (!['http:', 'https:'].includes(origin.protocol) || origin.username || origin.password) throw new Error('Invalid mirror origin')
  if (!/^[a-z0-9._:\[\]-]+$/i.test(origin.host)) throw new Error('Invalid mirror host')
  if (protocol !== undefined) {
    if (protocol !== 'http' && protocol !== 'https') throw new Error('Invalid mirror protocol')
    origin.protocol = protocol + ':'
  }
  return origin.origin
}

export function aptSources(groups: AptRepository[], format: AptFormat, sources: boolean) {
  if (format !== 'list' && format !== 'deb822') throw new Error('Invalid APT format')
  if (format === 'deb822') {
    return groups.map(group => [
      `Types: ${sources ? 'deb deb-src' : 'deb'}`,
      `URIs: ${group.uri}`,
      `Suites: ${group.suites.join(' ')}`,
      `Components: ${group.components}`,
      `Signed-By: ${group.keyring}`,
    ].join('\n')).join('\n\n') + '\n'
  }
  return groups.flatMap(group => group.suites.map(suite => [
    `deb ${group.uri} ${suite} ${group.components}`,
    `${sources ? '' : '# '}deb-src ${group.uri} ${suite} ${group.components}`,
  ].join('\n'))).join('\n\n') + '\n'
}
