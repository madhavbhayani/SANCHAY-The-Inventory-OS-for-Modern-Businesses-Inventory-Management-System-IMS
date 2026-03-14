const BASE = '/api'

async function request(path, { method = 'GET', body, auth = false } = {}) {
  const headers = { 'Content-Type': 'application/json' }

  if (auth) {
    const token = getToken()
    if (!token) {
      throw new Error('Session expired. Please log in again.')
    }
    headers.Authorization = `Bearer ${token}`
  }

  const res = await fetch(`${BASE}${path}`, {
    method,
    headers,
    ...(body !== undefined ? { body: JSON.stringify(body) } : {}),
  })

  let data = {}
  try {
    data = await res.json()
  } catch {
    data = {}
  }

  if (!res.ok) throw new Error(data.error || `Request failed (${res.status})`)
  return data
}

async function post(path, body) {
  return request(path, { method: 'POST', body })
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

/** Request forgot-password OTP for a registered email. */
export function apiRequestForgotPasswordOtp({ email }) {
  return post('/auth/forgot-password/request', { email })
}

/** Verify forgot-password OTP for email. */
export function apiVerifyForgotPasswordOtp({ email, otp }) {
  return post('/auth/forgot-password/verify', { email, otp })
}

/** Reset password after OTP verification. */
export function apiResetForgotPassword({ email, newPassword }) {
  return post('/auth/forgot-password/reset', {
    email,
    new_password: newPassword,
  })
}

/** Get all data required to render the Settings page. */
export function apiGetSettingsOverview() {
  return request('/settings', { auth: true })
}

/** Fetch dashboard counters and chart datasets. */
export function apiGetDashboardOverview() {
  return request('/dashboard/overview', { auth: true })
}

/** Create a warehouse entry. */
export function apiCreateWarehouse({ name, shortCode, address, description }) {
  return request('/settings/warehouses', {
    method: 'POST',
    auth: true,
    body: {
      name,
      short_code: shortCode,
      address,
      description,
    },
  })
}

/** Create a location and attach one or many warehouses. */
export function apiCreateLocation({ name, shortCode, warehouseIds }) {
  return request('/settings/locations', {
    method: 'POST',
    auth: true,
    body: {
      name,
      short_code: shortCode,
      warehouse_ids: warehouseIds,
    },
  })
}

/** Change the authenticated user's password. */
export function apiChangePassword({ currentPassword, newPassword }) {
  return request('/settings/change-password', {
    method: 'POST',
    auth: true,
    body: {
      current_password: currentPassword,
      new_password: newPassword,
    },
  })
}

/** Fetch categories and location options required for stock operations. */
export function apiGetStockMeta() {
  return request('/stocks/meta', { auth: true })
}

/** Create a product category in stocks.categories. */
export function apiCreateStockCategory({ name, description }) {
  return request('/stocks/categories', {
    method: 'POST',
    auth: true,
    body: {
      name,
      description,
    },
  })
}

/** Query products with server-side search. */
export function apiListStockProducts({ query = '', limit = 120 } = {}) {
  const params = new URLSearchParams()
  if (query.trim()) params.set('q', query.trim())
  params.set('limit', String(limit))
  const suffix = params.toString() ? `?${params.toString()}` : ''
  return request(`/stocks/products${suffix}`, { auth: true })
}

/** Create a product in stocks.products. */
export function apiCreateStockProduct(payload) {
  return request('/stocks/products', {
    method: 'POST',
    auth: true,
    body: payload,
  })
}

/** Update an existing product by id. */
export function apiUpdateStockProduct(productId, payload) {
  return request(`/stocks/products/${productId}`, {
    method: 'PUT',
    auth: true,
    body: payload,
  })
}

/** Delete a product by id. */
export function apiDeleteStockProduct(productId) {
  return request(`/stocks/products/${productId}`, {
    method: 'DELETE',
    auth: true,
  })
}

/** Fetch location and product options required for operations forms. */
export function apiGetOperationsMeta() {
  return request('/operations/meta', { auth: true })
}

/** List receipt orders (operation type IN). */
export function apiListReceiptOrders({ limit = 120, query = '' } = {}) {
  const params = new URLSearchParams()
  params.set('limit', String(limit))
  if (String(query || '').trim()) {
    params.set('q', String(query).trim())
  }
  return request(`/operations/receipts?${params.toString()}`, { auth: true })
}

/** Create a new receipt order (operation type IN). */
export function apiCreateReceiptOrder(payload) {
  return request('/operations/receipts', {
    method: 'POST',
    auth: true,
    body: payload,
  })
}

/** List delivery orders (operation type OUT). */
export function apiListDeliveryOrders({ limit = 120, query = '' } = {}) {
  const params = new URLSearchParams()
  params.set('limit', String(limit))
  if (String(query || '').trim()) {
    params.set('q', String(query).trim())
  }
  return request(`/operations/delivery?${params.toString()}`, { auth: true })
}

/** Create a new delivery order (operation type OUT). */
export function apiCreateDeliveryOrder(payload) {
  return request('/operations/delivery', {
    method: 'POST',
    auth: true,
    body: payload,
  })
}

/** Fetch operations adjustments overview data. */
export function apiGetAdjustmentsOverview({ limit = 320 } = {}) {
  const params = new URLSearchParams()
  params.set('limit', String(limit))
  return request(`/operations/adjustments?${params.toString()}`, { auth: true })
}

/** Move free-to-use stock between locations. */
export function apiTransferAdjustmentStock(payload) {
  return request('/operations/adjustments/transfer', {
    method: 'POST',
    auth: true,
    body: payload,
  })
}

/** Correct free-to-use stock quantity at a location. */
export function apiAdjustStockQuantity(payload) {
  return request('/operations/adjustments/quantity', {
    method: 'POST',
    auth: true,
    body: payload,
  })
}

/** Query stock ledger move history with optional filters. */
export function apiListMoveHistory({
  limit = 200,
  eventType = '',
  status = '',
  query = '',
  fromDate = '',
  toDate = '',
} = {}) {
  const params = new URLSearchParams()
  params.set('limit', String(limit))
  if (String(eventType || '').trim()) {
    params.set('event_type', String(eventType).trim())
  }
  if (String(status || '').trim()) {
    params.set('status', String(status).trim())
  }
  if (String(query || '').trim()) {
    params.set('q', String(query).trim())
  }
  if (String(fromDate || '').trim()) {
    params.set('from_date', String(fromDate).trim())
  }
  if (String(toDate || '').trim()) {
    params.set('to_date', String(toDate).trim())
  }

  return request(`/move-history?${params.toString()}`, { auth: true })
}

/** Delete an operations order by id. */
export function apiDeleteOperationOrder(orderId) {
  return request(`/operations/orders/${orderId}`, {
    method: 'DELETE',
    auth: true,
  })
}

/** Fetch operation detail by operation type and reference number. */
export function apiGetOperationOrderDetail(operationType, referenceNumber) {
  const safeType = String(operationType || '').toUpperCase() === 'OUT' ? 'OUT' : 'IN'
  return request(`/operations/orders/${safeType}/${encodeURIComponent(referenceNumber || '')}`, {
    auth: true,
  })
}

/** Update operation detail by operation type and reference number. */
export function apiUpdateOperationOrderDetail(operationType, referenceNumber, payload) {
  const safeType = String(operationType || '').toUpperCase() === 'OUT' ? 'OUT' : 'IN'
  return request(`/operations/orders/${safeType}/${encodeURIComponent(referenceNumber || '')}`, {
    method: 'PUT',
    auth: true,
    body: payload,
  })
}

/** Validate operation order lines against available stock. */
export function apiValidateOperationOrder(operationType, referenceNumber) {
  const safeType = String(operationType || '').toUpperCase() === 'OUT' ? 'OUT' : 'IN'
  return request(`/operations/orders/${safeType}/${encodeURIComponent(referenceNumber || '')}/validate`, {
    method: 'POST',
    auth: true,
    body: {},
  })
}

/** Cancel operation order by operation type and reference number. */
export function apiCancelOperationOrder(operationType, referenceNumber) {
  const safeType = String(operationType || '').toUpperCase() === 'OUT' ? 'OUT' : 'IN'
  return request(`/operations/orders/${safeType}/${encodeURIComponent(referenceNumber || '')}/cancel`, {
    method: 'POST',
    auth: true,
    body: {},
  })
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
