import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { RouterProvider } from 'react-router-dom'
import { ThemeProvider } from '@/components/theme-provider'
import { router } from '@/app/router'
import { I18nProvider } from '@/i18n'
import { SessionBootstrap } from '@/auth'
import { BadgeBootstrap } from '@/components/badge-bootstrap'
import { ContactConflictDialog } from '@/components/contact-conflict-dialog'
import './index.css'

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
