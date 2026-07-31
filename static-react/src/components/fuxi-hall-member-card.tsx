import { buildEveCharacterPortraitUrl } from '@/lib/eve-image'
import { useI18n } from '@/i18n'
import type { FuxiHallCard } from '@/types/api/fuxi-hall'

export function FuxiHallMemberCard({
  card,
  showStats = false,
}: {
  card: FuxiHallCard
  showStats?: boolean
}) {
  const { t } = useI18n()
  const avatarClass =
    card.avatar_shape === 'circle'
      ? 'rounded-full'
      : card.avatar_shape === 'rounded'
        ? 'rounded-xl'
        : 'rounded-md'
  const style = { borderColor: `${card.accent_color}50`, fontSize: `${card.font_scale}px` }
  return (
    <article className="min-w-0 rounded-2xl border bg-card" style={style}>
      <div className="p-4">
        <div className="flex gap-3">
          {card.main_character_id > 0 ? (
            <img
              className={`h-[72px] w-[72px] shrink-0 border-2 object-cover ${avatarClass}`}
              style={{ borderColor: `${card.accent_color}80` }}
              src={buildEveCharacterPortraitUrl(card.main_character_id, 256)}
              alt={card.nickname}
            />
          ) : null}
          <div className="min-w-0">
            <h3 className="font-semibold" style={{ fontSize: `${card.font_scale + 3}px` }}>
              {card.nickname}
            </h3>
            <p className="mt-1 italic text-muted-foreground">{card.main_character_name}</p>
            <div className="mt-2 flex flex-wrap gap-1">
              {card.title_tags.map((tag) => (
                <span
                  key={`${card.id}-${tag}`}
                  className="rounded-full border px-2 py-0.5 text-xs"
                  style={{ color: card.accent_color, borderColor: card.accent_color }}
                >
                  {tag}
                </span>
              ))}
            </div>
          </div>
        </div>
        {card.description_html ? (
          <div
            className="mt-3 leading-relaxed text-muted-foreground [&_a]:text-primary [&_a]:underline [&_img]:max-w-full"
            dangerouslySetInnerHTML={{ __html: card.description_html }}
          />
        ) : null}
        {showStats &&
        ((card.fleet_led_count ?? 0) > 0 || (card.welfare_delivery_count ?? 0) > 0) ? (
          <div className="mt-3 space-y-1 text-sm">
            {(card.fleet_led_count ?? 0) > 0 ? (
              <p>
                {t('fuxiHall.public.fleetLedCount')}: {card.fleet_led_count}
              </p>
            ) : null}
            {(card.welfare_delivery_count ?? 0) > 0 ? (
              <p>
                {t('fuxiHall.public.welfareDeliveryCount')}: {card.welfare_delivery_count}
              </p>
            ) : null}
          </div>
        ) : null}
      </div>
    </article>
  )
}
