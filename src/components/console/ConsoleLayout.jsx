import { useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import BottomNav from './BottomNav'
import { getToken } from '../../api/auth'
import '../../styles/dashboard/console.css'

function ConsoleLayout({ children, showBottomNav = true }) {
  const navigate = useNavigate()

  useEffect(() => {
    if (!getToken()) {
      navigate('/login', { replace: true })
    }
  }, [navigate])

  if (!getToken()) return null

  return (
    <div className="console-shell">
      <main className={`console-page${showBottomNav ? '' : ' is-no-bottom-nav'}`}>{children}</main>
      {showBottomNav ? <BottomNav /> : null}
    </div>
  )
}

export default ConsoleLayout
