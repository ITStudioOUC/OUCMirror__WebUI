<script setup lang="ts">
import { computed } from 'vue'
import { marked } from 'marked'
import DOMPurify from 'dompurify'
const props = defineProps<{ source: string }>()
const html = computed(() => DOMPurify.sanitize(marked.parse(props.source, { async: false, breaks: false }), {
  ALLOWED_TAGS: ['p', 'br', 'hr', 'h1', 'h2', 'h3', 'h4', 'h5', 'h6', 'strong', 'em', 'del', 'blockquote', 'pre', 'code', 'ul', 'ol', 'li', 'a', 'table', 'thead', 'tbody', 'tr', 'th', 'td', 'img'],
  ALLOWED_ATTR: ['href', 'src', 'alt', 'title', 'start'],
  ALLOW_DATA_ATTR: false,
  ALLOW_UNKNOWN_PROTOCOLS: false,
}))
</script>
<template><div class="markdown-content" v-html="html" /></template>
<style>
.markdown-content { color: var(--ink); font-size: 14px; line-height: 1.85; overflow-wrap: anywhere; }
.markdown-content > :first-child { margin-top: 0; }.markdown-content > :last-child { margin-bottom: 0; }
.markdown-content :is(h1,h2,h3,h4,h5,h6) { margin: 1.4em 0 .6em; font-size: 1.15em; line-height: 1.4; }
.markdown-content a { color: var(--blue); text-decoration: underline; text-underline-offset: 3px; }
.markdown-content pre { max-width: 100%; overflow: auto; padding: 14px; border: 1px solid var(--line); border-radius: 5px; background: var(--surface); color: var(--ink); font: 12px/1.65 ui-monospace, Consolas, monospace; white-space: pre; }
.markdown-content :not(pre)>code { padding: 2px 5px; border-radius: 3px; background: var(--surface); font: .9em ui-monospace, Consolas, monospace; }
.markdown-content img { max-width: 100%; height: auto; }.markdown-content blockquote { margin-left: 0; padding-left: 16px; border-left: 3px solid var(--line); }
.markdown-content table { display: block; max-width: 100%; overflow-x: auto; border-collapse: collapse; }.markdown-content th,.markdown-content td { padding: 6px 10px; border: 1px solid var(--line); }
</style>
