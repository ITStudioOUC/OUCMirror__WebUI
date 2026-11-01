import assert from 'node:assert/strict'
import { helpRoute, helpUrl } from '../src/help.ts'

for (const name of ['ubuntu', 'debian-cd', 'AOSP', 'gcc.git', 'archlinuxcn', 'test+repo']) {
  const url = helpUrl(name)
  assert.deepEqual(helpRoute(url), { active: true, name, invalid: false })
  assert.deepEqual(helpRoute(url.slice(0, -1)), { active: true, name, invalid: false })
}
for (const path of ['/help', '/help/']) assert.deepEqual(helpRoute(path), { active: true, name: null, invalid: false })
for (const path of ['/', '/helpful', '/announcements']) assert.equal(helpRoute(path).active, false)
for (const path of ['/help/../', '/help/ubuntu/extra', '/help/%2fsecret', '/help/%zz', '/help/%252fsecret']) assert.equal(helpRoute(path).invalid, true)
assert.equal(helpUrl('a/b?c#d'), '/help/a%2Fb%3Fc%23d/')
console.log('Help check passed: index and deep links, case-sensitive names, encoded identifiers and malformed paths.')
