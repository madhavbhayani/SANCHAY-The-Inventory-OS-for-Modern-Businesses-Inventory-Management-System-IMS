import { useEffect, useState } from 'react'
import { HiOutlineBars3, HiOutlineXMark } from 'react-icons/hi2'
import SanchayLogo from './SanchayLogo'
import '../../styles/landing/navbar.css'

const navItems = [
  { label: 'What We Do', sectionId: 'what-we-do' },
  { label: 'Services', sectionId: 'services' },
]

function Navbar({ activeSection, onNavigate }) {
  const [isDrawerOpen, setIsDrawerOpen] = useState(false)

  useEffect(() => {
    if (!isDrawerOpen) {
      return undefined
    }

    const closeOnEscape = (event) => {
      if (event.key === 'Escape') {
        setIsDrawerOpen(false)
      }
    }

    document.addEventListener('keydown', closeOnEscape)

    return () => {
      document.removeEventListener('keydown', closeOnEscape)
    }
  }, [isDrawerOpen])

  const handleNavigate = (sectionId) => {
    onNavigate(sectionId)
    setIsDrawerOpen(false)
  }

  return (
    <header className="navbar">
      <div className="navbar-inner">
        <button
          type="button"
          className="navbar-brand"
          onClick={() => handleNavigate('hero')}
          aria-label="Go to hero section"
        >
          <SanchayLogo size={36} wordmarkColor="#1A3A2A" wordmarkSize={20} />
        </button>

        <nav className="navbar-links" aria-label="Primary navigation">
          {navItems.map((item) => (
            <button
              key={item.sectionId}
              type="button"
              className={`nav-link ${activeSection === item.sectionId ? 'is-active' : ''}`}
              onClick={() => handleNavigate(item.sectionId)}
            >
              {item.label}
            </button>
          ))}
        </nav>

        <div className="navbar-actions">
          <button type="button" className="signin-btn">
            Sign In
          </button>
          <button type="button" className="get-started-btn" onClick={() => handleNavigate('services')}>
            Get Started
          </button>
        </div>

        <button
          type="button"
          className="navbar-menu-toggle"
          aria-label={isDrawerOpen ? 'Close menu' : 'Open menu'}
          aria-expanded={isDrawerOpen}
          aria-controls="mobile-drawer"
          onClick={() => setIsDrawerOpen((current) => !current)}
        >
          {isDrawerOpen ? <HiOutlineXMark size={24} /> : <HiOutlineBars3 size={24} />}
        </button>
      </div>

      <button
        type="button"
        className={`navbar-overlay ${isDrawerOpen ? 'is-open' : ''}`}
        onClick={() => setIsDrawerOpen(false)}
        aria-label="Close mobile menu overlay"
      />

      <aside id="mobile-drawer" className={`navbar-drawer ${isDrawerOpen ? 'is-open' : ''}`}>
        <nav className="navbar-drawer-links" aria-label="Mobile navigation">
          {navItems.map((item) => (
            <button
              key={`drawer-${item.sectionId}`}
              type="button"
              className={`drawer-link ${activeSection === item.sectionId ? 'is-active' : ''}`}
              onClick={() => handleNavigate(item.sectionId)}
            >
              {item.label}
            </button>
          ))}
        </nav>

        <div className="navbar-drawer-actions">
          <button type="button" className="signin-btn">
            Sign In
          </button>
          <button type="button" className="get-started-btn" onClick={() => handleNavigate('services')}>
            Get Started
          </button>
        </div>
      </aside>
    </header>
  )
}

export default Navbar
