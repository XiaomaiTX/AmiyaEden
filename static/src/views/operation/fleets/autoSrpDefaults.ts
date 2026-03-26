type AutoSrpMode = 'disabled' | 'submit_only' | 'auto_approve'

interface ResolveAutoSrpModeOnFleetConfigChangeOptions {
  isEditing: boolean
  selectedFleetConfigId?: number
  currentMode: AutoSrpMode
  userTouchedMode: boolean
}

export function resolveAutoSrpModeOnFleetConfigChange(
  options: ResolveAutoSrpModeOnFleetConfigChangeOptions
): AutoSrpMode {
  if (
    options.isEditing ||
    !options.selectedFleetConfigId ||
    options.userTouchedMode ||
    options.currentMode !== 'disabled'
  ) {
    return options.currentMode
  }

  return 'auto_approve'
}
