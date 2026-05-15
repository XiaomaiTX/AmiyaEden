<!-- 个人中心页面 -->
<template>
  <div class="w-full h-full p-0 bg-transparent border-none shadow-none">
    <div class="relative flex-b mt-2.5 max-md:block max-md:mt-1">
      <div class="w-112 mr-5 max-md:w-full max-md:mr-0">
        <div class="art-card-sm relative p-9 pb-6 overflow-hidden text-center">
          <img class="absolute top-0 left-0 w-full h-50 object-cover" src="@imgs/user/bg.webp" />
          <img
            class="relative z-10 w-20 h-20 mt-30 mx-auto object-cover border-2 border-white rounded-full"
            src="@imgs/user/avatar.webp"
          />
          <h2 class="mt-5 text-xl font-normal">{{ userInfo.userName }}</h2>
          <p class="mt-5 text-sm">{{ $t('userCenter.profile.bio') }}</p>

          <div class="w-75 mx-auto mt-7.5 text-left">
            <div class="mt-2.5">
              <ArtSvgIcon icon="ri:mail-line" class="text-g-700" />
              <span class="ml-2 text-sm">jdkjjfnndf@mall.com</span>
            </div>
            <div class="mt-2.5">
              <ArtSvgIcon icon="ri:user-3-line" class="text-g-700" />
              <span class="ml-2 text-sm">{{ $t('userCenter.profile.role') }}</span>
            </div>
            <div class="mt-2.5">
              <ArtSvgIcon icon="ri:map-pin-line" class="text-g-700" />
              <span class="ml-2 text-sm">{{ $t('userCenter.profile.location') }}</span>
            </div>
            <div class="mt-2.5">
              <ArtSvgIcon icon="ri:dribbble-fill" class="text-g-700" />
              <span class="ml-2 text-sm">{{ $t('userCenter.profile.organization') }}</span>
            </div>
          </div>

          <div class="mt-10">
            <h3 class="text-sm font-medium">{{ $t('userCenter.sections.tags') }}</h3>
            <div class="flex flex-wrap justify-center mt-3.5">
              <div
                v-for="item in labelList"
                :key="item"
                class="py-1 px-1.5 mr-2.5 mb-2.5 text-xs border border-g-300 rounded"
              >
                {{ item }}
              </div>
            </div>
          </div>
        </div>
      </div>
      <div class="flex-1 overflow-hidden max-md:w-full max-md:mt-3.5">
        <div class="art-card-sm">
          <h1 class="p-4 text-xl font-normal border-b border-g-300">
            {{ $t('userCenter.sections.basicSettings') }}
          </h1>

          <ElForm
            :model="form"
            class="box-border p-5 [&>.el-row_.el-form-item]:w-[calc(50%-10px)] [&>.el-row_.el-input]:w-full [&>.el-row_.el-select]:w-full"
            ref="ruleFormRef"
            :rules="rules"
            label-width="86px"
            label-position="top"
          >
            <ElRow>
              <ElFormItem :label="$t('userCenter.fields.realName')" prop="realName">
                <ElInput v-model="form.realName" :disabled="!isEdit" />
              </ElFormItem>
              <ElFormItem :label="$t('userCenter.fields.sex')" prop="sex" class="ml-5">
                <ElSelect
                  v-model="form.sex"
                  :placeholder="$t('userCenter.fields.sexPlaceholder')"
                  :disabled="!isEdit"
                >
                  <ElOption
                    v-for="item in options"
                    :key="item.value"
                    :label="item.label"
                    :value="item.value"
                  />
                </ElSelect>
              </ElFormItem>
            </ElRow>

            <ElRow>
              <ElFormItem :label="$t('userCenter.fields.nickName')" prop="nikeName">
                <ElInput v-model="form.nikeName" :disabled="!isEdit" />
              </ElFormItem>
              <ElFormItem :label="$t('userCenter.fields.email')" prop="email" class="ml-5">
                <ElInput v-model="form.email" :disabled="!isEdit" />
              </ElFormItem>
            </ElRow>

            <ElRow>
              <ElFormItem :label="$t('userCenter.fields.mobile')" prop="mobile">
                <ElInput v-model="form.mobile" :disabled="!isEdit" />
              </ElFormItem>
              <ElFormItem :label="$t('userCenter.fields.address')" prop="address" class="ml-5">
                <ElInput v-model="form.address" :disabled="!isEdit" />
              </ElFormItem>
            </ElRow>

            <ElFormItem :label="$t('userCenter.fields.description')" prop="des" class="h-32">
              <ElInput type="textarea" :rows="4" v-model="form.des" :disabled="!isEdit" />
            </ElFormItem>

            <div class="flex-c justify-end [&_.el-button]:!w-27.5">
              <ElButton type="primary" class="w-22.5" v-ripple @click="edit">
                {{ isEdit ? $t('common.save') : $t('common.edit') }}
              </ElButton>
            </div>
          </ElForm>
        </div>

        <div class="art-card-sm my-5">
          <h1 class="p-4 text-xl font-normal border-b border-g-300">
            {{ $t('userCenter.sections.changePassword') }}
          </h1>

          <ElForm :model="pwdForm" class="box-border p-5" label-width="86px" label-position="top">
            <ElFormItem :label="$t('userCenter.fields.currentPassword')" prop="password">
              <ElInput
                v-model="pwdForm.password"
                type="password"
                :disabled="!isEditPwd"
                show-password
              />
            </ElFormItem>

            <ElFormItem :label="$t('userCenter.fields.newPassword')" prop="newPassword">
              <ElInput
                v-model="pwdForm.newPassword"
                type="password"
                :disabled="!isEditPwd"
                show-password
              />
            </ElFormItem>

            <ElFormItem :label="$t('userCenter.fields.confirmPassword')" prop="confirmPassword">
              <ElInput
                v-model="pwdForm.confirmPassword"
                type="password"
                :disabled="!isEditPwd"
                show-password
              />
            </ElFormItem>

            <div class="flex-c justify-end [&_.el-button]:!w-27.5">
              <ElButton type="primary" class="w-22.5" v-ripple @click="editPwd">
                {{ isEditPwd ? $t('common.save') : $t('common.edit') }}
              </ElButton>
            </div>
          </ElForm>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
  import { useI18n } from 'vue-i18n'
  import { useUserStore } from '@/store/modules/user'
  import type { FormInstance, FormRules } from 'element-plus'

  defineOptions({ name: 'UserCenter' })

  const { t } = useI18n()
  const userStore = useUserStore()
  const userInfo = computed(() => userStore.getUserInfo)

  const isEdit = ref(false)
  const isEditPwd = ref(false)
  const date = ref('')
  const ruleFormRef = ref<FormInstance>()

  /**
   * 用户信息表单
   */
  const form = reactive({
    realName: 'John Snow',
    nikeName: t('userCenter.profile.nickname'),
    email: '59301283@mall.com',
    mobile: '18888888888',
    address: t('userCenter.profile.address'),
    sex: '2',
    des: t('userCenter.profile.description')
  })

  /**
   * 密码修改表单
   */
  const pwdForm = reactive({
    password: '123456',
    newPassword: '123456',
    confirmPassword: '123456'
  })

  /**
   * 表单验证规则
   */
  const rules = computed<FormRules>(() => ({
    realName: [
      { required: true, message: t('userCenter.validation.realNameRequired'), trigger: 'blur' },
      { min: 2, max: 50, message: t('userCenter.validation.length2To50'), trigger: 'blur' }
    ],
    nikeName: [
      { required: true, message: t('userCenter.validation.nickNameRequired'), trigger: 'blur' },
      { min: 2, max: 50, message: t('userCenter.validation.length2To50'), trigger: 'blur' }
    ],
    email: [{ required: true, message: t('userCenter.validation.emailRequired'), trigger: 'blur' }],
    mobile: [
      { required: true, message: t('userCenter.validation.mobileRequired'), trigger: 'blur' }
    ],
    address: [
      { required: true, message: t('userCenter.validation.addressRequired'), trigger: 'blur' }
    ],
    sex: [{ required: true, message: t('userCenter.validation.sexRequired'), trigger: 'blur' }]
  }))

  /**
   * 性别选项
   */
  const options = computed(() => [
    { value: '1', label: t('userCenter.sex.male') },
    { value: '2', label: t('userCenter.sex.female') }
  ])

  /**
   * 用户标签列表
   */
  const labelList = computed(() => [
    t('userCenter.tags.design'),
    t('userCenter.tags.creative'),
    t('userCenter.tags.bold'),
    t('userCenter.tags.tall'),
    t('userCenter.tags.sichuan'),
    t('userCenter.tags.inclusive')
  ])

  onMounted(() => {
    getDate()
  })

  /**
   * 根据当前时间获取问候语
   */
  const getDate = () => {
    const h = new Date().getHours()

    if (h >= 6 && h < 9) date.value = t('userCenter.greetings.earlyMorning')
    else if (h >= 9 && h < 11) date.value = t('userCenter.greetings.morning')
    else if (h >= 11 && h < 13) date.value = t('userCenter.greetings.noon')
    else if (h >= 13 && h < 18) date.value = t('userCenter.greetings.afternoon')
    else if (h >= 18 && h < 24) date.value = t('userCenter.greetings.evening')
    else date.value = t('userCenter.greetings.lateNight')
  }

  /**
   * 切换用户信息编辑状态
   */
  const edit = () => {
    isEdit.value = !isEdit.value
  }

  /**
   * 切换密码编辑状态
   */
  const editPwd = () => {
    isEditPwd.value = !isEditPwd.value
  }
</script>
