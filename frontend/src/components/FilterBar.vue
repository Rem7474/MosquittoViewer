<template>
  <div class="filterbar">
    <div class="levels">
      <button
        v-for="level in levels"
        :key="level"
        :class="['chip', { active: modelValue.level === level }]"
        @click="$emit('update:modelValue', { ...modelValue, level })"
      >
        {{ level }}
      </button>
    </div>

    <select
      v-if="sources.length > 0"
      class="source-select"
      :value="modelValue.source"
      @change="$emit('update:modelValue', { ...modelValue, source: ($event.target as HTMLSelectElement).value })"
    >
      <option value="">All sources</option>
      <option v-for="s in sources" :key="s.name" :value="s.name">{{ s.name }}</option>
    </select>

    <input
      class="search"
      type="text"
      placeholder="Search message..."
      :value="modelValue.search"
      @input="$emit('update:modelValue', { ...modelValue, search: ($event.target as HTMLInputElement).value })"
    />

    <input
      class="search"
      type="text"
      placeholder="Client ID"
      :value="modelValue.clientId"
      @input="$emit('update:modelValue', { ...modelValue, clientId: ($event.target as HTMLInputElement).value })"
    />

    <input
      class="search"
      type="text"
      placeholder="Topic"
      :value="modelValue.topic"
      @input="$emit('update:modelValue', { ...modelValue, topic: ($event.target as HTMLInputElement).value })"
    />

    <button class="action" @click="$emit('toggle-pause')">{{ paused ? 'Resume' : 'Pause' }}</button>
    <button class="action" @click="$emit('clear')">Clear</button>
    <button class="action" @click="$emit('export-json')">Export JSON</button>
    <button class="action" @click="$emit('export-csv')">Export CSV</button>
  </div>
</template>

<script setup lang="ts">
import type { LogFilters, LogSource } from '../types/log'

defineProps<{
  modelValue: LogFilters
  paused: boolean
  sources: LogSource[]
}>()

defineEmits<{
  (e: 'update:modelValue', value: LogFilters): void
  (e: 'toggle-pause'): void
  (e: 'clear'): void
  (e: 'export-json'): void
  (e: 'export-csv'): void
}>()

const levels: LogFilters['level'][] = ['ALL', 'INFO', 'WARN', 'ERROR', 'DEBUG']
</script>
