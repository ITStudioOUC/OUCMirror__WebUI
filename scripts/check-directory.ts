import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'

// Run after the build to check the actual Nginx fragments and hashed assets.
const header = await readFile('dist/fancy-index/header.html', 'utf8')
const footer = await readFile('dist/fancy-index/footer.html', 'utf8')
const labels = await readFile('dist/fancy-index/labels.conf', 'utf8')
assert.ok(!header.includes('directory-path'))
assert.ok(footer.includes('</html>'))
assert.ok(labels.includes('data-i18n=\\"directory.parent\\"'))
assert.ok(!/\{\{[\w.]+\}\}/.test(header + footer))
assert.ok(!header.includes('/src/'))
for (const [, url] of [...header.matchAll(/(?:src|href)="(\/assets\/[^"?]+)"/g)]) await readFile('dist' + url)
assert.ok(header.includes('data-locale="zh-CN"') && header.includes('data-locale="en-US"'))
assert.ok(header.includes('data-theme-choice="system"'))
console.log('Nginx directory fragments, locale substitution and referenced assets passed.')
