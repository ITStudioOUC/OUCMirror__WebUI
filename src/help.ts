const mirrorName = /^[a-zA-Z0-9][a-zA-Z0-9_.+-]{0,127}$/

export const helpUrl = (name: string) => `/help/${encodeURIComponent(name)}/`

export function helpRoute(path: string) {
  const normalized = path.replace(/\/$/, '')
  if (normalized === '/help') return { active: true, name: null, invalid: false }
  if (!normalized.startsWith('/help/')) return { active: false, name: null, invalid: false }
  let name: string
  try { name = decodeURIComponent(normalized.slice('/help/'.length)) }
  catch { return { active: true, name: null, invalid: true } }
  return { active: true, name, invalid: !mirrorName.test(name) }
}
