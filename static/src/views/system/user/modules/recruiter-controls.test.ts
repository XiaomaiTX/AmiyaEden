import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'
import { compileTemplate, parse } from '@vue/compiler-sfc'
import * as Vue from 'vue'
import { renderToString } from 'vue/server-renderer'
import { ElCheckbox, ElCheckboxGroup } from 'element-plus'
import type { RenderFunction } from 'vue'

// Render the actual role controls with Element Plus. The surrounding dialog
// needs a browser and unrelated profile APIs; these controls do not.
test('Vue role controls keep recruiter read-only for both admin tiers', async () => {
  const source = readFileSync(new URL('./user-manage-dialog.vue', import.meta.url), 'utf8')
  const template = parse(source).descriptor.template?.content
  const controls = template?.match(/<ElCheckboxGroup\b[\s\S]*?<\/ElCheckboxGroup>/)?.[0]
  assert.ok(controls)
  const compiled = compileTemplate({
    source: controls,
    filename: 'user-manage-dialog.vue',
    id: 'recruiter-controls',
    compilerOptions: { mode: 'function' }
  })
  assert.deepEqual(compiled.errors, [])
  const render = new Function('Vue', compiled.code)(Vue) as RenderFunction
  for (const isSuperAdmin of [false, true]) {
    const app = Vue.createSSRApp({
      setup: () => ({
        allRoles: ['recruiter', 'srp', 'admin'].map((code) => ({ code })),
        selectedRoleCodes: ['recruiter'],
        isSuperAdmin,
        isEditingSelf: false,
        getLocalizedRoleName: (role: { code: string }) => role.code
      }),
      render
    })
    app.component('ElCheckbox', ElCheckbox)
    app.component('ElCheckboxGroup', ElCheckboxGroup)
    const labels = [...(await renderToString(app)).matchAll(/<label\b[\s\S]*?<\/label>/g)]
    const input = (code: string) => {
      const label = labels.find(([html]) => html.includes(`>${code}<`))?.[0]
      assert.ok(label, `missing ${code} control`)
      const input = label.match(/<input\b[^>]*>/)?.[0]
      assert.ok(input)
      return input
    }
    assert.match(input('recruiter'), /\bdisabled\b/)
    assert.match(input('recruiter'), /\bchecked\b/)
    assert.doesNotMatch(input('srp'), /\bdisabled\b/)
    assert.equal(/\bdisabled\b/.test(input('admin')), !isSuperAdmin)
  }
})
