export function mergeSavedAdminIntoDirectory(
  directory: Api.FuxiAdmin.ManageDirectoryResponse,
  savedAdmin: Api.FuxiAdmin.ManageAdmin
): Api.FuxiAdmin.ManageDirectoryResponse {
  const targetTierIndex = directory.tiers.findIndex((tier) => tier.id === savedAdmin.tier_id)
  if (targetTierIndex < 0) {
    return directory
  }

  const existingTierIndex = directory.tiers.findIndex((tier) =>
    tier.admins.some((admin) => admin.id === savedAdmin.id)
  )
  const nextTiers = directory.tiers.map(
    (tier): Api.FuxiAdmin.ManageTierWithAdmins => ({ ...tier, admins: [...tier.admins] })
  )

  if (existingTierIndex >= 0) {
    const existingAdminIndex = nextTiers[existingTierIndex].admins.findIndex(
      (admin) => admin.id === savedAdmin.id
    )
    if (existingAdminIndex >= 0) {
      if (existingTierIndex === targetTierIndex) {
        nextTiers[existingTierIndex].admins[existingAdminIndex] = savedAdmin
      } else {
        nextTiers[existingTierIndex].admins.splice(existingAdminIndex, 1)
        nextTiers[targetTierIndex].admins.push(savedAdmin)
      }
      return { ...directory, tiers: nextTiers }
    }
  }

  nextTiers[targetTierIndex].admins.push(savedAdmin)
  return { ...directory, tiers: nextTiers }
}
