<template>
  <ElDialog
    v-model="dialogVisible"
    :title="row?.name || $t('corporationStructures.servicesDialog.title')"
    width="480px"
    destroy-on-close
  >
    <template v-if="services.length">
      <div class="mb-2 text-sm text-g-500">
        {{ $t('corporationStructures.servicesDialog.subtitle') }}
      </div>
      <ElTable :data="services" size="small" border>
        <ElTableColumn
          prop="name"
          :label="$t('corporationStructures.servicesDialog.serviceName')"
          min-width="200"
          show-overflow-tooltip
        />
        <ElTableColumn
          :label="$t('corporationStructures.servicesDialog.serviceState')"
          width="120"
          align="center"
        >
          <template #default="{ row }">
            <ElTag :type="row.state === 'online' ? 'success' : 'info'" size="small" effect="plain">
              {{ formatServiceStateLabel(row.state) }}
            </ElTag>
          </template>
        </ElTableColumn>
      </ElTable>
    </template>
    <ElEmpty
      v-else
      :description="$t('corporationStructures.servicesDialog.empty')"
      :image-size="60"
    />
  </ElDialog>
</template>

<script setup lang="ts">
  import { ElDialog, ElEmpty, ElTable, ElTableColumn, ElTag } from 'element-plus'
  import { useI18n } from 'vue-i18n'

  defineOptions({ name: 'StructureServicesDialog' })

  interface Props {
    visible: boolean
    row: Api.Dashboard.CorporationStructureRow | null
  }

  const props = defineProps<Props>()
  const emit = defineEmits<{ (e: 'update:visible', val: boolean): void }>()

  const { t } = useI18n()

  const dialogVisible = computed({
    get: () => props.visible,
    set: (val) => emit('update:visible', val)
  })

  const services = computed(() => props.row?.services ?? [])

  const formatServiceStateLabel = (state: string) => {
    const key = `corporationStructures.serviceStates.${state}`
    const translated = t(key)
    if (translated === key) {
      return state || '--'
    }
    return translated
  }
</script>
