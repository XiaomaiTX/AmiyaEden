import { useEffect, useRef, useState } from 'react'
import { Link, useLocation, useNavigate, useParams } from 'react-router-dom'
import {
  getAdmission,
  getAdmissionEvidence,
  listAdmissions,
  processAdmission,
  saveAdmission,
} from '@/api/admissions'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { useAdmissionContext } from '@/hooks/use-admission-context'
import { useI18n } from '@/i18n'
import { canAccessAdmission, compressAdmissionImage } from '@/lib/admissions'
import { operationSalaryMessage } from '@/lib/operation-salary'
import { useSessionStore, useBadgeStore } from '@/stores'
import type {
  AdmissionApplication,
  AdmissionBlacklistHit,
  AdmissionContext,
  AdmissionDetail,
  AdmissionKind,
  AdmissionList,
  AdmissionStatus,
  EvidenceKind,
} from '@/types/api/admissions'

const evidenceKinds: EvidenceKind[] = ['launcher', 'seat_profile']
const statusBadgeClass: Record<AdmissionStatus, string> = {
  pending: 'bg-amber-500/10 text-amber-700 dark:text-amber-400',
  invited: 'bg-emerald-500/10 text-emerald-700 dark:text-emerald-400',
  rejected: 'bg-destructive/10 text-destructive',
}
const fieldClass = 'w-full rounded-md border border-input bg-background px-3 py-2'
function errorKey(error: unknown) {
  return error instanceof Error && error.message.startsWith('admissions.errors.')
    ? error.message
    : 'admissions.errors.internal'
}

function AdmissionInstructions({ kind }: { kind: AdmissionKind }) {
  const { t } = useI18n()
  if (kind === 'alt')
    return <p className="font-semibold text-red-600 dark:text-red-400">{t('admissions.altHint')}</p>
  return (
    <p className="text-muted-foreground">
      {t('admissions.seatHint')
        .split('{seat}')
        .map((text, index) => (
          <span key={index}>
            {index > 0 && (
              <a
                className="text-blue-600 underline dark:text-blue-400"
                href="https://seat.winterco.org"
                target="_blank"
                rel="noreferrer"
              >
                {t('admissions.seat')}
              </a>
            )}
            {text}
          </span>
        ))}
    </p>
  )
}

function EvidenceImage({ id, kind }: { id: number; kind: EvidenceKind }) {
  const { t } = useI18n()
  const [url, setUrl] = useState('')
  const [error, setError] = useState('')
  useEffect(() => {
    let cancelled = false
    let objectUrl = ''
    void getAdmissionEvidence(id, kind)
      .then((blob) => {
        objectUrl = URL.createObjectURL(blob)
        if (!cancelled) setUrl(objectUrl)
        else URL.revokeObjectURL(objectUrl)
      })
      .catch((error) => {
        if (!cancelled) setError(errorKey(error))
      })
    return () => {
      cancelled = true
      if (objectUrl) URL.revokeObjectURL(objectUrl)
    }
  }, [id, kind])
  if (error) return <p role="alert">{t(error)}</p>
  return url ? (
    <a href={url} target="_blank" rel="noreferrer">
      <img
        src={url}
        alt={t(`admissions.${kind}`)}
        className="max-h-96 w-full object-contain object-left"
      />
    </a>
  ) : (
    <p>{t('admissions.loading')}</p>
  )
}

function UploadPreview({ blob, kind }: { blob: Blob; kind: EvidenceKind }) {
  const { t } = useI18n()
  const image = useRef<HTMLImageElement>(null)
  useEffect(() => {
    const value = URL.createObjectURL(blob)
    if (image.current) image.current.src = value
    return () => URL.revokeObjectURL(value)
  }, [blob])
  return (
    <img
      ref={image}
      alt={t(`admissions.${kind}`)}
      className="mt-2 max-h-64 max-w-full object-contain object-left"
    />
  )
}

function AdmissionForm({
  kind,
  context,
  application,
  onSaved,
}: {
  kind: AdmissionKind
  context: AdmissionContext
  application?: AdmissionApplication
  onSaved: (id: number) => void
}) {
  const { t } = useI18n()
  const [target, setTarget] = useState(application?.target_corporation_id ?? 98824507)
  const [characters, setCharacters] = useState<number[]>(
    application?.characters.map((char) => char.character_id) ?? []
  )
  const [files, setFiles] = useState<Partial<Record<EvidenceKind, Blob>>>({})
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const eligible = canAccessAdmission(context, kind)
  const [openedAt] = useState(() => Date.now())
  const mustResubmit =
    kind === 'new_member' &&
    application?.status === 'rejected' &&
    Boolean(application.evidence_expires_at) &&
    Date.parse(application.evidence_expires_at!) <= openedAt
  const chooseFile = async (kind: EvidenceKind, file?: File) => {
    if (!file) return
    setBusy(true)
    setError('')
    try {
      const blob = await compressAdmissionImage(file)
      setFiles((previous) => ({ ...previous, [kind]: blob }))
    } catch (error) {
      setError(errorKey(error))
    } finally {
      setBusy(false)
    }
  }
  const submit = async (resubmit: boolean) => {
    if (busy) return
    setBusy(true)
    setError('')
    try {
      const result = await saveAdmission(
        {
          kind,
          target_corporation_id: target,
          character_ids: characters,
          expected_version: application?.version ?? 0,
        },
        files,
        application?.id,
        resubmit
      )
      onSaved(result.id)
    } catch (error) {
      setError(errorKey(error))
    } finally {
      setBusy(false)
    }
  }
  if (!eligible) return <p role="alert">{t('admissions.denied')}</p>
  return (
    <form
      className="space-y-6"
      onSubmit={(event) => {
        event.preventDefault()
        void submit(mustResubmit)
      }}
    >
      <p>
        {t('admissions.bindHint')}{' '}
        <Link className="text-primary underline" to="/characters">
          {t('admissions.bind')}
        </Link>
      </p>
      <AdmissionInstructions kind={kind} />
      {kind === 'alt' && (
        <div>
          <h3 className="mb-1 font-medium">{t('admissions.contacts')}</h3>
          <p>
            {context.qq ? `QQ: ${context.qq}` : ''}{' '}
            {context.discord_id ? `Discord: ${context.discord_id}` : ''}
          </p>
        </div>
      )}
      <label className="block space-y-2">
        <span>{t('admissions.target')}</span>
        <select
          className={fieldClass}
          disabled={busy}
          value={target}
          onChange={(event) => {
            setTarget(Number(event.target.value))
            setCharacters([])
          }}
        >
          {context.targets.map((item) => (
            <option key={item.corporation_id} value={item.corporation_id}>
              {t(`admissions.corporations.${item.corporation_id}`)}
            </option>
          ))}
        </select>
      </label>
      <fieldset className="space-y-2">
        <legend className="mb-2 font-medium">{t('admissions.characters')}</legend>
        {context.characters.map((char) => (
          <label key={char.character_id} className="flex items-center gap-3 rounded-md border p-3">
            <input
              type="checkbox"
              disabled={busy || char.corporation_id === target}
              checked={characters.includes(char.character_id)}
              onChange={(event) =>
                setCharacters((previous) =>
                  event.target.checked
                    ? [...previous, char.character_id]
                    : previous.filter((id) => id !== char.character_id)
                )
              }
            />
            <span>{char.character_name}</span>
            <span className="ml-auto text-sm text-muted-foreground">{char.character_id}</span>
          </label>
        ))}
      </fieldset>
      {kind === 'new_member' && (
        <>
          <div className="grid gap-4 lg:grid-cols-2">
            {evidenceKinds.map((kind) => (
              <label className="block space-y-2 rounded-md border p-4" key={kind}>
                <span className="font-medium">
                  <span className="text-red-600 dark:text-red-400" aria-hidden="true">
                    *
                  </span>{' '}
                  {t(`admissions.${kind}`)} ({t('admissions.required')})
                </span>
                <input
                  aria-label={t(`admissions.${kind}`)}
                  aria-required="true"
                  className="block w-full text-sm"
                  type="file"
                  accept="image/jpeg,image/png"
                  disabled={busy}
                  onChange={(event) => {
                    void chooseFile(kind, event.target.files?.[0])
                    event.target.value = ''
                  }}
                />
                {files[kind] && <UploadPreview blob={files[kind]} kind={kind} />}
              </label>
            ))}
          </div>
          <details className="rounded-md border p-4">
            <summary className="cursor-pointer font-medium">
              {t('admissions.qqGuide')} / {t('admissions.doneGuide')}
            </summary>
            <div className="mt-4 grid gap-6 lg:grid-cols-2">
              {[
                ['qq-binding.png', 'qqGuide'],
                ['seat-completed.png', 'doneGuide'],
              ].map(([file, label]) => (
                <figure key={file}>
                  <figcaption className="mb-2">{t(`admissions.${label}`)}</figcaption>
                  <a href={`/admissions/${file}`} target="_blank" rel="noreferrer">
                    <img
                      src={`/admissions/${file}`}
                      alt={t(`admissions.${label}`)}
                      className="max-h-[32rem] max-w-full object-contain object-left"
                    />
                  </a>
                </figure>
              ))}
            </div>
          </details>
        </>
      )}
      {error && (
        <p role="alert" className="text-destructive">
          {t(error)}
        </p>
      )}
      <div className="flex flex-wrap gap-3">
        <Button type="submit" isDisabled={busy || characters.length === 0}>
          {t(
            mustResubmit
              ? 'admissions.resubmit'
              : application
                ? 'admissions.save'
                : 'admissions.submit'
          )}
        </Button>
        {application?.status === 'rejected' && !mustResubmit && (
          <Button
            type="button"
            isDisabled={busy || characters.length === 0}
            onClick={() => void submit(true)}
          >
            {t('admissions.resubmit')}
          </Button>
        )}
      </div>
    </form>
  )
}

export function AdmissionsPage() {
  const { t } = useI18n()
  const navigate = useNavigate()
  const location = useLocation()
  const { id } = useParams()
  const { context, loading: contextLoading, error: contextError, reload } = useAdmissionContext()
  const userID = useSessionStore((state) => state.characterId)
  const [detail, setDetail] = useState<AdmissionDetail | null>(null)
  const [list, setList] = useState<AdmissionList | null>(null)
  const [page, setPage] = useState(1)
  const [status, setStatus] = useState('pending')
  const [target, setTarget] = useState('')
  const [kind, setKind] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [revision, setRevision] = useState(0)
  const [reason, setReason] = useState('')
  const [busy, setBusy] = useState(false)
  // 处理失败后重新加载时保留的错误；加载开始会清空 error。
  const errorAfterReload = useRef('')
  const isEditing = Boolean(id) && location.pathname.endsWith('/edit')
  const mode = location.pathname.includes('/review')
    ? 'review'
    : location.pathname.endsWith('/new')
      ? 'new_member'
      : location.pathname.endsWith('/alt')
        ? 'alt'
        : 'history'
  useEffect(() => {
    let cancelled = false
    const load = async () => {
      if (mode === 'new_member' || mode === 'alt') {
        setDetail(null)
        setList(null)
        setLoading(false)
        return
      }
      setLoading(true)
      setError(errorAfterReload.current)
      errorAfterReload.current = ''
      setDetail(null)
      setList(null)
      const request = id
        ? getAdmission(Number(id)).then((value) => {
            if (!cancelled) setDetail(value)
          })
        : listAdmissions(
            mode === 'review',
            page,
            mode === 'review' ? status : '',
            target,
            kind
          ).then((value) => {
            if (!cancelled) setList(value)
          })
      await request
        .catch((error) => {
          if (!cancelled) setError(errorKey(error))
        })
        .finally(() => {
          if (!cancelled) setLoading(false)
        })
    }
    void load()
    return () => {
      cancelled = true
    }
  }, [id, mode, page, status, target, revision, userID, kind])
  const refresh = () => {
    void reload()
    setRevision((value) => value + 1)
  }
  const ownCharacterIDs = new Set(context?.characters.map((char) => char.character_id) ?? [])
  // 小号申请无需核对截图，可在列表直接登记已邀请；本人申请由后端拒绝处理，列表不显示入口。
  const canInviteFromList = (app: AdmissionApplication) =>
    mode === 'review' &&
    app.kind === 'alt' &&
    app.status === 'pending' &&
    !app.characters.some((char) => ownCharacterIDs.has(char.character_id))
  const blacklistValue = (hit: AdmissionBlacklistHit) =>
    hit.kind === 'character'
      ? t('blacklist.characterValue', { name: hit.display, id: hit.value })
      : hit.display
  const refreshBadges = () => {
    const characterId = useSessionStore.getState().characterId
    if (characterId !== null)
      void useBadgeStore
        .getState()
        .load(characterId, true)
        .catch(() => undefined)
  }
  const saved = (id: number) => {
    refreshBadges()
    void reload()
    navigate(`/admissions/applications/${id}`)
    setRevision((value) => value + 1)
  }
  const process = async (
    action: 'invited' | 'reject',
    application: AdmissionApplication | undefined = detail?.application
  ) => {
    if (!application || busy) return
    setBusy(true)
    setError('')
    try {
      const salary =
        action === 'invited' && application.kind === 'new_member'
          ? await operationSalaryMessage('recruitment', 1, t)
          : ''
      if (
        !window.confirm(
          [
            t(action === 'invited' ? 'admissions.confirmInvited' : 'admissions.confirmReject'),
            salary,
          ]
            .filter(Boolean)
            .join('\n\n')
        )
      )
        return
      await processAdmission(application.id, application.version, action, detail ? reason : '')
      refreshBadges()
      setRevision((value) => value + 1)
    } catch (error) {
      const key = errorKey(error)
      if (key === 'admissions.errors.blacklisted') {
        // 黑名单可能在打开页面后刷新；重新加载以显示最新命中。
        errorAfterReload.current = key
        setRevision((value) => value + 1)
      } else setError(key)
    } finally {
      setBusy(false)
    }
  }
  return (
    <section className="min-w-0 space-y-4">
      <header className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <h1 className="text-xl font-semibold">
            {t(`admissions.${isEditing ? 'editApplication' : id ? 'detail' : mode}`)}
            {id && ` #${id}`}
          </h1>
        </div>
        <Button
          variant="outline"
          onClick={isEditing ? () => navigate(`/admissions/applications/${id}`) : refresh}
        >
          {t(isEditing ? 'admissions.backToDetails' : 'admissions.refresh')}
        </Button>
      </header>
      {(contextLoading || loading) && <p role="status">{t('admissions.loading')}</p>}
      {(error || contextError) && (
        <div role="alert" className="text-destructive">
          {t(error || 'admissions.errors.internal')}{' '}
          <Button variant="outline" onClick={refresh}>
            {t('admissions.retry')}
          </Button>
        </div>
      )}
      {!contextLoading &&
        (mode === 'new_member' || mode === 'alt') &&
        (context ? (
          <Card>
            <CardContent>
              <AdmissionForm key={mode} kind={mode} context={context} onSaved={saved} />
            </CardContent>
          </Card>
        ) : (
          <p>{t('admissions.denied')}</p>
        ))}
      {!loading && !id && (mode === 'review' || mode === 'history') && (
        <>
          {mode === 'review' && (
            <div className="flex gap-3" role="tablist" aria-label={t('admissions.status')}>
              {['pending', 'processed'].map((value) => (
                <button
                  key={value}
                  role="tab"
                  aria-selected={status === value}
                  className={`rounded-md border px-3 py-2 ${status === value ? 'bg-primary text-primary-foreground' : 'bg-background'}`}
                  type="button"
                  onClick={() => {
                    setStatus(value)
                    setPage(1)
                  }}
                >
                  {t(`admissions.${value}`)}
                </button>
              ))}
            </div>
          )}
          {mode === 'review' && (
            <div className="flex flex-wrap gap-3">
              <select
                aria-label={t('admissions.kind')}
                className="rounded-md border bg-background p-2"
                value={kind}
                onChange={(event) => {
                  setKind(event.target.value)
                  setPage(1)
                }}
              >
                <option value="">{t('admissions.all')}</option>
                {['new_member', 'alt'].map((value) => (
                  <option key={value} value={value}>
                    {t(`admissions.${value}`)}
                  </option>
                ))}
              </select>
              <select
                aria-label={t('admissions.status')}
                className="rounded-md border bg-background p-2"
                value={status}
                onChange={(event) => {
                  setStatus(event.target.value)
                  setPage(1)
                }}
              >
                {['pending', 'processed', 'invited', 'rejected', ''].map((value) => (
                  <option key={value} value={value}>
                    {t(`admissions.${value || 'all'}`)}
                  </option>
                ))}
              </select>
              <select
                aria-label={t('admissions.target')}
                className="rounded-md border bg-background p-2"
                value={target}
                onChange={(event) => {
                  setTarget(event.target.value)
                  setPage(1)
                }}
              >
                <option value="">{t('admissions.all')}</option>
                {context?.targets
                  .filter((item) =>
                    context.reviewable_corporation_ids.includes(item.corporation_id)
                  )
                  .map((item) => (
                    <option key={item.corporation_id} value={item.corporation_id}>
                      {t(`admissions.corporations.${item.corporation_id}`)}
                    </option>
                  ))}
              </select>
            </div>
          )}
          <div className="overflow-x-auto rounded-md border">
            <table className="w-full text-left text-sm">
              <thead className="border-b bg-muted/40">
                <tr>
                  <th className="p-3">{t('admissions.applicant')}</th>
                  <th className="w-36 p-3">{t('admissions.kind')}</th>
                  <th className="p-3">{t('admissions.target')}</th>
                  <th className="p-3">{t('admissions.characters')}</th>
                  <th className="w-40 p-3">{t('admissions.status')}</th>
                  <th className="w-44 p-3">{t('admissions.submitted')}</th>
                  <th className="w-52 p-3">{t('common.operation')}</th>
                </tr>
              </thead>
              <tbody>
                {list?.list.map((app) => (
                  <tr key={app.id} className="border-b last:border-0">
                    <td className="p-3">{app.applicant_name}</td>
                    <td className="p-3">{t(`admissions.${app.kind}`)}</td>
                    <td className="p-3">
                      {t(`admissions.corporations.${app.target_corporation_id}`)}
                    </td>
                    <td className="p-3">
                      {app.characters.map((char) => char.character_name).join(', ')}
                    </td>
                    <td className="p-3">
                      <span
                        className={`inline-flex rounded-md px-2 py-0.5 text-xs font-medium ${statusBadgeClass[app.status]}`}
                      >
                        {t(`admissions.${app.status}`)}
                      </span>
                      {app.blacklisted && (
                        <span className="ml-2 inline-flex rounded-md bg-destructive px-2 py-0.5 text-xs font-medium text-white">
                          {t('admissions.blacklist.tag')}
                        </span>
                      )}
                      {app.closed_at && (
                        <div className="mt-1 space-y-1 text-xs text-muted-foreground">
                          <p>
                            {t('admissions.operator')}: {app.handler_name}
                          </p>
                          <time dateTime={app.closed_at}>
                            {new Date(app.closed_at).toLocaleString()}
                          </time>
                          {app.rejection_reason && (
                            <p
                              className="line-clamp-2 text-destructive [overflow-wrap:anywhere]"
                              title={app.rejection_reason}
                            >
                              {app.rejection_reason}
                            </p>
                          )}
                        </div>
                      )}
                    </td>
                    <td className="p-3 whitespace-nowrap">
                      {new Date(app.submitted_at).toLocaleString()}
                    </td>
                    <td className="p-3">
                      <div className="flex flex-wrap gap-2">
                        {canInviteFromList(app) && (
                          <Button
                            type="button"
                            size="sm"
                            isDisabled={busy || app.blacklisted}
                            onClick={() => void process('invited', app)}
                          >
                            {t('admissions.invited')}
                          </Button>
                        )}
                        <Button
                          type="button"
                          size="sm"
                          variant="outline"
                          onClick={() => navigate(`/admissions/applications/${app.id}`)}
                        >
                          {t('admissions.detail')}
                        </Button>
                        {mode === 'history' && app.status !== 'invited' && (
                          <Button
                            type="button"
                            size="sm"
                            variant="outline"
                            onClick={() => navigate(`/admissions/applications/${app.id}/edit`)}
                          >
                            {t('admissions.editApplication')}
                          </Button>
                        )}
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
            {!list?.list.length && <p className="p-6">{t('admissions.empty')}</p>}
          </div>
          <div className="flex items-center gap-4">
            <Button variant="outline" isDisabled={page === 1} onClick={() => setPage(page - 1)}>
              {t('admissions.previous')}
            </Button>
            <span>
              {t('admissions.page')} {page} · {t('admissions.total')} {list?.total ?? 0}
            </span>
            <Button
              variant="outline"
              isDisabled={page * 20 >= (list?.total ?? 0)}
              onClick={() => setPage(page + 1)}
            >
              {t('admissions.next')}
            </Button>
          </div>
        </>
      )}
      {!loading && detail && isEditing && (
        <Card>
          <CardContent>
            {context && detail.is_owner && detail.application.status !== 'invited' ? (
              <>
                {detail.application.rejection_reason && (
                  <p className="mb-4 text-destructive">{detail.application.rejection_reason}</p>
                )}
                <AdmissionForm
                  key={`${detail.application.id}:${detail.application.version}`}
                  kind={detail.application.kind}
                  context={context}
                  application={detail.application}
                  onSaved={saved}
                />
              </>
            ) : (
              <p role="alert">{t('admissions.denied')}</p>
            )}
          </CardContent>
        </Card>
      )}
      {!loading && detail && !isEditing && (
        <>
          <Card>
            <CardContent className="space-y-2 break-words">
              <h2 className="text-lg font-semibold">
                #{detail.application.id} · {detail.application.applicant_name}
              </h2>
              <p className="mt-2">
                {t(`admissions.${detail.application.status}`)} ·{' '}
                {t(`admissions.corporations.${detail.application.target_corporation_id}`)}
              </p>
              <p>{detail.application.characters.map((char) => char.character_name).join(', ')}</p>
              <p>{t(`admissions.${detail.application.kind}`)}</p>
              {detail.application.closed_at && (
                <p className="mt-2 text-sm">
                  {t('admissions.operator')}: {detail.application.handler_name} ·{' '}
                  <time dateTime={detail.application.closed_at}>
                    {new Date(detail.application.closed_at).toLocaleString()}
                  </time>
                </p>
              )}
              <p className="mt-2">
                {detail.application.qq ? `QQ: ${detail.application.qq}` : ''}{' '}
                {detail.application.discord_id ? `Discord: ${detail.application.discord_id}` : ''}
              </p>
              {detail.application.rejection_reason && (
                <p className="mt-3 text-destructive">
                  {t('admissions.reason')}: {detail.application.rejection_reason}
                </p>
              )}
              {detail.application.evidence_expires_at && (
                <p className="mt-2 text-sm">
                  {t('admissions.expires')}:{' '}
                  {new Date(detail.application.evidence_expires_at).toLocaleString()}
                </p>
              )}
              {detail.application.kind === 'new_member' && detail.evidence.length === 0 && (
                <p className="text-destructive">{t('admissions.expired')}</p>
              )}
            </CardContent>
          </Card>
          <div className="grid gap-4 lg:grid-cols-2">
            {detail.evidence.map((evidence) => (
              <figure
                key={`${evidence.kind}:${evidence.updated_at}`}
                className="rounded-md border p-4"
              >
                <figcaption className="mb-3 font-medium">
                  {t(`admissions.${evidence.kind}`)}
                </figcaption>
                <EvidenceImage id={detail.application.id} kind={evidence.kind} />
              </figure>
            ))}
          </div>
          {detail.can_review && detail.application.status === 'pending' && (
            <Card>
              <CardHeader>
                <CardTitle>{t('admissions.reviewApplication')}</CardTitle>
              </CardHeader>
              <CardContent className="space-y-4">
                {detail.blacklist_hits.length > 0 && (
                  <div
                    role="alert"
                    className="rounded-md border border-destructive p-3 text-destructive"
                  >
                    <p className="font-semibold">{t('admissions.blacklist.warning')}</p>
                    <ul className="mt-2 list-disc space-y-1 pl-5">
                      {detail.blacklist_hits.map((hit) => (
                        <li key={`${hit.kind}:${hit.value}`}>
                          <strong>{t(`blacklist.kinds.${hit.kind}`)}</strong>: {blacklistValue(hit)}
                          {hit.reason && (
                            <p className="whitespace-pre-line [overflow-wrap:anywhere]">
                              {hit.reason}
                            </p>
                          )}
                        </li>
                      ))}
                    </ul>
                  </div>
                )}
                <p>{t('admissions.manual')}</p>
                {detail.application.kind === 'new_member' && (
                  <p className="rounded-md bg-muted p-3 text-sm">
                    {t('admissions.reviewEvidence')}
                  </p>
                )}
                <label className="block space-y-2">
                  <span>{t('admissions.reason')}</span>
                  <textarea
                    className={fieldClass}
                    value={reason}
                    maxLength={4000}
                    rows={3}
                    placeholder={t('admissions.reasonHint')}
                    onChange={(event) => setReason(event.target.value)}
                  />
                </label>
                <div className="flex flex-wrap gap-3">
                  <Button
                    isDisabled={busy || detail.blacklist_hits.length > 0}
                    onClick={() => void process('invited')}
                  >
                    {t('admissions.invitedAction')}
                  </Button>
                  <Button
                    variant="destructive"
                    isDisabled={busy || !reason.trim()}
                    onClick={() => void process('reject')}
                  >
                    {t('admissions.rejectAction')}
                  </Button>
                </div>
              </CardContent>
            </Card>
          )}
          <Card>
            <CardContent>
              <h2 className="mb-3 text-lg font-semibold">{t('admissions.events')}</h2>
              <ol className="space-y-3 border-l pl-5">
                {detail.events.map((event) => (
                  <li key={event.id}>
                    <p>
                      {t(`admissions.${event.action}`)} · {event.actor_name} ·{' '}
                      {new Date(event.created_at).toLocaleString()}
                    </p>
                    {event.reason && <p className="text-muted-foreground">{event.reason}</p>}
                  </li>
                ))}
              </ol>
            </CardContent>
          </Card>
        </>
      )}
    </section>
  )
}
