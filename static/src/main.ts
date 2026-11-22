import App from './App.vue'
import { createApp } from 'vue'
import { initStore } from './store'                 // Store
import { initRouter } from './router'               // Router
import language, { defaultLanguage, loadLocaleMessages } from './locales' // 国际化
import '@styles/core/tailwind.css'                  // tailwind
import '@styles/index.scss'                         // 样式
import '@utils/sys/console.ts'                      // 控制台输出内容
import './utils/ui/iconify-loader'                  // 本地离线图标集
import { setupGlobDirectives } from './directives'
import { setupErrorHandle } from './utils/sys/error-handle'
import { useSysConfigStore } from './store/modules/sys-config'

async function bootstrap() {
  try {
    await loadLocaleMessages(defaultLanguage)
  } catch (error) {
    console.error('[i18n] 加载初始语言包失败:', error)
  }

  const app = createApp(App)
  initStore(app)
  initRouter(app)
  setupGlobDirectives(app)
  setupErrorHandle(app)

  const sysConfigStore = useSysConfigStore()
  sysConfigStore.ensureLoaded()

  app.use(language)
  app.mount('#app')
}

void bootstrap()
