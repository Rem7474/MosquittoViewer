<template>
  <section class="viewer">
    <header class="topbar">
      <div class="brand">⬡ MosquittoViewer</div>
      <ConnectionStatus :connected="wsConnected" />
      <div class="stats">
        <span>Total: {{ totalCount }}</span>
        <span>Rate: {{ ratePerSecond }}/s</span>
        <span class="err-count">E: {{ countByLevel.ERROR }}</span>
      </div>
      <button class="logout" @click="logout">Logout</button>
    </header>

    <FilterBar
      v-model="filters"
      :paused="paused"
      :sources="availableSources"
      @toggle-pause="togglePause"
      @clear="clear"
      @export-json="exportJSON"
      @export-csv="exportCSV"
    />

    <div class="content-grid">
      <div class="log-list" ref="listRef" @scroll="onScroll">
        <DynamicScroller
          v-if="filteredEntries.length > 500"
          class="scroller"
          :items="filteredEntries"
          :min-item-size="32"
          key-field="id"
        >
          <template #default="{ item, active }">
            <DynamicScrollerItem :item="item" :active="active" :size-dependencies="[(item as LogEntry).message]">
              <LogRow
                :entry="item as LogEntry"
                :selected="selected?.id === (item as LogEntry).id"
                :show-source="availableSources.length > 1"
                @select="onSelect"
              />
            </DynamicScrollerItem>
          </template>
        </DynamicScroller>

        <template v-else>
          <LogRow
            v-for="entry in filteredEntries"
            :key="entry.id"
            :entry="entry"
            :selected="selected?.id === entry.id"
            :show-source="availableSources.length > 1"
            @select="onSelect"
          />
        </template>
      </div>

      <aside class="detail-panel">
        <h3>Detail</h3>
        <pre v-if="selected">{{ JSON.stringify(selected, null, 2) }}</pre>
        <p v-else>Select a log row to inspect details.</p>
      </aside>
    </div>

    <footer class="statusbar">
      <span>{{ filteredEntries.length }} / {{ totalCount }} entrées en mémoire</span>
      <span v-if="filters.source">{{ currentSourcePath }}</span>
      <span v-else>{{ availableSources.length }} source(s)</span>
    </footer>
  </section>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { DynamicScroller, DynamicScrollerItem } from 'vue-virtual-scroller'
import ConnectionStatus from './ConnectionStatus.vue'
import FilterBar from './FilterBar.vue'
import LogRow from './LogRow.vue'
import { useAuth } from '../composables/useAuth'
import { useLogStore } from '../composables/useLogStore'
import { useWebSocket } from '../composables/useWebSocket'
import type { LogEntry, LogSource } from '../types/log'

const { logout, accessToken, authFetch } = useAuth()

const MAX_LOGS = 500
const { filters, filteredEntries, countByLevel, totalCount, ratePerSecond, push, replaceEntries, clear, exportJSON, exportCSV } =
  useLogStore(MAX_LOGS)

const { connected, paused, connect, disconnect, pause, resume, on } = useWebSocket(() => accessToken.value, MAX_LOGS)

const selected = ref<LogEntry | null>(null)
const wsConnected = computed(() => connected.value)
const listRef = ref<HTMLElement | null>(null)
const autoScroll = ref(true)
const availableSources = ref<LogSource[]>([])

const currentSourcePath = computed(() => {
  if (!filters.value.source) return ''
  return availableSources.value.find((s) => s.name === filters.value.source)?.path ?? filters.value.source
})

// Push new WebSocket entries into the store (unless paused).
on('entry', (entry) => {
  if (!paused.value) {
    push(entry as LogEntry)
  }
})

// Initial REST load – only runs once on mount.
async function syncLogs() {
  try {
    const source = filters.value.source
    const url = source ? `/api/logs?limit=${MAX_LOGS}&source=${encodeURIComponent(source)}` : `/api/logs?limit=${MAX_LOGS}`
    const res = await authFetch(url)
    const data = (await res.json()) as { data: LogEntry[] }
    replaceEntries(data.data)
  } catch {
    // Ignore transient errors on mount.
  }
}

async function fetchSources() {
  try {
    const res = await authFetch('/api/sources')
    const data = (await res.json()) as { sources: LogSource[] }
    availableSources.value = data.sources
  } catch {
    availableSources.value = []
  }
}

function togglePause() {
  if (paused.value) {
    resume()
  } else {
    pause()
  }
}

function onSelect(entry: LogEntry) {
  selected.value = entry
}

function onScroll() {
  if (!listRef.value) return
  const threshold = 24
  autoScroll.value = listRef.value.scrollTop < threshold
}

watch(filteredEntries, () => {
  if (!paused.value && autoScroll.value && listRef.value) {
    requestAnimationFrame(() => {
      if (listRef.value) listRef.value.scrollTop = 0
    })
  }
})

onMounted(async () => {
  await fetchSources()
  await syncLogs()
  connect()
})

onBeforeUnmount(() => {
  disconnect()
})
</script>
