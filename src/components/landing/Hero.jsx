import '../../styles/landing/hero.css'

function Hero({ onNavigate }) {
  return (
    <section id="hero" className="hero reveal-section">
      <div className="hero-content">
        <h1 className="hero-title">
          संचय
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
    </section>
  )
}

export default Hero
