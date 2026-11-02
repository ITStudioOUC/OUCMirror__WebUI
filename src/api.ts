export type Localized = Record<string, string>
export interface SiteLink { title: Localized; url: string }
export interface RelatedLink extends SiteLink { kind: 'url' | 'help' }
export type DownloadCategory = 'os' | 'software' | 'fonts'
export interface SiteDownload { id: string; name: string; category: DownloadCategory; description: string }
export interface SiteSettings {
  name: Localized; description: Localized; logoLight: string; logoDark: string; favicon: string; avatar: string
  email: string; address: Localized; links: SiteLink[]; relatedLinks: RelatedLink[]; downloads: SiteDownload[]
}
export interface AnnouncementContent { title: string; body: string; published: boolean; pinned: boolean; date: string }
export interface MirrorMetadata { name: string; description: Localized; help: string; category: string; visible: boolean; website: string }
export interface HelpSummary { name: string; description: Localized; website: string; browseUrl: string }
export interface HelpArticle extends HelpSummary { body: string }
export interface ContentRecord<T> { id: string; data: T; revision: number; updatedAt: number }
export interface AdminSession { username: string; role: 'admin' | 'editor'; csrf: string }
export interface SyncJob { name: string; worker: string; status: string; last_update: string; last_started: string; last_ended: string; next_schedule: string; upstream: string; size: string; error_msg: string }
export interface Overview { jobs: SyncJob[]; workers: { id: string; last_online: string }[]; worker: string; updatedAt: number; code: string; configAvailable: boolean; logsAvailable: boolean }
export interface SyncNumbers { concurrent: number; interval: number; retry: number; timeout: number }
export interface SyncConfigMirror { name: string; provider: string; upstream: string; interval: number; retry: number; timeout: number; source?: string }
export interface SyncConfig { worker: string; baseHash: string; global: SyncNumbers; mirrors: SyncConfigMirror[]; canAdd: boolean }
export interface ConfigVersion { id: string; files: Record<string, { before: string | null; after: string | null }> | null; baseHash: string; status: string; created: number; username: string }
export interface AuditEntry { id: number; username: string; action: string; target: string; result: string; created: number }
export interface AuditPage { items: AuditEntry[]; page: number; pageSize: number; total: number; totalPages: number }
export interface AdminUser { id: number; username: string; role: string; created: number }

let csrf = ''
export class ApiError extends Error { constructor(public code: string, public status: number) { super(code) } }
export function setSession(value: AdminSession | null) { csrf = value?.csrf ?? '' }
export async function api<T>(path: string, options: RequestInit = {}): Promise<T> {
  const headers = new Headers(options.headers)
  if (options.body && !(options.body instanceof FormData)) headers.set('Content-Type', 'application/json')
  if (options.method && !['GET', 'HEAD'].includes(options.method)) headers.set('X-CSRF-Token', csrf)
  let response: Response
  try { response = await fetch(`/api/v1${path}`, { ...options, headers, credentials: 'same-origin' }) }
  catch (error) { if ((error as Error).name === 'AbortError') throw error; throw new ApiError('NETWORK_ERROR', 0) }
  const data = await response.json().catch(() => ({ code: 'INVALID_RESPONSE' }))
  if (!response.ok) throw new ApiError(data.code || 'INTERNAL_ERROR', response.status)
  return data as T
}
export const jsonBody = (value: unknown) => JSON.stringify(value)
export function content(value: Localized | null | undefined, locale: string) { return value?.[locale] || value?.['zh-CN'] || Object.values(value || {}).find(Boolean) || '' }
