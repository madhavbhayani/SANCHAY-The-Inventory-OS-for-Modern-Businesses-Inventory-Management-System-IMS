import { useEffect, useMemo, useState } from 'react'
import {
  apiChangePassword,
  apiCreateLocation,
  apiCreateWarehouse,
  apiGetSettingsOverview,
} from '../../../api/auth'
import '../../../styles/dashboard/settings.css'

const SETTINGS_TABS = [
  { id: 'operations', label: 'Warehouse & Location' },
  { id: 'user', label: 'User Settings' },
]

function Settings() {
  const [activeTab, setActiveTab] = useState('operations')
  const [loading, setLoading] = useState(true)
  const [refreshing, setRefreshing] = useState(false)
  const [loadError, setLoadError] = useState('')

  const [user, setUser] = useState(null)
  const [warehouses, setWarehouses] = useState([])
  const [locations, setLocations] = useState([])
  const [loginHistory, setLoginHistory] = useState([])

  const [warehouseForm, setWarehouseForm] = useState({
    name: '',
    shortCode: '',
    address: '',
    description: '',
  })
  const [locationForm, setLocationForm] = useState({
    name: '',
    shortCode: '',
    warehouseIds: [],
  })
  const [passwordForm, setPasswordForm] = useState({
    currentPassword: '',
    newPassword: '',
    confirmPassword: '',
  })

  const [warehouseFeedback, setWarehouseFeedback] = useState({ type: '', message: '' })
  const [locationFeedback, setLocationFeedback] = useState({ type: '', message: '' })
  const [passwordFeedback, setPasswordFeedback] = useState({ type: '', message: '' })

  const [submittingWarehouse, setSubmittingWarehouse] = useState(false)
  const [submittingLocation, setSubmittingLocation] = useState(false)
  const [submittingPassword, setSubmittingPassword] = useState(false)

  const warehouseNameById = useMemo(
    () => new Map(warehouses.map((warehouse) => [warehouse.id, warehouse.name])),
    [warehouses],
  )

  useEffect(() => {
    loadSettings()
  }, [])

  const loadSettings = async ({ silent = false } = {}) => {
    if (silent) {
      setRefreshing(true)
    } else {
      setLoading(true)
      setLoadError('')
    }

    try {
      const data = await apiGetSettingsOverview()
      setUser(data.user || null)
      setWarehouses(data.warehouses || [])
      setLocations(data.locations || [])
      setLoginHistory(data.login_history || [])
      setLoadError('')
    } catch (error) {
      setLoadError(error?.message || 'Failed to load settings data')
    } finally {
      if (silent) {
        setRefreshing(false)
      } else {
        setLoading(false)
      }
    }
  }

  const onWarehouseInput = (event) => {
    const { name, value } = event.target
    setWarehouseForm((previous) => ({ ...previous, [name]: value }))
  }

  const onLocationInput = (event) => {
    const { name, value } = event.target
    setLocationForm((previous) => ({ ...previous, [name]: value }))
  }

  const toggleLocationWarehouse = (warehouseId) => {
    setLocationForm((previous) => {
      const selected = previous.warehouseIds.includes(warehouseId)
      return {
        ...previous,
        warehouseIds: selected
          ? previous.warehouseIds.filter((id) => id !== warehouseId)
          : [...previous.warehouseIds, warehouseId],
      }
    })
  }

  const onPasswordInput = (event) => {
    const { name, value } = event.target
    setPasswordForm((previous) => ({ ...previous, [name]: value }))
  }

  const submitWarehouse = async (event) => {
    event.preventDefault()
    setWarehouseFeedback({ type: '', message: '' })

    if (!warehouseForm.name.trim() || !warehouseForm.shortCode.trim()) {
      setWarehouseFeedback({
        type: 'error',
        message: 'Warehouse name and short code are required.',
      })
      return
    }

    setSubmittingWarehouse(true)
    try {
      const result = await apiCreateWarehouse({
        name: warehouseForm.name.trim(),
        shortCode: warehouseForm.shortCode.trim(),
        address: warehouseForm.address.trim(),
        description: warehouseForm.description.trim(),
      })

      if (result?.warehouse) {
        setWarehouses((previous) => [result.warehouse, ...previous])
      }

      setWarehouseForm({ name: '', shortCode: '', address: '', description: '' })
      setWarehouseFeedback({ type: 'success', message: 'Warehouse added successfully.' })
    } catch (error) {
      setWarehouseFeedback({
        type: 'error',
        message: error?.message || 'Could not create warehouse.',
      })
    } finally {
      setSubmittingWarehouse(false)
    }
  }

  const submitLocation = async (event) => {
    event.preventDefault()
    setLocationFeedback({ type: '', message: '' })

    if (!locationForm.name.trim() || !locationForm.shortCode.trim()) {
      setLocationFeedback({ type: 'error', message: 'Location name and short code are required.' })
      return
    }

    if (locationForm.warehouseIds.length === 0) {
      setLocationFeedback({
        type: 'error',
        message: 'Please select at least one warehouse for this location.',
      })
      return
    }

    setSubmittingLocation(true)
    try {
      await apiCreateLocation({
        name: locationForm.name.trim(),
        shortCode: locationForm.shortCode.trim(),
        warehouseIds: locationForm.warehouseIds,
      })

      setLocationForm({ name: '', shortCode: '', warehouseIds: [] })
      setLocationFeedback({ type: 'success', message: 'Location added successfully.' })
      await loadSettings({ silent: true })
    } catch (error) {
      setLocationFeedback({ type: 'error', message: error?.message || 'Could not create location.' })
    } finally {
      setSubmittingLocation(false)
    }
  }

  const submitPassword = async (event) => {
    event.preventDefault()
    setPasswordFeedback({ type: '', message: '' })

    if (!passwordForm.currentPassword || !passwordForm.newPassword || !passwordForm.confirmPassword) {
      setPasswordFeedback({ type: 'error', message: 'All password fields are required.' })
      return
    }

    if (passwordForm.newPassword.length < 8) {
      setPasswordFeedback({ type: 'error', message: 'New password must be at least 8 characters.' })
      return
    }

    if (passwordForm.newPassword !== passwordForm.confirmPassword) {
      setPasswordFeedback({ type: 'error', message: 'New password and confirm password must match.' })
      return
    }

    setSubmittingPassword(true)
    try {
      await apiChangePassword({
        currentPassword: passwordForm.currentPassword,
        newPassword: passwordForm.newPassword,
      })

      setPasswordForm({ currentPassword: '', newPassword: '', confirmPassword: '' })
      setPasswordFeedback({ type: 'success', message: 'Password updated successfully.' })
    } catch (error) {
      setPasswordFeedback({ type: 'error', message: error?.message || 'Could not update password.' })
    } finally {
      setSubmittingPassword(false)
    }
  }

  if (loading) {
    return (
      <section className="settings-shell">
        <div className="settings-loading-card">Loading settings...</div>
      </section>
    )
  }

  if (loadError) {
    return (
      <section className="settings-shell">
        <div className="settings-error-card">
          <h1>Unable to load settings</h1>
          <p>{loadError}</p>
          <button type="button" className="settings-primary-btn" onClick={() => loadSettings()}>
            Retry
          </button>
        </div>
      </section>
    )
  }

  return (
    <section className="settings-shell">
      <header className="settings-header">
        <div>
          <p className="settings-header-label">Platform Settings</p>
          <h1 className="settings-header-title">Warehouse, Location, and User Controls</h1>
          <p className="settings-header-subtitle">
            Keep navigation simple with focused top tabs for operational setup and account security.
          </p>
        </div>
        <button
          type="button"
          className="settings-secondary-btn"
          onClick={() => loadSettings({ silent: true })}
          disabled={refreshing}
        >
          {refreshing ? 'Refreshing...' : 'Refresh Data'}
        </button>
      </header>

      <div className="settings-tab-row" role="tablist" aria-label="Settings tabs">
        {SETTINGS_TABS.map((tab) => (
          <button
            key={tab.id}
            type="button"
            role="tab"
            aria-selected={activeTab === tab.id}
            className={`settings-tab${activeTab === tab.id ? ' is-active' : ''}`}
            onClick={() => setActiveTab(tab.id)}
          >
            {tab.label}
          </button>
        ))}
      </div>

      {activeTab === 'operations' ? (
        <div className="settings-content" role="tabpanel">
          <div className="settings-grid two-col">
            <article className="settings-card">
              <h2>Add Warehouse</h2>
              <p>Register a warehouse before assigning locations.</p>

              <form className="settings-form" onSubmit={submitWarehouse}>
                <label>
                  Warehouse Name
                  <input
                    name="name"
                    value={warehouseForm.name}
                    onChange={onWarehouseInput}
                    placeholder="Main Distribution Center"
                  />
                </label>

                <label>
                  Short Code
                  <input
                    name="shortCode"
                    value={warehouseForm.shortCode}
                    onChange={onWarehouseInput}
                    placeholder="MDC01"
                  />
                </label>

                <label>
                  Address
                  <textarea
                    name="address"
                    value={warehouseForm.address}
                    onChange={onWarehouseInput}
                    rows={2}
                    placeholder="Plot 18, Industrial Avenue, Pune"
                  />
                </label>

                <label>
                  Description
                  <textarea
                    name="description"
                    value={warehouseForm.description}
                    onChange={onWarehouseInput}
                    rows={2}
                    placeholder="Primary finished goods warehouse"
                  />
                </label>

                <button type="submit" className="settings-primary-btn" disabled={submittingWarehouse}>
                  {submittingWarehouse ? 'Saving...' : 'Add Warehouse'}
                </button>

                {warehouseFeedback.message ? (
                  <p className={`settings-feedback ${warehouseFeedback.type === 'error' ? 'is-error' : 'is-success'}`}>
                    {warehouseFeedback.message}
                  </p>
                ) : null}
              </form>
            </article>

            <article className="settings-card">
              <h2>Add Location</h2>
              <p>Attach one location to multiple warehouses, rooms, or zones.</p>

              <form className="settings-form" onSubmit={submitLocation}>
                <label>
                  Location Name
                  <input
                    name="name"
                    value={locationForm.name}
                    onChange={onLocationInput}
                    placeholder="Room A - Packing"
                  />
                </label>

                <label>
                  Short Code
                  <input
                    name="shortCode"
                    value={locationForm.shortCode}
                    onChange={onLocationInput}
                    placeholder="PK-A"
                  />
                </label>

                <div>
                  <p className="settings-field-title">Warehouses (multi-select)</p>
                  {warehouses.length === 0 ? (
                    <p className="settings-muted">No warehouses yet. Add a warehouse first.</p>
                  ) : (
                    <div className="warehouse-chip-grid">
                      {warehouses.map((warehouse) => {
                        const selected = locationForm.warehouseIds.includes(warehouse.id)
                        return (
                          <label
                            key={warehouse.id}
                            className={`warehouse-chip${selected ? ' is-selected' : ''}`}
                          >
                            <input
                              type="checkbox"
                              checked={selected}
                              onChange={() => toggleLocationWarehouse(warehouse.id)}
                            />
                            <span>{warehouse.name}</span>
                            <small>{warehouse.short_code}</small>
                          </label>
                        )
                      })}
                    </div>
                  )}
                </div>

                <button
                  type="submit"
                  className="settings-primary-btn"
                  disabled={submittingLocation || warehouses.length === 0}
                >
                  {submittingLocation ? 'Saving...' : 'Add Location'}
                </button>

                {locationFeedback.message ? (
                  <p className={`settings-feedback ${locationFeedback.type === 'error' ? 'is-error' : 'is-success'}`}>
                    {locationFeedback.message}
                  </p>
                ) : null}
              </form>
            </article>
          </div>

          <div className="settings-grid two-col list-grid">
            <article className="settings-card compact">
              <h3>Existing Warehouses</h3>
              {warehouses.length === 0 ? (
                <p className="settings-muted">No warehouses added yet.</p>
              ) : (
                <ul className="entity-list">
                  {warehouses.map((warehouse) => (
                    <li key={warehouse.id}>
                      <div>
                        <strong>{warehouse.name}</strong>
                        <p>{warehouse.address || 'No address provided'}</p>
                      </div>
                      <span className="entity-code">{warehouse.short_code}</span>
                    </li>
                  ))}
                </ul>
              )}
            </article>

            <article className="settings-card compact">
              <h3>Existing Locations</h3>
              {locations.length === 0 ? (
                <p className="settings-muted">No locations added yet.</p>
              ) : (
                <ul className="entity-list">
                  {locations.map((location) => {
                    const resolvedWarehouses = location.warehouse_names?.length
                      ? location.warehouse_names
                      : (location.warehouse_ids || [])
                        .map((warehouseId) => warehouseNameById.get(warehouseId) || warehouseId)
                    return (
                      <li key={location.id}>
                        <div>
                          <strong>{location.name}</strong>
                          <p>{resolvedWarehouses.join(', ')}</p>
                        </div>
                        <span className="entity-code">{location.short_code}</span>
                      </li>
                    )
                  })}
                </ul>
              )}
            </article>
          </div>
        </div>
      ) : (
        <div className="settings-content" role="tabpanel">
          <div className="settings-grid two-col">
            <article className="settings-card">
              <h2>User Profile</h2>
              <p>Core account identifiers are shown below.</p>

              <div className="readonly-stack">
                <div className="readonly-field">
                  <span>User ID</span>
                  <code>{user?.id || '--'}</code>
                </div>
                <div className="readonly-field">
                  <span>Login ID</span>
                  <strong>{user?.login_id || '--'}</strong>
                </div>
                <div className="readonly-field">
                  <span>Email</span>
                  <strong>{user?.email || '--'}</strong>
                </div>
              </div>
            </article>

            <article className="settings-card">
              <h2>Change Password</h2>
              <p>Use a strong password with at least 8 characters.</p>

              <form className="settings-form" onSubmit={submitPassword}>
                <label>
                  Current Password
                  <input
                    name="currentPassword"
                    type="password"
                    value={passwordForm.currentPassword}
                    onChange={onPasswordInput}
                    autoComplete="current-password"
                  />
                </label>

                <label>
                  New Password
                  <input
                    name="newPassword"
                    type="password"
                    value={passwordForm.newPassword}
                    onChange={onPasswordInput}
                    autoComplete="new-password"
                  />
                </label>

                <label>
                  Confirm Password
                  <input
                    name="confirmPassword"
                    type="password"
                    value={passwordForm.confirmPassword}
                    onChange={onPasswordInput}
                    autoComplete="new-password"
                  />
                </label>

                <button type="submit" className="settings-primary-btn" disabled={submittingPassword}>
                  {submittingPassword ? 'Updating...' : 'Update Password'}
                </button>

                {passwordFeedback.message ? (
                  <p className={`settings-feedback ${passwordFeedback.type === 'error' ? 'is-error' : 'is-success'}`}>
                    {passwordFeedback.message}
                  </p>
                ) : null}
              </form>
            </article>
          </div>

          <article className="settings-card history-card">
            <h2>Login History</h2>
            <p>Recent access details for this account.</p>

            {loginHistory.length === 0 ? (
              <p className="settings-muted">No login history recorded yet.</p>
            ) : (
              <div className="history-table-wrap">
                <table>
                  <thead>
                    <tr>
                      <th>Status</th>
                      <th>Browser</th>
                      <th>OS</th>
                      <th>IP Address</th>
                      <th>Time</th>
                    </tr>
                  </thead>
                  <tbody>
                    {loginHistory.map((entry) => (
                      <tr key={entry.id}>
                        <td>
                          <span className={`history-status ${entry.success ? 'is-success' : 'is-error'}`}>
                            {entry.success ? 'Success' : 'Failed'}
                          </span>
                        </td>
                        <td>{entry.browser || 'Unknown'}</td>
                        <td>{entry.os || 'Unknown'}</td>
                        <td>{entry.ip_address || '--'}</td>
                        <td>{formatDateTime(entry.created_at)}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </article>
        </div>
      )}
    </section>
  )
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

export default Settings
