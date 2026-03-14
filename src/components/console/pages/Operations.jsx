import { useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import {
  apiDeleteOperationOrder,
  apiListDeliveryOrders,
  apiListReceiptOrders,
} from '../../../api/auth'
import '../../../styles/dashboard/operations.css'

const OPERATIONS_TABS = [
  { id: 'receipts', label: 'Receipts', subtitle: 'Incoming Goods' },
  { id: 'delivery', label: 'Delivery', subtitle: 'Outgoing Goods' },
]

function Operations() {
  const [activeTab, setActiveTab] = useState('receipts')
  const [isBootstrapping, setIsBootstrapping] = useState(true)
  const [refreshing, setRefreshing] = useState(false)
  const [loadError, setLoadError] = useState('')
  const [feedback, setFeedback] = useState({ type: '', message: '' })

  const [receipts, setReceipts] = useState([])
  const [deliveries, setDeliveries] = useState([])
  const [deletingOrderId, setDeletingOrderId] = useState('')

  const allOrders = useMemo(() => {
    const combined = [...receipts, ...deliveries]
    return combined.sort((left, right) => {
      const leftTs = Date.parse(left.created_at || left.updated_at || left.scheduled_date || '')
      const rightTs = Date.parse(right.created_at || right.updated_at || right.scheduled_date || '')
      if (!Number.isNaN(leftTs) && !Number.isNaN(rightTs)) {
        return rightTs - leftTs
      }
      if (!Number.isNaN(leftTs)) return -1
      if (!Number.isNaN(rightTs)) return 1
      return String(right.reference_number || '').localeCompare(String(left.reference_number || ''))
    })
  }, [receipts, deliveries])

  useEffect(() => {
    bootstrap()
  }, [])

  const bootstrap = async () => {
    setIsBootstrapping(true)
    setLoadError('')

    try {
      const [receiptResponse, deliveryResponse] = await Promise.all([
        apiListReceiptOrders({ limit: 180 }),
        apiListDeliveryOrders({ limit: 180 }),
      ])

      setReceipts(receiptResponse.orders || [])
      setDeliveries(deliveryResponse.orders || [])
    } catch (error) {
      setLoadError(error?.message || 'Failed to load operations data.')
    } finally {
      setIsBootstrapping(false)
    }
  }

  const refreshData = async () => {
    setRefreshing(true)
    setLoadError('')

    try {
      const [receiptResponse, deliveryResponse] = await Promise.all([
        apiListReceiptOrders({ limit: 180 }),
        apiListDeliveryOrders({ limit: 180 }),
      ])

      setReceipts(receiptResponse.orders || [])
      setDeliveries(deliveryResponse.orders || [])
    } catch (error) {
      setLoadError(error?.message || 'Failed to refresh operations data.')
    } finally {
      setRefreshing(false)
    }
  }

  const deleteOrder = async (order) => {
    const confirmed = window.confirm(
      `Delete ${order.operation_type === 'OUT' ? 'delivery' : 'receipt'} ${order.reference_number}?`,
    )
    if (!confirmed) return

    setFeedback({ type: '', message: '' })
    setDeletingOrderId(String(order.id))
    try {
      await apiDeleteOperationOrder(order.id)
      await refreshData()
      setFeedback({ type: 'success', message: 'Order deleted successfully.' })
    } catch (error) {
      setFeedback({ type: 'error', message: error?.message || 'Failed to delete order.' })
    } finally {
      setDeletingOrderId('')
    }
  }

  if (isBootstrapping) {
    return (
      <section className="operations-shell">
        <div className="operations-status-card">Loading operations module...</div>
      </section>
    )
  }

  if (loadError && receipts.length === 0 && deliveries.length === 0) {
    return (
      <section className="operations-shell">
        <div className="operations-status-card">
          <h1>Unable to load operations</h1>
          <p>{loadError}</p>
          <button type="button" className="operations-btn primary" onClick={bootstrap}>
            Retry
          </button>
        </div>
      </section>
    )
  }

  return (
    <section className="operations-shell">
      <header className="operations-header">
        <div>
          <p className="operations-header-label">Inventory Operations</p>
          <h1 className="operations-header-title">Receipts and Delivery</h1>
          <p className="operations-header-subtitle">
            Track incoming and outgoing orders with reference number, schedule, and status.
          </p>
        </div>
        <button
          type="button"
          className="operations-btn secondary"
          onClick={refreshData}
          disabled={refreshing}
        >
          {refreshing ? 'Refreshing...' : 'Refresh Data'}
        </button>
      </header>

      <div className="operations-tab-row" role="tablist" aria-label="Operations tabs">
        {OPERATIONS_TABS.map((tab) => (
          <button
            key={tab.id}
            type="button"
            role="tab"
            aria-selected={activeTab === tab.id}
            className={`operations-tab${activeTab === tab.id ? ' is-active' : ''}`}
            onClick={() => setActiveTab(tab.id)}
          >
            <span className="operations-tab-label">{tab.label}</span>
            <span className="operations-tab-subtitle">{tab.subtitle}</span>
          </button>
        ))}
      </div>

      {feedback.message ? (
        <p className={`operations-feedback ${feedback.type === 'error' ? 'is-error' : 'is-success'}`}>
          {feedback.message}
        </p>
      ) : null}

      {loadError ? <p className="operations-feedback is-error">{loadError}</p> : null}

      {activeTab === 'receipts' ? (
        <article className="operations-card" role="tabpanel">
          <div className="operations-list-head">
            <div>
              <h2>All Receipt and Delivery Orders</h2>
              <p>
                Initial view includes both receipt and delivery orders with reference, from, to, contact,
                schedule date, and status.
              </p>
            </div>
            <div className="operations-form-actions">
              <Link to="/operations/receipts/create" className="operations-btn primary">
                Create Receipt
              </Link>
              <Link to="/operations/delivery/create" className="operations-btn ghost">
                Create Delivery
              </Link>
            </div>
          </div>

          {allOrders.length === 0 ? (
            <p className="operations-empty">No orders created yet.</p>
          ) : (
            <div className="operations-table-wrap">
              <table>
                <thead>
                  <tr>
                    <th>Reference</th>
                    <th>From</th>
                    <th>To</th>
                    <th>Contact</th>
                    <th>Schedule Date</th>
                    <th>Status</th>
                    <th>Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {allOrders.map((order) => (
                    <tr key={`${order.operation_type}-${order.id}`}>
                      <td>
                        <div className="operations-ref-cell">
                          <span className="operations-ref">{order.reference_number}</span>
                          <small>{order.operation_type === 'OUT' ? 'Delivery' : 'Receipt'}</small>
                        </div>
                      </td>
                      <td>{order.from_party || '--'}</td>
                      <td>{displayToParty(order)}</td>
                      <td>{order.contact_number || '--'}</td>
                      <td>{formatDate(order.scheduled_date)}</td>
                      <td>
                        <span className={`operations-status ${statusClassName(order.status)}`}>
                          {toTitleCase(order.status)}
                        </span>
                      </td>
                      <td>
                        <div className="operations-row-actions">
                          <Link
                            to={buildOrderDetailPath(order.operation_type, order.reference_number)}
                            className="operations-btn small secondary"
                          >
                            View
                          </Link>
                          <button
                            type="button"
                            className="operations-btn small danger"
                            disabled={deletingOrderId === String(order.id)}
                            onClick={() => deleteOrder(order)}
                          >
                            {deletingOrderId === String(order.id) ? 'Deleting...' : 'Delete'}
                          </button>
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </article>
      ) : (
        <article className="operations-card" role="tabpanel">
          <div className="operations-list-head">
            <div>
              <h2>Delivery Orders</h2>
              <p>Outgoing goods and shipment intent.</p>
            </div>
            <Link to="/operations/delivery/create" className="operations-btn primary">
              Create Delivery
            </Link>
          </div>

          {deliveries.length === 0 ? (
            <p className="operations-empty">No delivery orders created yet.</p>
          ) : (
            <div className="operations-table-wrap">
              <table>
                <thead>
                  <tr>
                    <th>Reference</th>
                    <th>From</th>
                    <th>To</th>
                    <th>Contact</th>
                    <th>Schedule Date</th>
                    <th>Status</th>
                    <th>Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {deliveries.map((order) => (
                    <tr key={order.id}>
                      <td>
                        <span className="operations-ref">{order.reference_number}</span>
                      </td>
                      <td>{order.from_party || '--'}</td>
                      <td>{displayToParty(order)}</td>
                      <td>{order.contact_number || '--'}</td>
                      <td>{formatDate(order.scheduled_date)}</td>
                      <td>
                        <span className={`operations-status ${statusClassName(order.status)}`}>
                          {toTitleCase(order.status)}
                        </span>
                      </td>
                      <td>
                        <div className="operations-row-actions">
                          <Link
                            to={buildOrderDetailPath(order.operation_type, order.reference_number)}
                            className="operations-btn small secondary"
                          >
                            View
                          </Link>
                          <button
                            type="button"
                            className="operations-btn small danger"
                            disabled={deletingOrderId === String(order.id)}
                            onClick={() => deleteOrder(order)}
                          >
                            {deletingOrderId === String(order.id) ? 'Deleting...' : 'Delete'}
                          </button>
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </article>
      )}
    </section>
  )
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

function toTitleCase(value) {
  const normalized = String(value || '').trim().toLowerCase()
  if (!normalized) return '--'
  return normalized.charAt(0).toUpperCase() + normalized.slice(1)
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

function displayToParty(order) {
  const to = String(order.to_party || '').trim()
  if (to) return to
  if (String(order.operation_type || '').toUpperCase() === 'IN') {
    const locationName = String(order.location_name || '').trim()
    const locationCode = String(order.location_short_code || '').trim()
    if (locationName && locationCode) return `${locationName} (${locationCode})`
    if (locationName) return locationName
  }
  return '--'
}

function buildOrderDetailPath(operationType, referenceNumber) {
  const type = String(operationType || '').toUpperCase() === 'OUT' ? 'OUT' : 'IN'
  return `/operations/${type}/${encodeURIComponent(referenceNumber || '')}`
}

export default Operations
