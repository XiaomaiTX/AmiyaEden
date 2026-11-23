import { useState, type CSSProperties, type ReactNode } from 'react'
import { Copy, Timer } from 'lucide-react'
import { Button } from '@/components/ui/button'
import '@/assets/fuxi-hall/hall.css'
import { buildEveCharacterPortraitUrl } from '@/lib/eve-image'
import { useI18n } from '@/i18n'
import type { FuxiHallCard } from '@/types/api/fuxi-hall'

const metrics = ['corp_pap', 'fleets', 'welfare', 'srp', 'recruitment', 'mentoring'] as const

export function FuxiHallMemberCard({ card, action }: { card: FuxiHallCard; action?: ReactNode }) {
  const { t } = useI18n()
  // Feedback is shown next to whichever value was copied.
  const [copyNotice, setCopyNotice] = useState<{ target: 'name' | 'qq'; text: string } | null>(null)
  const [failedPortrait, setFailedPortrait] = useState<number | null>(null)
  const qq = card.qq?.trim() || ''
  const copy = async (target: 'name' | 'qq', text: string) => {
    try {
      await navigator.clipboard.writeText(text)
      setCopyNotice({ target, text: t('common.copied') })
    } catch {
      setCopyNotice({ target, text: t('common.copyFailed') })
    }
  }
  // Values render without decimals, so filter on the rounded value to never show a "0" tag.
  const visibleMetrics = metrics.filter((key) => Math.round(card.contributions?.[key] ?? 0) > 0)
  const style = {
    '--fx-accent': card.accent_color,
    '--fx-size': `${card.font_scale}px`,
  } as CSSProperties
  return (
    <article className="fx-card" style={style}>
      <header className="fx-card__head">
        <div className={`fx-portrait fx-portrait--${card.avatar_shape}`}>
          {card.main_character_id > 0 && failedPortrait !== card.main_character_id && (
            <img
              loading="lazy"
              src={buildEveCharacterPortraitUrl(card.main_character_id, 256)}
              alt={card.nickname}
              onError={() => setFailedPortrait(card.main_character_id)}
            />
          )}
        </div>
        <div className="fx-identity">
          {card.is_current_management && (
            <span className="fx-current">{t('fuxiHall.fame.current')}</span>
          )}
          <h3 title={card.nickname}>{card.nickname}</h3>
          <div className="fx-character">
            <span title={card.main_character_name}>{card.main_character_name}</span>
            {card.main_character_name && (
              <Button
                type="button"
                variant="ghost"
                size="icon"
                className="fx-copy"
                aria-label={t('fuxiHall.fame.copyCharacterName')}
                onClick={() => void copy('name', card.main_character_name)}
              >
                <Copy className="size-3.5" />
              </Button>
            )}
          </div>
          {copyNotice?.target === 'name' && (
            <span className="fx-copy-status" role="status">
              {copyNotice.text}
            </span>
          )}
        </div>
      </header>
      <div className="fx-card__body">
        {card.title_tags.length > 0 && (
          <div className="fx-roles">
            {card.title_tags.map((tag) => (
              <span key={tag}>{tag}</span>
            ))}
          </div>
        )}
        <div className="fx-record">
          <Timer size={16} aria-hidden="true" />
          <span>{t('fuxiHall.fame.tenure')}</span>
          <strong>
            {card.tenure_months == null
              ? t('fuxiHall.fame.unknown')
              : t('fuxiHall.fame.duration', {
                  years: Math.floor(card.tenure_months / 12),
                  months: card.tenure_months % 12,
                })}
          </strong>
        </div>
        {qq && (
          <div className="fx-contact">
            <span>{t('fuxiHall.fame.qq')}</span>
            <strong>{qq}</strong>
            <Button
              type="button"
              variant="ghost"
              size="icon"
              className="fx-copy"
              aria-label={t('fuxiHall.fame.copyQQ')}
              onClick={() => void copy('qq', qq)}
            >
              <Copy className="size-3.5" />
            </Button>
            {copyNotice?.target === 'qq' && <span role="status">{copyNotice.text}</span>}
          </div>
        )}
        {card.description_html && (
          <div className="fx-bio" dangerouslySetInnerHTML={{ __html: card.description_html }} />
        )}
      </div>
      {visibleMetrics.length > 0 && (
        <div
          className="fx-contributions"
          title={
            card.contributions_updated_at
              ? t('fuxiHall.contributions.updated', {
                  date: new Date(card.contributions_updated_at).toLocaleDateString(),
                })
              : t('fuxiHall.contributions.pending')
          }
        >
          <dl aria-label={t('fuxiHall.contributions.title')}>
            {visibleMetrics.map((key) => (
              <div
                key={key}
                className={`fx-metric--${key}`}
                title={t(`fuxiHall.contributions.help.${key}`)}
              >
                <dt>{t(`fuxiHall.contributions.${key}`)}</dt>
                <dd>
                  {card.contributions?.[key]?.toLocaleString(undefined, {
                    maximumFractionDigits: 0,
                  })}
                </dd>
              </div>
            ))}
          </dl>
        </div>
      )}
      {action && <div className="fx-card__action">{action}</div>}
    </article>
  )
}
