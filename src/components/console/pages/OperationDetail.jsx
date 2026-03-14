import { useEffect, useMemo, useState } from 'react'
import { useLocation, useNavigate, useParams } from 'react-router-dom'
import {
  apiCancelOperationOrder,
  apiGetOperationOrderDetail,
  apiGetOperationsMeta,
  apiUpdateOperationOrderDetail,
  apiValidateOperationOrder,
} from '../../../api/auth'
import '../../../styles/dashboard/operations.css'

const RECEIPT_STATUS_OPTIONS = ['DRAFT', 'READY', 'DONE', 'CANCELLED']
const DELIVERY_STATUS_OPTIONS = ['DRAFT', 'WAITING', 'READY', 'DONE', 'CANCELLED']

function OperationDetail() {
  const { operationType = 'IN', referenceNumber = '' } = useParams()
  const navigate = useNavigate()
  const location = useLocation()

  const normalizedType = String(operationType).toUpperCase() === 'OUT' ? 'OUT' : 'IN'
  const decodedReference = safeDecode(referenceNumber)

  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [validating, setValidating] = useState(false)
  const [cancelling, setCancelling] = useState(false)

  const [loadError, setLoadError] = useState('')
  const [feedback, setFeedback] = useState({ type: '', message: '' })

  const [, setOrder] = useState(null)
  const [products, setProducts] = useState([])
  const [locations, setLocations] = useState([])

  const [form, setForm] = useState({
    from: '',
    to: '',
    locationId: '',
    contactNumber: '',
    scheduleDate: '',
    status: 'DRAFT',
    items: [],
  })

  const isReceipt = normalizedType === 'IN'
  const returnPath = resolveBackDestination(
    location.state?.from,
    isReceipt ? '/operations/receipts' : '/operations/delivery',
  )

  const locationById = useMemo(
    () => new Map(locations.map((location) => [location.id, location])),
    [locations],
  )

  const productById = useMemo(
    () => new Map(products.map((product) => [product.id, product])),
    [products],
  )

  const statusOptions = isReceipt ? RECEIPT_STATUS_OPTIONS : DELIVERY_STATUS_OPTIONS
  const canPrint = ['READY', 'DONE'].includes(String(form.status || '').toUpperCase())

  const itemsWithAvailability = useMemo(
    () =>
      form.items.map((item) => {
        const orderedQuantity = Number.parseInt(String(item.quantity || ''), 10) || 0
        const availableQuantity = getAvailableQuantityForItem(
          item.productId,
          form.locationId,
          products,
          item.fallbackAvailableQuantity,
        )

        return {
          ...item,
          orderedQuantity,
          availableQuantity,
          isInsufficient: !isReceipt && orderedQuantity > availableQuantity,
        }
      }),
    [form.items, form.locationId, isReceipt, products],
  )

  useEffect(() => {
    loadPage()
  }, [normalizedType, decodedReference])

  const loadPage = async () => {
    setLoading(true)
    setLoadError('')
    setFeedback({ type: '', message: '' })

    try {
      const [detailResponse, metaResponse] = await Promise.all([
        apiGetOperationOrderDetail(normalizedType, decodedReference),
        apiGetOperationsMeta(),
      ])

      const receivedOrder = detailResponse?.order
      if (!receivedOrder) {
        throw new Error('Order detail is unavailable.')
      }

      setOrder(receivedOrder)
      setProducts(metaResponse?.products || [])
      setLocations(metaResponse?.locations || [])
      setForm(toEditableForm(receivedOrder))
    } catch (error) {
      setLoadError(error?.message || 'Failed to load operation detail.')
    } finally {
      setLoading(false)
    }
  }

  const setField = (name, value) => {
    setForm((previous) => ({ ...previous, [name]: value }))
  }

  const updateItem = (index, field, value) => {
    setForm((previous) => ({
      ...previous,
      items: previous.items.map((item, itemIndex) =>
        itemIndex === index ? { ...item, [field]: value } : item,
      ),
    }))
  }

  const addItem = () => {
    setForm((previous) => ({
      ...previous,
      items: [
        ...previous.items,
        {
          productId: '',
          quantity: '1',
          fallbackAvailableQuantity: 0,
        },
      ],
    }))
  }

  const removeItem = (index) => {
    setForm((previous) => {
      if (previous.items.length === 1) return previous
      return {
        ...previous,
        items: previous.items.filter((_, itemIndex) => itemIndex !== index),
      }
    })
  }

  const saveChanges = async (event) => {
    event.preventDefault()
    setFeedback({ type: '', message: '' })

    const validationError = validateDetailForm(form, isReceipt)
    if (validationError) {
      setFeedback({ type: 'error', message: validationError })
      return
    }

    const payload = buildUpdatePayload({
      form,
      isReceipt,
      locationById,
    })

    setSaving(true)
    try {
      const response = await apiUpdateOperationOrderDetail(normalizedType, decodedReference, payload)
      const updatedOrder = response?.order
      if (!updatedOrder) {
        throw new Error('Updated order was not returned by the server.')
      }

      setOrder(updatedOrder)
      setForm(toEditableForm(updatedOrder))
      setFeedback({ type: 'success', message: 'Operation detail updated successfully.' })
    } catch (error) {
      setFeedback({ type: 'error', message: error?.message || 'Failed to update order detail.' })
    } finally {
      setSaving(false)
    }
  }

  const validateOrder = async () => {
    setFeedback({ type: '', message: '' })
    setValidating(true)
    try {
      const response = await apiValidateOperationOrder(normalizedType, decodedReference)
      const updatedOrder = response?.order
      if (!updatedOrder) {
        throw new Error('Validation response is incomplete.')
      }

      setOrder(updatedOrder)
      setForm(toEditableForm(updatedOrder))

      if (isReceipt) {
        setFeedback({ type: 'success', message: 'Validation successful. Receipt status updated to Ready.' })
      } else if (response.all_items_in_stock) {
        setFeedback({ type: 'success', message: 'Validation successful. All items are in stock.' })
      } else {
        setFeedback({
          type: 'error',
          message: 'Validation complete. Some items do not have enough available quantity.',
        })
      }
    } catch (error) {
      setFeedback({ type: 'error', message: error?.message || 'Failed to validate order.' })
    } finally {
      setValidating(false)
    }
  }

  const cancelOrder = async () => {
    const confirmed = window.confirm('Cancel this operation order?')
    if (!confirmed) return

    setFeedback({ type: '', message: '' })
    setCancelling(true)
    try {
      const response = await apiCancelOperationOrder(normalizedType, decodedReference)
      const updatedOrder = response?.order
      if (!updatedOrder) {
        throw new Error('Cancel response is incomplete.')
      }

      setOrder(updatedOrder)
      setForm(toEditableForm(updatedOrder))
      setFeedback({ type: 'success', message: 'Order has been cancelled.' })
    } catch (error) {
      setFeedback({ type: 'error', message: error?.message || 'Failed to cancel order.' })
    } finally {
      setCancelling(false)
    }
  }

  const printOrder = () => {
    if (!canPrint) {
      setFeedback({ type: 'error', message: 'Print is available only when status is Ready or Done.' })
      return
    }

    const printable = buildPrintableContent({
      operationType: normalizedType,
      referenceNumber: decodedReference,
      form,
      itemsWithAvailability,
      productById,
      locationById,
    })

    const blob = new Blob([printable], { type: 'text/plain;charset=utf-8' })
    const url = URL.createObjectURL(blob)
    const anchor = document.createElement('a')
    anchor.href = url
    anchor.download = `operation-${normalizedType}-${sanitizeFileToken(decodedReference)}.txt`
    anchor.click()
    URL.revokeObjectURL(url)

    setFeedback({ type: 'success', message: 'Printable file downloaded.' })
  }

  const goBack = () => {
    navigate(returnPath)
  }

  if (loading) {
    return (
      <section className="operation-detail-shell">
        <div className="operations-status-card">Loading operation detail...</div>
      </section>
    )
  }

  if (loadError) {
    return (
      <section className="operation-detail-shell">
        <div className="operations-status-card">
          <h1>Unable to load operation detail</h1>
          <p>{loadError}</p>
          <div className="operations-form-actions">
            <button type="button" className="operations-btn primary" onClick={loadPage}>
              Retry
            </button>
            <button type="button" className="operations-btn ghost" onClick={goBack}>
              Back to Operations
            </button>
          </div>
        </div>
      </section>
    )
  }

  return (
    <section className="operation-detail-shell">
      <div className="operation-page-topbar">
        <button type="button" className="operations-btn secondary" onClick={goBack}>
          Back to Operations
        </button>
      </div>

      <article className="operation-detail-card">
        <header className="operation-detail-header">
          <div>
            <h1>Operation Detail</h1>
            <p>Review and update operation information, quantities, and shipping details.</p>
          </div>
          <span className={`operations-status ${statusClassName(form.status)}`}>{toTitleCase(form.status)}</span>
        </header>

        <form className="operations-form" onSubmit={saveChanges}>
          <div className="operation-detail-meta-grid">
            <div className="operations-reference-preview">
              <span>Reference Number</span>
              <strong>{decodedReference || '--'}</strong>
            </div>
            <div className="operations-reference-preview">
              <span>Operation</span>
              <strong>{normalizedType}</strong>
            </div>
          </div>

          <div className="operations-field">
            <label htmlFor="detail-location">Shipping Location</label>
            <select
              id="detail-location"
              value={form.locationId}
              onChange={(event) => setField('locationId', event.target.value)}
            >
              <option value="">Select location</option>
              {locations.map((location) => (
                <option key={location.id} value={location.id}>
                  {buildLocationLabel(location)}
                </option>
              ))}
            </select>
          </div>

          <div className="operation-detail-two-col">
            <div className="operations-field">
              <label htmlFor="detail-from-party">Vendor Details (From)</label>
              <input
                id="detail-from-party"
                value={form.from}
                onChange={(event) => setField('from', event.target.value)}
                placeholder="Supplier / Source"
              />
            </div>

            <div className="operations-field">
              <label htmlFor="detail-to-party">Vendor Details (To)</label>
              <input
                id="detail-to-party"
                value={form.to}
                onChange={(event) => setField('to', event.target.value)}
                placeholder="Destination / Customer"
              />
            </div>
          </div>

          <div className="operation-detail-two-col">
            <div className="operations-field">
              <label htmlFor="detail-schedule-date">Shipping Date</label>
              <input
                id="detail-schedule-date"
                type="date"
                min={todayDateISO()}
                value={form.scheduleDate}
                onChange={(event) => setField('scheduleDate', event.target.value)}
              />
            </div>

            <div className="operations-field">
              <label htmlFor="detail-status">Status</label>
              <select
                id="detail-status"
                value={form.status}
                onChange={(event) => setField('status', event.target.value)}
              >
                {statusOptions.map((status) => (
                  <option key={status} value={status}>
                    {toTitleCase(status)}
                  </option>
                ))}
              </select>
            </div>
          </div>

          <div className="operations-field">
            <label htmlFor="detail-contact">Contact Details</label>
            <div className="operations-phone-input">
              <span>+91</span>
              <input
                id="detail-contact"
                inputMode="numeric"
                maxLength={10}
                value={form.contactNumber}
                onChange={(event) => setField('contactNumber', sanitizePhoneDigits(event.target.value))}
                placeholder="9876543210"
              />
            </div>
          </div>

          <div className="operations-field">
            <p className="operations-item-head">Products</p>
            <div className="operation-detail-items">
              {itemsWithAvailability.map((item, index) => (
                <div
                  key={`detail-item-${index}`}
                  className={`operations-item-row${item.isInsufficient ? ' is-insufficient' : ''}`}
                >
                  <div>
                    <select
                      value={item.productId}
                      onChange={(event) =>
                        updateItem(index, 'productId', event.target.value)
                      }
                    >
                      <option value="">Select product</option>
                      {products.map((product) => (
                        <option key={product.id} value={product.id}>
                          {product.sku} - {product.name}
                        </option>
                      ))}
                    </select>
                    {item.productId ? (
                      <p className="operations-item-meta">
                        {productById.get(item.productId)?.category_name || 'Selected category unavailable'}
                      </p>
                    ) : null}
                  </div>

                  <input
                    type="number"
                    min="1"
                    step="1"
                    value={item.quantity}
                    onChange={(event) =>
                      updateItem(index, 'quantity', sanitizeOrderedQuantity(event.target.value))
                    }
                    placeholder="Ordered Qty"
                  />

                  <div className={`operation-item-availability${item.isInsufficient ? ' is-insufficient' : ''}`}>
                    <span>Available: {item.availableQuantity}</span>
                  </div>

                  <button
                    type="button"
                    className="operations-btn ghost small"
                    onClick={() => removeItem(index)}
                    disabled={form.items.length === 1}
                  >
                    Remove
                  </button>
                </div>
              ))}
            </div>
          </div>

          <div className="operations-form-actions">
            <button type="button" className="operations-btn ghost" onClick={addItem}>
              Add Product
            </button>
            <button type="submit" className="operations-btn primary" disabled={saving}>
              {saving ? 'Saving...' : 'Save Changes'}
            </button>
            <button
              type="button"
              className="operations-btn secondary"
              onClick={validateOrder}
              disabled={validating}
            >
              {validating ? 'Validating...' : 'Validate'}
            </button>
            <button type="button" className="operations-btn secondary" onClick={printOrder} disabled={!canPrint}>
              Print
            </button>
            <button
              type="button"
              className="operations-btn danger"
              onClick={cancelOrder}
              disabled={cancelling || String(form.status).toUpperCase() === 'CANCELLED'}
            >
              {cancelling ? 'Cancelling...' : 'Cancel'}
            </button>
          </div>

          {feedback.message ? (
            <p className={`operations-feedback ${feedback.type === 'error' ? 'is-error' : 'is-success'}`}>
              {feedback.message}
            </p>
          ) : null}
        </form>
      </article>
    </section>
  )
}

function toEditableForm(order) {
  return {
    from: String(order?.from_party || ''),
    to: String(order?.to_party || ''),
    locationId: String(order?.location_id || ''),
    contactNumber: sanitizePhoneDigits(String(order?.contact_number || '')),
    scheduleDate: toInputDate(order?.scheduled_date),
    status: String(order?.status || 'DRAFT').toUpperCase(),
    items: (order?.items || []).map((item) => ({
      productId: String(item.product_id || ''),
      quantity: String(item.quantity || 1),
      fallbackAvailableQuantity: Number(item.available_quantity || 0),
    })),
  }
}

function validateDetailForm(form, isReceipt) {
  if (!form.locationId) return 'Shipping location is required.'
  if (!form.scheduleDate) return 'Shipping date is required.'
  if (form.scheduleDate < todayDateISO()) return 'Shipping date must be today or future only.'
  if (sanitizePhoneDigits(form.contactNumber).length !== 10) {
    return 'Contact number must be exactly 10 digits.'
  }

  if (isReceipt) {
    if (!String(form.from || '').trim()) return 'Vendor details (From) are required for receipts.'
  } else {
    if (!String(form.to || '').trim()) return 'Vendor details (To) are required for deliveries.'
  }

  const normalizedItems = normalizeItemsPayload(form.items)
  if (normalizedItems.length === 0) return 'Add at least one valid product with ordered quantity.'

  return ''
}

function buildUpdatePayload({ form, isReceipt, locationById }) {
  const selectedLocation = locationById.get(form.locationId)
  const destinationLocationLabel = selectedLocation ? buildLocationLabel(selectedLocation) : ''

  return {
    from: String(form.from || '').trim(),
    to: isReceipt ? String(form.to || '').trim() || destinationLocationLabel : String(form.to || '').trim(),
    location_id: String(form.locationId || '').trim(),
    contact_number: `+91${sanitizePhoneDigits(form.contactNumber)}`,
    schedule_date: String(form.scheduleDate || ''),
    status: String(form.status || '').toUpperCase(),
    items: normalizeItemsPayload(form.items),
  }
}

function normalizeItemsPayload(items) {
  return items
    .map((item) => ({
      product_id: String(item.productId || '').trim(),
      quantity: Number.parseInt(String(item.quantity || '').trim(), 10),
    }))
    .filter((item) => item.product_id && Number.isInteger(item.quantity) && item.quantity > 0)
}

function getAvailableQuantityForItem(productId, locationId, products, fallbackAvailableQuantity) {
  const normalizedProductId = String(productId || '').trim()
  const normalizedLocationId = String(locationId || '').trim()
  if (!normalizedProductId || !normalizedLocationId) {
    return Number(fallbackAvailableQuantity || 0)
  }

  const product = (products || []).find((entry) => entry.id === normalizedProductId)
  if (!product) return Number(fallbackAvailableQuantity || 0)

  const stockLevels = Array.isArray(product.stock_levels) ? product.stock_levels : []
  const level = stockLevels.find((entry) => entry.location_id === normalizedLocationId)
  if (level) {
    return Number(level.free_to_use_quantity || 0)
  }

  return Number(fallbackAvailableQuantity || product.free_to_use_quantity || 0)
}

function buildPrintableContent({ operationType, referenceNumber, form, itemsWithAvailability, productById, locationById }) {
  const location = locationById.get(form.locationId)
  const locationLabel = location ? buildLocationLabel(location) : '--'

  const lines = [
    'Sanchay IMS - Operation Order',
    '----------------------------------------',
    `Reference Number: ${referenceNumber || '--'}`,
    `Operation: ${operationType}`,
    `Status: ${toTitleCase(form.status)}`,
    `Shipping Date: ${formatDate(form.scheduleDate)}`,
    `Shipping Location: ${locationLabel}`,
    `Vendor (From): ${form.from || '--'}`,
    `Vendor (To): ${form.to || '--'}`,
    `Contact: ${sanitizePhoneDigits(form.contactNumber) ? `+91${sanitizePhoneDigits(form.contactNumber)}` : '--'}`,
    '',
    'Items',
    '----------------------------------------',
  ]

  itemsWithAvailability.forEach((item, index) => {
    const product = productById.get(item.productId)
    const productLabel = product ? `${product.sku} - ${product.name}` : item.productId || '--'
    lines.push(
      `${index + 1}. ${productLabel}`,
      `   Ordered Quantity: ${item.orderedQuantity}`,
      `   Available Quantity: ${item.availableQuantity}`,
      `   Status: ${item.isInsufficient ? 'Insufficient stock' : 'Available'}`,
    )
  })

  lines.push('', `Generated: ${new Date().toLocaleString('en-IN')}`)

  return lines.join('\n')
}

function buildLocationLabel(location) {
  const warehouses = Array.isArray(location?.warehouse_names) ? location.warehouse_names.join(', ') : ''
  if (!warehouses) {
    return `${location.name} (${location.short_code})`
  }
  return `${location.name} (${location.short_code}) - ${warehouses}`
}

function statusClassName(statusValue) {
  const status = String(statusValue || '').toUpperCase()
  if (status === 'DRAFT') return 'is-draft'
  if (status === 'WAITING') return 'is-waiting'
  if (status === 'READY') return 'is-ready'
  if (status === 'DONE') return 'is-done'
  if (status === 'CANCELLED') return 'is-cancelled'
  return ''
}

function toTitleCase(value) {
  const normalized = String(value || '').trim().toLowerCase()
  if (!normalized) return '--'
  return normalized.charAt(0).toUpperCase() + normalized.slice(1)
}

function formatDate(value) {
  if (!value) return '--'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '--'
  return date.toLocaleDateString('en-IN', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  })
}

function toInputDate(value) {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ''
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

function sanitizePhoneDigits(value) {
  const digitsOnly = String(value || '').replace(/\D/g, '')
  if (digitsOnly.startsWith('91') && digitsOnly.length > 10) {
    return digitsOnly.slice(2, 12)
  }
  return digitsOnly.slice(0, 10)
}

function sanitizeOrderedQuantity(value) {
  const numeric = String(value || '').replace(/\D/g, '')
  if (!numeric) return ''
  return String(Number.parseInt(numeric, 10))
}

function sanitizeFileToken(value) {
  return String(value || 'order').replace(/[^a-zA-Z0-9-_]+/g, '_')
}

function todayDateISO() {
  const now = new Date()
  const year = now.getFullYear()
  const month = String(now.getMonth() + 1).padStart(2, '0')
  const day = String(now.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

function safeDecode(value) {
  try {
    return decodeURIComponent(value)
  } catch {
    return value
  }
}

function resolveBackDestination(from, fallbackPath) {
  const normalized = String(from || '').trim()
  if (normalized.startsWith('/operations/')) {
    return normalized
  }
  return fallbackPath
}

export default OperationDetail
