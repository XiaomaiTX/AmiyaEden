<template>
  <ElDialog
    v-model="dialogVisible"
    :title="t('userAdmin.manageDialog.title')"
    width="560px"
    align-center
    @open="onOpen"
  >
    <ElForm
      ref="formRef"
      :model="form"
      :rules="rules"
      label-position="left"
      label-width="96px"
      class="user-manage-dialog__form"
    >
      <div class="user-manage-dialog__summary">
        <ElAvatar
          :size="36"
          :src="buildEveCharacterPortraitUrl(userData?.primary_character_id ?? 0, 36)"
        />
        <div class="user-manage-dialog__summary-content">
          <div class="user-manage-dialog__summary-name">
            {{ userData?.nickname || $t('userAdmin.unnamed') }}
          </div>
          <div class="user-manage-dialog__summary-meta">
            {{ $t('common.user') }}
          </div>
        </div>
      </div>

      <template v-if="canEditProfile">
        <div v-if="canEditRoles" class="user-manage-dialog__section-title">
          {{ t('userAdmin.profileDialog.title') }}
        </div>

        <div class="user-manage-dialog__grid">
          <ElFormItem
            class="user-manage-dialog__span-2"
            :label="$t('characters.profile.nickname')"
            prop="nickname"
          >
            <ElInput
              v-model="form.nickname"
              :maxlength="20"
              clearable
              show-word-limit
              :placeholder="$t('characters.profile.nicknamePlaceholder')"
            />
          </ElFormItem>

          <template v-if="canEditContacts">
            <ElFormItem :label="$t('characters.profile.qq')" prop="qq">
              <ElInput
                v-model="form.qq"
                :maxlength="20"
                clearable
                show-word-limit
                :placeholder="$t('characters.profile.qqPlaceholder')"
              />
            </ElFormItem>

            <ElFormItem :label="$t('characters.profile.discordId')" prop="discordId">
              <ElInput
                v-model="form.discordId"
                :maxlength="20"
                clearable
                show-word-limit
                :placeholder="$t('characters.profile.discordPlaceholder')"
              />
            </ElFormItem>
          </template>
        </div>
      </template>

      <ElDivider v-if="canEditProfile && canEditRoles" />

      <template v-if="canEditRoles">
        <div v-if="canEditProfile" class="user-manage-dialog__section-title">
          {{ t('userAdmin.roleManageTitle') }}
        </div>

        <ElFormItem :label="$t('common.role')" class="user-manage-dialog__roles-form-item">
          <ElCheckboxGroup v-model="selectedRoleCodes" class="user-manage-dialog__roles">
            <ElCheckbox
              v-for="role in allRoles"
              :key="role.code"
              :label="role.code"
              :disabled="
                role.code === 'super_admin' ||
                (role.code === 'admin' && !(isSuperAdmin || isEditingSelf))
              "
            >
              {{ getLocalizedRoleName(role) }}
            </ElCheckbox>
          </ElCheckboxGroup>
        </ElFormItem>
      </template>
    </ElForm>

    <template #footer>
      <div class="dialog-footer">
        <ElButton @click="dialogVisible = false">{{ $t('common.cancel') }}</ElButton>
        <ElButton type="primary" :loading="submitting" @click="handleSubmit">
          {{ $t('common.confirm') }}
        </ElButton>
      </div>
    </template>
  </ElDialog>
</template>

<script setup lang="ts">
  import type { FormInstance, FormRules } from 'element-plus'
  import { useI18n } from 'vue-i18n'
  import {
    fetchGetRoleDefinitions,
    fetchGetUserRoles,
    fetchSetUserRoles,
    fetchUpdateUser
  } from '@/api/system-manage'
  import { useUserStore } from '@/store/modules/user'
  import {
    buildUserManageUpdatePayload,
    validateDiscordIdInput,
    validateNicknameInput,
    validateQQInput,
    type UserManageDialogValidationError
  } from './user-manage-dialog.helpers'
  import { buildEveCharacterPortraitUrl } from '@/utils/eve-image'

  interface Props {
    visible: boolean
    userData?: Partial<Api.SystemManage.UserListItem>
    canEditProfile?: boolean
    canEditContacts?: boolean
    canEditRoles?: boolean
  }

  interface Emits {
    (e: 'update:visible', value: boolean): void
    (e: 'saved'): void
  }

  const props = withDefaults(defineProps<Props>(), {
    canEditProfile: false,
    canEditContacts: false,
    canEditRoles: false
  })
  const emit = defineEmits<Emits>()
  const { t, te } = useI18n()

  const validationMessageKeys: Record<UserManageDialogValidationError, string> = {
    nicknameRequired: 'characters.profile.validation.nicknameRequired',
    nicknameLength: 'characters.profile.validation.nicknameLength',
    qqLength: 'characters.profile.validation.qqLength',
    qqDigits: 'characters.profile.validation.qqDigits',
    discordLength: 'characters.profile.validation.discordLength'
  }

  const userStore = useUserStore()
  const isSuperAdmin = computed(() => userStore.info?.roles?.includes('super_admin'))
  const currentUserId = computed(() => userStore.info?.userId)
  const isEditingSelf = computed(
    () => props.userData?.id != null && currentUserId.value === props.userData.id
  )

  const dialogVisible = computed({
    get: () => props.visible,
    set: (value) => emit('update:visible', value)
  })

  const formRef = ref<FormInstance>()
  const allRoles = ref<Api.SystemManage.RoleDefinition[]>([])
  const selectedRoleCodes = ref<string[]>([])
  const roleSelectionReady = ref(false)
  const submitting = ref(false)
  const form = reactive({
    nickname: '',
    qq: '',
    discordId: ''
  })

  watch(selectedRoleCodes, (codes) => {
    if (codes.length <= 1 || !codes.includes('guest')) return

    const nonGuestCodes = codes.filter((code) => code !== 'guest')
    selectedRoleCodes.value = nonGuestCodes.length > 0 ? nonGuestCodes : ['guest']
  })

  const reportValidationResult = (
    errorCode: UserManageDialogValidationError | null,
    callback: (error?: Error) => void
  ) => {
    if (!errorCode) {
      callback()
      return
    }

    callback(new Error(t(validationMessageKeys[errorCode])))
  }

  const getLocalizedRoleName = (role: Api.SystemManage.RoleDefinition) => {
    const key = `userAdmin.roles.${role.code}`
    return te(key) ? t(key) : role.name
  }

  const rules = computed<FormRules>(() => {
    if (!props.canEditProfile) {
      return {}
    }

    const nextRules: FormRules = {
      nickname: [
        {
          validator: (_rule, value: string, callback: (error?: Error) => void) => {
            reportValidationResult(validateNicknameInput(value), callback)
          },
          trigger: 'blur'
        }
      ]
    }

    if (!props.canEditContacts) {
      return nextRules
    }

    nextRules.qq = [
      {
        validator: (_rule, value: string, callback: (error?: Error) => void) => {
          reportValidationResult(validateQQInput(value), callback)
        },
        trigger: 'blur'
      }
    ]

    nextRules.discordId = [
      {
        validator: (_rule, value: string, callback: (error?: Error) => void) => {
          reportValidationResult(validateDiscordIdInput(value), callback)
        },
        trigger: 'blur'
      }
    ]

    return nextRules
  })

  const getSuccessMessage = () => {
    if (props.canEditProfile && props.canEditRoles) {
      return t('userAdmin.manageDialog.saveSuccess')
    }

    if (props.canEditProfile) {
      return t('userAdmin.profileDialog.saveSuccess')
    }

    return t('roleUi.userRoleDialog.saveSuccess')
  }

  const getFailedMessage = () => {
    if (props.canEditProfile && props.canEditRoles) {
      return t('userAdmin.manageDialog.saveFailed')
    }

    if (props.canEditProfile) {
      return t('userAdmin.profileDialog.saveFailed')
    }

    return t('roleUi.userRoleDialog.saveFailed')
  }

  const syncForm = () => {
    form.nickname = props.userData?.nickname ?? ''
    form.qq = props.userData?.qq ?? ''
    form.discordId = props.userData?.discord_id ?? ''
  }

  const loadRoleSelection = async () => {
    roleSelectionReady.value = !props.canEditRoles
    allRoles.value = []
    selectedRoleCodes.value = []

    if (!props.canEditRoles || !props.userData?.id) {
      return
    }

    const [roleDefinitions, userRoles] = await Promise.all([
      fetchGetRoleDefinitions(),
      fetchGetUserRoles(props.userData.id)
    ])

    allRoles.value = roleDefinitions
    selectedRoleCodes.value = userRoles.map((role) => role.code)
    roleSelectionReady.value = true
  }

  const onOpen = async () => {
    syncForm()
    nextTick(() => {
      formRef.value?.clearValidate()
    })

    try {
      await loadRoleSelection()
    } catch (error) {
      roleSelectionReady.value = false
      console.error(t('roleUi.userRoleDialog.loadFailed'), error)
      ElMessage.error(t('roleUi.userRoleDialog.loadFailed'))
    }
  }

  const handleSubmit = async () => {
    if (!props.userData?.id) return

    if (props.canEditProfile) {
      await formRef.value?.validate()
    }

    if (props.canEditRoles && !roleSelectionReady.value) {
      ElMessage.error(t('roleUi.userRoleDialog.loadFailed'))
      return
    }

    submitting.value = true
    try {
      if (props.canEditProfile) {
        await fetchUpdateUser(
          props.userData.id,
          buildUserManageUpdatePayload(form, props.canEditContacts)
        )
      }

      if (props.canEditRoles) {
        await fetchSetUserRoles(props.userData.id, selectedRoleCodes.value)
      }

      ElMessage.success(getSuccessMessage())
      dialogVisible.value = false
      emit('saved')
    } catch (error: unknown) {
      ElMessage.error(error instanceof Error ? error.message : getFailedMessage())
    } finally {
      submitting.value = false
    }
  }
</script>

<style scoped>
  .user-manage-dialog__form {
    display: grid;
    gap: 12px;
  }

  .user-manage-dialog__summary {
    display: flex;
    align-items: center;
    gap: 12px;
    padding-bottom: 4px;
  }

  .user-manage-dialog__summary-content {
    min-width: 0;
  }

  .user-manage-dialog__summary-name {
    font-size: 14px;
    font-weight: 600;
    line-height: 1.2;
  }

  .user-manage-dialog__summary-meta {
    margin-top: 2px;
    font-size: 12px;
    color: var(--el-text-color-secondary);
    line-height: 1.2;
  }

  .user-manage-dialog__section-title {
    font-size: 13px;
    font-weight: 600;
    line-height: 1.2;
  }

  .user-manage-dialog__grid {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 12px 16px;
  }

  .user-manage-dialog__span-2 {
    grid-column: 1 / -1;
  }

  .user-manage-dialog__roles-form-item {
    margin-bottom: 0;
  }

  .user-manage-dialog__roles {
    display: flex;
    flex-wrap: wrap;
    gap: 8px 12px;
  }

  :deep(.user-manage-dialog__form .el-form-item) {
    margin-bottom: 0;
  }

  :deep(.user-manage-dialog__form .el-form-item__label) {
    padding-right: 10px;
  }

  @media (max-width: 640px) {
    .user-manage-dialog__grid {
      grid-template-columns: 1fr;
    }

    .user-manage-dialog__span-2 {
      grid-column: auto;
    }
  }
</style>
