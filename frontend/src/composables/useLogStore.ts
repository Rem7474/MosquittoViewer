import { computed, ref, shallowRef } from 'vue'
import type { LogEntry, LogFilters } from '../types/log'

export function useLogStore(maxEntries = 1000) {
  const entries = shallowRef<LogEntry[]>([])

  // Highest ID seen per source. push() rejects anything at or below this.
  // Keyed by source name ('' if absent).
  const watermark = new Map<string, number>()

  const filters = ref<LogFilters>({
    level: 'ALL',
    source: '',
    search: '',
    clientId: '',
    topic: '',
  })

  /** Prepend a live entry from WebSocket. Drops entries already loaded via REST. */
  function push(entry: LogEntry) {
    const src = entry.source ?? ''
    const hi = watermark.get(src) ?? -1
    if (entry.id <= hi) return   // already in buffer from REST load

    watermark.set(src, entry.id)
    const next = [entry, ...entries.value]
    if (next.length > maxEntries) next.length = maxEntries
    entries.value = next
  }

  /**
   * Replace the buffer with REST history.
   * Any live entries already pushed by WebSocket (id > REST max) are preserved.
   */
  function replaceEntries(restEntries: LogEntry[]) {
    const unique = new Map<string, LogEntry>()

    // Keep WS-delivered entries that are newer than anything in the REST payload.
    for (const e of entries.value) unique.set(`${e.source ?? ''}:${e.id}`, e)
    // REST data is canonical for its range.
    for (const e of restEntries) unique.set(`${e.source ?? ''}:${e.id}`, e)

    const sorted = [...unique.values()]
      .sort((a, b) => b.id - a.id || new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime())
      .slice(0, maxEntries)

    // Rebuild watermark from the merged result so push() knows what's already present.
    watermark.clear()
    for (const e of sorted) {
      const src = e.source ?? ''
      if ((watermark.get(src) ?? -1) < e.id) watermark.set(src, e.id)
    }

    entries.value = sorted
  }

  function clear() {
    entries.value = []
    watermark.clear()
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
    INFO:  entries.value.filter((e) => e.level === 'INFO').length,
    WARN:  entries.value.filter((e) => e.level === 'WARN').length,
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

  return { entries, filters, filteredEntries, countByLevel, totalCount, ratePerSecond, push, replaceEntries, clear, exportJSON, exportCSV }
}

function csvCell(value: string | number): string {
  return `"${String(value).replaceAll('"', '""')}"`
}

function downloadBlob(content: string, contentType: string, filename: string) {
  const blob = new Blob([content], { type: contentType })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  a.click()
  URL.revokeObjectURL(url)
}
