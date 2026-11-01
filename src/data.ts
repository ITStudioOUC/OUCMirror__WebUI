export type MirrorStatus = 'success' | 'syncing' | 'waiting' | 'failed' | 'paused' | 'unknown'
export type Category = 'all' | 'os' | 'language' | 'other'
export interface Mirror {
  name: string
  mark: string
  color: string
  category: Exclude<Category, 'all'>
  descriptionKey: string
  description?: Record<string, string>
  help?: string
  browseUrl?: string
  status: MirrorStatus
  lastSuccess: number | null
  nextSync: number | null
  size: number | null
  website: string
  upstream: string
  lastAttempt: number | null
  syncStarted: number | null
}

const started = Date.now()
const ago = (minutes: number) => started - minutes * 60_000
const after = (minutes: number) => started + minutes * 60_000
const mirrorEntries: Omit<Mirror, 'upstream' | 'lastAttempt' | 'syncStarted'>[] = [
  { name: 'alpine', mark: 'A', color: '#13799e', category: 'os', descriptionKey: 'mirrors.descriptions.alpine', status: 'success', lastSuccess: ago(26), nextSync: after(34), size: 842e9, website: 'https://alpinelinux.org/downloads/' },
  { name: 'anaconda', mark: '◎', color: '#278843', category: 'language', descriptionKey: 'mirrors.descriptions.anaconda', status: 'syncing', lastSuccess: ago(64), nextSync: null, size: 5.8e12, website: 'https://www.anaconda.com/download' },
  { name: 'archlinux', mark: 'Λ', color: '#1686b3', category: 'os', descriptionKey: 'mirrors.descriptions.archlinux', status: 'success', lastSuccess: ago(12), nextSync: after(48), size: 1.3e12, website: 'https://archlinux.org/download/' },
  { name: 'debian', mark: '∂', color: '#c83562', category: 'os', descriptionKey: 'mirrors.descriptions.debian', status: 'syncing', lastSuccess: ago(37), nextSync: null, size: 4.2e12, website: 'https://www.debian.org/distrib/' },
  { name: 'debian-cd', mark: '∂', color: '#c83562', category: 'os', descriptionKey: 'mirrors.descriptions.debian-cd', status: 'success', lastSuccess: ago(152), nextSync: after(88), size: 780e9, website: 'https://www.debian.org/CD/' },
  { name: 'docker-ce', mark: '▥', color: '#247bd2', category: 'language', descriptionKey: 'mirrors.descriptions.docker-ce', status: 'success', lastSuccess: ago(45), nextSync: after(75), size: 326e9, website: 'https://docs.docker.com/engine/install/' },
  { name: 'fedora', mark: 'f', color: '#456ba3', category: 'os', descriptionKey: 'mirrors.descriptions.fedora', status: 'waiting', lastSuccess: ago(116), nextSync: after(1), size: 2.1e12, website: 'https://fedoraproject.org/' },
  { name: 'homebrew', mark: 'H', color: '#9c7135', category: 'language', descriptionKey: 'mirrors.descriptions.homebrew', status: 'success', lastSuccess: ago(18), nextSync: after(42), size: 684e9, website: 'https://brew.sh/' },
  { name: 'linuxmint', mark: 'lm', color: '#548834', category: 'os', descriptionKey: 'mirrors.descriptions.linuxmint', status: 'paused', lastSuccess: ago(1480), nextSync: null, size: 96e9, website: 'https://linuxmint.com/download.php' },
  { name: 'npm', mark: 'n', color: '#bb3c3c', category: 'language', descriptionKey: 'mirrors.descriptions.npm', status: 'failed', lastSuccess: ago(183), nextSync: after(12), size: 3.6e12, website: 'https://www.npmjs.com/' },
  { name: 'pypi', mark: 'Py', color: '#3d6e9b', category: 'language', descriptionKey: 'mirrors.descriptions.pypi', status: 'syncing', lastSuccess: ago(8), nextSync: null, size: 18.7e12, website: 'https://pypi.org/' },
  { name: 'rustup', mark: 'R', color: '#9c6647', category: 'language', descriptionKey: 'mirrors.descriptions.rustup', status: 'success', lastSuccess: ago(72), nextSync: after(48), size: 1.1e12, website: 'https://rustup.rs/' },
  { name: 'ubuntu', mark: 'u', color: '#d96132', category: 'os', descriptionKey: 'mirrors.descriptions.ubuntu', status: 'success', lastSuccess: ago(21), nextSync: after(39), size: 3.4e12, website: 'https://ubuntu.com/download' },
  { name: 'ubuntu-releases', mark: 'u', color: '#d96132', category: 'os', descriptionKey: 'mirrors.descriptions.ubuntu-releases', status: 'success', lastSuccess: ago(126), nextSync: after(114), size: 124e9, website: 'https://releases.ubuntu.com/' },
  { name: 'vim', mark: 'V', color: '#30814c', category: 'other', descriptionKey: 'mirrors.descriptions.vim', status: 'unknown', lastSuccess: null, nextSync: null, size: null, website: 'https://www.vim.org/download.php' },
]

// Example upstream metadata for the prototype; no sync jobs use these URLs yet.
const demoUpstreams: Record<string, string> = {
  alpine: 'https://dl-cdn.alpinelinux.org/alpine/',
  anaconda: 'https://repo.anaconda.com/',
  archlinux: 'https://geo.mirror.pkgbuild.com/',
  debian: 'https://deb.debian.org/debian/',
  'debian-cd': 'https://cdimage.debian.org/debian-cd/',
  'docker-ce': 'https://download.docker.com/',
  fedora: 'https://dl.fedoraproject.org/pub/fedora/linux/',
  homebrew: 'https://formulae.brew.sh/api/',
  linuxmint: 'https://packages.linuxmint.com/',
  npm: 'https://registry.npmjs.org/',
  pypi: 'https://pypi.org/',
  rustup: 'https://static.rust-lang.org/rustup/',
  ubuntu: 'https://archive.ubuntu.com/ubuntu/',
  'ubuntu-releases': 'https://releases.ubuntu.com/',
  vim: 'https://www.vim.org/',
}
export const demoMirrors: Mirror[] = mirrorEntries.map((mirror, index) => ({
  ...mirror,
  upstream: demoUpstreams[mirror.name]!,
  lastAttempt: mirror.status === 'failed' ? ago(8) : mirror.status === 'syncing' ? ago(3 + index % 4) : mirror.lastSuccess,
  syncStarted: mirror.status === 'syncing' ? ago(3 + index % 4) : null,
}))

export const announcements = [
  {
    "id": "preview",
    "date": "2026-11-01T00:00:00+08:00",
    "pinned": true,
    "titleKey": "announcements.entries.preview.title",
    "bodyKey": "announcements.entries.preview.body"
  },
  {
    "id": "help",
    "date": "2026-11-01T00:00:00+08:00",
    "pinned": false,
    "titleKey": "announcements.entries.help.title",
    "bodyKey": "announcements.entries.help.body"
  }
]

export const distributions = [
  { name: 'Ubuntu', version: 'LTS', color: '#d96132', mark: 'u', url: 'https://ubuntu.com/download/desktop' },
  { name: 'Debian', version: '', color: '#c83562', mark: '∂', url: 'https://www.debian.org/distrib/' },
  { name: 'Arch Linux', version: '', color: '#1686b3', mark: 'Λ', url: 'https://archlinux.org/download/' },
  { name: 'Fedora', version: '', color: '#456ba3', mark: 'f', url: 'https://fedoraproject.org/workstation/download' },
]
