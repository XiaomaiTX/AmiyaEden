/** Keeps external navigation testable without asking jsdom to implement it. */
export function navigateExternal(url: string) {
  window.location.assign(url)
}
