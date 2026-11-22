import type { ToolBookmark } from '@/types/api/tool-bookmark'

// Preserve API order: groups follow their first bookmark; bookmarks keep their relative order.
export function groupToolBookmarks(bookmarks: ToolBookmark[]) {
  const groups = new Map<string, ToolBookmark[]>()
  for (const bookmark of bookmarks) {
    const name = bookmark.group_name
    const items = groups.get(name)
    if (items) items.push(bookmark)
    else groups.set(name, [bookmark])
  }
  return Array.from(groups, ([name, items]) => ({ name, items }))
}
