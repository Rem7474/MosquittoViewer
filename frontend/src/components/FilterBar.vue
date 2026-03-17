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
import type { LogFilters } from '../types/log'

defineProps<{
  modelValue: LogFilters
  paused: boolean
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
