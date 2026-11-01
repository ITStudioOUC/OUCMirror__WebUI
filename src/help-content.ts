import { marked, type Token, type TokensList } from 'marked'
import type { DebianFormat } from './debian-sources'

export type HelpBlock = { kind: 'markdown'; tokens: TokensList } | { kind: 'debian'; format: DebianFormat }

// Only an allowlisted data block can create a widget; Markdown never executes scripts.
export function helpBlocks(source: string): HelpBlock[] {
  const document = marked.lexer(source)
  const blocks: HelpBlock[] = []
  let pending: Token[] = []
  const flush = () => {
    if (pending.length) blocks.push({ kind: 'markdown', tokens: Object.assign(pending, { links: document.links }) })
    pending = []
  }
  for (const token of document) {
    let format: DebianFormat | undefined
    if (token.type === 'code' && token.lang === 'mirror-config' && token.text.length <= 2048) {
      try {
        const value: unknown = JSON.parse(token.text)
        if (value && typeof value === 'object' && !Array.isArray(value) && Object.keys(value).every(key => key === 'type' || key === 'format') && 'type' in value && value.type === 'debian' && 'format' in value && (value.format === 'list' || value.format === 'deb822')) format = value.format
      } catch { /* Invalid blocks remain visible as ordinary code for the author to correct. */ }
    }
    if (format) { flush(); blocks.push({ kind: 'debian', format }) }
    else pending.push(token)
  }
  flush()
  return blocks
}
