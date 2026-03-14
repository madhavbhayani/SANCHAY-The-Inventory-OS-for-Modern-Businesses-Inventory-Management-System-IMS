const BASE = '/api'

/**
 * Internal: POST JSON to a backend endpoint.
 * Throws with the server's error message on non-2xx responses.
 */
async function post(path, body) {
  const res = await fetch(`${BASE}${path}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  const data = await res.json()
  if (!res.ok) throw new Error(data.error || `Request failed (${res.status})`)
  return data
}

/**
 * Register a new user.
 * @param {{ loginId: string, email: string, password: string }} params
 * @returns {Promise<{ token: string, user: object }>}
 */
export function apiSignup({ loginId, email, password }) {
  return post('/auth/signup', { login_id: loginId, email, password })
}

/**
 * Log in with login ID or email.
 * @param {{ identifier: string, password: string }} params
 * @returns {Promise<{ token: string, user: object }>}
 */
export function apiLogin({ identifier, password }) {
  return post('/auth/login', { identifier, password })
}

/** Persist JWT + user info to localStorage after successful auth. */
export function saveSession(token, user) {
  localStorage.setItem('sanchay_token', token)
  localStorage.setItem('sanchay_user', JSON.stringify(user))
}

/** Remove stored session (logout). */
export function clearSession() {
  localStorage.removeItem('sanchay_token')
  localStorage.removeItem('sanchay_user')
}

/** Return the stored JWT or null. */
export function getToken() {
  return localStorage.getItem('sanchay_token')
}

/** Return the stored user object or null. */
export function getUser() {
  const raw = localStorage.getItem('sanchay_user')
  return raw ? JSON.parse(raw) : null
}
