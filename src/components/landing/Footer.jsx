import SanchayLogo from './SanchayLogo'
import '../../styles/landing/Footer.css'

function Footer() {
  return (
    <footer className="site-footer reveal-section">
      <div className="site-footer-main">
        <section>
          <SanchayLogo size={28} wordmarkColor="#F5F0E8" wordmarkSize={18} />
          <p className="site-footer-tagline">The Inventory OS for Modern Businesses.</p>
        </section>

        <section className="site-footer-contact">
          <p className="site-footer-contact-title">Hackathon Contact</p>
          <a
            className="site-footer-link"
            href="https://github.com/madhavbhayani/SANCHAY-The-Inventory-OS-for-Modern-Businesses-Inventory-Management-System-IMS.git"
            target="_blank"
            rel="noreferrer"
          >
            GitHub Repository
          </a>
          <a className="site-footer-link" href="mailto:madhavbhayani21@gmail.com">
            madhavbhayani21@gmail.com
          </a>
        </section>

        <section className="site-footer-stack">Built with GoLang · React · PostgreSQL</section>
      </div>

      <div className="site-footer-bottom">
        <p>© 2025 Sanchay IMS. Made in India.</p>
      </div>
    </footer>
  )
}

export default Footer
