import '../../styles/landing/services.css'

const roleCards = [
  {
    key: 'manager',
    role: 'Inventory Manager',
    title: 'Full visibility.\nTotal control.',
    description:
      'Manage incoming and outgoing stock, approve transfers, set reorder thresholds, and generate audit-ready reports.',
    features: [
      'Dashboard with live metrics',
      'Approve / reject transfers',
      'Low stock & expiry alerts',
      'Reports & export to CSV',
      'Supplier & PO management',
    ],
  },
  {
    key: 'staff',
    role: 'Warehouse Staff',
    title: 'Pick. Shelve.\nCount. Done.',
    description:
      'A distraction-free interface for daily warehouse tasks - scan barcodes, complete picks, and record stock counts.',
    features: [
      'Barcode & QR scanning',
      'Picking & packing lists',
      'Bin / zone shelving guide',
      'Stock count sheets',
      'Transfer request & receive',
    ],
  },
]

function CheckIcon() {
  return (
    <svg viewBox="0 0 18 18" fill="none" aria-hidden="true">
      <circle cx="9" cy="9" r="8" stroke="#1E7B4B" strokeWidth="1.5" />
      <path d="m5.5 9.2 2.2 2.2 4.8-4.8" stroke="#1E7B4B" strokeWidth="1.8" />
    </svg>
  )
}

function Services({ onNavigate }) {
  return (
    <section id="services" className="services reveal-section">
      <div className="services-header">
        <p className="section-label section-label--light">SERVICES</p>
        <h2>
          Built for Every Role
          <br />
          in Your Warehouse.
        </h2>
      </div>

      <div className="role-grid">
        {roleCards.map((card) => (
          <article key={card.role} className={`role-card role-card--${card.key}`}>
            <p className={`role-badge role-badge--${card.key}`}>{card.role}</p>
            <h3>
              {card.title.split('\n').map((line) => (
                <span key={`${card.role}-${line}`}>{line}</span>
              ))}
            </h3>
            <p className="role-description">{card.description}</p>
            <ul className="role-feature-list">
              {card.features.map((feature) => (
                <li key={feature}>
                  <CheckIcon />
                  <span>{feature}</span>
                </li>
              ))}
            </ul>
          </article>
        ))}
      </div>

      <div className="services-cta-strip">
        <h3>Ready to digitize your godown?</h3>
        <p>Start free. No credit card. No Excel.</p>
        <button type="button" className="hero-primary-btn" onClick={() => onNavigate('hero')}>
          Get Started Free
        </button>
      </div>
    </section>
  )
}

export default Services
