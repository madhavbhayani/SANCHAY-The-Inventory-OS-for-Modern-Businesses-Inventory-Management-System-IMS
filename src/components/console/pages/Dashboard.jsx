import { useEffect, useMemo, useState } from 'react'
import { apiGetDashboardOverview, getUser } from '../../../api/auth'
import '../../../styles/dashboard/dashboard.css'

function Dashboard() {
  const user = getUser()
  const [loading, setLoading] = useState(true)
  const [loadError, setLoadError] = useState('')
  const [overview, setOverview] = useState(null)

  useEffect(() => {
    loadOverview()
  }, [])

  const loadOverview = async () => {
    setLoading(true)
    setLoadError('')
    try {
      const data = await apiGetDashboardOverview()
      setOverview(data)
    } catch (error) {
      setLoadError(error?.message || 'Failed to load dashboard overview.')
    } finally {
      setLoading(false)
    }
  }

  const locationRows = overview?.stock_by_location || []
  const categoryRows = overview?.free_to_use_by_category || []

  const maxLocationValue = useMemo(
    () => locationRows.reduce((max, row) => Math.max(max, Number(row.free_to_use_quantity || 0)), 0),
    [locationRows],
  )

  const totalCategoryFreeToUse = useMemo(
    () => categoryRows.reduce((total, row) => total + Number(row.free_to_use_quantity || 0), 0),
    [categoryRows],
  )

  const pieSlices = useMemo(() => {
    if (categoryRows.length === 0 || totalCategoryFreeToUse <= 0) {
      return []
    }

    let cursor = 0
    return categoryRows.map((row, index) => {
      const value = Number(row.free_to_use_quantity || 0)
      const angle = (value / totalCategoryFreeToUse) * 360
      const start = cursor
      const end = cursor + angle
      cursor = end
      return {
        ...row,
        color: PIE_COLORS[index % PIE_COLORS.length],
        start,
        end,
        percent: Math.round((value / totalCategoryFreeToUse) * 1000) / 10,
      }
    })
  }, [categoryRows, totalCategoryFreeToUse])

  const pieGradient = useMemo(() => {
    if (pieSlices.length === 0) {
      return 'conic-gradient(#d6d2c4 0deg 360deg)'
    }
    return `conic-gradient(${pieSlices
      .map((slice) => `${slice.color} ${slice.start}deg ${slice.end}deg`)
      .join(', ')})`
  }, [pieSlices])

  if (loading) {
    return (
      <section className="dashboard-shell">
        <div className="dashboard-status-card">Loading dashboard...</div>
      </section>
    )
  }

  if (loadError) {
    return (
      <section className="dashboard-shell">
        <div className="dashboard-status-card is-error">
          <h1>Unable to load dashboard</h1>
          <p>{loadError}</p>
          <button type="button" className="dashboard-btn primary" onClick={loadOverview}>
            Retry
          </button>
        </div>
      </section>
    )
  }

  return (
    <section className="dashboard-shell">
      <header className="dashboard-header">
        <div>
          <p className="dashboard-header-label">Overview</p>
          <h1 className="dashboard-header-title">Welcome back, {user?.login_id || 'User'}</h1>
          <p className="dashboard-header-subtitle">
            Sanchay IMS analytics for receipts, delivery, and current stock posture.
          </p>
        </div>
        <button type="button" className="dashboard-btn secondary" onClick={loadOverview}>
          Refresh
        </button>
      </header>

      <section className="dashboard-stat-grid">
        <article className="dashboard-stat-card">
          <h2>Receipts</h2>
          <div className="dashboard-stat-list">
            <div><span>Current Receipt Orders</span><strong>{overview?.receipts?.current_orders ?? 0}</strong></div>
            <div><span>Receipt Ready</span><strong>{overview?.receipts?.ready ?? 0}</strong></div>
            <div><span>Receipt Done</span><strong>{overview?.receipts?.done ?? 0}</strong></div>
            <div><span>Late Receipts</span><strong>{overview?.receipts?.late ?? 0}</strong></div>
          </div>
        </article>

        <article className="dashboard-stat-card">
          <h2>Delivery</h2>
          <div className="dashboard-stat-list">
            <div><span>Current Delivery Orders</span><strong>{overview?.delivery?.current_orders ?? 0}</strong></div>
            <div><span>Delivery Ready</span><strong>{overview?.delivery?.ready ?? 0}</strong></div>
            <div><span>Delivery Done</span><strong>{overview?.delivery?.done ?? 0}</strong></div>
            <div><span>Late Delivery</span><strong>{overview?.delivery?.late ?? 0}</strong></div>
          </div>
        </article>
      </section>

      <section className="dashboard-chart-grid">
        <article className="dashboard-chart-card">
          <div className="dashboard-chart-head">
            <h3>Stock Availability by Location</h3>
            <p>Free-to-use and on-hand quantities across locations.</p>
          </div>

          {locationRows.length === 0 ? (
            <p className="dashboard-empty">No location stock data found.</p>
          ) : (
            <div className="dashboard-location-bars">
              {locationRows.map((row) => {
                const freeToUse = Number(row.free_to_use_quantity || 0)
                const onHand = Number(row.on_hand_quantity || 0)
                const freeWidth = maxLocationValue > 0 ? (freeToUse / maxLocationValue) * 100 : 0
                const onHandWidth = maxLocationValue > 0 ? (onHand / maxLocationValue) * 100 : 0

                return (
                  <div key={row.location_id} className="dashboard-location-row">
                    <div className="dashboard-location-label">
                      <strong>{row.location_name}</strong>
                      <span>{row.location_short_code}</span>
                    </div>
                    <div className="dashboard-location-bar-group">
                      <div className="dashboard-bar-track">
                        <span className="dashboard-bar-fill free" style={{ width: `${freeWidth}%` }} />
                      </div>
                      <div className="dashboard-bar-track subtle">
                        <span className="dashboard-bar-fill onhand" style={{ width: `${onHandWidth}%` }} />
                      </div>
                    </div>
                    <div className="dashboard-location-values">
                      <span>Free {freeToUse}</span>
                      <span>On Hand {onHand}</span>
                    </div>
                  </div>
                )
              })}
            </div>
          )}
        </article>

        <article className="dashboard-chart-card">
          <div className="dashboard-chart-head">
            <h3>Free-To-Use by Category</h3>
            <p>Distribution of currently available stock.</p>
          </div>

          {pieSlices.length === 0 ? (
            <p className="dashboard-empty">No category stock data found.</p>
          ) : (
            <div className="dashboard-pie-layout">
              <div className="dashboard-pie" style={{ backgroundImage: pieGradient }} />
              <div className="dashboard-pie-legend">
                {pieSlices.map((slice) => (
                  <div key={slice.category_id} className="dashboard-pie-legend-row">
                    <span className="dashboard-pie-dot" style={{ backgroundColor: slice.color }} />
                    <span>{slice.category_name}</span>
                    <strong>{slice.free_to_use_quantity} ({slice.percent}%)</strong>
                  </div>
                ))}
              </div>
            </div>
          )}
        </article>
      </section>
    </section>
  )
}

const PIE_COLORS = ['#1f7a50', '#e8652a', '#2f4d8f', '#8e5b2b', '#b03a2e', '#4a6f22']

export default Dashboard
