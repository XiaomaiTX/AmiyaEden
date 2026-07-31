import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Checkbox } from '@/components/ui/checkbox'
import {
  Table,
  TableHeader,
  TableBody,
  TableRow,
  TableHead,
  TableCell,
} from '@/components/ui/table'
import { Fragment, useCallback, useEffect, useMemo, useState } from 'react'
import { fetchGetUserInfo } from '@/api/auth'
import { toSessionSnapshot } from '@/auth'
import {
  fetchCharacterESIRestrictionConfig,
  updateCharacterESIRestrictionConfig,
} from '@/api/sys-config'
import {
  fetchDeleteUser,
  fetchGetRoleDefinitions,
  fetchGetUserList,
  fetchGetUserRoles,
  fetchImpersonateUser,
  fetchSetUserRoles,
  fetchUpdateUser,
} from '@/api/system-manage'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { buildEveCharacterPortraitUrl } from '@/lib/eve-image'
import { useI18n } from '@/i18n'
import { useSessionStore } from '@/stores'
import type { RoleDefinition, UserListItem } from '@/types/api/system-manage'
import { getErrorMessage, ShopDialog } from './shop-page-utils'

type SearchDraft = {
  keyword: string
  status: string
  role: string
}

type EditFormState = {
  nickname: string
  qq: string
  discordId: string
  status: number
  roleCodes: string[]
}

const defaultSearchDraft: SearchDraft = {
  keyword: '',
  status: '',
  role: '',
}

const defaultEditForm: EditFormState = {
  nickname: '',
  qq: '',
  discordId: '',
  status: 1,
  roleCodes: ['user'],
}

const numberFormatter = new Intl.NumberFormat('en-US')

function formatTime(value: string | null | undefined) {
  if (!value) return '-'
  const parsed = new Date(value)
  return Number.isNaN(parsed.getTime()) ? value : parsed.toLocaleString()
}

function isProtectedRole(role: string) {
  return role === 'super_admin' || role === 'admin'
}

function getRoleLabel(t: ReturnType<typeof useI18n>['t'], role: string) {
  const key = `userAdmin.roles.${role}`
  const translated = t(key)
  return translated === key ? role : translated
}

function getRoleTone(role: string) {
  switch (role) {
    case 'super_admin':
      return 'bg-red-100 text-red-700 dark:bg-red-500/10 dark:text-red-300'
    case 'admin':
      return 'bg-amber-100 text-amber-700 dark:bg-amber-500/10 dark:text-amber-300'
    case 'srp':
    case 'shop_order_manage':
      return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-300'
    case 'senior_fc':
    case 'fc':
      return 'bg-sky-100 text-sky-700 dark:bg-sky-500/10 dark:text-sky-300'
    case 'captain':
      return 'bg-indigo-100 text-indigo-700 dark:bg-indigo-500/10 dark:text-indigo-300'
    case 'mentor':
    case 'welfare':
      return 'bg-teal-100 text-teal-700 dark:bg-teal-500/10 dark:text-teal-300'
    default:
      return 'bg-muted text-muted-foreground'
  }
}

function getStatusTone(status: number) {
  return status === 1
    ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-300'
    : 'bg-rose-100 text-rose-700 dark:bg-rose-500/10 dark:text-rose-300'
}

function normalizeRoles(roles: string[]) {
  if (roles.includes('guest') && roles.length > 1) {
    return roles.filter((role) => role !== 'guest')
  }

  return roles.length > 0 ? roles : ['guest']
}

export function SystemUserPage() {
  const { t } = useI18n()
  const currentSessionRoles = useSessionStore((state) => state.roles)
  const setSessionSnapshot = useSessionStore((state) => state.setSessionSnapshot)

  const isSuperAdmin = currentSessionRoles.includes('super_admin')

  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [refreshSeed, setRefreshSeed] = useState(0)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)
  const [total, setTotal] = useState(0)
  const [users, setUsers] = useState<UserListItem[]>([])
  const [roleDefinitions, setRoleDefinitions] = useState<RoleDefinition[]>([])
  const [searchDraft, setSearchDraft] = useState<SearchDraft>(defaultSearchDraft)
  const [searchState, setSearchState] = useState<SearchDraft>(defaultSearchDraft)
  const [expandedUserIds, setExpandedUserIds] = useState<number[]>([])
  const [restrictionEnabled, setRestrictionEnabled] = useState(true)
  const [restrictionLoading, setRestrictionLoading] = useState(false)
  const [restrictionSaving, setRestrictionSaving] = useState(false)
  const [dialogOpen, setDialogOpen] = useState(false)
  const [dialogSaving, setDialogSaving] = useState(false)
  const [editingUser, setEditingUser] = useState<UserListItem | null>(null)
  const [userRolesLoading, setUserRolesLoading] = useState(false)
  const [dialogForm, setDialogForm] = useState<EditFormState>(defaultEditForm)

  const pageCount = useMemo(() => Math.max(1, Math.ceil(total / pageSize) || 1), [pageSize, total])

  const loadRoleDefinitions = useCallback(async () => {
    try {
      setRoleDefinitions(await fetchGetRoleDefinitions())
    } catch {
      setRoleDefinitions([])
    }
  }, [])

  const loadRestriction = useCallback(async () => {
    if (!isSuperAdmin) {
      return
    }

    setRestrictionLoading(true)
    try {
      const config = await fetchCharacterESIRestrictionConfig()
      setRestrictionEnabled(config.enforce_character_esi_restriction)
    } catch (caughtError) {
      setError(getErrorMessage(caughtError, t('userAdmin.characterEsiRestriction.loadFailed')))
    } finally {
      setRestrictionLoading(false)
    }
  }, [isSuperAdmin, t])

  const loadData = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const response = await fetchGetUserList({
        current: page,
        size: pageSize,
        keyword: searchState.keyword.trim() || undefined,
        status: searchState.status === '' ? undefined : Number(searchState.status),
        role: searchState.role || undefined,
      })
      setUsers(response.list ?? [])
      setTotal(response.total ?? 0)
      setPage(response.page ?? page)
      setPageSize(response.pageSize ?? pageSize)
      setExpandedUserIds((current) =>
        current.filter((id) => (response.list ?? []).some((user) => user.id === id))
      )
    } catch (caughtError) {
      setError(getErrorMessage(caughtError, t('userAdmin.loadFailed')))
      setUsers([])
      setTotal(0)
    } finally {
      setLoading(false)
    }
  }, [page, pageSize, searchState, t])

  useEffect(() => {
    const timer = window.setTimeout(() => {
      void loadData()
    }, 0)
    return () => window.clearTimeout(timer)
  }, [loadData, refreshSeed])

  useEffect(() => {
    const timer = window.setTimeout(() => {
      void loadRoleDefinitions()
      void loadRestriction()
    }, 0)
    return () => window.clearTimeout(timer)
  }, [loadRoleDefinitions, loadRestriction])

  const canEditProfile = (user: UserListItem) => isSuperAdmin || !isProtectedRoleSet(user.roles)
  const canEditContacts = (user: UserListItem) =>
    isSuperAdmin && !user.roles.includes('super_admin')
  const canEditRoles = (user: UserListItem) => isSuperAdmin || !user.roles.includes('super_admin')
  const canDeleteUser = (user: UserListItem) => isSuperAdmin || !isProtectedRoleSet(user.roles)

  const openEditDialog = async (user: UserListItem) => {
    if (!canEditProfile(user) && !canEditRoles(user)) {
      setError(t('userAdmin.editProtectedDenied'))
      return
    }

    setEditingUser(user)
    setDialogOpen(true)
    setUserRolesLoading(true)
    setDialogForm({
      nickname: user.nickname ?? '',
      qq: user.qq ?? '',
      discordId: user.discord_id ?? '',
      status: user.status,
      roleCodes: normalizeRoles(user.roles ?? ['guest']),
    })

    try {
      const roles = await fetchGetUserRoles(user.id)
      setDialogForm((current) => ({
        ...current,
        roleCodes: normalizeRoles(roles.map((role) => role.code)),
      }))
    } catch {
      setDialogForm((current) => ({
        ...current,
        roleCodes: normalizeRoles(user.roles ?? ['guest']),
      }))
    } finally {
      setUserRolesLoading(false)
    }
  }

  const submitEdit = async () => {
    if (!editingUser) return

    if (canEditProfile(editingUser)) {
      const nickname = dialogForm.nickname.trim()
      if (!nickname) {
        setError(t('userAdmin.manageDialog.nicknameRequired'))
        return
      }
    }

    setDialogSaving(true)
    setError(null)
    try {
      if (canEditProfile(editingUser)) {
        await fetchUpdateUser(editingUser.id, {
          nickname: dialogForm.nickname.trim(),
          qq: canEditContacts(editingUser) ? dialogForm.qq.trim() : undefined,
          discord_id: canEditContacts(editingUser) ? dialogForm.discordId.trim() : undefined,
          status: dialogForm.status,
        })
      }

      if (canEditRoles(editingUser)) {
        await fetchSetUserRoles(editingUser.id, normalizeRoles(dialogForm.roleCodes))
      }

      setDialogOpen(false)
      setEditingUser(null)
      setRefreshSeed((current) => current + 1)
    } catch (caughtError) {
      setError(getErrorMessage(caughtError, t('userAdmin.manageDialog.saveFailed')))
    } finally {
      setDialogSaving(false)
    }
  }

  const deleteUser = async (user: UserListItem) => {
    if (!canDeleteUser(user)) {
      setError(t('userAdmin.deleteProtectedDenied'))
      return
    }

    if (!window.confirm(t('userAdmin.deleteConfirm', { name: user.nickname || user.id }))) {
      return
    }

    setError(null)
    try {
      await fetchDeleteUser(user.id)
      setRefreshSeed((current) => current + 1)
    } catch (caughtError) {
      setError(getErrorMessage(caughtError, t('userAdmin.deleteFailed')))
    }
  }

  const impersonateUser = async (user: UserListItem) => {
    if (!window.confirm(t('userAdmin.impersonateConfirm', { name: user.nickname || user.id }))) {
      return
    }

    setError(null)
    try {
      const result = await fetchImpersonateUser(user.id)
      setSessionSnapshot({
        isLoggedIn: true,
        accessToken: result.token,
      } satisfies Partial<import('@/stores/session-store').SessionSnapshot>)
      const userInfo = await fetchGetUserInfo()
      setSessionSnapshot({
        accessToken: result.token,
        ...toSessionSnapshot(userInfo),
      })
      window.location.assign('/')
    } catch (caughtError) {
      setError(getErrorMessage(caughtError, t('userAdmin.impersonateFailed')))
    }
  }

  const toggleRestriction = async () => {
    if (restrictionSaving || restrictionLoading) {
      return
    }

    const nextValue = !restrictionEnabled
    setRestrictionSaving(true)
    try {
      await updateCharacterESIRestrictionConfig({
        enforce_character_esi_restriction: nextValue,
      })
      setRestrictionEnabled(nextValue)
    } catch (caughtError) {
      setError(getErrorMessage(caughtError, t('userAdmin.characterEsiRestriction.saveFailed')))
    } finally {
      setRestrictionSaving(false)
    }
  }

  const resetSearch = () => {
    setSearchDraft(defaultSearchDraft)
    setSearchState(defaultSearchDraft)
    setPage(1)
    setRefreshSeed((current) => current + 1)
  }

  const applySearch = () => {
    setSearchState(searchDraft)
    setPage(1)
    setRefreshSeed((current) => current + 1)
  }

  const getDisplayRoles = (user: UserListItem) => (user.roles?.length ? user.roles : ['guest'])

  const getUserCharacters = (user: UserListItem) => user.characters ?? []

  return (
    <section className="space-y-4">
      <div className="rounded-lg border bg-card p-5">
        <div className="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
          <div>
            <h1 className="text-xl font-semibold">{t('userAdmin.title')}</h1>
            <p className="mt-1 text-sm text-muted-foreground">{t('userAdmin.subtitle')}</p>
          </div>
          <div className="flex flex-wrap items-end gap-3">
            <label className="space-y-1">
              <span className="text-sm text-muted-foreground">{t('userAdmin.search.keyword')}</span>
              <Input
                className="w-60"
                value={searchDraft.keyword}
                onChange={(event) =>
                  setSearchDraft((current) => ({ ...current, keyword: event.target.value }))
                }
                placeholder={t('userAdmin.search.keywordPlaceholder')}
              />
            </label>
            <label className="space-y-1">
              <span className="text-sm text-muted-foreground">{t('common.status')}</span>
              <Select
                selectedKey={String(searchDraft.status ?? '')}
                onSelectionChange={(key) => ((value) =>
                  setSearchDraft((current) => ({ ...current, status: value })))(String(key))}
              >
                <SelectTrigger className="h-10">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem id="">{t('userAdmin.search.statusPlaceholder')}</SelectItem>
                  <SelectItem id="1">{t('userAdmin.status.active')}</SelectItem>
                  <SelectItem id="0">{t('userAdmin.status.disabled')}</SelectItem>
                </SelectContent>
              </Select>
            </label>
            <label className="space-y-1">
              <span className="text-sm text-muted-foreground">{t('common.role')}</span>
              <Select
                selectedKey={String(searchDraft.role ?? '')}
                onSelectionChange={(key) => ((value) =>
                  setSearchDraft((current) => ({ ...current, role: value })))(String(key))}
              >
                <SelectTrigger className="h-10">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem id="">{t('userAdmin.search.rolePlaceholder')}</SelectItem>
                  {roleDefinitions.map((role) => (
                    <SelectItem key={role.code} id={String(role.code ?? '')}>
                      {getRoleLabel(t, role.code)}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </label>
            <Button type="button" variant="outline" onClick={applySearch}>
              {t('common.search')}
            </Button>
            <Button type="button" variant="outline" onClick={resetSearch}>
              {t('common.reset')}
            </Button>
          </div>
        </div>
      </div>

      {isSuperAdmin ? (
        <div className="rounded-lg border bg-card p-5">
          <div className="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
            <div>
              <h2 className="text-base font-semibold">
                {t('userAdmin.characterEsiRestriction.title')}
              </h2>
              <p className="mt-1 text-sm text-muted-foreground">
                {t('userAdmin.characterEsiRestriction.description')}
              </p>
            </div>
            <div className="flex flex-wrap items-center gap-3">
              <span
                className={`inline-flex rounded-full px-3 py-1 text-xs font-medium ${
                  restrictionEnabled
                    ? 'bg-rose-100 text-rose-700 dark:bg-rose-500/10 dark:text-rose-300'
                    : 'bg-slate-100 text-slate-700 dark:bg-slate-500/10 dark:text-slate-300'
                }`}
              >
                {restrictionEnabled
                  ? t('userAdmin.characterEsiRestriction.enabled')
                  : t('userAdmin.characterEsiRestriction.disabled')}
              </span>
              <Button type="button" variant="outline" onClick={() => void toggleRestriction()}>
                {restrictionSaving || restrictionLoading
                  ? t('userAdmin.characterEsiRestriction.loading')
                  : t('userAdmin.characterEsiRestriction.switchLabel')}
              </Button>
            </div>
          </div>
        </div>
      ) : null}

      {error ? <p className="text-sm text-destructive">{error}</p> : null}
      {loading ? <p className="text-sm text-muted-foreground">{t('userAdmin.loading')}</p> : null}

      <div className="overflow-hidden rounded-lg border bg-card">
        <div className="border-b px-4 py-3 text-sm font-medium">
          {t('userAdmin.title')} ({total})
        </div>
        <div className="overflow-x-auto">
          <Table className="min-w-full text-sm">
            <TableHeader>
              <TableRow className="border-b bg-muted/40 text-left">
                <TableHead className="px-3 py-2">{t('userAdmin.table.userInfo')}</TableHead>
                <TableHead className="px-3 py-2">{t('common.role')}</TableHead>
                <TableHead className="px-3 py-2">{t('userAdmin.table.contact')}</TableHead>
                <TableHead className="px-3 py-2">{t('common.status')}</TableHead>
                <TableHead className="px-3 py-2">{t('userAdmin.table.lastLogin')}</TableHead>
                <TableHead className="px-3 py-2">{t('common.operation')}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {users.map((user) => {
                const expanded = expandedUserIds.includes(user.id)
                return (
                  <Fragment key={user.id}>
                    <TableRow
                      className="border-b align-top hover:bg-muted/40"
                      onClick={() =>
                        setExpandedUserIds((current) =>
                          current.includes(user.id)
                            ? current.filter((id) => id !== user.id)
                            : [...current, user.id]
                        )
                      }
                    >
                      <TableCell className="px-3 py-2">
                        <div className="flex items-center gap-3">
                          <img
                            alt={user.nickname || String(user.id)}
                            className="h-10 w-10 rounded-full border object-cover"
                            src={buildEveCharacterPortraitUrl(user.primary_character_id, 64)}
                          />
                          <div className="min-w-0">
                            <div className="font-medium">
                              {user.nickname || t('userAdmin.unnamed')}
                            </div>
                            <div className="text-xs text-muted-foreground">
                              {t('common.id')}: {user.id}
                            </div>
                          </div>
                        </div>
                      </TableCell>
                      <TableCell className="px-3 py-2">
                        <div className="flex flex-wrap gap-1">
                          {getDisplayRoles(user).map((role) => (
                            <span
                              key={role}
                              className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${getRoleTone(role)}`}
                            >
                              {getRoleLabel(t, role)}
                            </span>
                          ))}
                        </div>
                      </TableCell>
                      <TableCell className="px-3 py-2">
                        <div className="space-y-1 text-xs">
                          <div>
                            <span className="text-muted-foreground">
                              {t('characters.profile.qq')}:{' '}
                            </span>
                            <span>{user.qq || '-'}</span>
                          </div>
                          <div>
                            <span className="text-muted-foreground">
                              {t('characters.profile.discordId')}:
                            </span>
                            <span>{user.discord_id || '-'}</span>
                          </div>
                        </div>
                      </TableCell>
                      <TableCell className="px-3 py-2">
                        <span
                          className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${getStatusTone(
                            user.status
                          )}`}
                        >
                          {user.status === 1
                            ? t('userAdmin.status.active')
                            : t('userAdmin.status.disabled')}
                        </span>
                      </TableCell>
                      <TableCell className="px-3 py-2">
                        <div>{formatTime(user.last_login_at)}</div>
                        <div className="text-xs text-muted-foreground">
                          {user.last_login_ip || '-'}
                        </div>
                      </TableCell>
                      <TableCell className="px-3 py-2">
                        <div className="flex flex-wrap gap-2">
                          {isSuperAdmin ? (
                            <Button
                              type="button"
                              size="sm"
                              variant="outline"
                              onClick={(event) => {
                                event.stopPropagation()
                                void impersonateUser(user)
                              }}
                            >
                              {t('userAdmin.impersonate')}
                            </Button>
                          ) : null}
                          <Button
                            type="button"
                            size="sm"
                            variant="outline"
                            isDisabled={!canEditProfile(user) && !canEditRoles(user)}
                            onClick={(event) => {
                              event.stopPropagation()
                              void openEditDialog(user)
                            }}
                          >
                            {t('common.edit')}
                          </Button>
                          <Button
                            type="button"
                            size="sm"
                            variant="outline"
                            isDisabled={!canDeleteUser(user)}
                            onClick={(event) => {
                              event.stopPropagation()
                              void deleteUser(user)
                            }}
                          >
                            {t('common.delete')}
                          </Button>
                        </div>
                      </TableCell>
                    </TableRow>
                    {expanded ? (
                      <TableRow className="border-b bg-muted/20">
                        <TableCell className="px-3 py-3" colSpan={6}>
                          <div className="overflow-hidden rounded-lg border bg-background">
                            <div className="border-b px-3 py-2 text-xs font-medium text-muted-foreground">
                              {t('userAdmin.characters.title')} ({getUserCharacters(user).length})
                            </div>
                            <div className="overflow-x-auto">
                              {getUserCharacters(user).length > 0 ? (
                                <Table className="min-w-full text-xs">
                                  <TableHeader>
                                    <TableRow className="border-b bg-muted/40 text-left">
                                      <TableHead className="px-3 py-2">
                                        {t('userAdmin.characters.character')}
                                      </TableHead>
                                      <TableHead className="px-3 py-2">
                                        {t('userAdmin.characters.characterIdLabel')}
                                      </TableHead>
                                      <TableHead className="px-3 py-2">
                                        {t('userAdmin.characters.tokenHealth')}
                                      </TableHead>
                                      <TableHead className="px-3 py-2">
                                        {t('userAdmin.characters.seat')}
                                      </TableHead>
                                      <TableHead className="px-3 py-2">
                                        {t('userAdmin.characters.totalSkillPointsLabel')}
                                      </TableHead>
                                    </TableRow>
                                  </TableHeader>
                                  <TableBody>
                                    {getUserCharacters(user).map((character) => (
                                      <TableRow key={character.character_id} className="border-b">
                                        <TableCell className="px-3 py-2">
                                          <div className="flex items-center gap-2">
                                            <img
                                              alt={character.character_name}
                                              className="h-8 w-8 rounded-full border"
                                              src={buildEveCharacterPortraitUrl(
                                                character.character_id,
                                                32
                                              )}
                                            />
                                            <span className="font-medium">
                                              {character.character_name}
                                            </span>
                                          </div>
                                        </TableCell>
                                        <TableCell className="px-3 py-2">
                                          {character.character_id}
                                        </TableCell>
                                        <TableCell className="px-3 py-2">
                                          <span
                                            className={`inline-flex rounded-full px-2 py-0.5 font-medium ${
                                              character.token_invalid
                                                ? 'bg-rose-100 text-rose-700 dark:bg-rose-500/10 dark:text-rose-300'
                                                : 'bg-emerald-100 text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-300'
                                            }`}
                                          >
                                            {character.token_invalid
                                              ? t('userAdmin.characters.tokenExpired')
                                              : t('userAdmin.characters.tokenValid')}
                                          </span>
                                        </TableCell>
                                        <TableCell className="px-3 py-2">
                                          <a
                                            className="text-primary hover:underline"
                                            href={`https://seat.winterco.space/character/view/sheet/${character.character_id}`}
                                            rel="noreferrer noopener"
                                            target="_blank"
                                          >
                                            {character.character_id}
                                          </a>
                                        </TableCell>
                                        <TableCell className="px-3 py-2">
                                          {numberFormatter.format(character.total_sp)}
                                        </TableCell>
                                      </TableRow>
                                    ))}
                                  </TableBody>
                                </Table>
                              ) : (
                                <div className="px-3 py-5 text-muted-foreground">
                                  {t('userAdmin.characters.empty')}
                                </div>
                              )}
                            </div>
                          </div>
                        </TableCell>
                      </TableRow>
                    ) : null}
                  </Fragment>
                )
              })}
              {!loading && users.length === 0 ? (
                <TableRow>
                  <TableCell className="px-3 py-6 text-center text-muted-foreground" colSpan={6}>
                    {t('userAdmin.empty')}
                  </TableCell>
                </TableRow>
              ) : null}
            </TableBody>
          </Table>
        </div>
      </div>

      <div className="flex flex-wrap items-center gap-3 text-sm">
        <span>
          {page}/{pageCount}
        </span>
        <Button
          type="button"
          size="sm"
          variant="outline"
          onClick={() => setPage((current) => Math.max(1, current - 1))}
          isDisabled={page <= 1}
        >
          {t('welfareMy.pagination.prev')}
        </Button>
        <Button
          type="button"
          size="sm"
          variant="outline"
          onClick={() => setPage((current) => current + 1)}
          isDisabled={users.length < pageSize || page * pageSize >= total}
        >
          {t('welfareMy.pagination.next')}
        </Button>
        <label className="flex items-center gap-2">
          <span>{t('welfareMy.pageSize')}</span>
          <Select
            selectedKey={String(pageSize ?? '')}
            onSelectionChange={(key) => ((value) => {
              setPageSize(Number(value))
              setPage(1)
            })(String(key))}
          >
            <SelectTrigger className="h-8">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {[10, 20, 50].map((size) => (
                <SelectItem key={size} id={String(size ?? '')}>
                  {size}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </label>
      </div>

      <ShopDialog
        open={dialogOpen}
        title={t('userAdmin.manageDialog.title')}
        onClose={() => {
          setDialogOpen(false)
          setEditingUser(null)
        }}
        closeLabel={t('common.close')}
        widthClass="w-full sm:max-w-2xl"
        footer={
          <>
            <Button
              type="button"
              variant="outline"
              onClick={() => {
                setDialogOpen(false)
                setEditingUser(null)
              }}
              isDisabled={dialogSaving}
            >
              {t('common.cancel')}
            </Button>
            <Button type="button" onClick={() => void submitEdit()} isDisabled={dialogSaving}>
              {dialogSaving ? t('userAdmin.manageDialog.saving') : t('common.confirm')}
            </Button>
          </>
        }
      >
        {editingUser ? (
          <div className="space-y-4">
            <div className="flex items-center gap-3">
              <img
                alt={editingUser.nickname || String(editingUser.id)}
                className="h-11 w-11 rounded-full border object-cover"
                src={buildEveCharacterPortraitUrl(editingUser.primary_character_id, 64)}
              />
              <div>
                <div className="font-medium">{editingUser.nickname || t('userAdmin.unnamed')}</div>
                <div className="text-xs text-muted-foreground">#{editingUser.id}</div>
              </div>
            </div>
            {userRolesLoading ? (
              <p className="text-sm text-muted-foreground">{t('userAdmin.manageDialog.loading')}</p>
            ) : null}
            {canEditProfile(editingUser) ? (
              <div className="grid gap-4 md:grid-cols-2">
                <label className="space-y-2 md:col-span-2">
                  <span className="text-sm text-muted-foreground">
                    {t('characters.profile.nickname')}
                  </span>
                  <Input
                    value={dialogForm.nickname}
                    onChange={(event) =>
                      setDialogForm((current) => ({ ...current, nickname: event.target.value }))
                    }
                  />
                </label>
                {canEditContacts(editingUser) ? (
                  <>
                    <label className="space-y-2">
                      <span className="text-sm text-muted-foreground">
                        {t('characters.profile.qq')}
                      </span>
                      <Input
                        value={dialogForm.qq}
                        onChange={(event) =>
                          setDialogForm((current) => ({ ...current, qq: event.target.value }))
                        }
                      />
                    </label>
                    <label className="space-y-2">
                      <span className="text-sm text-muted-foreground">
                        {t('characters.profile.discordId')}
                      </span>
                      <Input
                        value={dialogForm.discordId}
                        onChange={(event) =>
                          setDialogForm((current) => ({
                            ...current,
                            discordId: event.target.value,
                          }))
                        }
                      />
                    </label>
                  </>
                ) : null}
                <label className="space-y-2">
                  <span className="text-sm text-muted-foreground">{t('common.status')}</span>
                  <Select
                    selectedKey={String(dialogForm.status)}
                    onSelectionChange={(key) => ((value) =>
                      setDialogForm((current) => ({ ...current, status: Number(value) })))(String(key))}
                  >
                    <SelectTrigger className="h-10">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem id="1">{t('userAdmin.status.active')}</SelectItem>
                      <SelectItem id="0">{t('userAdmin.status.disabled')}</SelectItem>
                    </SelectContent>
                  </Select>
                </label>
              </div>
            ) : null}
            {canEditRoles(editingUser) ? (
              <div className="space-y-2">
                <div className="text-sm font-medium">{t('userAdmin.roleManageTitle')}</div>
                <div className="grid gap-x-4 gap-y-3 sm:grid-cols-2">
                  {roleDefinitions.map((role) => {
                    const checked = dialogForm.roleCodes.includes(role.code)
                    const disabled =
                      role.code === 'super_admin' || (role.code === 'admin' && !isSuperAdmin)
                    return (
                      <label key={role.code} className="flex min-w-0 items-center gap-2 text-sm">
                        <Checkbox
                          isSelected={checked}
                          isDisabled={disabled}
                          onChange={(nextChecked) =>
                            setDialogForm((current) => ({
                              ...current,
                              roleCodes:
                                nextChecked === true
                                  ? normalizeRoles([...current.roleCodes, role.code])
                                  : normalizeRoles(
                                      current.roleCodes.filter((item) => item !== role.code)
                                    ),
                            }))
                          }
                        />
                        <span className="truncate whitespace-nowrap">
                          {getRoleLabel(t, role.code)}
                        </span>
                      </label>
                    )
                  })}
                </div>
              </div>
            ) : null}
          </div>
        ) : null}
      </ShopDialog>
    </section>
  )
}

function isProtectedRoleSet(roles: string[]) {
  return roles.some((role) => isProtectedRole(role))
}
