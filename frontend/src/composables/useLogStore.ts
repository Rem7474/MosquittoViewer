import { computed, ref, shallowRef } from 'vue'
import type { LogEntry, LogFilters } from '../types/log'

export function useLogStore(maxEntries = 1000) {
  const entries = shallowRef<LogEntry[]>([])
  const filters = ref<LogFilters>({
    level: 'ALL',
    search: '',
    clientId: '',
    topic: '',
  })

  function push(entry: LogEntry) {
    entries.value = [...entries.value, entry].slice(-maxEntries)
  }

  function clear() {
    entries.value = []
  }

  const filteredEntries = computed(() => {
    const f = filters.value
    const s = f.search.toLowerCase()
    return entries.value.filter((e) => {
      if (f.level !== 'ALL' && e.level !== f.level) return false
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
    const first = new Date(entries.value[0].timestamp).getTime()
    const last = new Date(entries.value[entries.value.length - 1].timestamp).getTime()
    const seconds = Math.max((last - first) / 1000, 1)
    return Number((entries.value.length / seconds).toFixed(2))
  })

  function exportJSON() {
    downloadBlob(JSON.stringify(filteredEntries.value, null, 2), 'application/json', 'logs.json')
  }

  function exportCSV() {
    const header = 'id,timestamp,level,message,client_id,topic,plugin,raw'
    const rows = filteredEntries.value.map((e) => [
      e.id,
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
