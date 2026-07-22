import type { EveCharacter, RegisteredScope } from '@/types/api/auth'

export interface ScopeRow extends RegisteredScope {
  authorized: boolean
}

export interface ScopeCoverage {
  scopeRows: ScopeRow[]
  requiredScopes: RegisteredScope[]
  grantedRequiredCount: number
  hasMissingRequiredScopes: boolean
}

function parseScopeSet(scopesText: string) {
  return new Set(scopesText.split(' ').filter(Boolean))
}

export function findInvalidCharacters(scopes: RegisteredScope[], characters: EveCharacter[]) {
  return characters.filter((character) => {
    if (character.token_invalid) {
      return true
    }

    const scopeSet = parseScopeSet(character.scopes)
    return scopes.some((scope) => scope.required && !scopeSet.has(scope.scope))
  })
}

export function calculateScopeCoverage(
  scopes: RegisteredScope[],
  character: EveCharacter | null
): ScopeCoverage {
  const requiredScopes = scopes.filter((scope) => scope.required)

  if (!character) {
    return {
      scopeRows: [],
      requiredScopes,
      grantedRequiredCount: 0,
      hasMissingRequiredScopes: false,
    }
  }

  const authorizedScopes = character.token_invalid
    ? new Set<string>()
    : parseScopeSet(character.scopes)
  const scopeRows = scopes.map((scope) => ({
    ...scope,
    authorized: authorizedScopes.has(scope.scope),
  }))
  const grantedRequiredCount = requiredScopes.filter((scope) => authorizedScopes.has(scope.scope)).length

  return {
    scopeRows,
    requiredScopes,
    grantedRequiredCount,
    hasMissingRequiredScopes: scopeRows.some((row) => row.required && !row.authorized),
  }
}
