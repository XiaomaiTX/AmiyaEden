import assert from 'node:assert/strict'
import test from 'node:test'
import { readFileSync } from 'node:fs'

import {
  buildUserManageUpdatePayload,
  validateDiscordIdInput,
  validateNicknameInput,
  validateQQInput
} from './user-manage-dialog.helpers'

test('user manage dialog payload trims nickname and allows clearing both contacts for super admins', () => {
  const payload = buildUserManageUpdatePayload(
    {
      nickname: '  Test User  ',
      qq: ' ',
      discordId: ' '
    },
    true
  )

  assert.deepEqual(payload, {
    nickname: 'Test User',
    qq: '',
    discord_id: ''
  })
})

test('user manage dialog payload excludes contacts when contact editing is disabled', () => {
  const payload = buildUserManageUpdatePayload(
    {
      nickname: '  Test User  ',
      qq: '123456',
      discordId: 'discord-1'
    },
    false
  )

  assert.deepEqual(payload, {
    nickname: 'Test User'
  })
})

test('user manage dialog uses a compact left-labeled form layout', () => {
  const source = readFileSync(new URL('./user-manage-dialog.vue', import.meta.url), 'utf8')

  assert.match(source, /label-position="left"/)
  assert.match(source, /class="user-manage-dialog__grid"/)
  assert.doesNotMatch(source, /label-position="top"/)
})

test('user manage dialog localizes role labels through the component i18n scope', () => {
  const source = readFileSync(new URL('./user-manage-dialog.vue', import.meta.url), 'utf8')

  assert.match(source, /const \{ t, te \} = useI18n\(\)/)
  assert.match(
    source,
    /const getLocalizedRoleName = \(role: Api\.SystemManage\.RoleDefinition\) =>/
  )
  assert.match(source, /return te\(key\) \? t\(key\) : role\.name/)
  assert.match(source, /\{\{ getLocalizedRoleName\(role\) \}\}/)
  assert.doesNotMatch(source, /\{\{ role\.name \}\}/)
})

test('user manage dialog nickname validation still requires a non-empty nickname', () => {
  assert.equal(validateNicknameInput('   '), 'nicknameRequired')
  assert.equal(validateNicknameInput('Valid Name'), null)
})

test('user manage dialog contact validation allows blank contacts but still enforces QQ format', () => {
  assert.equal(validateQQInput('   '), null)
  assert.equal(validateQQInput('123456'), null)
  assert.equal(validateQQInput('abc123'), 'qqDigits')
})

test('user manage dialog discord validation only enforces max length', () => {
  assert.equal(validateDiscordIdInput('   '), null)
  assert.equal(validateDiscordIdInput('discord-user'), null)
  assert.equal(validateDiscordIdInput('1'.repeat(21)), 'discordLength')
})
