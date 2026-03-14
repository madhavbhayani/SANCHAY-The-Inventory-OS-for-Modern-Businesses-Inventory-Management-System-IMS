import { useEffect, useMemo, useState } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import {
  apiCreateDeliveryOrder,
  apiCreateReceiptOrder,
  apiGetOperationsMeta,
} from '../../../api/auth'
import '../../../styles/dashboard/operations.css'

const RECEIPT_STATUS_OPTIONS = ['DRAFT', 'READY', 'DONE']
const DELIVERY_STATUS_OPTIONS = ['DRAFT', 'WAITING', 'READY', 'DONE']
const EMPTY_ITEM = { productId: '', quantity: '1' }

function OperationCreateOrder({ mode = 'receipt' }) {
  const isReceipt = mode === 'receipt'
  const navigate = useNavigate()
  const location = useLocation()
  const returnPath = resolveBackDestination(
    location.state?.from,
    isReceipt ? '/operations/receipts' : '/operations/delivery',
  )

  const [loading, setLoading] = useState(true)
  const [loadError, setLoadError] = useState('')
  const [saving, setSaving] = useState(false)
  const [feedback, setFeedback] = useState({ type: '', message: '' })

  const [locations, setLocations] = useState([])
  const [products, setProducts] = useState([])

  const [form, setForm] = useState({
    fromVendor: '',
    toVendor: '',
    locationId: '',
    contactNumber: '',
    scheduleDate: '',
    status: isReceipt ? 'DRAFT' : 'DRAFT',
    items: [{ ...EMPTY_ITEM }],
  })

  const locationById = useMemo(
    () => new Map(locations.map((location) => [location.id, location])),
    [locations],
  )

  const statusOptions = isReceipt ? RECEIPT_STATUS_OPTIONS : DELIVERY_STATUS_OPTIONS

  useEffect(() => {
    loadMeta()
  }, [])

  const loadMeta = async () => {
    setLoading(true)
    setLoadError('')

    try {
      const data = await apiGetOperationsMeta()
      setLocations(data.locations || [])
      setProducts(data.products || [])
    } catch (error) {
      setLoadError(error?.message || 'Failed to load create-order page.')
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
      items: [...previous.items, { ...EMPTY_ITEM }],
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

  const submit = async (event) => {
    event.preventDefault()
    setFeedback({ type: '', message: '' })

    const validationError = validateForm(form, isReceipt)
    if (validationError) {
      setFeedback({ type: 'error', message: validationError })
      return
    }

    const sourceLocation = locationById.get(form.locationId)
    const receiptDestination = sourceLocation ? formatLocationLabel(sourceLocation) : ''
    const payload = {
      from: isReceipt
        ? form.fromVendor.trim()
        : sourceLocation
          ? `${sourceLocation.name} (${sourceLocation.short_code})`
          : 'Warehouse Location',
      to: isReceipt ? receiptDestination : form.toVendor.trim(),
      location_id: form.locationId,
      contact_number: `+91${sanitizePhoneDigits(form.contactNumber)}`,
      schedule_date: form.scheduleDate,
      status: form.status,
      items: normalizeItems(form.items),
    }

    setSaving(true)
    try {
      if (isReceipt) {
        await apiCreateReceiptOrder(payload)
      } else {
        await apiCreateDeliveryOrder(payload)
      }
      navigate(returnPath, { replace: true })
    } catch (error) {
      setFeedback({ type: 'error', message: error?.message || 'Failed to create order.' })
    } finally {
      setSaving(false)
    }
  }

  const goBack = () => {
    navigate(returnPath)
  }

  const productsForLocation = isReceipt ? products : getProductsForLocation(products, form.locationId)

  if (loading) {
    return (
      <section className="operation-detail-shell">
        <div className="operations-status-card">Loading create page...</div>
      </section>
    )
  }

  if (loadError) {
    return (
      <section className="operation-detail-shell">
        <div className="operations-status-card">
          <h1>Unable to load create page</h1>
          <p>{loadError}</p>
          <div className="operations-form-actions">
            <button type="button" className="operations-btn primary" onClick={loadMeta}>
              Retry
            </button>
            <button type="button" className="operations-btn ghost" onClick={goBack}>
              Back
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

      <article className="operations-card">
        <h2>{isReceipt ? 'Create Receipt' : 'Create Delivery'}</h2>
        <p>
          {isReceipt
            ? 'Use this when items arrive from vendors.'
            : 'Use this when items are prepared for outbound delivery.'}
        </p>

        <form className="operations-form" onSubmit={submit}>
          <div className="operations-reference-preview">
            <span>Reference Number</span>
            <strong>{buildReferencePreview(form.locationId, locationById, isReceipt ? 'IN' : 'OUT')}</strong>
          </div>

          {isReceipt ? (
            <div className="operations-field">
              <label htmlFor="receipt-from-vendor">From (Vendor Name)</label>
              <input
                id="receipt-from-vendor"
                value={form.fromVendor}
                onChange={(event) => setField('fromVendor', event.target.value)}
                placeholder="Vendor / Supplier Name"
              />
            </div>
          ) : null}

          <div className="operations-field">
            <label htmlFor="create-location">
              {isReceipt ? 'Location of Warehouse' : 'From (Location of Goods)'}
            </label>
            <select
              id="create-location"
              value={form.locationId}
              onChange={(event) => setField('locationId', event.target.value)}
            >
              <option value="">Select location</option>
              {locations.map((location) => (
                <option key={location.id} value={location.id}>
                  {formatLocationLabel(location)}
                </option>
              ))}
            </select>
          </div>

          {isReceipt ? null : (
            <div className="operations-field">
              <label htmlFor="delivery-to-vendor">To (Vendor Name)</label>
              <input
                id="delivery-to-vendor"
                value={form.toVendor}
                onChange={(event) => setField('toVendor', event.target.value)}
                placeholder="Destination Vendor Name"
              />
            </div>
          )}

          <div className="operations-field">
            <label htmlFor="create-contact-number">Contact Number</label>
            <div className="operations-phone-input">
              <span>+91</span>
              <input
                id="create-contact-number"
                inputMode="numeric"
                maxLength={10}
                value={form.contactNumber}
                onChange={(event) => setField('contactNumber', sanitizePhoneDigits(event.target.value))}
                placeholder="9876543210"
              />
            </div>
          </div>

          <div className="operations-field">
            <label htmlFor="create-schedule-date">Schedule Date</label>
            <input
              id="create-schedule-date"
              type="date"
              value={form.scheduleDate}
              min={todayDateISO()}
              onChange={(event) => setField('scheduleDate', event.target.value)}
            />
          </div>

          <div className="operations-field">
            <label htmlFor="create-status">Status</label>
            <select
              id="create-status"
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

          <div className="operations-field">
            <p className="operations-item-head">Product Details</p>
            <div className="operations-items">
              {form.items.map((item, index) => (
                <div key={`create-item-${index}`} className="operations-item-row">
                  <div>
                    <select
                      value={item.productId}
                      onChange={(event) => updateItem(index, 'productId', event.target.value)}
                    >
                      <option value="">Select product</option>
                      {productsForLocation.map((product) => (
                        <option key={product.id} value={product.id}>
                          {product.sku} - {product.name}
                        </option>
                      ))}
                    </select>
                    {item.productId ? (
                      <p className="operations-item-meta">
                        {productMetaById(item.productId, products) || 'Selected product details unavailable'}
                      </p>
                    ) : null}
                  </div>

                  <input
                    type="number"
                    min="1"
                    step="1"
                    value={item.quantity}
                    onChange={(event) => updateItem(index, 'quantity', event.target.value)}
                    placeholder="Qty"
                  />

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
              Add Product Row
            </button>
            <button type="submit" className="operations-btn primary" disabled={saving}>
              {saving ? 'Saving...' : isReceipt ? 'Create Receipt' : 'Create Delivery'}
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

function normalizeItems(items) {
  return items
    .map((item) => ({
      product_id: String(item.productId || '').trim(),
      quantity: Number(String(item.quantity || '').trim()),
    }))
    .filter((item) => item.product_id && Number.isInteger(item.quantity) && item.quantity > 0)
}

function validateForm(form, isReceipt) {
  if (!form.locationId) return 'Location is required.'
  if (!form.scheduleDate) return 'Schedule date is required.'
  if (form.scheduleDate < todayDateISO()) return 'Schedule date must be today or future only.'
  if (sanitizePhoneDigits(form.contactNumber).length !== 10) {
    return 'Contact number must be exactly 10 digits.'
  }
  if (isReceipt) {
    if (!form.fromVendor.trim()) return 'From (vendor name) is required.'
  } else if (!form.toVendor.trim()) {
    return 'To (vendor name) is required.'
  }
  if (normalizeItems(form.items).length === 0) return 'Add at least one valid product with quantity.'
  return ''
}

function getProductsForLocation(products, locationId) {
  if (!locationId) return products
  const scoped = products.filter((product) => {
    const stockLevels = Array.isArray(product.stock_levels) ? product.stock_levels : []
    if (stockLevels.some((level) => level.location_id === locationId)) {
      return true
    }
    return product.location_id === locationId
  })
  return scoped.length > 0 ? scoped : products
}

function formatLocationLabel(location) {
  const warehouses = (location.warehouse_names || []).join(', ')
  if (!warehouses) {
    return `${location.name} (${location.short_code})`
  }
  return `${location.name} (${location.short_code}) - ${warehouses}`
}

function buildReferencePreview(locationId, locationById, operationType) {
  const location = locationById.get(locationId)
  const rawWarehouse = location?.warehouse_names?.[0] || 'WAREHOUSE'
  const warehouseToken = rawWarehouse
    .toUpperCase()
    .replace(/[^A-Z0-9]+/g, '')
    .slice(0, 10)

  return `${warehouseToken || 'WH'}/${operationType}/<Auto ID>`
}

function productMetaById(productId, products) {
  const product = products.find((candidate) => candidate.id === productId)
  if (!product) return ''
  const stockLevels = Array.isArray(product.stock_levels) ? product.stock_levels : []
  const freeTotal =
    stockLevels.length > 0
      ? stockLevels.reduce((total, level) => total + Number(level.free_to_use_quantity || 0), 0)
      : Number(product.free_to_use_quantity || 0)
  if (stockLevels.length === 0) {
    return `${product.category_name} | ${product.location_name} (${product.location_short_code}) | Free ${product.free_to_use_quantity}`
  }

  const locationPreview = stockLevels
    .slice(0, 2)
    .map((level) => `${level.location_name} (${level.location_short_code})`)
    .join(', ')

  return `${product.category_name} | ${stockLevels.length} locations | Free ${freeTotal}${locationPreview ? ` | ${locationPreview}` : ''}`
}

function sanitizePhoneDigits(value) {
  const digitsOnly = String(value || '').replace(/\D/g, '')
  if (digitsOnly.startsWith('91') && digitsOnly.length > 10) {
    return digitsOnly.slice(2, 12)
  }
  return digitsOnly.slice(0, 10)
}

function resolveBackDestination(from, fallbackPath) {
  const normalized = String(from || '').trim()
  if (normalized.startsWith('/operations/')) {
    return normalized
  }
  return fallbackPath
}

function todayDateISO() {
  const now = new Date()
  const year = now.getFullYear()
  const month = String(now.getMonth() + 1).padStart(2, '0')
  const day = String(now.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

function toTitleCase(value) {
  const normalized = String(value || '').trim().toLowerCase()
  if (!normalized) return '--'
  return normalized.charAt(0).toUpperCase() + normalized.slice(1)
}

export default OperationCreateOrder
