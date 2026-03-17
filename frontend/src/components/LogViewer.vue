<template>
  <section class="viewer">
    <header class="topbar">
      <div class="brand">⬡ MosquittoViewer</div>
      <ConnectionStatus :connected="wsConnected" />
      <div class="stats">
        <span>Total: {{ totalCount }}</span>
        <span>Rate: {{ ratePerSecond }}/s</span>
        <span>E: {{ countByLevel.ERROR }}</span>
      </div>
      <button class="logout" @click="logout">Logout</button>
    </header>

    <FilterBar
      v-model="filters"
      :paused="paused"
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
              <LogRow :entry="item as LogEntry" :selected="selected?.id === (item as LogEntry).id" @select="onSelect" />
            </DynamicScrollerItem>
          </template>
        </DynamicScroller>

        <template v-else>
          <LogRow
            v-for="entry in filteredEntries"
            :key="entry.id"
            :entry="entry"
            :selected="selected?.id === entry.id"
            @select="onSelect"
          />
        </template>
      </div>

      <aside class="detail-panel">
        <h3>Detail</h3>
        <pre v-if="selected">{{ selected }}</pre>
        <p v-else>Select a log row to inspect details.</p>
      </aside>
    </div>

    <footer class="statusbar">
      <span>{{ filteredEntries.length }} logs / {{ totalCount }} en memoire</span>
      <span>{{ logPath }}</span>
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
import type { LogEntry } from '../types/log'

const { logout, accessToken, authFetch } = useAuth()
const { filters, filteredEntries, countByLevel, totalCount, ratePerSecond, push, replaceEntries, clear, exportJSON, exportCSV } = useLogStore(1000)
const { connected, paused, connect, disconnect, pause, resume, on } = useWebSocket(() => accessToken.value, 1000)

const selected = ref<LogEntry | null>(null)
const wsConnected = computed(() => connected.value)
const listRef = ref<HTMLElement | null>(null)
const autoScroll = ref(true)
const logPath = '/var/log/mosquitto/mosquitto.log'
let pollTimer: number | null = null

on('entry', (entry) => {
  if (!paused.value) {
    push(entry as LogEntry)
  }
})

async function syncLogs() {
  if (paused.value) return
  const res = await authFetch('/api/logs?limit=200')
  const data = await res.json() as { data: LogEntry[] }
  replaceEntries(data.data)
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
      if (listRef.value) {
        listRef.value.scrollTop = 0
      }
    })
  }
})

onMounted(async () => {
  await syncLogs()
  connect()
  pollTimer = window.setInterval(() => {
    void syncLogs()
  }, 1000)
})

onBeforeUnmount(() => {
  if (pollTimer != null) {
    window.clearInterval(pollTimer)
    pollTimer = null
  }
  disconnect()
})
</script>
