import { buildEveCharacterPortraitUrl } from '@/lib/eve-image'
import { useI18n } from '@/i18n'
import type { FuxiHallCard } from '@/types/api/fuxi-hall'

const metrics = ['fleets', 'welfare', 'srp', 'recruitment', 'mentoring'] as const

export function FuxiHallMemberCard({ card }: { card: FuxiHallCard }) {
  const { t } = useI18n()
  const visibleMetrics = metrics.filter((key) => (card.contributions?.[key] ?? 0) > 0)
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
      <div className="flex-1 space-y-4 p-5">
        <div className="flex items-start gap-3">
          {card.main_character_id > 0 && (
            <img
              loading="lazy"
              className={`h-14 w-14 shrink-0 object-cover ${avatarClass}`}
              src={buildEveCharacterPortraitUrl(card.main_character_id, 256)}
              alt={card.nickname}
            />
          )}
          <div className="min-w-0 flex-1">
            <h3
              className="break-words font-semibold leading-snug tracking-tight"
              style={{ fontSize: `${card.font_scale + 4}px` }}
            >
              {card.nickname}
            </h3>
            <p className="mt-1 break-words text-xs text-muted-foreground">
              {card.main_character_name}
            </p>
            {card.is_current_management && (
              <p className="mt-2 inline-flex items-center gap-1.5 text-xs font-medium text-green-700 dark:text-green-400">
                <span aria-hidden="true" className="h-2 w-2 rounded-full bg-green-500" />
                {t('fuxiHall.fame.current')}
              </p>
            )}
          </div>
        </div>
        {card.title_tags.length > 0 && (
          <div className="flex flex-wrap gap-1.5">
            {card.title_tags.map((tag) => (
              <span
                key={tag}
                className="rounded px-2 py-1 text-xs font-medium"
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
        {card.description_html && (
          <div
            className="break-words leading-relaxed text-foreground/80 [overflow-wrap:anywhere] [&_a]:text-primary [&_a]:underline [&_img]:max-w-full"
            dangerouslySetInnerHTML={{ __html: card.description_html }}
          />
        )}
      </div>
      {visibleMetrics.length > 0 && (
        <div
          className="border-t border-border bg-muted/20 px-5 py-3"
          title={
            card.contributions_updated_at
              ? t('fuxiHall.contributions.updated', {
                  date: new Date(card.contributions_updated_at).toLocaleDateString(),
                })
              : t('fuxiHall.contributions.pending')
          }
        >
          <dl
            aria-label={t('fuxiHall.contributions.title')}
            className="flex flex-wrap gap-x-5 gap-y-2"
          >
            {visibleMetrics.map((key) => {
              const value = card.contributions?.[key] ?? 0
              return (
                <div
                  key={key}
                  title={t(`fuxiHall.contributions.help.${key}`)}
                  className="inline-flex items-baseline gap-1.5"
                >
                  <dt className="text-xs text-muted-foreground">
                    {t(`fuxiHall.contributions.${key}`)}
                  </dt>
                  <dd className="text-sm font-semibold tabular-nums">{value.toLocaleString()}</dd>
                </div>
              )
            })}
          </dl>
        </div>
      )}
    </article>
  )
}
