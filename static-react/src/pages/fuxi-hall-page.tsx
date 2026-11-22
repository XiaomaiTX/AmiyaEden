import { lazy, Suspense } from 'react'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { useRole } from '@/hooks/use-permission'
import { useI18n } from '@/i18n'
import { FuxiHallPublicPage } from '@/pages/fuxi-hall-public-page'

const FuxiHallManagePage = lazy(() =>
  import('@/pages/fuxi-hall-manage-page').then((module) => ({ default: module.FuxiHallManagePage }))
)

export function FuxiHallPage() {
  const { t } = useI18n()
  const canManage = useRole(['super_admin', 'admin'])

  if (!canManage) return <FuxiHallPublicPage />

  return (
    <Tabs defaultSelectedKey="fame" className="min-w-0">
      <TabsList aria-label={t('nav.fuxiHall.fame')} className="mx-4 mt-4 md:mx-6" variant="line">
        <TabsTrigger id="fame">{t('nav.fuxiHall.fame')}</TabsTrigger>
        <TabsTrigger id="manage">{t('nav.fuxiHall.manage')}</TabsTrigger>
      </TabsList>
      <TabsContent id="fame">
        <FuxiHallPublicPage />
      </TabsContent>
      <TabsContent id="manage">
        <Suspense fallback={<p className="p-4">{t('common.loading')}</p>}>
          <FuxiHallManagePage />
        </Suspense>
      </TabsContent>
    </Tabs>
  )
}
