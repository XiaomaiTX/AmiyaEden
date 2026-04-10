import { execSync } from 'child_process'

const getChangedFiles = () => {
  try {
    const output = execSync('git diff --name-only HEAD~1 HEAD', { encoding: 'utf-8' })
    return output.trim().split('\n').filter(Boolean)
  } catch {
    return []
  }
}

const getStagedFiles = () => {
  try {
    const output = execSync('git diff --cached --name-only', { encoding: 'utf-8' })
    return output.trim().split('\n').filter(Boolean)
  } catch {
    return []
  }
}

const checkVersionChanged = () => {
  try {
    const output = execSync('git diff HEAD~1 HEAD -- package.json', { encoding: 'utf-8' })
    return output.includes('"version"')
  } catch {
    return false
  }
}

const analyzeChangeType = (files) => {
  let hasNewFeatures = false
  let hasBugFixes = false
  let hasBreakingChanges = false

  files.forEach((file) => {
    if (file.startsWith('static/src/views/') || file.startsWith('server/internal/handler/')) {
      if (file.includes('new') || file.includes('add')) {
        hasNewFeatures = true
      }
    }
    if (file.includes('fix') || file.includes('bug')) {
      hasBugFixes = true
    }
    if (file.includes('server/migrations/')) {
      hasBreakingChanges = true
    }
  })

  if (hasBreakingChanges) return 'MAJOR'
  if (hasNewFeatures) return 'MINOR'
  if (hasBugFixes) return 'PATCH'
  return 'NONE'
}

const main = () => {
  const changedFiles = getChangedFiles()
  const stagedFiles = getStagedFiles()
  const allFiles = [...new Set([...changedFiles, ...stagedFiles])]

  if (allFiles.length === 0) {
    console.log('ℹ️  没有检测到文件变更')
    process.exit(0)
  }

  const relevantFiles = allFiles.filter(
    (file) => file.startsWith('static/src/') || file.startsWith('server/internal/')
  )

  if (relevantFiles.length === 0) {
    console.log('ℹ️  没有检测到源代码变更')
    process.exit(0)
  }

  const versionChanged = checkVersionChanged()
  const changeType = analyzeChangeType(relevantFiles)

  if (changeType === 'NONE') {
    console.log('ℹ️  检测到的变更不需要更新版本号')
    process.exit(0)
  }

  if (!versionChanged) {
    console.error('⚠️  检测到源代码变更，但版本号未更新！')
    console.error(`📋 变更类型: ${changeType}`)
    console.error('📝 请根据版本号规范更新 package.json 中的版本号：')
    console.error(`   npm version ${changeType.toLowerCase()}`)
    console.error('')
    console.error('📚 版本号规范：docs/standards/versioning.md')
    process.exit(1)
  }

  console.log('✅ 版本号已正确更新')
}

main()
