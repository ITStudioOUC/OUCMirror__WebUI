import { marked, type Token, type TokensList } from 'marked'
import { parseMirrorConfig, type MirrorConfig } from './mirror-config.ts'
import { defaultGeneratorCatalog, type GeneratorCatalog } from './generator-catalog.ts'

export type HelpBlock = { kind: 'markdown'; tokens: TokensList } | { kind: 'config'; config: MirrorConfig }

// Only an allowlisted data block can create a widget; Markdown never executes scripts.
export function helpBlocks(source: string, catalog: GeneratorCatalog = defaultGeneratorCatalog): HelpBlock[] {
  const document = marked.lexer(source)
  const blocks: HelpBlock[] = []
  let pending: Token[] = []
  const flush = () => {
    if (pending.length) blocks.push({ kind: 'markdown', tokens: Object.assign(pending, { links: document.links }) })
    pending = []
  }
  for (const token of document) {
    let config: MirrorConfig | null = null
    if (token.type === 'code' && token.lang === 'mirror-config' && token.text.length <= 2048) {
      try {
        config = parseMirrorConfig(JSON.parse(token.text), catalog)
      } catch { /* Invalid blocks remain visible as ordinary code for the author to correct. */ }
    }
    if (config) { flush(); blocks.push({ kind: 'config', config }) }
    else pending.push(token)
  }
  flush()
  return blocks
}
