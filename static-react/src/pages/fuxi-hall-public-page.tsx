import { useEffect, useState } from 'react'
import { fetchFuxiHallContributors, fetchFuxiHallLeadership } from '@/api/fuxi-hall'
import { FuxiHallMemberCard } from '@/components/fuxi-hall-member-card'
import { useI18n } from '@/i18n'
import type { FuxiHallPublicPageResponse } from '@/types/api/fuxi-hall'

export function FuxiHallPublicPage({ page }: { page: 'leadership' | 'contributors' }) {
  const { t } = useI18n()
  const [data, setData] = useState<FuxiHallPublicPageResponse | null>(null)
  const [error, setError] = useState<string | null>(null)
  useEffect(() => {
    void (async () => {
      try {
        setError(null)
        setData(
          await (page === 'leadership' ? fetchFuxiHallLeadership() : fetchFuxiHallContributors())
        )
      } catch (caught) {
        setError(caught instanceof Error ? caught.message : t('fuxiHall.public.loadFailed'))
      }
    })()
  }, [page, t])
  const eyebrow =
    page === 'leadership'
      ? t('fuxiHall.public.leadershipEyebrow')
      : t('fuxiHall.public.contributorsEyebrow')
  const fallbackTitle =
    page === 'leadership'
      ? t('fuxiHall.public.defaultLeadershipTitle')
      : t('fuxiHall.public.defaultContributorsTitle')
  return (
    <section className="min-h-full space-y-5 rounded-xl bg-[radial-gradient(circle_at_top_left,rgba(59,130,246,.16),transparent_32%),radial-gradient(circle_at_top_right,rgba(2,132,199,.12),transparent_30%)] p-4 md:p-6">
      {error ? <p className="text-sm text-destructive">{error}</p> : null}
      {!data && !error ? (
        <p className="text-sm text-muted-foreground">{t('common.loading')}</p>
      ) : null}
      {data ? (
        <div className="rounded-xl border bg-card p-5">
          <header>
            <p className="text-xs tracking-widest text-muted-foreground uppercase">{eyebrow}</p>
            <h1 className="mt-2 text-3xl font-semibold">{data.page.title || fallbackTitle}</h1>
            {data.page.subtitle ? (
              <p className="mt-2 text-muted-foreground">{data.page.subtitle}</p>
            ) : null}
          </header>
          {data.page.description_html ? (
            <article
              className="mt-5 rounded-xl bg-muted/50 p-4 leading-relaxed [&_a]:text-primary [&_a]:underline [&_img]:max-w-full"
              dangerouslySetInnerHTML={{ __html: data.page.description_html }}
            />
          ) : null}
          {data.cards.length ? (
            <div className="mt-5 grid gap-4 md:grid-cols-2 xl:grid-cols-3">
              {data.cards.map((card) => (
                <FuxiHallMemberCard key={card.id} card={card} />
              ))}
            </div>
          ) : (
            <div className="mt-5 rounded-xl border border-dashed p-8 text-center">
              <h2 className="font-medium">{t('fuxiHall.public.emptyTitle')}</h2>
              <p className="mt-1 text-sm text-muted-foreground">
                {t('fuxiHall.public.emptySubtitle')}
              </p>
            </div>
          )}
        </div>
      ) : null}
    </section>
  )
}

export const FuxiHallLeadershipPage = () => <FuxiHallPublicPage page="leadership" />
export const FuxiHallContributorsPage = () => <FuxiHallPublicPage page="contributors" />
