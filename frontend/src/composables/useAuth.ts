import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'

const ACCESS_KEY = 'mv_access_token'
const REFRESH_KEY = 'mv_refresh_token'
const USERNAME_KEY = 'mv_username'

const accessToken = ref(localStorage.getItem(ACCESS_KEY) ?? '')
const refreshToken = ref(localStorage.getItem(REFRESH_KEY) ?? '')
const username = ref(localStorage.getItem(USERNAME_KEY) ?? '')

function decodePayload(token: string): Record<string, unknown> | null {
  try {
    const base64 = token.split('.')[1]
    if (!base64) return null
    const normalized = base64.replace(/-/g, '+').replace(/_/g, '/')
    const json = atob(normalized)
    return JSON.parse(json) as Record<string, unknown>
  } catch {
    return null
  }
}

function isExpired(token: string): boolean {
  const payload = decodePayload(token)
  const exp = Number(payload?.exp ?? 0)
  if (!exp) return true
  return Date.now() >= exp * 1000
}

async function refreshTokens(): Promise<boolean> {
  if (!refreshToken.value) return false
  const res = await fetch('/api/auth/refresh', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ refresh_token: refreshToken.value }),
  })
  if (!res.ok) return false
  const data = await res.json() as { access_token: string; refresh_token: string; username: string }
  setTokens(data.access_token, data.refresh_token, data.username)
  return true
}

function setTokens(access: string, refresh: string, user: string) {
  accessToken.value = access
  refreshToken.value = refresh
  username.value = user
  localStorage.setItem(ACCESS_KEY, access)
  localStorage.setItem(REFRESH_KEY, refresh)
  localStorage.setItem(USERNAME_KEY, user)
}

export function useAuth() {
  const router = useRouter()

  async function login(user: string, password: string): Promise<boolean> {
    const res = await fetch('/api/auth/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username: user, password }),
    })
    if (!res.ok) return false
    const data = await res.json() as { access_token: string; refresh_token: string; username: string }
    setTokens(data.access_token, data.refresh_token, data.username)
    return true
  }

  function logout() {
    accessToken.value = ''
    refreshToken.value = ''
    username.value = ''
    localStorage.removeItem(ACCESS_KEY)
    localStorage.removeItem(REFRESH_KEY)
    localStorage.removeItem(USERNAME_KEY)
    router.push('/login')
  }

  async function authFetch(input: RequestInfo | URL, init: RequestInit = {}): Promise<Response> {
    if (accessToken.value && isExpired(accessToken.value)) {
      const refreshed = await refreshTokens()
      if (!refreshed) {
        logout()
        throw new Error('session expired')
      }
    }

    const headers = new Headers(init.headers)
    if (accessToken.value) {
      headers.set('Authorization', `Bearer ${accessToken.value}`)
    }

    let res = await fetch(input, { ...init, headers })
    if (res.status === 401) {
      const refreshed = await refreshTokens()
      if (!refreshed) {
        logout()
        throw new Error('unauthorized')
      }
      headers.set('Authorization', `Bearer ${accessToken.value}`)
      res = await fetch(input, { ...init, headers })
      if (res.status === 401) {
        logout()
        throw new Error('unauthorized')
      }
    }
    return res
  }

  return {
    login,
    logout,
    authFetch,
    username: computed(() => username.value),
    accessToken: computed(() => accessToken.value),
    isAuthenticated: computed(() => Boolean(accessToken.value) && !isExpired(accessToken.value)),
  }
}
