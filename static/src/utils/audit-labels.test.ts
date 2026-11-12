import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'
import { createI18n } from 'vue-i18n'
import { getAuditActionLabel, getAuditCategoryLabel } from './audit-labels'
import {
  getTaskDisplayName,
  getTaskDisplayDescription
} from '../views/system/task-manager/task-labels'

type Messages = {
  taskManager: { tasks: Record<string, { name: string; description: string }> }
  auditAdmin: { actionLabels: Record<string, string>; categories: Record<string, string> }
}

for (const locale of ['zh', 'en']) {
  const messages: Messages = JSON.parse(
    readFileSync(new URL(`../locales/langs/${locale}.json`, import.meta.url), 'utf8')
  )
  const { t } = createI18n({
    legacy: false,
    locale,
    messages: { [locale]: messages },
    missingWarn: false,
    fallbackWarn: false
  }).global
  test(`${locale}: task and audit labels use the locale and preserve unknown identifiers`, () => {
    assert.equal(
      getTaskDisplayName(t, 'admission_evidence_cleanup', 'raw'),
      messages.taskManager.tasks.admission_evidence_cleanup.name
    )
    assert.equal(
      getTaskDisplayDescription(t, 'audit_archive_daily', 'raw'),
      messages.taskManager.tasks.audit_archive_daily.description
    )
    assert.equal(
      getTaskDisplayName(t, 'qq_group_governance_reconcile', 'raw'),
      messages.taskManager.tasks.qq_group_governance_reconcile.name
    )
    assert.equal(getTaskDisplayName(t, 'unknown_task', 'raw'), 'raw')
    assert.equal(
      getTaskDisplayDescription(t, 'unknown_task', 'server description'),
      'server description'
    )
    assert.equal(
      getAuditActionLabel(t, 'audit_archive_daily'),
      messages.auditAdmin.actionLabels.audit_archive_daily
    )
    assert.equal(getAuditCategoryLabel(t, 'system'), messages.auditAdmin.categories.system)
    assert.equal(getAuditActionLabel(t, 'unknown_event'), 'unknown_event')
    assert.equal(getAuditActionLabel(t, 'create'), 'create')
    assert.equal(
      getAuditActionLabel(t, 'create', 'corporation_admissions'),
      messages.auditAdmin.actionLabels.admission_create
    )
    assert.equal(getAuditActionLabel(t, 'unknown_event', 'corporation_admissions'), 'unknown_event')
    assert.equal(getAuditCategoryLabel(t, 'unknown_category'), 'unknown_category')
  })
}
