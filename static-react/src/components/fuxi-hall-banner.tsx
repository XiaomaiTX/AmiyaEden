import type { ReactNode } from 'react'
import { SYSTEM_IDENTITY } from '@/constants/system-identity'
import insignia from '@/assets/fuxi-hall/legion.png'
import '@/assets/fuxi-hall/hall.css'

export function FuxiHallBanner({
  title,
  subtitle,
  preview = false,
  children,
}: {
  title: string
  subtitle?: string
  preview?: boolean
  children?: ReactNode
}) {
  const Heading = preview ? 'h3' : 'h1'
  return (
    <header className={preview ? 'fx-banner fx-banner--preview' : 'fx-banner'}>
      <div className="fx-banner__scene">
        <div className="fx-banner__signature">
          <img src={insignia} width="30" height="30" alt="" />
          {SYSTEM_IDENTITY.displayName}
        </div>
        <Heading className="fx-banner__title">{title}</Heading>
        {subtitle && <p className="fx-banner__subtitle">{subtitle}</p>}
      </div>
      {children}
    </header>
  )
}
