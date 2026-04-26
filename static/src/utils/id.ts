let localUidCounter = 0

function nextLocalUidCounter() {
  localUidCounter = (localUidCounter + 1) % Number.MAX_SAFE_INTEGER
  return localUidCounter
}

export function createLocalUid(prefix = 'local'): string {
  const cryptoRef = globalThis.crypto
  if (cryptoRef?.randomUUID) {
    return `${prefix}_${cryptoRef.randomUUID()}`
  }

  return `${prefix}_${Date.now()}_${nextLocalUidCounter()}`
}
