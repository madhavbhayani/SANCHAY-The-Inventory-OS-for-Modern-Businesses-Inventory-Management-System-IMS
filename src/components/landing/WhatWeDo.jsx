import '../../styles/landing/whatwedo.css'

function BoxIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" aria-hidden="true">
      <path d="M4 8.5 12 4l8 4.5-8 4.5L4 8.5Z" />
      <path d="M4 8.5V16l8 4.5 8-4.5V8.5" />
      <path d="M12 13v7.5" />
    </svg>
  )
}

function CycleIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" aria-hidden="true">
      <path d="M6 9a6 6 0 0 1 10.2-4.2" />
      <path d="m16.2 3.8.3 3.4-3.4.3" />
      <path d="M18 15a6 6 0 0 1-10.2 4.2" />
      <path d="m7.8 20.2-.3-3.4 3.4-.3" />
    </svg>
  )
}

function BarcodeIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" aria-hidden="true">
      <path d="M5 6v12" />
      <path d="M8 6v12" />
      <path d="M11 6v12" />
      <path d="M14 6v12" />
      <path d="M17 6v12" />
      <path d="M19 8v8" />
      <path d="M4 18h16" />
    </svg>
  )
}

function BarsIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" aria-hidden="true">
      <path d="M5 18h14" />
      <path d="M7 18V10" />
      <path d="M12 18V7" />
      <path d="M17 18V4" />
    </svg>
  )
}

function ClipboardIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" aria-hidden="true">
      <path d="M8 5h8" />
      <path d="M9 3h6v4H9z" />
      <path d="M6 6h12v15H6z" />
      <path d="m9 14 2 2 4-4" />
    </svg>
  )
}

function UsersIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" aria-hidden="true">
      <path d="M9 11a3 3 0 1 0 0-6 3 3 0 0 0 0 6Z" />
      <path d="M16 12a2.5 2.5 0 1 0 0-5 2.5 2.5 0 0 0 0 5Z" />
      <path d="M4.5 19a4.5 4.5 0 0 1 9 0" />
      <path d="M13 19a3.5 3.5 0 0 1 7 0" />
    </svg>
  )
}

const features = [
  {
    title: 'Real-time Stock Tracking',
    description:
      'Every inbound and outbound movement updates instantly across all warehouse zones.',
    Icon: BoxIcon,
  },
  {
    title: 'Smart Transfers',
    description:
      'Initiate, approve, and track inter-warehouse transfers with full audit trail.',
    Icon: CycleIcon,
  },
  {
    title: 'Barcode & SKU Scanning',
    description:
      'Scan items during picking, shelving, and stock counts. Works offline too.',
    Icon: BarcodeIcon,
  },
  {
    title: 'Low Stock Alerts',
    description:
      'Automated alerts when items fall below reorder threshold. Never run out again.',
    Icon: BarsIcon,
  },
  {
    title: 'Stock Counting',
    description:
      'Schedule cycle counts or full audits. Compare physical vs system quantities in real time.',
    Icon: ClipboardIcon,
  },
  {
    title: 'Role-based Access',
    description:
      'Inventory Managers and Warehouse Staff each get a purpose-built interface.',
    Icon: UsersIcon,
  },
]

function WhatWeDo() {
  return (
    <section id="what-we-do" className="what-we-do reveal-section">
      <div className="what-we-do-header">
        <p className="section-label">WHAT WE DO</p>
        <h2>
          Every Stock Operation,
          <br />
          In One Place.
        </h2>
        <p>
          Sanchay replaces fragmented spreadsheets and manual counts with a single
          real-time source of truth for your warehouse.
        </p>
      </div>

      <div className="feature-grid">
        {features.map((feature) => (
          <article key={feature.title} className="feature-card">
            <div className="feature-icon-box">
              <feature.Icon />
            </div>
            <h3>{feature.title}</h3>
            <p>{feature.description}</p>
          </article>
        ))}
      </div>
    </section>
  )
}

export default WhatWeDo
