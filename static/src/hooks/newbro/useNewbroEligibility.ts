import { ElMessage } from 'element-plus'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { useMenuStore } from '../../store/modules/menu'
import { getNewbroIneligibilityReasonKey } from './newbroEligibility'
import { formatNewbroDateTime } from './useNewbroFormatters'

export function useNewbroEligibility() {
  const { t } = useI18n()
  const router = useRouter()
  const menuStore = useMenuStore()

  const redirectIfIneligible = async (state: Api.Newbro.MyAffiliationResponse) => {
    if (state.is_currently_newbro) {
      return false
    }

    ElMessage.warning(
      t('newbro.select.currentlyIneligibleWithReason', {
        reason: t(getNewbroIneligibilityReasonKey(state.disqualified_reason)),
        evaluatedAt: formatNewbroDateTime(state.evaluated_at)
      })
    )
    await router.replace(menuStore.getHomePath() || '/')
    return true
  }

  return {
    redirectIfIneligible
  }
}
