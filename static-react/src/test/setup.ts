import '@testing-library/jest-dom/vitest'

// Existing page fixtures isolate sidebar eligibility from their ordered API
// responses. Admissions tests override this mock to exercise their own context.
vi.mock('@/hooks/use-admission-context', () => ({
  useAdmissionContext: () => ({
    context: null,
    loading: false,
    error: false,
    reload: async () => {},
  }),
}))

Object.defineProperty(window, 'matchMedia', {
  writable: true,
  value: (query: string) => ({
    matches: false,
    media: query,
    onchange: null,
    addEventListener: () => undefined,
    removeEventListener: () => undefined,
    addListener: () => undefined,
    removeListener: () => undefined,
    dispatchEvent: () => false,
  }),
})
import { configure } from '@testing-library/react'

configure({ asyncUtilTimeout: 10_000 })

// Radix Select probes the Pointer Events capture APIs when a trigger is
// activated. jsdom does not implement them, so provide the browser no-op
// behavior for interaction tests.
if (!HTMLElement.prototype.hasPointerCapture) {
  Object.assign(HTMLElement.prototype, {
    hasPointerCapture: () => false,
    releasePointerCapture: () => undefined,
    setPointerCapture: () => undefined,
  })
}

if (!HTMLElement.prototype.scrollIntoView) {
  HTMLElement.prototype.scrollIntoView = () => undefined
}
