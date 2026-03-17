<template>
  <span class="connection" :class="connected ? 'ok' : 'down'">
    <span class="dot" />
    {{ connected ? 'LIVE' : 'OFFLINE' }}
  </span>
</template>

<script setup lang="ts">
defineProps<{ connected: boolean }>()
</script>
