// Folders first, then case-insensitive by name. Returns a new sorted array.
export function sortEntries(nodes) {
  return [...(nodes || [])].sort((a, b) => {
    if (a.isDir !== b.isDir) return a.isDir ? -1 : 1
    return a.name.localeCompare(b.name, undefined, { sensitivity: 'base' })
  })
}
