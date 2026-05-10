import { useMemo, useState } from 'react'
import { buildEveCharacterPortraitUrl } from '@/lib/eve-image'
import { notifySuccess } from '@/feedback'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { useI18n } from '@/i18n'
import { useSessionStore } from '@/stores'

type ProfileFormState = {
  nickname: string
  realName: string
  email: string
  mobile: string
  address: string
  sex: string
  description: string
}

type PasswordFormState = {
  currentPassword: string
  newPassword: string
  confirmPassword: string
}

const defaultProfileForm: ProfileFormState = {
  nickname: '',
  realName: '',
  email: '',
  mobile: '',
  address: '',
  sex: '2',
  description: '',
}

const defaultPasswordForm: PasswordFormState = {
  currentPassword: '',
  newPassword: '',
  confirmPassword: '',
}

export function SystemUserCenterPage() {
  const { t } = useI18n()
  const characterName = useSessionStore((state) => state.characterName)
  const characterId = useSessionStore((state) => state.characterId)
  const roles = useSessionStore((state) => state.roles)

  const [profileEditing, setProfileEditing] = useState(false)
  const [passwordEditing, setPasswordEditing] = useState(false)
  const [profileForm, setProfileForm] = useState<ProfileFormState>({
    ...defaultProfileForm,
    nickname: characterName ?? '',
    realName: characterName ?? '',
    email: '',
    mobile: '',
    address: '',
    sex: '2',
    description: '',
  })
  const [passwordForm, setPasswordForm] = useState<PasswordFormState>(defaultPasswordForm)

  const profileTags = useMemo(
    () => [
      t('userCenter.tags.focus'),
      t('userCenter.tags.design'),
      t('userCenter.tags.chair'),
      t('userCenter.tags.team'),
    ],
    [t]
  )

  const toggleProfileEdit = () => {
    if (profileEditing) {
      notifySuccess(t('userCenter.profile.saveSuccess'))
    }
    setProfileEditing((current) => !current)
  }

  const togglePasswordEdit = () => {
    if (passwordEditing) {
      notifySuccess(t('userCenter.password.saveSuccess'))
    }
    setPasswordEditing((current) => !current)
  }

  return (
    <section className="grid gap-4 xl:grid-cols-[352px_minmax(0,1fr)]">
      <div className="overflow-hidden rounded-lg border bg-card">
        <div className="relative h-40 overflow-hidden bg-gradient-to-br from-sky-500 via-cyan-500 to-emerald-400">
          <div className="absolute inset-0 bg-[radial-gradient(circle_at_top_left,_rgba(255,255,255,0.32),_transparent_32%),radial-gradient(circle_at_bottom_right,_rgba(255,255,255,0.18),_transparent_28%)]" />
          {characterId ? (
            <img
              alt={characterName ?? t('userCenter.unnamed')}
              className="absolute left-1/2 top-10 h-24 w-24 -translate-x-1/2 rounded-full border-4 border-white object-cover shadow-lg"
              src={buildEveCharacterPortraitUrl(characterId, 128)}
            />
          ) : (
            <div className="absolute left-1/2 top-10 flex h-24 w-24 -translate-x-1/2 items-center justify-center rounded-full border-4 border-white bg-white/25 text-2xl font-semibold text-white shadow-lg">
              {characterName?.slice(0, 1)?.toUpperCase() ?? '?'}
            </div>
          )}
        </div>
        <div className="px-6 pb-6 pt-16 text-center">
          <h1 className="text-2xl font-semibold">{characterName ?? t('userCenter.unnamed')}</h1>
          <p className="mt-2 text-sm text-muted-foreground">{t('userCenter.subtitle')}</p>

          <div className="mt-6 flex flex-wrap justify-center gap-2">
            {profileTags.map((tag) => (
              <span
                key={tag}
                className="inline-flex rounded-full bg-muted px-3 py-1 text-xs font-medium text-muted-foreground"
              >
                {tag}
              </span>
            ))}
          </div>

          <div className="mt-6 space-y-3 text-left text-sm">
            <div className="rounded-lg border bg-muted/20 px-4 py-3">
              <div className="text-muted-foreground">{t('userCenter.contact.email')}</div>
              <div className="mt-1 font-medium">{profileForm.email || '-'}</div>
            </div>
            <div className="rounded-lg border bg-muted/20 px-4 py-3">
              <div className="text-muted-foreground">{t('userCenter.contact.title')}</div>
              <div className="mt-1 font-medium">{t('userCenter.contact.designer')}</div>
            </div>
            <div className="rounded-lg border bg-muted/20 px-4 py-3">
              <div className="text-muted-foreground">{t('common.role')}</div>
              <div className="mt-1 font-medium">{roles.length > 0 ? roles.join(', ') : '-'}</div>
            </div>
          </div>
        </div>
      </div>

      <div className="space-y-4">
        <div className="rounded-lg border bg-card p-5">
          <div className="flex flex-col gap-4 border-b pb-4 lg:flex-row lg:items-start lg:justify-between">
            <div>
              <h2 className="text-xl font-semibold">{t('userCenter.profile.title')}</h2>
              <p className="mt-1 text-sm text-muted-foreground">{t('userCenter.profile.subtitle')}</p>
            </div>
            <Button type="button" onClick={toggleProfileEdit}>
              {profileEditing ? t('userCenter.profile.save') : t('userCenter.profile.edit')}
            </Button>
          </div>

          <div className="mt-5 grid gap-4 md:grid-cols-2">
            <label className="space-y-2 md:col-span-2">
              <span className="text-sm text-muted-foreground">{t('userCenter.profile.nickname')}</span>
              <Input
                value={profileForm.nickname}
                disabled={!profileEditing}
                onChange={(event) =>
                  setProfileForm((current) => ({ ...current, nickname: event.target.value }))
                }
              />
            </label>
            <label className="space-y-2">
              <span className="text-sm text-muted-foreground">{t('userCenter.profile.realName')}</span>
              <Input
                value={profileForm.realName}
                disabled={!profileEditing}
                onChange={(event) =>
                  setProfileForm((current) => ({ ...current, realName: event.target.value }))
                }
              />
            </label>
            <label className="space-y-2">
              <span className="text-sm text-muted-foreground">{t('userCenter.profile.sex')}</span>
              <select
                className="h-10 w-full rounded-md border border-input bg-background px-3 text-sm disabled:cursor-not-allowed disabled:opacity-60"
                value={profileForm.sex}
                disabled={!profileEditing}
                onChange={(event) =>
                  setProfileForm((current) => ({ ...current, sex: event.target.value }))
                }
              >
                <option value="1">{t('userCenter.profile.sexMale')}</option>
                <option value="2">{t('userCenter.profile.sexFemale')}</option>
              </select>
            </label>
            <label className="space-y-2">
              <span className="text-sm text-muted-foreground">{t('userCenter.profile.email')}</span>
              <Input
                value={profileForm.email}
                disabled={!profileEditing}
                onChange={(event) =>
                  setProfileForm((current) => ({ ...current, email: event.target.value }))
                }
              />
            </label>
            <label className="space-y-2">
              <span className="text-sm text-muted-foreground">{t('userCenter.profile.mobile')}</span>
              <Input
                value={profileForm.mobile}
                disabled={!profileEditing}
                onChange={(event) =>
                  setProfileForm((current) => ({ ...current, mobile: event.target.value }))
                }
              />
            </label>
            <label className="space-y-2 md:col-span-2">
              <span className="text-sm text-muted-foreground">{t('userCenter.profile.address')}</span>
              <Input
                value={profileForm.address}
                disabled={!profileEditing}
                onChange={(event) =>
                  setProfileForm((current) => ({ ...current, address: event.target.value }))
                }
              />
            </label>
            <label className="space-y-2 md:col-span-2">
              <span className="text-sm text-muted-foreground">{t('userCenter.profile.description')}</span>
              <textarea
                className="min-h-28 w-full rounded-lg border border-input bg-background px-3 py-2 text-sm outline-none disabled:cursor-not-allowed disabled:opacity-60"
                value={profileForm.description}
                disabled={!profileEditing}
                onChange={(event) =>
                  setProfileForm((current) => ({ ...current, description: event.target.value }))
                }
              />
            </label>
          </div>
        </div>

        <div className="rounded-lg border bg-card p-5">
          <div className="flex flex-col gap-4 border-b pb-4 lg:flex-row lg:items-start lg:justify-between">
            <div>
              <h2 className="text-xl font-semibold">{t('userCenter.password.title')}</h2>
              <p className="mt-1 text-sm text-muted-foreground">{t('userCenter.password.subtitle')}</p>
            </div>
            <Button type="button" variant="outline" onClick={togglePasswordEdit}>
              {passwordEditing ? t('userCenter.password.save') : t('userCenter.password.edit')}
            </Button>
          </div>

          <div className="mt-5 grid gap-4">
            <label className="space-y-2">
              <span className="text-sm text-muted-foreground">{t('userCenter.password.current')}</span>
              <Input
                type="password"
                value={passwordForm.currentPassword}
                disabled={!passwordEditing}
                onChange={(event) =>
                  setPasswordForm((current) => ({ ...current, currentPassword: event.target.value }))
                }
              />
            </label>
            <label className="space-y-2">
              <span className="text-sm text-muted-foreground">{t('userCenter.password.next')}</span>
              <Input
                type="password"
                value={passwordForm.newPassword}
                disabled={!passwordEditing}
                onChange={(event) =>
                  setPasswordForm((current) => ({ ...current, newPassword: event.target.value }))
                }
              />
            </label>
            <label className="space-y-2">
              <span className="text-sm text-muted-foreground">{t('userCenter.password.confirm')}</span>
              <Input
                type="password"
                value={passwordForm.confirmPassword}
                disabled={!passwordEditing}
                onChange={(event) =>
                  setPasswordForm((current) => ({ ...current, confirmPassword: event.target.value }))
                }
              />
            </label>
          </div>
        </div>
      </div>
    </section>
  )
}
