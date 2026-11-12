// Localize both current and historical recruitment entries without rewriting ledger history.
export function walletReason(
  row: { ref_type: string; ref_id?: string; reason?: string },
  translate: (key: string, values: Record<string, string>) => string
): string {
  const id =
    row.ref_type === 'recruitment_invitation_salary'
      ? /^admission:(\d+)$/.exec(row.ref_id ?? '')?.[1]
      : undefined
  return id ? translate('admissions.walletReason', { id }) : row.reason || '-'
}
