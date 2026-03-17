<template>
  <form class="login-card" @submit.prevent="submit">
    <h1>MosquittoViewer</h1>
    <p class="subtitle">Secure real-time MQTT log dashboard</p>

    <label>
      Username
      <input v-model="username" type="text" required autocomplete="username" />
    </label>

    <label>
      Password
      <input v-model="password" type="password" required autocomplete="current-password" />
    </label>

    <button type="submit" :disabled="loading">{{ loading ? 'Signing in...' : 'Login' }}</button>
    <p v-if="error" class="error">{{ error }}</p>
  </form>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuth } from '../composables/useAuth'

const router = useRouter()
const { login } = useAuth()

const username = ref('admin')
const password = ref('')
const loading = ref(false)
const error = ref('')

async function submit() {
  loading.value = true
  error.value = ''
  const ok = await login(username.value, password.value)
  loading.value = false
  if (!ok) {
    error.value = 'Invalid credentials'
    return
  }
  router.push('/')
}
</script>
