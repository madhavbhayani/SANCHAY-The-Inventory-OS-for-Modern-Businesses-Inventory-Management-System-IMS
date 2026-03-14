import { useCallback, useEffect, useState } from 'react'
import { Navigate, Route, Routes } from 'react-router-dom'
import Login from './components/auth/Login'
import Signup from './components/auth/Signup'
import Footer from './components/landing/Footer'
import Hero from './components/landing/Hero'
import Navbar from './components/landing/Navbar'
import Services from './components/landing/Services'
import WhatWeDo from './components/landing/WhatWeDo'

function LandingPage() {
  const [activeSection, setActiveSection] = useState('what-we-do')

  const handleNavigate = useCallback((sectionId) => {
    const section = document.getElementById(sectionId)

    if (section) {
      section.scrollIntoView({ behavior: 'smooth', block: 'start' })
    }

    if (sectionId === 'what-we-do' || sectionId === 'services') {
      setActiveSection(sectionId)
    }
  }, [])

  useEffect(() => {
    const revealTargets = document.querySelectorAll('.reveal-section')

    if (revealTargets.length === 0) {
      return undefined
    }

    const revealObserver = new IntersectionObserver(
      (entries) => {
        entries.forEach((entry) => {
          if (entry.isIntersecting) {
            entry.target.classList.add('is-visible')
          }
        })
      },
      { threshold: 0.2 },
    )

    revealTargets.forEach((target) => revealObserver.observe(target))

    return () => {
      revealObserver.disconnect()
    }
  }, [])

  useEffect(() => {
    const trackedSections = ['what-we-do', 'services']
      .map((sectionId) => document.getElementById(sectionId))
      .filter(Boolean)

    if (trackedSections.length === 0) {
      return undefined
    }

    const activeSectionObserver = new IntersectionObserver(
      (entries) => {
        const visibleEntries = entries
          .filter((entry) => entry.isIntersecting)
          .sort((left, right) => right.intersectionRatio - left.intersectionRatio)

        if (visibleEntries.length > 0) {
          setActiveSection(visibleEntries[0].target.id)
        }
      },
      {
        threshold: [0.2, 0.5, 0.8],
        rootMargin: '-20% 0px -55% 0px',
      },
    )

    trackedSections.forEach((section) => activeSectionObserver.observe(section))

    return () => {
      activeSectionObserver.disconnect()
    }
  }, [])

  return (
    <div className="app-shell">
      <Navbar activeSection={activeSection} onNavigate={handleNavigate} />
      <Hero onNavigate={handleNavigate} />
      <WhatWeDo />
      <Services onNavigate={handleNavigate} />
      <Footer />
    </div>
  )
}

function App() {
  return (
    <Routes>
      <Route path="/" element={<LandingPage />} />
      <Route path="/login" element={<Login />} />
      <Route path="/signup" element={<Signup />} />
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  )
}

export default App
