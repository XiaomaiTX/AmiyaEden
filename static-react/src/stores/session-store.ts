import { create } from 'zustand'
import { createJSONStorage, persist } from 'zustand/middleware'
import { SESSION_STORE_KEY } from '@/stores/persistence-keys'

export interface SessionSnapshot {
  isLoggedIn: boolean
  characterId: number | null
  characterName: string | null
  roles: string[]
  authList: string[]
  hydratedAt: string | null
}

interface SessionStoreState extends SessionSnapshot {
  setSessionSnapshot: (snapshot: Partial<SessionSnapshot>) => void
  setRouteAuthList: (authList: string[]) => void
  clearSession: () => void
}

const defaultSnapshot: SessionSnapshot = {
  isLoggedIn: false,
  characterId: null,
  characterName: null,
  roles: [],
  authList: [],
  hydratedAt: null,
}

export const useSessionStore = create<SessionStoreState>()(
  persist(
    (set) => ({
      ...defaultSnapshot,
      setSessionSnapshot: (snapshot) => {
        set((state) => ({
          isLoggedIn: snapshot.isLoggedIn ?? state.isLoggedIn,
          characterId: snapshot.characterId ?? state.characterId,
          characterName: snapshot.characterName ?? state.characterName,
          roles: snapshot.roles ?? state.roles,
          authList: snapshot.authList ?? state.authList,
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
        characterId: state.characterId,
        characterName: state.characterName,
        roles: state.roles,
        authList: state.authList,
        hydratedAt: state.hydratedAt,
      }),
    }
  )
)
