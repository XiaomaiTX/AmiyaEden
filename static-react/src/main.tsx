import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { RouterProvider } from 'react-router-dom'
import { ThemeProvider } from '@/components/theme-provider'
import { TooltipProvider } from '@/components/ui/tooltip'
import { router } from '@/app/router'
import { I18nProvider } from '@/i18n'
import './index.css'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <ThemeProvider>
      <TooltipProvider>
        <I18nProvider>
          <RouterProvider router={router} />
        </I18nProvider>
      </TooltipProvider>
    </ThemeProvider>
  </StrictMode>
)
