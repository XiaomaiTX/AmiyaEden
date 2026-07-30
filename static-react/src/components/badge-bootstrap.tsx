import { useEffect } from 'react'
import { useBadgeStore, useSessionStore } from '@/stores'

export function BadgeBootstrap() {
  const isLoggedIn = useSessionStore((state) => state.isLoggedIn)
  const characterId = useSessionStore((state) => state.characterId)
  const bootstrapRequired = useSessionStore((state) => state.bootstrapRequired)
  const load = useBadgeStore((state) => state.load)
  const clear = useBadgeStore((state) => state.clear)

  useEffect(() => {
    if (!isLoggedIn || characterId === null) {
      clear()
      return
    }
    if (bootstrapRequired) {
      return
    }

    void load(characterId).catch(() => undefined)
  }, [bootstrapRequired, characterId, clear, isLoggedIn, load])

  return null
}
