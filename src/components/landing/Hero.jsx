import '../../styles/landing/hero.css'

const metrics = [
  { value: '1,248', label: 'Items in Stock', valueClass: 'metric-value--forest' },
  { value: '34', label: 'Pending Transfers', valueClass: 'metric-value--saffron' },
  { value: '98%', label: 'Accuracy Rate', valueClass: 'metric-value--success' },
]

const tableRows = [
  {
    sku: 'SKU-00421',
    itemName: 'Cement Bags 50kg',
    qty: '420',
    status: 'In Stock',
    statusClass: 'status-badge--stock',
  },
  {
    sku: 'SKU-00847',
    itemName: 'Steel Rods 12mm',
    qty: '38',
    status: 'Low Stock',
    statusClass: 'status-badge--low',
  },
  {
    sku: 'SKU-01102',
    itemName: 'Paint Drums 20L',
    qty: '156',
    status: 'In Stock',
    statusClass: 'status-badge--stock',
  },
  {
    sku: 'SKU-01340',
    itemName: 'PVC Pipes 4"',
    qty: '89',
    status: 'In Transit',
    statusClass: 'status-badge--transit',
  },
  {
    sku: 'SKU-01891',
    itemName: 'Bricks (Red Clay)',
    qty: '2400',
    status: 'In Stock',
    statusClass: 'status-badge--stock',
  },
]

function Hero({ onNavigate }) {
  return (
    <section id="hero" className="hero reveal-section">
      <div className="hero-content">
        <h1 className="hero-title">
          Sanchay -
          <br />
          <span>The Inventory OS for Modern Businesses</span>
        </h1>

        <p className="hero-sub-description">
          Digitizing India&apos;s Warehouses - Real-time. Centralized. Zero Guesswork.
        </p>

        <p className="hero-copy">
          Replace manual registers, Excel sheets, and scattered methods with one
          centralized, real-time platform built for Indian warehouses.
        </p>

        <div className="hero-cta-row">
          <button type="button" className="hero-primary-btn" onClick={() => onNavigate('services')}>
            Start Managing Stock
          </button>
          <button type="button" className="hero-secondary-btn" onClick={() => onNavigate('what-we-do')}>
            See How It Works
          </button>
        </div>
      </div>

      <div className="hero-dashboard-card" aria-label="Sanchay live dashboard mockup">
        <div className="dashboard-mini-topbar">
          <div className="dashboard-mini-dots" aria-hidden="true">
            <span className="dot-saffron" />
            <span className="dot-turmeric" />
            <span className="dot-jungle" />
          </div>
          <p>Sanchay - Live Dashboard</p>
        </div>

        <div className="dashboard-metrics-row">
          {metrics.map((metric) => (
            <article key={metric.label} className="dashboard-metric-card">
              <p className={`metric-value ${metric.valueClass}`}>{metric.value}</p>
              <p className="metric-label">{metric.label}</p>
            </article>
          ))}
        </div>

        <div className="dashboard-table-wrap">
          <table className="dashboard-table">
            <thead>
              <tr>
                <th>SKU</th>
                <th>Item Name</th>
                <th>Qty</th>
                <th>Status</th>
              </tr>
            </thead>
            <tbody>
              {tableRows.map((row) => (
                <tr key={row.sku}>
                  <td className="sku-cell">{row.sku}</td>
                  <td>{row.itemName}</td>
                  <td>{row.qty}</td>
                  <td>
                    <span className={`status-badge ${row.statusClass}`}>{row.status}</span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        <div className="dashboard-bottom-bar">
          <p>Last synced: just now</p>
          <span className="dashboard-live-pill">
            <span className="pulse-dot pulse-dot--small" aria-hidden="true" />
            Live
          </span>
        </div>
      </div>
    </section>
  )
}

export default Hero
