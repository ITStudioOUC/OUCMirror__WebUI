import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import { errorCodes, errorMessages, apiErrorCodes } from '../src/error-pages.ts'

assert.deepEqual(Object.keys(errorMessages['zh-CN'].errors.codes), Object.keys(errorMessages['en-US'].errors.codes))
for (const status of errorCodes) {
  const html = await readFile(`dist/errors/${status}.html`, 'utf8')
  assert.ok(html.includes(`data-error-status="${status}"`))
  assert.ok(html.includes(errorMessages['zh-CN'].errors.codes[status as keyof typeof errorMessages['zh-CN']['errors']['codes']].title))
  assert.ok(!html.includes('{{') && !html.includes('/src/'))
  assert.ok(html.includes('href="/"') && html.includes('noindex, nofollow'))
  for (const [, asset] of html.matchAll(/(?:src|href)="(\/assets\/[^"]+)"/g)) await readFile('dist' + asset)
  const runtimePath = html.match(/defer src="([^"]+)"/)![1]!
  const runtime = await readFile('dist' + runtimePath, 'utf8')
  assert.ok(runtime.includes('mirrorouc.locale') && runtime.includes('mirrorouc.theme'))
  assert.ok(!/fetch\(|XMLHttpRequest|\/api\//.test(runtime), 'Error page must work without the backend')
  assert.deepEqual(JSON.parse(await readFile(`dist/errors/api-${status}.json`, 'utf8')), { code: apiErrorCodes[status] })
}
console.log(`Error pages passed: ${errorCodes.length} statuses, language parity, local assets and backend-independent rendering.`)
