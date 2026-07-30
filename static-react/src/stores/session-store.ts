import { create } from 'zustand'
import { createJSONStorage, persist } from 'zustand/middleware'
import { SESSION_STORE_KEY } from '@/stores/persistence-keys'

export interface SessionSnapshot {
  isLoggedIn: boolean
  accessToken: string | null
  characterId: number | null
  characterName: string | null
  roles: string[]
  corpCapabilities: string[]
  isCurrentlyNewbro: boolean
  isMentorMenteeEligible: boolean
  profileComplete: boolean
  enforceCharacterESIRestriction: boolean
  primaryCharacterId: number | null
  characters: Array<{ characterId: number; tokenInvalid: boolean }>
  hydratedAt: string | null
}

interface SessionStoreState extends SessionSnapshot {
  setSessionSnapshot: (snapshot: Partial<SessionSnapshot>) => void
  markBootstrapRequired: () => void
  markBootstrapComplete: () => void
  bootstrapRequired: boolean
  clearSession: () => void
}

const defaultSnapshot: SessionSnapshot = {
  isLoggedIn: false,
  accessToken: null,
  characterId: null,
  characterName: null,
  roles: [],
  corpCapabilities: [],
  isCurrentlyNewbro: false,
  isMentorMenteeEligible: false,
  profileComplete: true,
  enforceCharacterESIRestriction: true,
  primaryCharacterId: null,
  characters: [],
  hydratedAt: null,
}

export const useSessionStore = create<SessionStoreState>()(
  persist(
    (set) => ({
      ...defaultSnapshot,
      bootstrapRequired: false,
      setSessionSnapshot: (snapshot) => {
        set((state) => ({
          isLoggedIn: snapshot.isLoggedIn ?? state.isLoggedIn,
          accessToken: snapshot.accessToken ?? state.accessToken,
          characterId: snapshot.characterId ?? state.characterId,
          characterName: snapshot.characterName ?? state.characterName,
          roles: snapshot.roles ?? state.roles,
          corpCapabilities: snapshot.corpCapabilities ?? state.corpCapabilities,
          isCurrentlyNewbro: snapshot.isCurrentlyNewbro ?? state.isCurrentlyNewbro,
          isMentorMenteeEligible: snapshot.isMentorMenteeEligible ?? state.isMentorMenteeEligible,
          profileComplete: snapshot.profileComplete ?? state.profileComplete,
          enforceCharacterESIRestriction:
            snapshot.enforceCharacterESIRestriction ?? state.enforceCharacterESIRestriction,
          primaryCharacterId: snapshot.primaryCharacterId ?? state.primaryCharacterId,
          characters: snapshot.characters ?? state.characters,
          hydratedAt: new Date().toISOString(),
          bootstrapRequired: false,
        }))
      },
      markBootstrapRequired: () => set({ bootstrapRequired: true }),
      markBootstrapComplete: () => set({ bootstrapRequired: false }),
      clearSession: () => {
        set({
          ...defaultSnapshot,
          bootstrapRequired: false,
        })
      },
    }),
    {
      name: SESSION_STORE_KEY,
      storage: createJSONStorage(() => localStorage),
      partialize: (state) => ({
        isLoggedIn: state.isLoggedIn,
        accessToken: state.accessToken,
        characterId: state.characterId,
        characterName: state.characterName,
        roles: state.roles,
        corpCapabilities: state.corpCapabilities,
        isCurrentlyNewbro: state.isCurrentlyNewbro,
        isMentorMenteeEligible: state.isMentorMenteeEligible,
        profileComplete: state.profileComplete,
        enforceCharacterESIRestriction: state.enforceCharacterESIRestriction,
        primaryCharacterId: state.primaryCharacterId,
        characters: state.characters,
        hydratedAt: state.hydratedAt,
      }),
      onRehydrateStorage: () => (state) => {
        if (state?.accessToken) {
          state.markBootstrapRequired()
        }
      },
    }
  )
)
