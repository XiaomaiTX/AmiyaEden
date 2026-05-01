import { create } from 'zustand'
import { createJSONStorage, persist } from 'zustand/middleware'
import { SESSION_STORE_KEY } from '@/stores/persistence-keys'

export interface SessionSnapshot {
  isLoggedIn: boolean
  accessToken: string | null
  characterId: number | null
  characterName: string | null
  roles: string[]
  authList: string[]
  isCurrentlyNewbro: boolean
  isMentorMenteeEligible: boolean
  hydratedAt: string | null
}

interface SessionStoreState extends SessionSnapshot {
  setSessionSnapshot: (snapshot: Partial<SessionSnapshot>) => void
  setRouteAuthList: (authList: string[]) => void
  clearSession: () => void
}

const defaultSnapshot: SessionSnapshot = {
  isLoggedIn: false,
  accessToken: null,
  characterId: null,
  characterName: null,
  roles: [],
  authList: [],
  isCurrentlyNewbro: false,
  isMentorMenteeEligible: false,
  hydratedAt: null,
}

export const useSessionStore = create<SessionStoreState>()(
  persist(
    (set) => ({
      ...defaultSnapshot,
      setSessionSnapshot: (snapshot) => {
        set((state) => ({
          isLoggedIn: snapshot.isLoggedIn ?? state.isLoggedIn,
          accessToken: snapshot.accessToken ?? state.accessToken,
          characterId: snapshot.characterId ?? state.characterId,
          characterName: snapshot.characterName ?? state.characterName,
          roles: snapshot.roles ?? state.roles,
          authList: snapshot.authList ?? state.authList,
          isCurrentlyNewbro: snapshot.isCurrentlyNewbro ?? state.isCurrentlyNewbro,
          isMentorMenteeEligible: snapshot.isMentorMenteeEligible ?? state.isMentorMenteeEligible,
          hydratedAt: new Date().toISOString(),
        }))
      },
      setRouteAuthList: (authList) => {
        set({
          authList: Array.from(new Set(authList)),
        })
      },
      clearSession: () => {
        set({
          ...defaultSnapshot,
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
        authList: state.authList,
        isCurrentlyNewbro: state.isCurrentlyNewbro,
        isMentorMenteeEligible: state.isMentorMenteeEligible,
        hydratedAt: state.hydratedAt,
      }),
    }
  )
)
