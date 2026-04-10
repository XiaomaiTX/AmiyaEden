# 版本号显示功能实施方案

**创建日期**: 2026-04-10
**最后更新**: 2026-04-10
**状态**: Completed
**预计工期**: 0.25 天
**实际工期**: 0.25 天
**优先级**: Medium

**版本号规范**: 本实施遵循 [版本号管理规范](../../standards/versioning.md)，前后端统一使用前端 `package.json` 中的版本号。

---

## 需求背景

IT 运维人员需要快速查看当前部署的版本信息，以便：
- 确认部署的版本是否正确
- 快速定位问题发生的版本
- 便于版本回滚和问题排查
- 提升运维效率

## 功能要求

### 核心功能

1. **统一版本号**
   - 从 `static/package.json` 自动读取版本号
   - 在顶部栏显示，格式：`v{version}`
   - 构建时自动同步到环境变量

2. **展示方式**
   - 顶部栏右侧显示，位于用户菜单之前
   - 显示格式：`v{version}`
   - 移动端自动隐藏（≤768px）

3. **技术要求**
   - 遵循项目现有架构模式
   - 前端使用 Vue 3 Composition API
   - 无需后端改动
   - 版本信息通过环境变量获取

---

## 实施方案

### 前端实施

#### 1. 创建 `static/scripts/sync-version.js`

创建版本同步脚本，构建前自动执行：

```javascript
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
```

#### 2. 修改 `static/package.json`

添加构建前钩子：

```json
{
  "scripts": {
    "dev": "vite --open",
    "prebuild": "node scripts/sync-version.js",
    "build": "vite build",
    "serve": "vite preview"
  }
}
```

#### 3. 修改 `static/.env.development`

添加版本号环境变量（会被脚本自动更新）：

```env
# 应用版本号（从 package.json 读取）
VITE_VERSION = 0.0.0
```

#### 4. 修改 `static/.env.production`

添加版本号环境变量（会被脚本自动更新）：

```env
# 应用版本号（从 package.json 读取）
VITE_VERSION = 0.0.0
```

#### 5. 修改 `static/vite.config.ts`

确保环境变量类型定义：

```typescript
export default defineConfig(({ mode }) => {
  return {
    define: {
      __APP_VERSION__: JSON.stringify(process.env.VITE_VERSION),
    },
    // ... 其他配置
  }
})
```

#### 6. 修改 `static/src/env.d.ts`

添加环境变量类型定义：

```typescript
declare const __APP_VERSION__: string
```

#### 7. 创建 `static/src/components/core/layouts/art-header-bar/widget/ArtVersionDisplay.vue`

创建版本显示组件：

```vue
<template>
  <div class="version-display">
    <div class="version-info">
      <span class="version-label">v</span>
      <span class="version-text">{{ version }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
const version = __APP_VERSION__
</script>

<style lang="scss" scoped>
.version-display {
  display: flex;
  align-items: center;
  font-size: 12px;
  color: var(--art-text-color-3);
  padding: 4px 8px;

  .version-info {
    display: flex;
    align-items: center;
    gap: 2px;

    .version-label {
      font-weight: 600;
      color: var(--art-primary-color);
    }

    .version-text {
      font-family: 'Courier New', monospace;
      font-weight: 500;
    }
  }
}

@media screen and (max-width: 768px) {
  .version-display {
    display: none;
  }
}
</style>
```

#### 8. 修改 `static/src/components/core/layouts/art-header-bar/index.vue`

在顶部栏集成版本显示组件：

```vue
<script setup lang="ts">
import { useHeaderBar } from '@/hooks/core/useHeaderBar'
import { fetchUnreadCount } from '@/api/notification'
import ArtUserMenu from './widget/ArtUserMenu.vue'
import ArtVersionDisplay from './widget/ArtVersionDisplay.vue'
// ...
</script>

<template>
  <div class="art-header-bar">
    <!-- 其他内容... -->

    <!-- 版本信息显示 -->
    <ArtVersionDisplay />

    <!-- 用户头像、菜单 -->
    <ArtUserMenu />
  </div>
</template>
```

---

## 实施步骤

### 阶段一：脚本和环境配置（0.1 天）

- [x] 创建 `static/scripts/sync-version.js`
- [x] 修改 `static/package.json`，添加 prebuild 钩子
- [x] 修改环境变量文件（`.env.development`、`.env.production`）
- [x] 测试版本同步脚本

### 阶段二：配置类型定义（0.05 天）

- [x] 修改 `static/vite.config.ts`，添加环境变量定义
- [x] 修改 `static/src/env.d.ts`，添加类型定义

### 阶段三：组件开发（0.1 天）

- [x] 创建 `ArtVersionDisplay.vue` 组件
- [x] 修改顶部栏组件，集成版本显示
- [x] 测试版本显示和悬停效果

---

## 构建命令

### 开发环境

```bash
cd static
pnpm dev
```

### 生产环境

```bash
cd static
pnpm build
```

### Docker 构建

```bash
# 在 docker build 过程中，版本同步脚本会自动执行
docker build -t amiya-eden:latest .
```

---

## 验收标准

### 功能验收

- [x] 顶部栏正确显示版本号（格式：`v{version}`）
- [x] 版本号与 `package.json` 一致
- [x] 移动端（≤768px）版本显示自动隐藏

### 技术验收

- [x] 前端使用 Vue 3 Composition API
- [x] 版本同步脚本正确工作
- [x] 构建流程正确注入版本信息
- [x] 代码符合项目编码规范
- [x] 无 TypeScript 类型错误
- [x] 无 ESLint 警告

---

## 风险评估

| 风险项 | 影响 | 概率 | 缓解措施 |
|--------|------|------|----------|
| Git 命令执行失败 | 低 | 低 | 捕获错误并提供默认值 |
| 版本号不一致 | 中 | 低 | 自动化同步脚本，构建前验证 |
| UI 布局影响 | 低 | 低 | 使用最小化样式，不影响现有布局 |
| 环境变量注入失败 | 低 | 低 | 提供 fallback 默认值 |

---

## 相关文件清单

### 前端文件

- `static/scripts/sync-version.js`
- `static/package.json`
- `static/.env.development`
- `static/.env.production`
- `static/vite.config.ts`
- `static/src/env.d.ts`
- `static/src/components/core/layouts/art-header-bar/widget/ArtVersionDisplay.vue`
- `static/src/components/core/layouts/art-header-bar/index.vue`

---

## 后续优化

- [ ] 添加版本更新检查功能
- [ ] 支持版本回滚功能
- [ ] 集成发布日志展示
- [ ] 添加版本兼容性检查
- [ ] 支持多环境版本标识（dev/test/prod）
