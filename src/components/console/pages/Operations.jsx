import { useEffect, useMemo, useState } from 'react'
import { Link, useLocation } from 'react-router-dom'
import {
  apiAdjustStockQuantity,
  apiDeleteOperationOrder,
  apiGetAdjustmentsOverview,
  apiListDeliveryOrders,
  apiListReceiptOrders,
  apiTransferAdjustmentStock,
} from '../../../api/auth'
import '../../../styles/dashboard/operations.css'

const OPERATIONS_TABS = [
  { id: 'receipts', label: 'Receipts', subtitle: 'Incoming Goods', path: '/operations/receipts' },
  { id: 'delivery', label: 'Delivery', subtitle: 'Outgoing Goods', path: '/operations/delivery' },
  { id: 'adjustments', label: 'Adjustments', subtitle: 'Internal Transfers', path: '/operations/adjustments' },
]

const ORDER_KANBAN_COLUMNS = [
  { id: 'DRAFT', label: 'Draft' },
  { id: 'WAITING', label: 'Waiting' },
  { id: 'READY', label: 'Ready' },
  { id: 'DONE', label: 'Done' },
  { id: 'CANCELLED', label: 'Cancelled' },
  { id: 'OTHER', label: 'Other' },
]

const ORDER_LIMIT = 180

function Operations({ activeTab = 'receipts' }) {
  const location = useLocation()

  const [isBootstrapping, setIsBootstrapping] = useState(true)
  const [refreshing, setRefreshing] = useState(false)
  const [loadError, setLoadError] = useState('')
  const [feedback, setFeedback] = useState({ type: '', message: '' })

  const [viewMode, setViewMode] = useState('list')
  const [receiptQueryInput, setReceiptQueryInput] = useState('')
  const [receiptQueryApplied, setReceiptQueryApplied] = useState('')

  const [receipts, setReceipts] = useState([])
  const [deliveries, setDeliveries] = useState([])
  const [deletingOrderId, setDeletingOrderId] = useState('')

  const [adjustmentsOverview, setAdjustmentsOverview] = useState({
    locations: [],
    rows: [],
    history: [],
  })
  const [adjustmentDrafts, setAdjustmentDrafts] = useState({})
  const [submittingAdjustmentKey, setSubmittingAdjustmentKey] = useState('')
  const [submittingAdjustmentAction, setSubmittingAdjustmentAction] = useState('')

  const currentPath = location.pathname

  useEffect(() => {
    setFeedback({ type: '', message: '' })

    if (activeTab === 'adjustments') {
      bootstrapAdjustments()
      return
    }

    bootstrapOrders(receiptQueryApplied)
  }, [activeTab])

  const loadOrders = async (queryText = '') => {
    const [receiptResponse, deliveryResponse] = await Promise.all([
      apiListReceiptOrders({ limit: ORDER_LIMIT, query: queryText }),
      apiListDeliveryOrders({ limit: ORDER_LIMIT }),
    ])

    setReceipts(receiptResponse.orders || [])
    setDeliveries(deliveryResponse.orders || [])
  }

  const bootstrapOrders = async (queryText = receiptQueryApplied) => {
    setIsBootstrapping(true)
    setLoadError('')

    try {
      await loadOrders(queryText)
    } catch (error) {
      setLoadError(error?.message || 'Failed to load operations data.')
    } finally {
      setIsBootstrapping(false)
    }
  }

  const bootstrapAdjustments = async () => {
    setIsBootstrapping(true)
    setLoadError('')

    try {
      const overview = await apiGetAdjustmentsOverview({ limit: 360 })
      setAdjustmentsOverview({
        locations: overview.locations || [],
        rows: overview.rows || [],
        history: overview.history || [],
      })
      setAdjustmentDrafts({})
    } catch (error) {
      setLoadError(error?.message || 'Failed to load adjustments data.')
    } finally {
      setIsBootstrapping(false)
    }
  }

  const refreshData = async () => {
    setRefreshing(true)
    setLoadError('')

    try {
      if (activeTab === 'adjustments') {
        const overview = await apiGetAdjustmentsOverview({ limit: 360 })
        setAdjustmentsOverview({
          locations: overview.locations || [],
          rows: overview.rows || [],
          history: overview.history || [],
        })
        setAdjustmentDrafts({})
      } else {
        await loadOrders(receiptQueryApplied)
      }
    } catch (error) {
      setLoadError(
        error?.message ||
          (activeTab === 'adjustments'
            ? 'Failed to refresh adjustments data.'
            : 'Failed to refresh operations data.'),
      )
    } finally {
      setRefreshing(false)
    }
  }

  const submitReceiptSearch = async (event) => {
    event.preventDefault()
    const nextQuery = receiptQueryInput.trim()
    setReceiptQueryApplied(nextQuery)

    setRefreshing(true)
    setLoadError('')
    try {
      await loadOrders(nextQuery)
    } catch (error) {
      setLoadError(error?.message || 'Failed to search receipt orders.')
    } finally {
      setRefreshing(false)
    }
  }

  const clearReceiptSearch = async () => {
    setReceiptQueryInput('')
    setReceiptQueryApplied('')

    setRefreshing(true)
    setLoadError('')
    try {
      await loadOrders('')
    } catch (error) {
      setLoadError(error?.message || 'Failed to clear receipt search.')
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

  const updateAdjustmentDraft = (row, field, value) => {
    const rowKey = buildAdjustmentRowKey(row)
    setAdjustmentDrafts((previous) => ({
      ...previous,
      [rowKey]: {
        ...createAdjustmentDraft(row),
        ...(previous[rowKey] || {}),
        [field]: value,
      },
    }))
  }

  const submitTransfer = async (row) => {
    const rowKey = buildAdjustmentRowKey(row)
    const draft = {
      ...createAdjustmentDraft(row),
      ...(adjustmentDrafts[rowKey] || {}),
    }

    const quantity = Number.parseInt(String(draft.transferQuantity || '').trim(), 10)
    if (!draft.transferLocationId) {
      setFeedback({ type: 'error', message: 'Select a destination location for the transfer.' })
      return
    }
    if (draft.transferLocationId === row.location_id) {
      setFeedback({ type: 'error', message: 'Destination location must be different from the current location.' })
      return
    }
    if (!Number.isInteger(quantity) || quantity <= 0) {
      setFeedback({ type: 'error', message: 'Transfer quantity must be greater than zero.' })
      return
    }
    if (quantity > Number(row.free_to_use_quantity || 0)) {
      setFeedback({ type: 'error', message: 'Transfer quantity cannot exceed the current free-to-use quantity.' })
      return
    }
    if (!String(draft.reason || '').trim()) {
      setFeedback({ type: 'error', message: 'Reason is required for an internal transfer.' })
      return
    }

    setFeedback({ type: '', message: '' })
    setSubmittingAdjustmentKey(rowKey)
    setSubmittingAdjustmentAction('transfer')

    try {
      await apiTransferAdjustmentStock({
        product_id: row.product_id,
        from_location_id: row.location_id,
        to_location_id: draft.transferLocationId,
        quantity,
        reason: String(draft.reason || '').trim(),
      })

      await refreshData()
      setFeedback({ type: 'success', message: 'Internal transfer saved successfully.' })
    } catch (error) {
      setFeedback({ type: 'error', message: error?.message || 'Failed to save the internal transfer.' })
    } finally {
      setSubmittingAdjustmentKey('')
      setSubmittingAdjustmentAction('')
    }
  }

  const submitQuantityCorrection = async (row) => {
    const rowKey = buildAdjustmentRowKey(row)
    const draft = {
      ...createAdjustmentDraft(row),
      ...(adjustmentDrafts[rowKey] || {}),
    }

    const correctedFreeToUse = Number.parseInt(String(draft.correctedFreeToUse || '').trim(), 10)
    if (!Number.isInteger(correctedFreeToUse) || correctedFreeToUse < 0) {
      setFeedback({ type: 'error', message: 'Corrected free-to-use quantity must be zero or more.' })
      return
    }
    if (correctedFreeToUse === Number(row.free_to_use_quantity || 0)) {
      setFeedback({ type: 'error', message: 'Enter a different free-to-use quantity to save a correction.' })
      return
    }
    if (!String(draft.reason || '').trim()) {
      setFeedback({ type: 'error', message: 'Reason is required for a stock correction.' })
      return
    }

    setFeedback({ type: '', message: '' })
    setSubmittingAdjustmentKey(rowKey)
    setSubmittingAdjustmentAction('quantity')

    try {
      await apiAdjustStockQuantity({
        product_id: row.product_id,
        location_id: row.location_id,
        free_to_use_quantity: correctedFreeToUse,
        reason: String(draft.reason || '').trim(),
      })

      await refreshData()
      setFeedback({ type: 'success', message: 'Free-to-use quantity corrected successfully.' })
    } catch (error) {
      setFeedback({ type: 'error', message: error?.message || 'Failed to update free-to-use quantity.' })
    } finally {
      setSubmittingAdjustmentKey('')
      setSubmittingAdjustmentAction('')
    }
  }

  const activeOrders = activeTab === 'receipts' ? receipts : deliveries
  const activeEmptyMessage =
    activeTab === 'receipts' && receiptQueryApplied
      ? 'No receipt orders found for this search.'
      : activeTab === 'receipts'
        ? 'No receipt orders created yet.'
        : 'No delivery orders created yet.'

  const createPath = activeTab === 'receipts' ? '/operations/receipts/create' : '/operations/delivery/create'
  const createLabel = activeTab === 'receipts' ? 'Create Receipt' : 'Create Delivery'
  const sectionTitle = activeTab === 'receipts' ? 'Receipt Orders' : 'Delivery Orders'
  const sectionDescription =
    activeTab === 'receipts'
      ? 'Incoming goods planned for warehouse receipt.'
      : 'Outgoing goods and shipment intent.'

  const kanbanColumns = useMemo(() => {
    const grouped = activeOrders.reduce((result, order) => {
      const key = resolveKanbanStatus(order)
      return {
        ...result,
        [key]: [...(result[key] || []), order],
      }
    }, {})

    return ORDER_KANBAN_COLUMNS.map((column) => ({
      ...column,
      orders: grouped[column.id] || [],
    })).filter((column) => column.id !== 'OTHER' || column.orders.length > 0)
  }, [activeOrders])

  if (isBootstrapping) {
    return (
      <section className="operations-shell">
        <div className="operations-status-card">
          {activeTab === 'adjustments' ? 'Loading adjustments module...' : 'Loading operations module...'}
        </div>
      </section>
    )
  }

  if (
    loadError &&
    ((activeTab === 'adjustments' && adjustmentsOverview.rows.length === 0 && adjustmentsOverview.history.length === 0) ||
      (activeTab !== 'adjustments' && receipts.length === 0 && deliveries.length === 0))
  ) {
    return (
      <section className="operations-shell">
        <div className="operations-status-card">
          <h1>Unable to load {activeTab === 'adjustments' ? 'adjustments' : 'operations'}</h1>
          <p>{loadError}</p>
          <button
            type="button"
            className="operations-btn primary"
            onClick={activeTab === 'adjustments' ? bootstrapAdjustments : () => bootstrapOrders(receiptQueryApplied)}
          >
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
          <h1 className="operations-header-title">Receipts, Delivery, and Adjustments</h1>
          <p className="operations-header-subtitle">
            Track incoming orders, outbound deliveries, and internal stock corrections in one place.
          </p>
        </div>
        <button type="button" className="operations-btn secondary" onClick={refreshData} disabled={refreshing}>
          {refreshing ? 'Refreshing...' : 'Refresh Data'}
        </button>
      </header>

      <div className="operations-tab-row" role="tablist" aria-label="Operations tabs">
        {OPERATIONS_TABS.map((tab) => (
          <Link
            key={tab.id}
            role="tab"
            to={tab.path}
            aria-selected={activeTab === tab.id}
            className={`operations-tab${activeTab === tab.id ? ' is-active' : ''}`}
          >
            <span className="operations-tab-label">{tab.label}</span>
            <span className="operations-tab-subtitle">{tab.subtitle}</span>
          </Link>
        ))}
      </div>

      {feedback.message ? (
        <p className={`operations-feedback ${feedback.type === 'error' ? 'is-error' : 'is-success'}`}>{feedback.message}</p>
      ) : null}

      {loadError ? <p className="operations-feedback is-error">{loadError}</p> : null}

      {activeTab === 'adjustments' ? (
        <article className="operations-card" role="tabpanel">
          <div className="operations-list-head">
            <div>
              <h2>Adjustments</h2>
              <p>Move stock across locations and correct free-to-use mismatches with a reason trail.</p>
            </div>
          </div>

          {adjustmentsOverview.rows.length === 0 ? (
            <p className="operations-empty">No stock rows available for internal transfers yet.</p>
          ) : (
            <div className="operations-table-wrap">
              <table className="operations-adjustments-table">
                <thead>
                  <tr>
                    <th>Product</th>
                    <th>Category</th>
                    <th>Location</th>
                    <th>
                      <span className="operations-th-with-help">
                        On Hand
                        <InfoHint text="On hand means quantity going to receive soon from receipt." />
                      </span>
                    </th>
                    <th>
                      <span className="operations-th-with-help">
                        Free To Use
                        <InfoHint text="Free to use means available in stock at warehouse." />
                      </span>
                    </th>
                    <th>Move To</th>
                    <th>Move Qty</th>
                    <th>Correct Free Qty</th>
                    <th>Reason</th>
                    <th>Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {adjustmentsOverview.rows.map((row) => {
                    const rowKey = buildAdjustmentRowKey(row)
                    const draft = {
                      ...createAdjustmentDraft(row),
                      ...(adjustmentDrafts[rowKey] || {}),
                    }
                    const isSubmitting = submittingAdjustmentKey === rowKey

                    return (
                      <tr key={rowKey}>
                        <td>
                          <div className="operations-products-cell">
                            <strong>{row.sku}</strong>
                            <span>{row.name}</span>
                          </div>
                        </td>
                        <td>{row.category_name}</td>
                        <td>
                          <div className="operations-location-cell">
                            <strong>
                              {row.location_name} ({row.location_short_code})
                            </strong>
                            <span>{(row.warehouse_names || []).join(', ') || '--'}</span>
                          </div>
                        </td>
                        <td>{row.on_hand_quantity}</td>
                        <td>{row.free_to_use_quantity}</td>
                        <td>
                          <select
                            value={draft.transferLocationId}
                            onChange={(event) => updateAdjustmentDraft(row, 'transferLocationId', event.target.value)}
                            disabled={isSubmitting}
                          >
                            <option value="">Select location</option>
                            {adjustmentsOverview.locations
                              .filter((entry) => entry.id !== row.location_id)
                              .map((entry) => (
                                <option key={entry.id} value={entry.id}>
                                  {entry.name} ({entry.short_code})
                                </option>
                              ))}
                          </select>
                        </td>
                        <td>
                          <input
                            type="number"
                            min="1"
                            step="1"
                            value={draft.transferQuantity}
                            onChange={(event) => updateAdjustmentDraft(row, 'transferQuantity', sanitizeWholeNumber(event.target.value))}
                            disabled={isSubmitting}
                            placeholder="0"
                          />
                        </td>
                        <td>
                          <input
                            type="number"
                            min="0"
                            step="1"
                            value={draft.correctedFreeToUse}
                            onChange={(event) => updateAdjustmentDraft(row, 'correctedFreeToUse', sanitizeWholeNumber(event.target.value))}
                            disabled={isSubmitting}
                            placeholder="0"
                          />
                        </td>
                        <td>
                          <input
                            value={draft.reason}
                            onChange={(event) => updateAdjustmentDraft(row, 'reason', event.target.value)}
                            disabled={isSubmitting}
                            placeholder="Why is this changing?"
                          />
                        </td>
                        <td>
                          <div className="operations-row-actions">
                            <button
                              type="button"
                              className="operations-btn small secondary"
                              disabled={isSubmitting}
                              onClick={() => submitTransfer(row)}
                            >
                              {isSubmitting && submittingAdjustmentAction === 'transfer' ? 'Moving...' : 'Move'}
                            </button>
                            <button
                              type="button"
                              className="operations-btn small primary"
                              disabled={isSubmitting}
                              onClick={() => submitQuantityCorrection(row)}
                            >
                              {isSubmitting && submittingAdjustmentAction === 'quantity' ? 'Saving...' : 'Correct Qty'}
                            </button>
                          </div>
                        </td>
                      </tr>
                    )
                  })}
                </tbody>
              </table>
            </div>
          )}

          <div className="operations-history-card">
            <div className="operations-list-head">
              <div>
                <h2>Internal Transfer History</h2>
                <p>Every internal move and free-to-use correction is logged with reason and timestamp.</p>
              </div>
            </div>

            {adjustmentsOverview.history.length === 0 ? (
              <p className="operations-empty">No adjustments recorded yet.</p>
            ) : (
              <div className="operations-table-wrap">
                <table>
                  <thead>
                    <tr>
                      <th>Time</th>
                      <th>Type</th>
                      <th>Product</th>
                      <th>From</th>
                      <th>To</th>
                      <th>Qty</th>
                      <th>Free Qty</th>
                      <th>Reason</th>
                    </tr>
                  </thead>
                  <tbody>
                    {adjustmentsOverview.history.map((entry) => (
                      <tr key={entry.id}>
                        <td>{formatDateTime(entry.created_at)}</td>
                        <td>{formatActionType(entry.action_type)}</td>
                        <td>
                          <div className="operations-products-cell">
                            <strong>{entry.sku}</strong>
                            <span>{entry.name}</span>
                          </div>
                        </td>
                        <td>{formatHistoryLocation(entry.from_location_name, entry.from_location_short_code)}</td>
                        <td>{formatHistoryLocation(entry.to_location_name, entry.to_location_short_code)}</td>
                        <td>{entry.quantity_changed}</td>
                        <td>
                          {entry.previous_free_to_use_quantity} -{'>'} {entry.updated_free_to_use_quantity}
                        </td>
                        <td>{entry.reason}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        </article>
      ) : (
        <article className="operations-card" role="tabpanel">
          <div className="operations-list-head">
            <div>
              <h2>{sectionTitle}</h2>
              <p>{sectionDescription}</p>
            </div>
            <Link to={createPath} state={{ from: currentPath }} className="operations-btn primary">
              {createLabel}
            </Link>
          </div>

          <div className="operations-controls-row">
            {activeTab === 'receipts' ? (
              <form className="operations-search-form" onSubmit={submitReceiptSearch}>
                <input
                  value={receiptQueryInput}
                  onChange={(event) => setReceiptQueryInput(event.target.value)}
                  placeholder="Search receipts by ref, party, contact, SKU"
                />
                <button type="submit" className="operations-btn small secondary" disabled={refreshing}>
                  Search
                </button>
                <button
                  type="button"
                  className="operations-btn small ghost"
                  onClick={clearReceiptSearch}
                  disabled={refreshing || (!receiptQueryInput && !receiptQueryApplied)}
                >
                  Clear
                </button>
              </form>
            ) : (
              <div />
            )}

            <div className="operations-view-toggle" aria-label="Order view selector">
              <button
                type="button"
                className={`operations-view-btn${viewMode === 'list' ? ' is-active' : ''}`}
                onClick={() => setViewMode('list')}
              >
                List View
              </button>
              <button
                type="button"
                className={`operations-view-btn${viewMode === 'kanban' ? ' is-active' : ''}`}
                onClick={() => setViewMode('kanban')}
              >
                Kanban View
              </button>
            </div>
          </div>

          {activeTab === 'receipts' && receiptQueryApplied ? (
            <p className="operations-search-meta">Showing receipt search results for: "{receiptQueryApplied}"</p>
          ) : null}

          {activeOrders.length === 0 ? (
            <p className="operations-empty">{activeEmptyMessage}</p>
          ) : viewMode === 'list' ? (
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
                  {activeOrders.map((order) => (
                    <tr key={order.id}>
                      <td>
                        <div className="operations-ref-cell">
                          <span className="operations-ref">{order.reference_number}</span>
                          <small>{String(order.operation_type || '').toUpperCase() === 'OUT' ? 'Delivery' : 'Receipt'}</small>
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
                            state={{ from: currentPath }}
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
          ) : (
            <div className="operations-kanban-board">
              {kanbanColumns.map((column) => (
                <section key={column.id} className="operations-kanban-column">
                  <header>
                    <h3>{column.label}</h3>
                    <span>{column.orders.length}</span>
                  </header>

                  <div className="operations-kanban-cards">
                    {column.orders.length === 0 ? (
                      <p className="operations-kanban-empty">No orders</p>
                    ) : (
                      column.orders.map((order) => (
                        <article key={order.id} className="operations-kanban-card">
                          <div className="operations-kanban-card-top">
                            <span className="operations-ref">{order.reference_number}</span>
                            <span className={`operations-status ${statusClassName(order.status)}`}>
                              {toTitleCase(order.status)}
                            </span>
                          </div>

                          <div className="operations-kanban-card-meta">
                            <p><strong>From:</strong> {order.from_party || '--'}</p>
                            <p><strong>To:</strong> {displayToParty(order)}</p>
                            <p><strong>Contact:</strong> {order.contact_number || '--'}</p>
                            <p><strong>Schedule:</strong> {formatDate(order.scheduled_date)}</p>
                          </div>

                          <div className="operations-row-actions">
                            <Link
                              to={buildOrderDetailPath(order.operation_type, order.reference_number)}
                              state={{ from: currentPath }}
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
                        </article>
                      ))
                    )}
                  </div>
                </section>
              ))}
            </div>
          )}
        </article>
      )}
    </section>
  )
}

function createAdjustmentDraft(row) {
  return {
    transferLocationId: '',
    transferQuantity: '',
    correctedFreeToUse: String(row.free_to_use_quantity ?? 0),
    reason: '',
  }
}

function buildAdjustmentRowKey(row) {
  return `${row.product_id}:${row.location_id}`
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

function formatDateTime(value) {
  if (!value) return '--'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '--'
  return date.toLocaleString('en-IN', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
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

function resolveKanbanStatus(order) {
  const status = String(order?.status || '').toUpperCase()
  if (status === 'DRAFT') return 'DRAFT'
  if (status === 'WAITING') return 'WAITING'
  if (status === 'READY') return 'READY'
  if (status === 'DONE') return 'DONE'
  if (status === 'CANCELLED') return 'CANCELLED'
  return 'OTHER'
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

function sanitizeWholeNumber(value) {
  return String(value || '').replace(/\D/g, '')
}

function formatActionType(value) {
  const normalized = String(value || '').trim().toUpperCase()
  if (normalized === 'QUANTITY_ADJUSTMENT') return 'Qty Correction'
  if (normalized === 'TRANSFER') return 'Transfer'
  return toTitleCase(normalized.replace(/_/g, ' '))
}

function formatHistoryLocation(name, shortCode) {
  if (!name && !shortCode) return '--'
  if (name && shortCode) return `${name} (${shortCode})`
  return name || shortCode
}

function InfoHint({ text }) {
  return (
    <span className="operations-info-hint" tabIndex={0} aria-label={text}>
      i
      <span className="operations-info-tooltip">{text}</span>
    </span>
  )
}

export default Operations
