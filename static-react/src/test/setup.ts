import '@testing-library/jest-dom/vitest'

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
