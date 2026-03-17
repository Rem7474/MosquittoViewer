import { ref } from 'vue'
import type { LogEntry } from '../types/log'

export type WebSocketEventName = 'connected' | 'disconnected' | 'entry'

export function useWebSocket(tokenProvider: () => string, maxEntries = 1000) {
  const entries = ref<LogEntry[]>([])
  const connected = ref(false)
  const paused = ref(false)

  const listeners: Record<WebSocketEventName, Array<(payload?: unknown) => void>> = {
    connected: [],
    disconnected: [],
    entry: [],
  }

  let ws: WebSocket | null = null
  let reconnectDelay = 1000
  let reconnectTimer: number | null = null
  let heartbeatTimer: number | null = null

  function emit(event: WebSocketEventName, payload?: unknown) {
    listeners[event].forEach((fn) => fn(payload))
  }

  function on(event: WebSocketEventName, callback: (payload?: unknown) => void) {
    listeners[event].push(callback)
  }

  function pause() {
    paused.value = true
  }

  function resume() {
    paused.value = false
  }

  function scheduleReconnect() {
    if (reconnectTimer != null) return
    reconnectTimer = window.setTimeout(() => {
      reconnectTimer = null
      connect()
    }, reconnectDelay)
    reconnectDelay = Math.min(reconnectDelay * 2, 30000)
  }

  function startHeartbeat() {
    if (!ws) return
    heartbeatTimer = window.setInterval(() => {
      if (ws?.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: 'ping', at: Date.now() }))
      }
    }, 25000)
  }

  function stopHeartbeat() {
    if (heartbeatTimer != null) {
      window.clearInterval(heartbeatTimer)
      heartbeatTimer = null
    }
  }

  function connect() {
    const token = tokenProvider()
    if (!token) return

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const url = `${protocol}//${window.location.host}/api/ws?token=${encodeURIComponent(token)}`
    ws = new WebSocket(url)

    ws.onopen = () => {
      connected.value = true
      reconnectDelay = 1000
      emit('connected')
      startHeartbeat()
    }

    ws.onmessage = (event) => {
      try {
        const entry = JSON.parse(event.data) as LogEntry
        if (!paused.value) {
          entries.value = [...entries.value, entry].slice(-maxEntries)
        }
        emit('entry', entry)
      } catch {
        // Ignore malformed payloads.
      }
    }

    ws.onclose = () => {
      connected.value = false
      emit('disconnected')
      stopHeartbeat()
      scheduleReconnect()
    }

    ws.onerror = () => {
      connected.value = false
      stopHeartbeat()
      ws?.close()
    }
  }

  function disconnect() {
    stopHeartbeat()
    if (reconnectTimer != null) {
      window.clearTimeout(reconnectTimer)
      reconnectTimer = null
    }
    ws?.close()
    ws = null
  }

  return {
    entries,
    connected,
    paused,
    on,
    connect,
    disconnect,
    pause,
    resume,
  }
}
