<template>
  <div class="log-row" :class="[`lvl-${entry.level.toLowerCase()}`, { selected }]" @click="$emit('select', entry)">
    <span class="ts">{{ ts }}</span>
    <span class="lvl">{{ entry.level }}</span>
    <span v-if="showSource && entry.source" class="src">{{ entry.source }}</span>
    <span class="msg">{{ entry.message }}</span>
  </div>
</template>

<script setup lang="ts">
import dayjs from 'dayjs'
import { computed } from 'vue'
import type { LogEntry } from '../types/log'

const props = defineProps<{
  entry: LogEntry
  selected: boolean
  showSource?: boolean
}>()

defineEmits<{
  (e: 'select', value: LogEntry): void
}>()

const ts = computed(() => dayjs(props.entry.timestamp).format('HH:mm:ss.SSS'))
</script>
