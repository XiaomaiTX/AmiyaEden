import { readFileSync, writeFileSync } from 'fs'
import { join } from 'path'
import { fileURLToPath } from 'url'

const __filename = fileURLToPath(import.meta.url)
const __dirname = join(__filename, '..')

const packageJsonPath = join(__dirname, '..', 'package.json')
const envDevPath = join(__dirname, '..', '.env.development')
const envProdPath = join(__dirname, '..', '.env.production')

try {
  const packageJson = JSON.parse(readFileSync(packageJsonPath, 'utf-8'))
  const version = packageJson.version

  const updateEnvFile = (envPath) => {
    let content = readFileSync(envPath, 'utf-8')

    const envVars = {
      VITE_VERSION: version
    }

    Object.entries(envVars).forEach(([key, value]) => {
      const regex = new RegExp(`^${key}\\s*=.*`, 'm')
      if (content.includes(key)) {
        content = content.replace(regex, `${key} = ${value}`)
      } else {
        content += `\n${key} = ${value}\n`
      }
    })

    writeFileSync(envPath, content)
  }

  updateEnvFile(envDevPath)
  updateEnvFile(envProdPath)

  console.log(`✅ 版本号已同步: ${version}`)
} catch (error) {
  console.error('❌ 同步版本号失败:', error)
  process.exit(1)
}
