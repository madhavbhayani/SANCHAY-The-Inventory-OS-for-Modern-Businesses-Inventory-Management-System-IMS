import { useEffect, useState } from 'react'
import { apiListMoveHistory } from '../../../api/auth'
import '../../../styles/dashboard/movehistory.css'

const EVENT_OPTIONS = [
  { value: '', label: 'All Events' },
  { value: 'RECEIPT', label: 'Receipt' },
  { value: 'DELIVERY', label: 'Delivery' },
  { value: 'INTERNAL_TRANSFER', label: 'Internal Transfer' },
  { value: 'QUANTITY_ADJUSTMENT', label: 'Quantity Adjustment' },
]

const STATUS_OPTIONS = [
  { value: '', label: 'All Statuses' },
  { value: 'DRAFT', label: 'Draft' },
  { value: 'WAITING', label: 'Waiting' },
  { value: 'READY', label: 'Ready' },
  { value: 'DONE', label: 'Done' },
  { value: 'CANCELLED', label: 'Cancelled' },
]

function MoveHistory() {
  const [loading, setLoading] = useState(true)
  const [refreshing, setRefreshing] = useState(false)
  const [loadError, setLoadError] = useState('')
  const [entries, setEntries] = useState([])

  const [filters, setFilters] = useState({
    eventType: '',
    status: '',
    query: '',
    fromDate: '',
    toDate: '',
  })

  useEffect(() => {
    loadEntries({ bootstrap: true })
  }, [])

  const loadEntries = async ({ bootstrap = false } = {}) => {
    if (bootstrap) {
      setLoading(true)
    } else {
      setRefreshing(true)
    }
    setLoadError('')

    try {
      const response = await apiListMoveHistory({
        limit: 260,
        eventType: filters.eventType,
        status: filters.status,
        query: filters.query,
        fromDate: filters.fromDate,
        toDate: filters.toDate,
      })
      setEntries(response.entries || [])
    } catch (error) {
      setLoadError(error?.message || 'Failed to load move history data.')
    } finally {
      setLoading(false)
      setRefreshing(false)
    }
  }

  const setFilter = (name, value) => {
    setFilters((previous) => ({ ...previous, [name]: value }))
  }

  const applyFilters = async (event) => {
    event.preventDefault()
    await loadEntries()
  }

  const resetFilters = async () => {
    setFilters({ eventType: '', status: '', query: '', fromDate: '', toDate: '' })
    setTimeout(() => {
      loadEntries()
    }, 0)
  }

  if (loading) {
    return (
      <section className="movehistory-shell">
        <div className="movehistory-status-card">Loading move history...</div>
      </section>
    )
  }

  return (
    <section className="movehistory-shell">
      <header className="movehistory-header">
        <div>
          <p className="movehistory-label">Stock Ledger</p>
          <h1>Move History</h1>
          <p>Receipts, delivery, internal transfers, and quantity corrections with state transitions.</p>
        </div>
        <button type="button" className="movehistory-btn secondary" onClick={() => loadEntries()} disabled={refreshing}>
          {refreshing ? 'Refreshing...' : 'Refresh'}
        </button>
      </header>

      <form className="movehistory-filter-card" onSubmit={applyFilters}>
        <div className="movehistory-filter-grid">
          <label>
            Event
            <select value={filters.eventType} onChange={(event) => setFilter('eventType', event.target.value)}>
              {EVENT_OPTIONS.map((option) => (
                <option key={option.value} value={option.value}>{option.label}</option>
              ))}
            </select>
          </label>

          <label>
            Status
            <select value={filters.status} onChange={(event) => setFilter('status', event.target.value)}>
              {STATUS_OPTIONS.map((option) => (
                <option key={option.value} value={option.value}>{option.label}</option>
              ))}
            </select>
          </label>

          <label>
            Search
            <input
              value={filters.query}
              onChange={(event) => setFilter('query', event.target.value)}
              placeholder="Reference / SKU / Product / Reason"
            />
          </label>

          <label>
            From Date
            <input type="date" value={filters.fromDate} onChange={(event) => setFilter('fromDate', event.target.value)} />
          </label>

          <label>
            To Date
            <input type="date" value={filters.toDate} onChange={(event) => setFilter('toDate', event.target.value)} />
          </label>
        </div>

        <div className="movehistory-filter-actions">
          <button type="submit" className="movehistory-btn primary">Apply Filters</button>
          <button type="button" className="movehistory-btn ghost" onClick={resetFilters}>Reset</button>
        </div>
      </form>

      {loadError ? <p className="movehistory-feedback is-error">{loadError}</p> : null}

      <article className="movehistory-table-card">
        <div className="movehistory-table-head">
          <h2>Ledger Entries</h2>
          <span>{entries.length} records</span>
        </div>

        {entries.length === 0 ? (
          <p className="movehistory-empty">No ledger entries found for the selected filters.</p>
        ) : (
          <div className="movehistory-table-wrap">
            <table>
              <thead>
                <tr>
                  <th>Time</th>
                  <th>Event</th>
                  <th>Reference</th>
                  <th>Product</th>
                  <th>Route</th>
                  <th>Prev State</th>
                  <th>Current State</th>
                  <th>Status</th>
                  <th>Delta</th>
                  <th>Reason</th>
                </tr>
              </thead>
              <tbody>
                {entries.map((entry) => (
                  <tr key={entry.id}>
                    <td>{formatDateTime(entry.created_at)}</td>
                    <td>{toTitleCase(String(entry.event_type || '').replace(/_/g, ' '))}</td>
                    <td>{entry.reference_number || '--'}</td>
                    <td>
                      <div className="movehistory-product">
                        <strong>{entry.sku}</strong>
                        <span>{entry.product_name}</span>
                        <small>{entry.category_name}</small>
                      </div>
                    </td>
                    <td>{formatRoute(entry)}</td>
                    <td>On Hand {entry.previous_on_hand_quantity} | Free {entry.previous_free_to_use_quantity}</td>
                    <td>On Hand {entry.current_on_hand_quantity} | Free {entry.current_free_to_use_quantity}</td>
                    <td>
                      <span className="movehistory-status">
                        {entry.previous_status || '--'} → {entry.current_status || '--'}
                      </span>
                    </td>
                    <td>On {withSign(entry.on_hand_delta)} | Free {withSign(entry.free_to_use_delta)}</td>
                    <td>{entry.reason || '--'}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </article>
    </section>
  )
}

function withSign(value) {
  const number = Number(value || 0)
  if (number > 0) return `+${number}`
  return String(number)
}

function toTitleCase(value) {
  const normalized = String(value || '').trim().toLowerCase()
  if (!normalized) return '--'
  return normalized.charAt(0).toUpperCase() + normalized.slice(1)
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

function formatLocation(name, shortCode) {
  if (!name && !shortCode) return '--'
  if (name && shortCode) return `${name} (${shortCode})`
  return name || shortCode
}

function formatRoute(entry) {
  const from = formatLocation(entry.from_location_name, entry.from_location_short_code)
  const to = formatLocation(entry.to_location_name, entry.to_location_short_code)
  if (from === '--' && to === '--') {
    return formatLocation(entry.location_name, entry.location_short_code)
  }
  return `${from} -> ${to}`
}

export default MoveHistory
