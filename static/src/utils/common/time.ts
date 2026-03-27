export const formatTime = (v: string | null | undefined) => (v ? new Date(v).toLocaleString() : '-')
