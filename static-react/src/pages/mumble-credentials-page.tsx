import { useEffect, useState } from 'react'
import { fetchGetUserInfo } from '@/api/auth'
import { MumbleCredentialCard } from '@/components/mumble-credential-card'

export function MumbleCredentialsPage() {
  const [canonicalName, setCanonicalName] = useState('')

  useEffect(() => {
    let active = true
    void (async () => {
      try {
        const info = await fetchGetUserInfo()
        if (!active) return
        const primary =
          info.characters.find(
            (character) => character.character_id === info.primaryCharacterId
          ) ?? info.characters[0]
        setCanonicalName(primary?.character_name ?? '')
      } catch {
        if (active) {
          setCanonicalName('')
        }
      }
    })()
    return () => {
      active = false
    }
  }, [])

  return (
    <section className="space-y-4">
      <MumbleCredentialCard canonicalName={canonicalName} />
    </section>
  )
}
