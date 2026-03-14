import { useCallback, useEffect, useState } from 'react'
import { Navigate, Route, Routes } from 'react-router-dom'
import Login from './components/auth/Login'
import Signup from './components/auth/Signup'
import ForgotPassword from './components/auth/ForgotPassword'
import ConsoleLayout from './components/console/ConsoleLayout'
import Dashboard from './components/console/pages/Dashboard'
import MoveHistory from './components/console/pages/MoveHistory'
import OperationCreateOrder from './components/console/pages/OperationCreateOrder'
import OperationDetail from './components/console/pages/OperationDetail'
import Operations from './components/console/pages/Operations'
import Settings from './components/console/pages/Settings'
import Stock from './components/console/pages/Stock'
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
      <Route path="/forgot-password" element={<ForgotPassword />} />
      <Route path="/dashboard" element={<ConsoleLayout><Dashboard /></ConsoleLayout>} />
      <Route path="/operations" element={<Navigate to="/operations/receipts" replace />} />
      <Route path="/operations/receipts" element={<ConsoleLayout><Operations activeTab="receipts" /></ConsoleLayout>} />
      <Route path="/operations/delivery" element={<ConsoleLayout><Operations activeTab="delivery" /></ConsoleLayout>} />
      <Route path="/operations/adjustments" element={<ConsoleLayout><Operations activeTab="adjustments" /></ConsoleLayout>} />
      <Route path="/operations/receipts/create" element={<ConsoleLayout showBottomNav={false}><OperationCreateOrder mode="receipt" /></ConsoleLayout>} />
      <Route path="/operations/delivery/create" element={<ConsoleLayout showBottomNav={false}><OperationCreateOrder mode="delivery" /></ConsoleLayout>} />
      <Route path="/operations/:operationType/:referenceNumber" element={<ConsoleLayout showBottomNav={false}><OperationDetail /></ConsoleLayout>} />
      <Route path="/stock" element={<ConsoleLayout><Stock /></ConsoleLayout>} />
      <Route path="/move-history" element={<ConsoleLayout><MoveHistory /></ConsoleLayout>} />
      <Route path="/settings" element={<ConsoleLayout><Settings /></ConsoleLayout>} />
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  )
}

export default App
