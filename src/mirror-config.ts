import { aptSources, mirrorOrigin, type AptFormat } from './apt-sources.ts'
import { debianSources } from './debian-sources.ts'
import { ubuntuSources } from './ubuntu-sources.ts'
import { defaultGeneratorCatalog, type GeneratorCatalog, type VersionedGenerator } from './generator-catalog.ts'

export type MirrorConfigType = 'debian' | 'ubuntu' | 'debian-security' | 'archlinux' | 'archlinuxarm' | 'archlinuxcn' | 'anaconda' | 'kubernetes' | 'tailscale' | 'proxmox' | 'openwrt' | 'elrepo'
export type ConfigValues = Record<string, string | boolean>
export interface ConfigChoice { value: string; label?: string; labelKey?: string }
export interface ConfigField {
  key: string
  labelKey: string
  kind: 'select' | 'switch'
  choices?: ConfigChoice[]
  disabled?: (values: ConfigValues) => boolean
}
export interface ConfigFile { path?: string; titleKey?: string; code: string }
interface GeneratorContext { origin: string; format: string; values: ConfigValues; catalog: GeneratorCatalog }
export interface MirrorGenerator {
  nameKey: string
  formats: ConfigChoice[]
  defaultFormat: string
  defaults: ConfigValues
  fields: ConfigField[]
  notes?: { key: string; when?: (values: ConfigValues) => boolean }[]
  normalize?: (values: ConfigValues) => void
  generate: (context: GeneratorContext) => ConfigFile[]
}
export interface MirrorConfig { type: MirrorConfigType; format?: string; options?: ConfigValues }

const aptFormats: ConfigChoice[] = [{ value: 'list', labelKey: 'mirrors.generator.formats.list' }, { value: 'deb822', label: 'DEB822' }]
const format = (value: string, label = value): ConfigChoice[] => [{ value, label }]
const versionField = (choices: ConfigChoice[]): ConfigField => ({ key: 'release', kind: 'select', labelKey: 'mirrors.generator.version', choices })
const sourceField: ConfigField = { key: 'sources', kind: 'switch', labelKey: 'mirrors.generator.sources' }
const securityField: ConfigField = { key: 'securityMirror', kind: 'switch', labelKey: 'mirrors.generator.securityMirror' }
const nonFreeField: ConfigField = { key: 'nonFree', kind: 'switch', labelKey: 'mirrors.generator.nonFree' }
const choices = (type: VersionedGenerator) => defaultGeneratorCatalog[type].releases.map(release => ({ value: release.id, label: release.label }))
const aptFile = (name: string, selected: string) => selected === 'list' ? '/etc/apt/sources.list' : `/etc/apt/sources.list.d/${name}.sources`
const configText = (path: string, code: string): ConfigFile[] => [{ path, code }]
const aptKey = (url: string, path: string): ConfigFile => ({ titleKey: 'mirrors.generator.signingKey', code: `sudo install -d -m 0755 ${path.slice(0, path.lastIndexOf('/'))}\ncurl -fsSL ${url} | sudo gpg --dearmor -o ${path}\nsudo chmod 0644 ${path}\n` })

// A new generator supplies its fields and configuration rules here; the Markdown
// parser, controls, copy behavior and responsive layout are shared by all sources.
export const mirrorGenerators: Record<MirrorConfigType, MirrorGenerator> = {
  debian: {
    nameKey: 'mirrors.generator.names.debian', formats: aptFormats, defaultFormat: 'list',
    defaults: { release: defaultGeneratorCatalog.debian.defaultRelease, sources: false, nonFree: true, securityMirror: false },
    fields: [versionField(choices('debian')), sourceField, nonFreeField, securityField],
    generate: ({ origin, format, values, catalog }) => configText(aptFile('debian', format), debianSources({ origin, format: format as AptFormat, release: String(values.release), sources: Boolean(values.sources), nonFree: Boolean(values.nonFree), securityMirror: Boolean(values.securityMirror) }, catalog.debian.releases)),
  },
  ubuntu: {
    nameKey: 'mirrors.generator.names.ubuntu', formats: aptFormats, defaultFormat: 'deb822',
    defaults: { release: defaultGeneratorCatalog.ubuntu.defaultRelease, sources: false, proposed: false, securityMirror: false },
    fields: [versionField(choices('ubuntu')), sourceField, { key: 'proposed', kind: 'switch', labelKey: 'mirrors.generator.proposed' }, securityField],
    notes: [{ key: 'mirrors.generator.ubuntuScope' }, { key: 'mirrors.generator.proposedNotice', when: values => Boolean(values.proposed) }],
    generate: ({ origin, format, values, catalog }) => configText(aptFile('ubuntu', format), ubuntuSources({ origin, format: format as AptFormat, release: String(values.release), sources: Boolean(values.sources), proposed: Boolean(values.proposed), securityMirror: Boolean(values.securityMirror) }, catalog.ubuntu.releases)),
  },
  'debian-security': {
    nameKey: 'mirrors.generator.names.debian-security', formats: aptFormats, defaultFormat: 'deb822',
    defaults: { release: defaultGeneratorCatalog.debian.defaultRelease, sources: false, nonFree: true }, fields: [versionField(defaultGeneratorCatalog.debian.releases.filter(release => release.stable).map(release => ({ value: release.id, label: release.label }))), sourceField, nonFreeField],
    generate: ({ origin, format, values, catalog }) => {
      const release = catalog.debian.releases.find(entry => entry.id === values.release)!
      const components = ['main', ...(values.nonFree ? ['contrib', 'non-free', ...(release.firmware ? ['non-free-firmware'] : [])] : [])].join(' ')
      return configText(format === 'list' ? '/etc/apt/sources.list' : '/etc/apt/sources.list.d/debian-security.sources', aptSources([{ uri: `${origin}/mirrors/debian-security/`, suites: [`${release.id}-security`], components, keyring: '/usr/share/keyrings/debian-archive-keyring.gpg' }], format as AptFormat, Boolean(values.sources)))
    },
  },
  archlinux: {
    nameKey: 'mirrors.generator.names.archlinux', formats: format('pacman', 'pacman'), defaultFormat: 'pacman', defaults: {}, fields: [],
    generate: ({ origin }) => configText('/etc/pacman.d/mirrorlist', `Server = ${origin}/mirrors/archlinux/$repo/os/$arch\n`),
  },
  archlinuxarm: {
    nameKey: 'mirrors.generator.names.archlinuxarm', formats: format('pacman', 'pacman'), defaultFormat: 'pacman', defaults: {}, fields: [],
    generate: ({ origin }) => configText('/etc/pacman.d/mirrorlist', `Server = ${origin}/mirrors/archlinuxarm/$arch/$repo\n`),
  },
  archlinuxcn: {
    nameKey: 'mirrors.generator.names.archlinuxcn', formats: format('pacman', 'pacman'), defaultFormat: 'pacman', defaults: {}, fields: [],
    generate: ({ origin }) => configText('/etc/pacman.conf', `[archlinuxcn]\nServer = ${origin}/mirrors/archlinuxcn/$arch\n`),
  },
  anaconda: {
    nameKey: 'mirrors.generator.names.anaconda', formats: format('condarc', '.condarc'), defaultFormat: 'condarc', defaults: { channel: 'conda-forge' },
    fields: [{ key: 'channel', kind: 'select', labelKey: 'mirrors.generator.channel', choices: ['conda-forge', 'bioconda', 'defaults'].map(value => ({ value, label: value })) }],
    generate: ({ origin, values }) => {
      const prefix = `${origin}/mirrors/anaconda/`
      if (values.channel === 'defaults') return configText('~/.condarc', `channels:\n  - defaults\nshow_channel_urls: true\ndefault_channels:\n  - ${prefix}pkgs/main\n  - ${prefix}pkgs/r\n  - ${prefix}pkgs/msys2\n`)
      const channels = values.channel === 'bioconda' ? ['conda-forge', 'bioconda'] : ['conda-forge']
      return configText('~/.condarc', `channels:\n${channels.map(name => `  - ${name}`).join('\n')}\nshow_channel_urls: true\nchannel_priority: strict\ncustom_channels:\n${channels.map(name => `  ${name}: ${prefix}cloud`).join('\n')}\n`)
    },
  },
  kubernetes: {
    nameKey: 'mirrors.generator.names.kubernetes', formats: [{ value: 'deb', label: 'APT (Debian / Ubuntu)' }, { value: 'rpm', label: 'RPM' }], defaultFormat: 'deb', defaults: { release: defaultGeneratorCatalog.kubernetes.defaultRelease },
    fields: [versionField(choices('kubernetes'))],
    notes: [{ key: 'mirrors.generator.kubernetesNotice' }],
    generate: ({ origin, format, values }) => {
      const version = String(values.release)
      const suffix = `core:/stable:/${version}/${format}/`
      const uri = `${origin}/mirrors/kubernetes/${suffix}`
      if (format === 'rpm') return configText('/etc/yum.repos.d/kubernetes.repo', `[kubernetes]\nname=Kubernetes\nbaseurl=${uri}\nenabled=1\ngpgcheck=1\ngpgkey=https://pkgs.k8s.io/${suffix}repodata/repomd.xml.key\nexclude=kubelet kubeadm kubectl cri-tools kubernetes-cni\n`)
      const keyring = '/etc/apt/keyrings/kubernetes-apt-keyring.gpg'
      return [aptKey(`https://pkgs.k8s.io/${suffix}Release.key`, keyring), { path: '/etc/apt/sources.list.d/kubernetes.list', code: `deb [signed-by=${keyring}] ${uri} /\n` }]
    },
  },
  tailscale: {
    nameKey: 'mirrors.generator.names.tailscale', formats: aptFormats, defaultFormat: 'list', defaults: { release: defaultGeneratorCatalog.tailscale.defaultRelease },
    fields: [versionField(choices('tailscale'))],
    generate: ({ origin, format, values }) => {
      const [distribution, suite] = String(values.release).split('/')
      const keyring = '/usr/share/keyrings/tailscale-archive-keyring.gpg'
      const key: ConfigFile = { titleKey: 'mirrors.generator.signingKey', code: `sudo install -d -m 0755 /usr/share/keyrings\ncurl -fsSL https://pkgs.tailscale.com/stable/${distribution}/${suite}.noarmor.gpg | sudo tee ${keyring} >/dev/null\nsudo chmod 0644 ${keyring}\n` }
      const uri = `${origin}/mirrors/tailscale/${distribution}`
      const code = format === 'list' ? `deb [signed-by=${keyring}] ${uri} ${suite} main\n` : aptSources([{ uri, suites: [suite], components: 'main', keyring }], 'deb822', false)
      return [key, { path: `/etc/apt/sources.list.d/tailscale.${format === 'list' ? 'list' : 'sources'}`, code }]
    },
  },
  proxmox: {
    nameKey: 'mirrors.generator.names.proxmox', formats: aptFormats, defaultFormat: 'deb822', defaults: { release: defaultGeneratorCatalog.proxmox.defaultRelease },
    fields: [versionField(choices('proxmox'))],
    generate: ({ origin, format, values, catalog }) => {
      const uri = `${origin}/mirrors/proxmox/debian/pve`
      const keyring = catalog.proxmox.releases.find(release => release.id === values.release)!.keyring!
      const code = format === 'list' ? `deb ${uri} ${values.release} pve-no-subscription\n` : aptSources([{ uri, suites: [String(values.release)], components: 'pve-no-subscription', keyring }], 'deb822', false)
      return configText(`/etc/apt/sources.list.d/pve-no-subscription.${format === 'list' ? 'list' : 'sources'}`, code)
    },
  },
  openwrt: {
    nameKey: 'mirrors.generator.names.openwrt', formats: [{ value: 'apk', label: 'APK (OpenWrt ≥ 25.12)' }, { value: 'opkg', label: 'OPKG (OpenWrt ≤ 24.10)' }], defaultFormat: 'apk', defaults: {}, fields: [],
    notes: [{ key: 'mirrors.generator.openwrtNotice' }],
    generate: ({ origin, format }) => {
      const path = format === 'apk' ? '/etc/apk/repositories.d/distfeeds.list' : '/etc/opkg/distfeeds.conf'
      return [{ titleKey: 'mirrors.generator.applyCommands', code: `cp ${path} ${path}.bak\nsed -i 's|https\\?://downloads.openwrt.org|${origin}/mirrors/openwrt|g' ${path}\n${format} update\n` }]
    },
  },
  elrepo: {
    nameKey: 'mirrors.generator.names.elrepo', formats: format('shell', 'Shell'), defaultFormat: 'shell', defaults: {}, fields: [],
    notes: [{ key: 'mirrors.generator.elrepoNotice' }],
    generate: ({ origin }) => [{ titleKey: 'mirrors.generator.applyCommands', code: `sudo cp /etc/yum.repos.d/elrepo.repo /etc/yum.repos.d/elrepo.repo.bak\nsudo sed -i -e '/^[[:space:]]*mirrorlist=/s/^/#/' -e 's|^#baseurl=|baseurl=|' -e 's|https\\?://elrepo.org/linux|${origin}/mirrors/elrepo|g' /etc/yum.repos.d/elrepo.repo\nsudo dnf makecache\n` }],
  },
}

export const mirrorConfigTypes = Object.keys(mirrorGenerators) as MirrorConfigType[]
export const protocolField: ConfigField = { key: 'protocol', kind: 'select', labelKey: 'mirrors.generator.protocol', choices: [{ value: 'https', label: 'HTTPS' }, { value: 'http', label: 'HTTP' }] }
export function getMirrorGenerator(type: MirrorConfigType, catalog: GeneratorCatalog = defaultGeneratorCatalog): MirrorGenerator {
  const base = mirrorGenerators[type]
  const catalogType = type === 'debian-security' ? 'debian' : type
  if (!Object.hasOwn(catalog, catalogType)) return base
  const versions = catalog[catalogType as VersionedGenerator]
  const releases = type === 'debian-security' ? versions.releases.filter(release => release.stable) : versions.releases
  const defaultRelease = releases.find(release => release.id === versions.defaultRelease)?.id || releases[0]!.id
  const stable = (values: ConfigValues) => Boolean(versions.releases.find(release => release.id === values.release)?.stable)
  return {
    ...base,
    defaults: { ...base.defaults, release: defaultRelease },
    fields: base.fields.map(field => field.key === 'release' ? versionField(releases.map(release => ({ value: release.id, label: release.label }))) : type === 'debian' && field.key === 'securityMirror' ? { ...field, disabled: values => !stable(values) } : field),
    ...(type === 'debian' ? { normalize: values => { if (!stable(values)) values.securityMirror = false }, notes: [{ key: 'mirrors.debian.rollingNotice', when: values => !stable(values) }] } : {}),
  }
}
export function configFields(type: MirrorConfigType, catalog: GeneratorCatalog = defaultGeneratorCatalog) { return [...getMirrorGenerator(type, catalog).fields, protocolField] }

// JSON config selects a registered generator and allowlisted values; it cannot
// supply scripts, HTML, functions, arbitrary templates or package-manager flags.
export function parseMirrorConfig(value: unknown, catalog: GeneratorCatalog = defaultGeneratorCatalog): MirrorConfig | null {
  if (!value || typeof value !== 'object' || Array.isArray(value)) return null
  if (Object.keys(value).some(key => !['type', 'format', 'options'].includes(key)) || !('type' in value) || typeof value.type !== 'string' || !Object.hasOwn(mirrorGenerators, value.type)) return null
  const type = value.type as MirrorConfigType
  const generator = getMirrorGenerator(type, catalog)
  if ('format' in value && (typeof value.format !== 'string' || !generator.formats.some(choice => choice.value === value.format))) return null
  if ('options' in value) {
    const options = value.options
    if (!options || typeof options !== 'object' || Array.isArray(options)) return null
    const fields = configFields(type, catalog)
    if (Object.entries(options).some(([key, entry]) => {
      const field = fields.find(field => field.key === key)
      return !field || (field.kind === 'switch' ? typeof entry !== 'boolean' : typeof entry !== 'string' || !field.choices!.some(choice => choice.value === entry))
    })) return null
  }
  return { type, ...('format' in value ? { format: value.format as string } : {}), ...('options' in value ? { options: { ...value.options as ConfigValues } } : {}) }
}

export function configDefaults(config: MirrorConfig, origin: string, catalog: GeneratorCatalog = defaultGeneratorCatalog): ConfigValues {
  const parsed = parseMirrorConfig(config, catalog)
  if (!parsed) throw new Error('Invalid mirror configuration')
  return { ...getMirrorGenerator(parsed.type, catalog).defaults, protocol: new URL(mirrorOrigin(origin)).protocol.slice(0, -1), ...parsed.options }
}

export function generateMirrorConfig(config: MirrorConfig, origin: string, values: ConfigValues = {}, catalog: GeneratorCatalog = defaultGeneratorCatalog) {
  const parsed = parseMirrorConfig({ ...config, options: { ...config.options, ...values } }, catalog)
  if (!parsed) throw new Error('Invalid mirror configuration')
  const generator = getMirrorGenerator(parsed.type, catalog)
  const state = configDefaults(parsed, origin, catalog)
  generator.normalize?.(state)
  return generator.generate({ origin: mirrorOrigin(origin, String(state.protocol)), format: parsed.format || generator.defaultFormat, values: state, catalog })
}
