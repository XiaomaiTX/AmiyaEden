const assert = require('node:assert/strict')
const { test } = require('node:test')
const { extractVueScripts } = require('./inventory-vue.cjs')

test('extracts both Vue scripts in source order and accepts whitespace in a closing tag', () => {
  const setup = "import { ref } from 'vue'; const ready = ref(true)"
  const script = 'export default { name: "InventoryFixture" }'
  const component = `<script setup lang="ts">${setup}</script >\n<script lang="ts">${script}</script>`

  assert.equal(extractVueScripts(component, 'InventoryFixture.vue'), `${setup}\n${script}`)
})

test('ignores script-like text in comments and custom blocks', () => {
  const component = `
    <!-- <script>export const commented = true</script> -->
    <template><div>Example</div></template>
    <docs><script>export const documentation = true</script></docs>
  `

  assert.equal(extractVueScripts(component, 'InventoryFixture.vue'), '')
})

test('reports malformed components rather than silently omitting scripts', () => {
  assert.throws(
    () => extractVueScripts('<script>export default {}', 'BrokenFixture.vue'),
    /^Error: BrokenFixture\.vue: Element is missing end tag\./
  )
})
