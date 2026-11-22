import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { RouterProvider } from 'react-router-dom'
import { ThemeProvider } from '@/components/theme-provider'
import { router } from '@/app/router'
import { ensureLocaleLoaded, I18nProvider } from '@/i18n'
import { SessionBootstrap } from '@/auth'
import { BadgeBootstrap } from '@/components/badge-bootstrap'
import { ContactConflictDialog } from '@/components/contact-conflict-dialog'
import { usePreferenceStore } from '@/stores'
import './index.css'

async function bootstrap() {
  const initialLocale = usePreferenceStore.getState().locale
  await ensureLocaleLoaded(initialLocale)

  createRoot(document.getElementById('root')!).render(
    <StrictMode>
      <ThemeProvider>
        <I18nProvider>
          <SessionBootstrap />
          <BadgeBootstrap />
          <ContactConflictDialog />
          <RouterProvider router={router} />
        </I18nProvider>
      </ThemeProvider>
    </StrictMode>
  )
}

void bootstrap()
