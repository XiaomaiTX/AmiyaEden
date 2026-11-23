import { useState, type ReactNode } from 'react'
import { Copy } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { buildEveCharacterPortraitUrl } from '@/lib/eve-image'
import { useI18n } from '@/i18n'
import type { FuxiHallCard } from '@/types/api/fuxi-hall'

const metrics = ['corp_pap', 'fleets', 'welfare', 'srp', 'recruitment', 'mentoring'] as const
const metricColors = {
  corp_pap: 'bg-amber-500/10 text-amber-700 dark:text-amber-300 ring-amber-500/20',
  fleets: 'bg-sky-500/10 text-sky-700 dark:text-sky-300 ring-sky-500/20',
  welfare: 'bg-emerald-500/10 text-emerald-700 dark:text-emerald-300 ring-emerald-500/20',
  srp: 'bg-violet-500/10 text-violet-700 dark:text-violet-300 ring-violet-500/20',
  recruitment: 'bg-cyan-500/10 text-cyan-700 dark:text-cyan-300 ring-cyan-500/20',
  mentoring: 'bg-rose-500/10 text-rose-700 dark:text-rose-300 ring-rose-500/20',
}

export function FuxiHallMemberCard({ card, action }: { card: FuxiHallCard; action?: ReactNode }) {
  const { t } = useI18n()
  // Feedback is shown next to whichever value was copied.
  const [copyNotice, setCopyNotice] = useState<{ target: 'name' | 'qq'; text: string } | null>(null)
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
  const avatarClass =
    card.avatar_shape === 'circle'
      ? 'rounded-full'
      : card.avatar_shape === 'rounded'
        ? 'rounded-xl'
        : 'rounded-md'
  return (
    <article
      className="flex h-full min-w-0 flex-col overflow-hidden rounded-xl border border-border bg-card"
      style={{ fontSize: `${card.font_scale}px`, borderTop: `2px solid ${card.accent_color}` }}
    >
      <div className="flex-1 space-y-3 p-4">
        <div className="flex items-start gap-3">
          {card.main_character_id > 0 && (
            <img
              loading="lazy"
              className={`h-12 w-12 shrink-0 object-cover ${avatarClass}`}
              src={buildEveCharacterPortraitUrl(card.main_character_id, 256)}
              alt={card.nickname}
            />
          )}
          <div className="min-w-0 flex-1">
            <h3
              className="truncate font-semibold leading-snug tracking-tight"
              style={{ fontSize: `${card.font_scale + 4}px` }}
              title={card.nickname}
            >
              {card.nickname}
            </h3>
            <div className="mt-1 flex min-w-0 items-center gap-1 text-xs text-muted-foreground">
              <span className="truncate" title={card.main_character_name}>
                {card.main_character_name}
              </span>
              {card.main_character_name && (
                <Button
                  variant="ghost"
                  size="icon"
                  className="h-5 w-5 shrink-0"
                  aria-label={t('fuxiHall.fame.copyCharacterName')}
                  onClick={() => void copy('name', card.main_character_name)}
                >
                  <Copy className="size-3" />
                </Button>
              )}
              {copyNotice?.target === 'name' && (
                <span role="status" className="shrink-0">
                  {copyNotice.text}
                </span>
              )}
            </div>
          </div>
          {card.is_current_management && (
            <p className="inline-flex shrink-0 items-center gap-1.5 whitespace-nowrap rounded-full bg-green-500/10 px-2 py-0.5 text-xs font-medium text-green-700 dark:text-green-400">
              <span aria-hidden="true" className="h-2 w-2 rounded-full bg-green-500" />
              {t('fuxiHall.fame.current')}
            </p>
          )}
        </div>
        {card.title_tags.length > 0 && (
          <div className="flex flex-wrap gap-1.5">
            {card.title_tags.map((tag) => (
              <span
                key={tag}
                className="rounded px-2 py-0.5 text-xs font-medium"
                style={{ color: card.accent_color, backgroundColor: `${card.accent_color}12` }}
              >
                {tag}
              </span>
            ))}
          </div>
        )}
        <p className="text-xs leading-relaxed text-muted-foreground">
          {t('fuxiHall.fame.tenure')} ·{' '}
          {card.tenure_months == null
            ? t('fuxiHall.fame.unknown')
            : t('fuxiHall.fame.duration', {
                years: Math.floor(card.tenure_months / 12),
                months: card.tenure_months % 12,
              })}
        </p>
        {qq && (
          <div className="flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
            <span>
              {t('fuxiHall.fame.qq')}:{' '}
              <span className="font-medium tabular-nums text-foreground">{qq}</span>
            </span>
            <Button
              variant="ghost"
              size="icon"
              className="h-6 w-6"
              aria-label={t('fuxiHall.fame.copyQQ')}
              onClick={() => void copy('qq', qq)}
            >
              <Copy className="size-3.5" />
            </Button>
            {copyNotice?.target === 'qq' && <span role="status">{copyNotice.text}</span>}
          </div>
        )}
        {card.description_html && (
          <div
            className="break-words leading-relaxed text-foreground/80 [overflow-wrap:anywhere] [&_a]:text-primary [&_a]:underline [&_img]:max-w-full"
            dangerouslySetInnerHTML={{ __html: card.description_html }}
          />
        )}
      </div>
      {visibleMetrics.length > 0 && (
        <div
          className="border-t border-border/60 px-4 py-2"
          title={
            card.contributions_updated_at
              ? t('fuxiHall.contributions.updated', {
                  date: new Date(card.contributions_updated_at).toLocaleDateString(),
                })
              : t('fuxiHall.contributions.pending')
          }
        >
          <dl aria-label={t('fuxiHall.contributions.title')} className="flex flex-wrap gap-1.5">
            {visibleMetrics.map((key) => {
              const value = card.contributions?.[key] ?? 0
              return (
                <div
                  key={key}
                  title={t(`fuxiHall.contributions.help.${key}`)}
                  className={`inline-flex items-baseline gap-1 rounded px-1.5 py-0.5 text-xs leading-4 ring-1 ring-inset ${metricColors[key]}`}
                >
                  <dt className="text-xs font-medium leading-4">{t(`fuxiHall.contributions.${key}`)}</dt>
                  <dd className="text-xs font-semibold leading-4 tabular-nums">
                    {value.toLocaleString(undefined, { maximumFractionDigits: 0 })}
                  </dd>
                </div>
              )
            })}
          </dl>
        </div>
      )}
      {action && <div className="px-4 pb-4">{action}</div>}
    </article>
  )
}
