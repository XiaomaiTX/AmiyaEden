const path = require('node:path')
const { parse } = require(require.resolve('@vue/compiler-sfc', { paths: [path.resolve(__dirname, '../static')] }))

function extractVueScripts(text, filename) {
  const { descriptor, errors } = parse(text, { filename, sourceMap: false })
  if (errors.length) {
    throw new Error(`${filename}: ${errors.map((error) => error.message ?? error).join('; ')}`)
  }
  return [descriptor.script, descriptor.scriptSetup]
    .filter(Boolean)
    .sort((a, b) => a.loc.start.offset - b.loc.start.offset)
    .map((block) => block.content)
    .join('\n')
}

module.exports = { extractVueScripts }
