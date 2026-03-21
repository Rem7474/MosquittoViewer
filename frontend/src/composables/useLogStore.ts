import { computed, ref, shallowRef } from 'vue'
import type { LogEntry, LogFilters } from '../types/log'

export function useLogStore(maxEntries = 1000) {
  const entries = shallowRef<LogEntry[]>([])

  // O(1) duplicate detection – tracks source:id of every entry currently in the buffer.
  const entryIds = new Set<string>()

  const filters = ref<LogFilters>({
    level: 'ALL',
    source: '',
    search: '',
    clientId: '',
    topic: '',
  })

  function entryKey(e: LogEntry): string {
    return `${e.source ?? ''}:${e.id}`
  }

  /** Prepend a single new entry (from WebSocket). Silently drops duplicates. */
  function push(entry: LogEntry) {
    const key = entryKey(entry)
    if (entryIds.has(key)) return

    entryIds.add(key)
    const next = [entry, ...entries.value]
    // When the buffer is full, evict the oldest entry and remove its key.
    if (next.length > maxEntries) {
      const evicted = next[next.length - 1]
      entryIds.delete(entryKey(evicted))
      next.length = maxEntries
    }
    entries.value = next
  }

  /**
   * Merge REST history with any entries already delivered by WebSocket
   * (to avoid losing entries that arrived between connect() and the REST response).
   */
  function replaceEntries(restEntries: LogEntry[]) {
    const unique = new Map<string, LogEntry>()

    // Seed with current buffer (WS-delivered entries, may include entries newer than REST).
    for (const e of entries.value) {
      unique.set(entryKey(e), e)
    }
    // REST data is canonical – overwrite duplicates.
    for (const e of restEntries) {
      unique.set(entryKey(e), e)
    }

    const sorted = [...unique.values()]
      .sort((a, b) => {
        if (a.id !== b.id) return b.id - a.id
        return new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime()
      })
      .slice(0, maxEntries)

    // Rebuild ID set to match the new buffer exactly.
    entryIds.clear()
    for (const e of sorted) entryIds.add(entryKey(e))
    entries.value = sorted
  }

  function clear() {
    entries.value = []
    entryIds.clear()
  }

  const filteredEntries = computed(() => {
    const f = filters.value
    const s = f.search.toLowerCase()
    return entries.value.filter((e) => {
      if (f.level !== 'ALL' && e.level !== f.level) return false
      if (f.source && e.source !== f.source) return false
      if (f.clientId && e.client_id !== f.clientId) return false
      if (f.topic && !e.topic?.toLowerCase().includes(f.topic.toLowerCase())) return false
      if (s && !(`${e.message} ${e.raw}`.toLowerCase().includes(s))) return false
      return true
    })
  })

  const countByLevel = computed(() => ({
    INFO: entries.value.filter((e) => e.level === 'INFO').length,
    WARN: entries.value.filter((e) => e.level === 'WARN').length,
    ERROR: entries.value.filter((e) => e.level === 'ERROR').length,
    DEBUG: entries.value.filter((e) => e.level === 'DEBUG').length,
  }))

  const totalCount = computed(() => entries.value.length)

  const ratePerSecond = computed(() => {
    if (entries.value.length < 2) return 0
    const newest = new Date(entries.value[0].timestamp).getTime()
    const oldest = new Date(entries.value[entries.value.length - 1].timestamp).getTime()
    const seconds = Math.max((newest - oldest) / 1000, 1)
    return Number((entries.value.length / seconds).toFixed(2))
  })

  function exportJSON() {
    downloadBlob(JSON.stringify(filteredEntries.value, null, 2), 'application/json', 'logs.json')
  }

  function exportCSV() {
    const header = 'id,source,timestamp,level,message,client_id,topic,plugin,raw'
    const rows = filteredEntries.value.map((e) => [
      e.id,
      csvCell(e.source ?? ''),
      csvCell(e.timestamp),
      csvCell(e.level),
      csvCell(e.message),
      csvCell(e.client_id ?? ''),
      csvCell(e.topic ?? ''),
      csvCell(e.plugin ?? ''),
      csvCell(e.raw),
    ].join(','))
    downloadBlob([header, ...rows].join('\n'), 'text/csv', 'logs.csv')
  }

  return {
    entries,
    filters,
    filteredEntries,
    countByLevel,
    totalCount,
    ratePerSecond,
    push,
    replaceEntries,
    clear,
    exportJSON,
    exportCSV,
  }
}

function csvCell(value: string | number): string {
  const text = String(value).replaceAll('"', '""')
  return `"${text}"`
}

function downloadBlob(content: string, contentType: string, filename: string) {
  const blob = new Blob([content], { type: contentType })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = filename
  link.click()
  URL.revokeObjectURL(url)
}
