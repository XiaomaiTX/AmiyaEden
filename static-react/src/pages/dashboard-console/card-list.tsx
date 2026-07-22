import { useI18n } from '@/i18n'
import { formatIskSmart } from '@/lib/isk'
import { humanizeNumber } from '@/lib/format'
import type { DashboardCards } from '@/types/api/dashboard'

interface DashboardCardListProps {
  cards?: DashboardCards
}

interface CardItem {
  key: string
  label: string
  value: number
  decimals: number
  description: string
}

const numberFormatter = new Intl.NumberFormat('en-US', {
  minimumFractionDigits: 0,
  maximumFractionDigits: 2,
})

function formatCardValue(value: number, decimals: number): string {
  return numberFormatter.format(Number(value.toFixed(decimals)))
}

export function DashboardCardList({ cards }: DashboardCardListProps) {
  const { t } = useI18n()

  const items: CardItem[] = [
    {
      key: 'eveWallet',
      label: t('dashboardConsole.cards.eveWalletBalance'),
      value: cards?.eve_wallet_balance ?? 0,
      decimals: 2,
      description: formatIskSmart(cards?.eve_wallet_balance ?? 0),
    },
    {
      key: 'skillPoints',
      label: t('dashboardConsole.cards.eveSkillPoints'),
      value: cards?.eve_skill_points ?? 0,
      decimals: 0,
      description: humanizeNumber(cards?.eve_skill_points ?? 0),
    },
    {
      key: 'systemWallet',
      label: t('dashboardConsole.cards.systemWallet'),
      value: cards?.system_wallet_balance ?? 0,
      decimals: 2,
      description: '',
    },
    {
      key: 'alliancePap',
      label: t('dashboardConsole.cards.currentAlliancePap'),
      value: cards?.alliance_pap ?? 0,
      decimals: 1,
      description: '',
    },
  ]

  return (
    <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
      {items.map((item) => (
        <article
          key={item.key}
          className="flex flex-col justify-center rounded-lg border bg-card p-5"
        >
          <p className="text-sm text-muted-foreground">{item.label}</p>
          <p className="mt-2 text-2xl font-semibold">
            {formatCardValue(item.value, item.decimals)}
          </p>
          {item.description ? (
            <p className="mt-1 text-xs text-muted-foreground">{item.description}</p>
          ) : null}
        </article>
      ))}
    </div>
  )
}
