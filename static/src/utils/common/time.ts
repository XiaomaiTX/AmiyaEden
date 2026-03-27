export const formatTime = (v: string | null | undefined) =>
  v ? new Date(v).toLocaleString('en-GB', { hour12: false }) : '-'
